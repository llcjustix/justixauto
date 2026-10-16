package service

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

type returnTestStore struct {
	*saleTestStore
	returns                  map[string]model.ConsignmentReturn
	stock                    *returnTestStock
	failCreate, failDecision bool
}

func (s *returnTestStore) InTx(_ context.Context, fn func(Store) error) error {
	units, events, returns := maps.Clone(s.units), append([]model.Event(nil), s.events...), maps.Clone(s.returns)
	moves := s.stock.moves
	outer := s.depth == 0
	s.depth++
	err := fn(s)
	s.depth--
	if outer && err == nil && s.failOuterCommit {
		err = errors.New("outer commit failed")
	}
	if err != nil {
		s.units, s.events, s.returns, s.stock.moves = units, events, returns, moves
	}
	if outer {
		s.locks = map[string]bool{}
	}
	return err
}

func (s *returnTestStore) ConsignmentReturns() ConsignmentReturnRepository {
	return returnTestRepo{s: s}
}

type returnTestRepo struct {
	ConsignmentReturnRepository
	s *returnTestStore
}

func (r returnTestRepo) Create(_ context.Context, row *model.ConsignmentReturn) error {
	if r.s.failCreate {
		return errors.New("request write failed")
	}
	if !r.s.locks[row.UnitID] {
		return errors.New("unit was not locked before request")
	}
	for _, old := range r.s.returns {
		if old.UnitID == row.UnitID && old.Status == model.ConsignmentReturnPending {
			return apperr.ErrConflict
		}
	}
	r.s.returns[row.ID] = *row
	return nil
}

func (r returnTestRepo) Get(_ context.Context, id string) (*model.ConsignmentReturn, error) {
	row, ok := r.s.returns[id]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	return &row, nil
}

func (r returnTestRepo) List(_ context.Context, companyID, orderID string, limit, offset int) ([]model.ConsignmentReturn, error) {
	rows := []model.ConsignmentReturn{}
	for _, row := range r.s.returns {
		if (row.SupplierCompanyID == companyID || row.DealerCompanyID == companyID) && (orderID == "" || row.OrderID == orderID) {
			rows = append(rows, row)
		}
	}
	if offset >= len(rows) {
		return []model.ConsignmentReturn{}, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], nil
}

func (r returnTestRepo) Lock(_ context.Context, id string) (*model.ConsignmentReturn, error) {
	row, ok := r.s.returns[id]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	if !r.s.locks[row.UnitID] {
		return nil, errors.New("request locked before unit")
	}
	return &row, nil
}

func (r returnTestRepo) Decide(_ context.Context, row *model.ConsignmentReturn, expected int64) error {
	if r.s.failDecision {
		return errors.New("decision write failed")
	}
	old, ok := r.s.returns[row.ID]
	if !ok {
		return apperr.ErrNotFound
	}
	if old.Version != expected {
		return apperr.ErrStale
	}
	row.Version = expected + 1
	r.s.returns[row.ID] = *row
	return nil
}

type returnTestStock struct {
	Stock
	warehouse, supplier, dealer string
	fact                        string
	moves                       int
	last                        ReturnCustodyMove
	failWarehouse, failMove     bool
}

func (s *returnTestStock) OwnsWarehouse(ctx context.Context, companyID, id string) error {
	if ctx.Value(saleBoundKey{}) != true {
		return errors.New("warehouse check not bound")
	}
	if s.failWarehouse || companyID != s.supplier || id != s.warehouse {
		return apperr.ErrNotFound
	}
	return nil
}

func (s *returnTestStock) ReturnCustody(ctx context.Context, move ReturnCustodyMove) (string, error) {
	if ctx.Value(saleBoundKey{}) != true {
		return "", errors.New("custody move not bound")
	}
	if s.failMove {
		return "", errors.New("capacity or hold rejected")
	}
	if move.OwnerCompanyID != s.supplier || move.SourceCustodianCompanyID != s.dealer || move.DestinationWarehouseID != s.warehouse ||
		move.VehicleID == "" || move.RequestID == "" || move.AgreementID == "" || move.At.IsZero() {
		return "", errors.New("incorrect return movement")
	}
	s.last = move
	s.moves++
	return s.fact, nil
}

func returnFixture(t *testing.T) (*ConsignmentReturn, *returnTestStore, *auth.Principal, *auth.Principal, string) {
	t.Helper()
	setup := saleFixture(t)
	base, dealerID := setup.store, setup.dealerID
	u := saleOnlyUnit(base)
	stock := &returnTestStock{warehouse: uuid.NewString(), supplier: u.SupplierCompanyID, dealer: dealerID, fact: uuid.NewString()}
	st := &returnTestStore{saleTestStore: base, returns: map[string]model.ConsignmentReturn{}, stock: stock}
	clock := func() time.Time { return time.Date(2026, 10, 11, 2, 0, 0, 0, time.UTC) }
	s := NewConsignmentReturn(NewDeps(st, nil, nil, stock, nil, clock))
	supplier := &auth.Principal{UserID: uuid.NewString(), CompanyID: u.SupplierCompanyID, Permissions: map[string]bool{model.PermTrade: true, model.PermRead: true}}
	dealer := &auth.Principal{UserID: uuid.NewString(), CompanyID: dealerID, Permissions: map[string]bool{model.PermTrade: true, model.PermRead: true}}
	return s, st, supplier, dealer, u.VehicleID
}

func returnContext() context.Context {
	return context.WithValue(context.Background(), saleBoundKey{}, true)
}

func requestReturn(t *testing.T, s *ConsignmentReturn, st *returnTestStore, supplier *auth.Principal, vehicle string) *model.ConsignmentReturn {
	t.Helper()
	u := saleOnlyUnit(st.saleTestStore)
	row, revision, err := s.Request(returnContext(), supplier, vehicle, u.Version, ReturnRequestInput{DestinationWarehouseID: st.stock.warehouse, Reason: "unsold stock"})
	if err != nil || row == nil || revision != u.Version+1 {
		t.Fatalf("request=%+v revision=%d err=%v", row, revision, err)
	}
	return row
}

func TestConsignmentReturnRequestAndConfirmation(t *testing.T) {
	s, st, supplier, dealer, vehicle := returnFixture(t)
	other := saleOnlyUnit(st.saleTestStore)
	other.ID, other.VehicleID, other.VIN = uuid.NewString(), uuid.NewString(), "OTHER-VIN"
	st.units[other.ID] = other
	row := requestReturn(t, s, st, supplier, vehicle)
	for _, viewer := range []*auth.Principal{supplier, dealer} {
		listed, err := s.List(context.Background(), viewer, row.OrderID, 0, 0)
		if err != nil || len(listed) != 1 || listed[0].ID != row.ID {
			t.Fatalf("party list=%+v err=%v", listed, err)
		}
	}
	outsider := &auth.Principal{UserID: uuid.NewString(), CompanyID: uuid.NewString(), Permissions: map[string]bool{model.PermRead: true}}
	if listed, err := s.List(context.Background(), outsider, row.OrderID, 0, 0); err != nil || len(listed) != 0 {
		t.Fatalf("outsider list=%+v err=%v", listed, err)
	}
	if st.stock.moves != 0 || saleOnlyUnitByVehicle(st, vehicle).PendingReturnID == nil {
		t.Fatal("request moved stock or lost pending association")
	}
	if saleOnlyUnitByVehicle(st, other.VehicleID).PendingReturnID != nil {
		t.Fatal("other VIN was blocked")
	}
	if _, err := s.Get(context.Background(), &auth.Principal{UserID: uuid.NewString(), CompanyID: uuid.NewString(), Permissions: map[string]bool{model.PermRead: true}}, row.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("other company read: %v", err)
	}
	at := time.Date(2026, 10, 11, 1, 30, 0, 0, time.UTC)
	got, err := s.Decide(returnContext(), dealer, row.ID, row.Version, true, ReturnDecisionInput{MovedAt: at})
	if err != nil || got.Status != model.ConsignmentReturnConfirmed || got.MovementFactID == nil || *got.MovementFactID != st.stock.fact || got.MovementOperationRef == nil || *got.MovementOperationRef != row.ID || st.stock.moves != 1 {
		t.Fatalf("confirm=%+v moves=%d err=%v", got, st.stock.moves, err)
	}
	if st.stock.last.RequestID != row.ID || st.stock.last.ActorUserID != dealer.UserID {
		t.Fatalf("wrong physical reference: %+v", st.stock.last)
	}
	u := saleOnlyUnitByVehicle(st, vehicle)
	if u.Status != model.ConsignmentReturned || u.PendingReturnID != nil || u.ReturnFactID == nil || *u.ReturnFactID != st.stock.fact || st.order.ID != row.OrderID {
		t.Fatalf("terminal unit=%+v", u)
	}
	for _, viewer := range []*auth.Principal{supplier, dealer} {
		listed, err := s.List(context.Background(), viewer, row.OrderID, 0, 0)
		if err != nil || len(listed) != 1 || listed[0].Status != model.ConsignmentReturnConfirmed || listed[0].MovementFactID == nil {
			t.Fatalf("confirmed history=%+v err=%v", listed, err)
		}
		if wrong, err := s.List(context.Background(), viewer, uuid.NewString(), 0, 0); err != nil || len(wrong) != 0 {
			t.Fatalf("wrong order history=%+v err=%v", wrong, err)
		}
	}
	if _, err := s.Decide(returnContext(), dealer, row.ID, row.Version, true, ReturnDecisionInput{MovedAt: at}); err == nil || st.stock.moves != 1 {
		t.Fatalf("replay moved again: %v", err)
	}
}

func saleOnlyUnitByVehicle(st *returnTestStore, vehicle string) model.ConsignmentUnit {
	for _, u := range st.units {
		if u.VehicleID == vehicle {
			return u
		}
	}
	return model.ConsignmentUnit{}
}

func TestConsignmentReturnExclusionAndRejection(t *testing.T) {
	s, st, supplier, dealer, vehicle := returnFixture(t)
	u := saleOnlyUnit(st.saleTestStore)
	retail := uuid.NewString()
	u.CurrentRetailDealID = &retail
	st.units[u.ID] = u
	if _, _, err := s.Request(returnContext(), supplier, vehicle, u.Version, ReturnRequestInput{DestinationWarehouseID: st.stock.warehouse, Reason: "recall"}); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("retail held VIN: %v", err)
	}
	u.CurrentRetailDealID = nil
	st.units[u.ID] = u
	row := requestReturn(t, s, st, supplier, vehicle)
	if _, _, err := s.Request(returnContext(), supplier, vehicle, u.Version+1, ReturnRequestInput{DestinationWarehouseID: st.stock.warehouse, Reason: "repeat"}); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("repeat request: %v", err)
	}
	if _, err := s.Decide(returnContext(), supplier, row.ID, row.Version, false, ReturnDecisionInput{Reason: "dealer reason"}); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("supplier decision: %v", err)
	}
	rejected, err := s.Decide(returnContext(), dealer, row.ID, row.Version, false, ReturnDecisionInput{Reason: "still needed"})
	if err != nil || rejected.Status != model.ConsignmentReturnRejected || rejected.DecisionReason != "still needed" || st.stock.moves != 0 || saleOnlyUnitByVehicle(st, vehicle).PendingReturnID != nil {
		t.Fatalf("reject=%+v err=%v", rejected, err)
	}
	listed, err := s.List(context.Background(), supplier, row.OrderID, 0, 0)
	if err != nil || len(listed) != 1 || listed[0].Status != model.ConsignmentReturnRejected || listed[0].DecisionReason != "still needed" {
		t.Fatalf("rejected history=%+v err=%v", listed, err)
	}
	if _, _, err := s.Request(returnContext(), supplier, vehicle, u.Version+2, ReturnRequestInput{DestinationWarehouseID: st.stock.warehouse, Reason: "new request"}); err != nil {
		t.Fatalf("re-request after rejection: %v", err)
	}
}

func TestConsignmentReturnRollbackAndStale(t *testing.T) {
	for _, failure := range []string{"warehouse", "custody", "decision", "unit", "event", "outer"} {
		t.Run(failure, func(t *testing.T) {
			s, st, supplier, dealer, vehicle := returnFixture(t)
			row := requestReturn(t, s, st, supplier, vehicle)
			switch failure {
			case "warehouse":
				st.stock.failWarehouse = true
			case "custody":
				st.stock.failMove = true
			case "decision":
				st.failDecision = true
			case "unit":
				st.failUpdate = true
			case "event":
				st.failEvent = true
			case "outer":
				st.failOuterCommit = true
			}
			at := time.Date(2026, 10, 11, 1, 30, 0, 0, time.UTC)
			_, err := s.Decide(returnContext(), dealer, row.ID, row.Version, true, ReturnDecisionInput{MovedAt: at})
			if err == nil {
				t.Fatal("expected failure")
			}
			u := saleOnlyUnitByVehicle(st, vehicle)
			if u.Status != model.ConsignmentHeld || u.PendingReturnID == nil || st.returns[row.ID].Status != model.ConsignmentReturnPending || st.stock.moves != 0 {
				t.Fatalf("partial result after %s: unit=%+v return=%+v moves=%d", failure, u, st.returns[row.ID], st.stock.moves)
			}
		})
	}
	s, st, supplier, dealer, vehicle := returnFixture(t)
	row := requestReturn(t, s, st, supplier, vehicle)
	if _, err := s.Decide(returnContext(), dealer, row.ID, row.Version+1, true, ReturnDecisionInput{MovedAt: time.Date(2026, 10, 11, 1, 0, 0, 0, time.UTC)}); !errors.Is(err, apperr.ErrStale) || st.stock.moves != 0 {
		t.Fatalf("stale decision=%v moves=%d", err, st.stock.moves)
	}
}
