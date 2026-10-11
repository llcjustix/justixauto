package service

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
)

type saleBoundKey struct{}

// The fake owns state and transaction snapshots. Its return decision uses the
// same locked unit state as the production sale service, without copying the
// production sale predicate into the repository fake.
type saleTestStore struct {
	Store
	agreement model.ConsignmentAgreement
	order model.Order
	units map[string]model.ConsignmentUnit
	events []model.Event
	depth int
	locks map[string]bool
	failRead, failAgreement, failUpdate, failEvent, failOuterCommit bool
}

func (s *saleTestStore) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, saleBoundKey{}, true)
}

func (s *saleTestStore) InTx(ctx context.Context, fn func(Store) error) error {
	if s.depth > 0 && ctx.Value(saleBoundKey{}) != true { return errors.New("nested commerce call lost outer transaction") }
	units, events := maps.Clone(s.units), append([]model.Event(nil), s.events...)
	outer := s.depth == 0
	s.depth++
	err := fn(s)
	s.depth--
	if outer && err == nil && s.failOuterCommit { err = errors.New("outer transaction failed") }
	if err != nil { s.units, s.events = units, events }
	if outer { s.locks = map[string]bool{} }
	return err
}

func (s *saleTestStore) Consignments() ConsignmentRepository { return saleTestUnits{s:s} }
func (s *saleTestStore) Deals() DealRepository { return saleTestDeals{s:s} }
func (s *saleTestStore) Events() EventRepository { return saleTestEvents{s:s} }

type saleTestUnits struct { ConsignmentRepository; s *saleTestStore }
func (r saleTestUnits) LockUnitByVehicle(ctx context.Context, vehicleID string) (*model.ConsignmentUnit, error) {
	if r.s.depth == 0 || ctx.Value(saleBoundKey{}) != true { return nil, errors.New("VIN lookup did not join bound outer transaction") }
	if r.s.failRead { return nil, errors.New("unit read failed") }
	for _, u := range r.s.units {
		if u.VehicleID == vehicleID && (u.Status == model.ConsignmentPrepared || u.Status == model.ConsignmentHeld) {
			r.s.locks[u.ID] = true
			return &u, nil
		}
	}
	return nil, apperr.ErrNotFound
}
func (r saleTestUnits) Agreement(_ context.Context, id string) (*model.ConsignmentAgreement, error) {
	if r.s.failAgreement { return nil, errors.New("agreement read failed") }
	if r.s.agreement.ID != id { return nil, apperr.ErrNotFound }
	a := r.s.agreement
	return &a, nil
}
func (r saleTestUnits) UpdateUnit(ctx context.Context, u *model.ConsignmentUnit, expected int64) error {
	if ctx.Value(saleBoundKey{}) != true || !r.s.locks[u.ID] { return errors.New("unlocked unit update") }
	if r.s.failUpdate { return apperr.ErrStale }
	old, ok := r.s.units[u.ID]
	if !ok { return apperr.ErrNotFound }
	if old.Version != expected { return apperr.ErrStale }
	u.Version = expected + 1
	r.s.units[u.ID] = *u
	return nil
}

type saleTestDeals struct { DealRepository; s *saleTestStore }
func (r saleTestDeals) Order(_ context.Context, companyID, id string) (*model.Order, error) {
	if r.s.order.ID != id || r.s.order.SupplierCompanyID != companyID { return nil, apperr.ErrNotFound }
	o := r.s.order
	return &o, nil
}

type saleTestEvents struct { EventRepository; s *saleTestStore }
func (r saleTestEvents) Append(_ context.Context, e *model.Event) error {
	if r.s.failEvent { return errors.New("event write failed") }
	r.s.events = append(r.s.events, *e)
	return nil
}

func saleFixture(t *testing.T) (*Consignment, *saleTestStore, string, string, string, string) {
	t.Helper()
	supplier, dealer, outsider, vehicle, deal, actor := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	agreementID, orderID, reservationID, orderLineID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	terms, err := json.Marshal(model.Terms{Reservation:&model.ReservationOrderTerms{
		ReservationID:reservationID, BookingDigest:"booking", ContinuationDigest:"continuation", ContinuationRevision:2, Mode:"consignment",
	}, Lines:[]model.Line{{LineID:orderLineID}}, PaymentSchedule:[]model.Installment{{ID:uuid.NewString(), DueDate:"2026-12-15"}}})
	if err != nil { t.Fatal(err) }
	at := time.Date(2026, 10, 11, 1, 0, 0, 0, time.UTC)
	fact, operation := uuid.NewString(), uuid.NewString()
	u := model.ConsignmentUnit{ID:uuid.NewString(), AgreementID:agreementID, OrderID:orderID,
		OrderLineID:orderLineID, VehicleID:vehicle, VIN:"VIN-123", SupplierCompanyID:supplier, DealerCompanyID:dealer,
		Status:model.ConsignmentHeld, HandoverFactID:&fact, HandoverOperationRef:&operation,
		HandedOverAt:&at, HandedOverBy:&actor, Version:3}
	st := &saleTestStore{agreement:model.ConsignmentAgreement{ID:agreementID, OrderID:orderID,
		ReservationID:reservationID, SupplierCompanyID:supplier, DealerCompanyID:dealer,
		BookingDigest:"booking", ContinuationDigest:"continuation", ContinuationRevision:2},
		order:model.Order{ID:orderID, BuyerCompanyID:dealer, SupplierCompanyID:supplier, Source:"reservation", Terms:terms},
		units:map[string]model.ConsignmentUnit{u.ID:u}, locks:map[string]bool{}}
	return NewConsignment(NewDeps(st,nil,nil,nil,nil,func() time.Time{return at})), st, dealer, outsider, deal, actor
}

func saleOuter(st *saleTestStore, fn func(context.Context) error) error {
	ctx := context.Background()
	return st.InTx(ctx, func(Store) error { return fn(st.Bind(ctx)) })
}

func saleOnlyUnit(st *saleTestStore) model.ConsignmentUnit {
	for _, u := range st.units { return u }
	return model.ConsignmentUnit{}
}

func TestConsignmentSaleAuthorityAbsenceAndDenial(t *testing.T) {
	c, st, dealer, outsider, _, _ := saleFixture(t)
	u := saleOnlyUnit(st)
	if err := saleOuter(st, func(ctx context.Context) error {
		available, err := c.CheckSale(ctx, dealer, uuid.NewString())
		if err != nil || available != nil { t.Fatalf("ordinary VIN: %+v, %v", available, err) }
		_, err = c.CheckSale(ctx, u.SupplierCompanyID, u.VehicleID)
		if !errors.Is(err, apperr.ErrForbidden) { t.Fatalf("supplier bypass: %v", err) }
		_, err = c.CheckSale(ctx, outsider, u.VehicleID)
		if !errors.Is(err, apperr.ErrForbidden) { t.Fatalf("other dealer: %v", err) }
		available, err = c.CheckSale(ctx, dealer, u.VehicleID)
		if err != nil || available == nil || available.OrderID != u.OrderID || available.VIN != u.VIN || available.HandoverFactID != *u.HandoverFactID {
			t.Fatalf("actual dealer authority: %+v, %v", available, err)
		}
		return nil
	}); err != nil { t.Fatal(err) }
	for _, status := range []model.ConsignmentUnitStatus{model.ConsignmentPrepared, model.ConsignmentSold, model.ConsignmentReturned} {
		row := u; row.Status = status; st.units[row.ID] = row
		err := saleOuter(st, func(ctx context.Context) error {
			got, err := c.CheckSale(ctx, dealer, row.VehicleID)
			if status == model.ConsignmentPrepared {
				if !errors.Is(err, apperr.ErrForbidden) { t.Fatalf("prepared unit: %v", err) }
			} else if err != nil || got != nil { t.Fatalf("terminal row exposed: %+v %v", got, err) }
			return nil
		})
		if err != nil { t.Fatal(err) }
	}
	st.units[u.ID] = u
	ret := uuid.NewString(); u.PendingReturnID = &ret; st.units[u.ID] = u
	if err := saleOuter(st, func(ctx context.Context) error {
		_, err := c.CheckSale(ctx, dealer, u.VehicleID)
		if !errors.Is(err, apperr.ErrForbidden) { t.Fatalf("pending return: %v", err) }
		return nil
	}); err != nil { t.Fatal(err) }
	u.PendingReturnID = nil; st.units[u.ID] = u
	st.agreement.DealerCompanyID = outsider
	if err := saleOuter(st, func(ctx context.Context) error {
		_, err := c.CheckSale(ctx, dealer, u.VehicleID)
		if !errors.Is(err, apperr.ErrConflict) { t.Fatalf("wrong agreement party: %v", err) }
		return nil
	}); err != nil { t.Fatal(err) }
	st.agreement.DealerCompanyID = dealer
	u.OrderLineID = uuid.NewString(); st.units[u.ID] = u
	if err := saleOuter(st, func(ctx context.Context) error {
		_, err := c.CheckSale(ctx, dealer, u.VehicleID)
		if !errors.Is(err, apperr.ErrConflict) { t.Fatalf("wrong accepted line: %v", err) }
		return nil
	}); err != nil { t.Fatal(err) }
	u = saleOnlyUnit(st); u.OrderLineID = st.order.DecodeTerms().Lines[0].LineID; st.units[u.ID] = u
	st.failAgreement = true
	if err := saleOuter(st, func(ctx context.Context) error {
		_, err := c.CheckSale(ctx, dealer, u.VehicleID)
		if err == nil { t.Fatal("agreement read failure was ignored") }
		return nil
	}); err != nil { t.Fatal(err) }
	st.failAgreement = false
	st.failRead = true
	if err := saleOuter(st, func(ctx context.Context) error {
		_, err := c.CheckSale(ctx, dealer, u.VehicleID)
		if err == nil || errors.Is(err, apperr.ErrNotFound) { t.Fatalf("read failure became absence: %v", err) }
		return nil
	}); err != nil { t.Fatal(err) }
}

func TestConsignmentSaleAuthorityReserveRollback(t *testing.T) {
	for _, failure := range []string{"cas", "event", "outer"} {
		t.Run(failure, func(t *testing.T) {
			c, st, dealer, _, deal, actor := saleFixture(t)
			u := saleOnlyUnit(st)
			switch failure { case "cas": st.failUpdate = true; case "event": st.failEvent = true; case "outer": st.failOuterCommit = true }
			err := saleOuter(st, func(ctx context.Context) error {
				_, err := c.ReserveSale(ctx, dealer, u.VehicleID, deal, actor)
				return err
			})
			got := st.units[u.ID]
			if err == nil || got.CurrentRetailDealID != nil || got.Version != u.Version || len(st.events) != 0 {
				t.Fatalf("partial reserve after %s: %+v events=%d err=%v", failure, got, len(st.events), err)
			}
		})
	}
}

func TestConsignmentSaleAuthorityReserveCancelAndOtherVIN(t *testing.T) {
	c, st, dealer, _, deal, actor := saleFixture(t)
	u := saleOnlyUnit(st)
	other := u; other.ID = uuid.NewString(); other.VehicleID = uuid.NewString(); other.VIN = "VIN-456"
	st.units[other.ID] = other
	ret := uuid.NewString(); other.PendingReturnID = &ret; st.units[other.ID] = other
	originalTerms := string(st.order.Terms)
	if err := saleOuter(st, func(ctx context.Context) error {
		a, err := c.ReserveSale(ctx, dealer, u.VehicleID, deal, actor)
		if err != nil || a == nil || a.Revision != u.Version+1 { t.Fatalf("reserve: %+v %v", a, err) }
		if !st.locks[u.ID] { t.Fatal("unit lock released before outer stock operation") }
		if st.units[other.ID].PendingReturnID == nil { t.Fatal("other VIN return was changed") }
		if _, err := c.ReserveSale(ctx, dealer, u.VehicleID, uuid.NewString(), actor); !errors.Is(err, apperr.ErrConflict) { t.Fatalf("second deal: %v", err) }
		if err := c.CancelSale(ctx, dealer, u.VehicleID, uuid.NewString(), actor); !errors.Is(err, apperr.ErrConflict) { t.Fatalf("wrong-deal cancel: %v", err) }
		if err := c.CancelSale(ctx, dealer, u.VehicleID, deal, actor); err != nil { return err }
		return nil
	}); err != nil { t.Fatal(err) }
	got := st.units[u.ID]
	if got.CurrentRetailDealID != nil || got.Status != model.ConsignmentHeld || got.OrderID != u.OrderID ||
		got.SupplierCompanyID != u.SupplierCompanyID || string(st.order.Terms) != originalTerms || len(st.events) != 2 {
		t.Fatalf("cancel changed obligation or history: %+v events=%d", got, len(st.events))
	}
}

func TestConsignmentSaleAuthorityReturnSaleOrderings(t *testing.T) {
	for _, saleFirst := range []bool{false, true} {
		t.Run(map[bool]string{false:"return-first", true:"sale-first"}[saleFirst], func(t *testing.T) {
			c, st, dealer, _, deal, actor := saleFixture(t)
			u := saleOnlyUnit(st)
			if err := saleOuter(st, func(ctx context.Context) error {
				requestReturn := func() error {
					row, err := st.Consignments().LockUnitByVehicle(ctx, u.VehicleID)
					if err != nil { return err }
					if row.CurrentRetailDealID != nil || row.PendingReturnID != nil { return apperr.ErrConflict }
					id := uuid.NewString(); row.PendingReturnID = &id
					return st.Consignments().UpdateUnit(ctx, row, row.Version)
				}
				if saleFirst {
					if _, err := c.ReserveSale(ctx, dealer, u.VehicleID, deal, actor); err != nil { return err }
					if err := requestReturn(); !errors.Is(err, apperr.ErrConflict) { t.Fatalf("return after sale: %v", err) }
				} else {
					if err := requestReturn(); err != nil { return err }
					if _, err := c.ReserveSale(ctx, dealer, u.VehicleID, deal, actor); !errors.Is(err, apperr.ErrForbidden) { t.Fatalf("sale after return: %v", err) }
				}
				if !st.locks[u.ID] { t.Fatal("shared VIN row was never locked") }
				return nil
			}); err != nil { t.Fatal(err) }
		})
	}
}

func TestConsignmentSaleAuthorityCompletionAndRollback(t *testing.T) {
	for _, failure := range []string{"none", "cas", "event", "outer"} {
		t.Run(failure, func(t *testing.T) {
			c, st, dealer, _, deal, actor := saleFixture(t)
			u := saleOnlyUnit(st)
			if err := saleOuter(st, func(ctx context.Context) error { _, err := c.ReserveSale(ctx, dealer, u.VehicleID, deal, actor); return err }); err != nil { t.Fatal(err) }
			before := st.units[u.ID]
			originalTerms, originalEvents := string(st.order.Terms), len(st.events)
			at := time.Date(2026, 10, 12, 3, 0, 0, 0, time.UTC)
			ref := uuid.NewString()
			switch failure { case "cas": st.failUpdate = true; case "event": st.failEvent = true; case "outer": st.failOuterCommit = true }
			err := saleOuter(st, func(ctx context.Context) error {
				if _, err := c.SaleForDeal(ctx, dealer, u.VehicleID, uuid.NewString()); !errors.Is(err, apperr.ErrConflict) { t.Fatalf("wrong deal delivery: %v", err) }
				a, err := c.SaleForDeal(ctx, dealer, u.VehicleID, deal)
				if err != nil || a == nil || a.OrderID != u.OrderID { t.Fatalf("settlement authority: %+v %v", a, err) }
				return c.CompleteSale(ctx, dealer, u.VehicleID, deal, ref, actor, at)
			})
			got := st.units[u.ID]
			if failure == "none" {
				if err != nil || got.Status != model.ConsignmentSold || got.SoldAt == nil || !got.SoldAt.Equal(at) || got.SoldBy == nil || *got.SoldBy != actor || got.RetailDeliveryRef != ref || got.CurrentRetailDealID == nil || *got.CurrentRetailDealID != deal || len(st.events) != originalEvents+1 {
					t.Fatalf("completion: %+v events=%d err=%v", got, len(st.events), err)
				}
				if err := saleOuter(st, func(ctx context.Context) error { return c.CompleteSale(ctx, dealer, u.VehicleID, deal, ref, actor, at) }); !errors.Is(err, apperr.ErrNotFound) { t.Fatalf("repeated completion: %v", err) }
			} else if err == nil || got.Status != before.Status || got.Version != before.Version || got.SoldAt != nil || len(st.events) != originalEvents {
				t.Fatalf("partial completion after %s: %+v events=%d err=%v", failure, got, len(st.events), err)
			}
			if string(st.order.Terms) != originalTerms || got.OrderID != u.OrderID || got.SupplierCompanyID != u.SupplierCompanyID { t.Fatal("accepted supplier settlement was altered") }
		})
	}
}
