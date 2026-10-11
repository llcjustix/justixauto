package service

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
)

type supplyTestStore struct {
	Store
	lots   map[string]*model.SupplyLot
	claims map[string][]model.SupplyClaim
	active bool
}

func (s *supplyTestStore) InTx(_ context.Context, fn func(Store) error) error { return fn(s) }
func (s *supplyTestStore) Bind(ctx context.Context) context.Context           { return ctx }
func (s *supplyTestStore) Supply() SupplyRepository                           { return supplyTestRepo{s} }
func (s *supplyTestStore) Partnerships() PartnershipRepository                { return supplyPartners{s: s} }
func (s *supplyTestStore) Events() EventRepository                            { return supplyEvents{} }

type supplyEvents struct{ EventRepository }

func (supplyEvents) Append(context.Context, *model.Event) error { return nil }

type supplyPartners struct {
	PartnershipRepository
	s *supplyTestStore
}

func (r supplyPartners) ActiveBetween(context.Context, string, string) (bool, error) {
	return r.s.active, nil
}

type supplyTestRepo struct{ s *supplyTestStore }

func (r supplyTestRepo) Create(_ context.Context, lot *model.SupplyLot) error {
	r.s.lots[lot.ID] = lot
	return nil
}
func (r supplyTestRepo) Lot(_ context.Context, id string) (*model.SupplyLot, error) {
	if x := r.s.lots[id]; x != nil {
		return x, nil
	}
	return nil, apperr.ErrNotFound
}
func (r supplyTestRepo) LockLot(ctx context.Context, id string) (*model.SupplyLot, error) {
	return r.Lot(ctx, id)
}
func (r supplyTestRepo) ListSupplier(_ context.Context, supplier string, _, _ int) ([]model.SupplyLot, error) {
	var out []model.SupplyLot
	for _, x := range r.s.lots {
		if x.SupplierCompanyID == supplier {
			out = append(out, *x)
		}
	}
	return out, nil
}
func (r supplyTestRepo) ListVisible(ctx context.Context, dealer string, limit, offset int) ([]model.SupplyLot, error) {
	return r.ListSupplier(ctx, "supplier", limit, offset)
}
func (r supplyTestRepo) Update(_ context.Context, lot *model.SupplyLot, expected int64) error {
	if lot.Version != expected {
		return apperr.ErrStale
	}
	lot.Version++
	return nil
}
func (r supplyTestRepo) Capacity(_ context.Context, lotID string) (model.SupplyCapacity, error) {
	lot := r.s.lots[lotID]
	if lot == nil {
		return model.SupplyCapacity{}, apperr.ErrNotFound
	}
	c := model.SupplyCapacity{}
	for _, claim := range r.s.claims[lotID] {
		if claim.State == "held" {
			c.Held++
		}
		if claim.State == "materialized" {
			c.Materialized++
		}
	}
	c.Available = lot.TotalQuantity - c.Held - c.Materialized
	return c, nil
}
func (r supplyTestRepo) ClaimsForUpdate(_ context.Context, lotID string) ([]model.SupplyClaim, error) {
	return slices.Clone(r.s.claims[lotID]), nil
}
func (r supplyTestRepo) CreateClaims(_ context.Context, claims []model.SupplyClaim) error {
	for _, c := range claims {
		r.s.claims[c.SupplyLotID] = append(r.s.claims[c.SupplyLotID], c)
	}
	return nil
}
func (r supplyTestRepo) UpdateClaim(_ context.Context, claim *model.SupplyClaim) error {
	for i := range r.s.claims[claim.SupplyLotID] {
		if r.s.claims[claim.SupplyLotID][i].ID == claim.ID {
			r.s.claims[claim.SupplyLotID][i] = *claim
			return nil
		}
	}
	return apperr.ErrNotFound
}

type supplyDirectory struct{ companies map[string]*Company }

func (d supplyDirectory) Company(_ context.Context, id string) (*Company, error) {
	if c := d.companies[id]; c != nil {
		return c, nil
	}
	return nil, apperr.ErrNotFound
}

type supplyCatalog struct{ model *Model }

func (c supplyCatalog) Model(_ context.Context, id string) (*Model, error) {
	if c.model != nil && c.model.ID == id {
		return c.model, nil
	}
	return nil, apperr.ErrNotFound
}

func supplyFixture() (*Supply, *supplyTestStore, *auth.Principal, string) {
	supplier := uuid.NewString()
	dealer := uuid.NewString()
	modelID := uuid.NewString()
	st := &supplyTestStore{lots: map[string]*model.SupplyLot{}, claims: map[string][]model.SupplyClaim{}, active: true}
	d := supplyDirectory{companies: map[string]*Company{
		supplier: {ID: supplier, Name: "Distributor", Kind: "seller", Active: true, UserType: supplyString("distributor")},
		dealer:   {ID: dealer, Name: "Dealer", Kind: "seller", Active: true, UserType: supplyString("dealer")},
	}}
	c := supplyCatalog{model: &Model{ID: modelID, CurrentSpecificationVersion: "2", Specifications: []ModelSpecification{{Version: "2", ExteriorColors: []string{"white"}, InteriorColors: []string{"black"}}}}}
	return NewSupply(NewDeps(st, d, c, nil, nil, func() time.Time { return time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC) })), st, &auth.Principal{CompanyID: supplier, UserID: uuid.NewString(), CommercialAccess: true}, modelID
}

func supplyString(v string) *string { return &v }
func supplyInput(modelID string, quantity int64) SupplyInput {
	return SupplyInput{ModelID: modelID, ModelSpecificationVersion: "2", ExteriorColor: "white", InteriorColor: "black", TotalQuantity: jsonx.Quantity(quantity)}
}

func TestSupplyCreateAndCapacityProjection(t *testing.T) {
	s, st, p, modelID := supplyFixture()
	v, err := s.Create(context.Background(), p, supplyInput(modelID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if v.Capacity.Available != 10 || v.Capacity.Held != 0 || v.Capacity.Materialized != 0 {
		t.Fatalf("capacity = %+v", v.Capacity)
	}
	st.claims[v.Lot.ID] = []model.SupplyClaim{{ID: uuid.NewString(), SupplyLotID: v.Lot.ID, State: "held"}, {ID: uuid.NewString(), SupplyLotID: v.Lot.ID, State: "held"}, {ID: uuid.NewString(), SupplyLotID: v.Lot.ID, State: "materialized"}}
	got, err := s.Get(context.Background(), p, v.Lot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Capacity.Available != 7 || got.Capacity.Held != 2 || got.Capacity.Materialized != 1 {
		t.Fatalf("capacity = %+v", got.Capacity)
	}
}

func TestSupplyRejectsInvalidColorQuantityAndOutsider(t *testing.T) {
	s, st, p, modelID := supplyFixture()
	bad := supplyInput(modelID, 0)
	if _, err := s.Create(context.Background(), p, bad); err == nil {
		t.Fatal("zero quantity accepted")
	}
	bad = supplyInput(modelID, 1)
	bad.ExteriorColor = "red"
	if _, err := s.Create(context.Background(), p, bad); err == nil {
		t.Fatal("wrong color accepted")
	}
	before := len(st.lots)
	outsider := *p
	outsider.CompanyID = uuid.NewString()
	_, err := s.Create(context.Background(), &outsider, supplyInput(modelID, 1))
	var validation *apperr.ValidationError
	if !errors.As(err, &validation) || validation.Fields["companyId"] == "" {
		t.Fatalf("outsider err = %v", err)
	}
	if len(st.lots) != before {
		t.Fatal("rejected outsider create persisted a lot")
	}
}

func TestSupplyUpdateCannotReduceCommittedQuantity(t *testing.T) {
	s, st, p, modelID := supplyFixture()
	v, err := s.Create(context.Background(), p, supplyInput(modelID, 4))
	if err != nil {
		t.Fatal(err)
	}
	st.claims[v.Lot.ID] = []model.SupplyClaim{{ID: uuid.NewString(), SupplyLotID: v.Lot.ID, State: "held"}, {ID: uuid.NewString(), SupplyLotID: v.Lot.ID, State: "materialized"}}
	if _, err := s.Update(context.Background(), p, v.Lot.ID, 1, supplyInput(modelID, 1)); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("reduce err = %v", err)
	}
	updated, err := s.Update(context.Background(), p, v.Lot.ID, 1, supplyInput(modelID, 3))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Capacity.Available != 1 || updated.Capacity.Held != 1 || updated.Capacity.Materialized != 1 {
		t.Fatalf("capacity = %+v", updated.Capacity)
	}
}

func TestSupplyRejectsQuantityBeyondStorageRange(t *testing.T) {
	s, st, p, modelID := supplyFixture()
	tooLarge := supplyInput(modelID, math.MaxInt32+1)
	_, err := s.Create(context.Background(), p, tooLarge)
	var validation *apperr.ValidationError
	if !errors.As(err, &validation) || validation.Fields["totalQuantity"] == "" {
		t.Fatalf("create overflow err = %v", err)
	}
	if len(st.lots) != 0 {
		t.Fatal("overflow create persisted a lot")
	}

	v, err := s.Create(context.Background(), p, supplyInput(modelID, math.MaxInt32))
	if err != nil {
		t.Fatalf("max storage quantity rejected: %v", err)
	}
	_, err = s.Update(context.Background(), p, v.Lot.ID, v.Lot.Version, tooLarge)
	if !errors.As(err, &validation) || validation.Fields["totalQuantity"] == "" {
		t.Fatalf("update overflow err = %v", err)
	}
	if st.lots[v.Lot.ID].TotalQuantity != math.MaxInt32 || st.lots[v.Lot.ID].Version != 1 {
		t.Fatalf("overflow update changed lot: %+v", st.lots[v.Lot.ID])
	}
}

func TestSupplyCommittedLotIdentityIsLocked(t *testing.T) {
	for _, state := range []string{"held", "materialized"} {
		for _, change := range []string{"model", "specification", "exterior", "interior"} {
			t.Run(state+"/"+change, func(t *testing.T) {
				s, st, p, modelID := supplyFixture()
				v, err := s.Create(context.Background(), p, supplyInput(modelID, 3))
				if err != nil {
					t.Fatal(err)
				}
				st.claims[v.Lot.ID] = []model.SupplyClaim{{ID: uuid.NewString(), SupplyLotID: v.Lot.ID, State: state}}
				in := supplyInput(modelID, 4)
				arrival := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
				in.ExpectedArrival = &arrival
				catalogModel := &Model{ID: modelID, Specifications: []ModelSpecification{
					{Version: "2", ExteriorColors: []string{"white", "red"}, InteriorColors: []string{"black", "tan"}},
					{Version: "3", ExteriorColors: []string{"white"}, InteriorColors: []string{"black"}},
				}}
				switch change {
				case "model":
					in.ModelID = uuid.NewString()
					catalogModel.ID = in.ModelID
				case "specification":
					in.ModelSpecificationVersion = "3"
				case "exterior":
					in.ExteriorColor = "red"
				case "interior":
					in.InteriorColor = "tan"
				}
				s.catalog = supplyCatalog{model: catalogModel}
				if _, err := s.Update(context.Background(), p, v.Lot.ID, 1, in); !errors.Is(err, apperr.ErrConflict) {
					t.Fatalf("committed identity edit err = %v", err)
				}
				lot := st.lots[v.Lot.ID]
				if lot.ModelID != modelID || lot.ModelSpecificationVersion != "2" || lot.ExteriorColor != "white" || lot.InteriorColor != "black" || lot.TotalQuantity != 3 || lot.ExpectedArrival != nil || lot.Version != 1 {
					t.Fatalf("committed identity edit changed lot: %+v", lot)
				}
				catalogModel.ID = modelID
				updated, err := s.Update(context.Background(), p, v.Lot.ID, 1, SupplyInput{ModelID: modelID, ModelSpecificationVersion: "2", ExteriorColor: "white", InteriorColor: "black", TotalQuantity: 4, ExpectedArrival: &arrival})
				if err != nil {
					t.Fatalf("quantity/arrival update rejected: %v", err)
				}
				if updated.Lot.TotalQuantity != 4 || updated.Lot.ExpectedArrival == nil || !updated.Lot.ExpectedArrival.Equal(arrival) || updated.Capacity.Available != 3 {
					t.Fatalf("quantity/arrival update = %+v", updated)
				}
			})
		}
	}
}

func TestSupplyUncommittedLotIdentityCanChange(t *testing.T) {
	s, st, p, modelID := supplyFixture()
	v, err := s.Create(context.Background(), p, supplyInput(modelID, 2))
	if err != nil {
		t.Fatal(err)
	}
	st.claims[v.Lot.ID] = []model.SupplyClaim{{ID: uuid.NewString(), SupplyLotID: v.Lot.ID, State: "released"}}
	newModelID := uuid.NewString()
	s.catalog = supplyCatalog{model: &Model{ID: newModelID, Specifications: []ModelSpecification{{Version: "3", ExteriorColors: []string{"red"}, InteriorColors: []string{"tan"}}}}}
	in := SupplyInput{ModelID: newModelID, ModelSpecificationVersion: "3", ExteriorColor: "red", InteriorColor: "tan", TotalQuantity: 1}
	updated, err := s.Update(context.Background(), p, v.Lot.ID, 1, in)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Lot.ModelID != newModelID || updated.Lot.ModelSpecificationVersion != "3" || updated.Lot.ExteriorColor != "red" || updated.Lot.InteriorColor != "tan" || updated.Capacity.Available != 1 {
		t.Fatalf("uncommitted identity edit = %+v", updated)
	}
}
