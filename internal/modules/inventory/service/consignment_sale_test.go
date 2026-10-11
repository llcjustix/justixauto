package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
)

type consignmentStockContextKey struct{}

var errConsignmentStockInjected = errors.New("injected stock failure")

type consignmentStockState struct {
	units      map[string]*model.VehicleUnit
	places     map[string]*model.Placement
	warehouses map[string]*model.Warehouse
	holds      map[string]*model.Reservation
	facts      []model.Fact
}

func (x consignmentStockState) clone() consignmentStockState {
	out := consignmentStockState{units: map[string]*model.VehicleUnit{}, places: map[string]*model.Placement{}, warehouses: map[string]*model.Warehouse{}, holds: map[string]*model.Reservation{}, facts: append([]model.Fact(nil), x.facts...)}
	for id, u := range x.units {
		c := *u
		if u.OwnerCompanyID != nil {
			v := *u.OwnerCompanyID
			c.OwnerCompanyID = &v
		}
		if u.CustodianCompanyID != nil {
			v := *u.CustodianCompanyID
			c.CustodianCompanyID = &v
		}
		out.units[id] = &c
	}
	for id, p := range x.places {
		c := *p
		out.places[id] = &c
	}
	for id, w := range x.warehouses {
		c := *w
		out.warehouses[id] = &c
	}
	for id, h := range x.holds {
		c := *h
		out.holds[id] = &c
	}
	return out
}

type consignmentStockStore struct {
	Store
	state consignmentStockState
	fail  string
	calls []string
}

func (s *consignmentStockStore) step(ctx context.Context, name string) error {
	if ctx.Value(consignmentStockContextKey{}) != "outer-retail-tx" {
		return errors.New("lost outer transaction context")
	}
	s.calls = append(s.calls, name)
	if s.fail == name {
		return errConsignmentStockInjected
	}
	return nil
}

func (s *consignmentStockStore) InTx(ctx context.Context, fn func(Store) error) error {
	if err := s.step(ctx, "InTx"); err != nil {
		return err
	}
	work := &consignmentStockStore{state: s.state.clone(), fail: s.fail}
	err := fn(work)
	s.calls = append(s.calls, work.calls...)
	if err == nil {
		s.state = work.state
	}
	return err
}

func (s *consignmentStockStore) Vehicles() VehicleRepository { return consignmentStockVehicles{s: s} }
func (s *consignmentStockStore) Warehouses() WarehouseRepository {
	return consignmentStockWarehouses{s: s}
}
func (s *consignmentStockStore) Reservations() ReservationRepository {
	return consignmentStockReservations{s: s}
}
func (s *consignmentStockStore) Facts() FactRepository { return consignmentStockFacts{s: s} }

type consignmentStockVehicles struct {
	VehicleRepository
	s *consignmentStockStore
}

func (r consignmentStockVehicles) LockOwned(ctx context.Context, id string) (*model.VehicleUnit, error) {
	if err := r.s.step(ctx, "LockOwned"); err != nil {
		return nil, err
	}
	if x := r.s.state.units[id]; x != nil {
		return x, nil
	}
	return nil, apperr.ErrNotFound
}
func (r consignmentStockVehicles) LockPlacement(ctx context.Context, id string) (*model.Placement, error) {
	if err := r.s.step(ctx, "LockPlacement"); err != nil {
		return nil, err
	}
	if x := r.s.state.places[id]; x != nil {
		return x, nil
	}
	return nil, apperr.ErrNotFound
}
func (r consignmentStockVehicles) DeletePlacement(ctx context.Context, id string) (*model.Placement, error) {
	if err := r.s.step(ctx, "DeletePlacement"); err != nil {
		return nil, err
	}
	x := r.s.state.places[id]
	if x == nil {
		return nil, apperr.ErrNotFound
	}
	delete(r.s.state.places, id)
	return x, nil
}
func (r consignmentStockVehicles) ClearOwnership(ctx context.Context, id string) error {
	if err := r.s.step(ctx, "ClearOwnership"); err != nil {
		return err
	}
	r.s.state.units[id].OwnerCompanyID = nil
	r.s.state.units[id].CustodianCompanyID = nil
	return nil
}

type consignmentStockWarehouses struct {
	WarehouseRepository
	s *consignmentStockStore
}

func (r consignmentStockWarehouses) Lock(ctx context.Context, companyID, id string) (*model.Warehouse, error) {
	if err := r.s.step(ctx, "LockWarehouse"); err != nil {
		return nil, err
	}
	w := r.s.state.warehouses[id]
	if w == nil || w.CompanyID != companyID {
		return nil, apperr.ErrNotFound
	}
	return w, nil
}
func (r consignmentStockWarehouses) Touch(ctx context.Context, w *model.Warehouse, _ time.Time) error {
	if err := r.s.step(ctx, "TouchWarehouse"); err != nil {
		return err
	}
	w.Version++
	return nil
}

type consignmentStockReservations struct {
	ReservationRepository
	s *consignmentStockStore
}

func (r consignmentStockReservations) LockHeldByVehicle(ctx context.Context, vehicleID string) (*model.Reservation, error) {
	if err := r.s.step(ctx, "LockHeldByVehicle"); err != nil {
		return nil, err
	}
	h := r.s.state.holds[vehicleID]
	if h == nil || h.Status != "held" {
		return nil, apperr.ErrNotFound
	}
	return h, nil
}
func (r consignmentStockReservations) HeldByVehicle(ctx context.Context, vehicleID string) (*model.Reservation, error) {
	if err := r.s.step(ctx, "HeldByVehicle"); err != nil {
		return nil, err
	}
	h := r.s.state.holds[vehicleID]
	if h == nil || h.Status != "held" {
		return nil, apperr.ErrNotFound
	}
	return h, nil
}
func (r consignmentStockReservations) Create(ctx context.Context, h *model.Reservation) error {
	if err := r.s.step(ctx, "CreateHold"); err != nil {
		return err
	}
	if old := r.s.state.holds[h.VehicleID]; old != nil && old.Status == "held" {
		return apperr.ErrConflict
	}
	c := *h
	r.s.state.holds[h.VehicleID] = &c
	return nil
}
func (r consignmentStockReservations) FinalizeOne(ctx context.Context, id string, at time.Time) error {
	if err := r.s.step(ctx, "FinalizeHold"); err != nil {
		return err
	}
	for _, h := range r.s.state.holds {
		if h.ID == id && h.Status == "held" {
			h.Status, h.ClosedAt = "finalized", &at
			return nil
		}
	}
	return apperr.ErrNotFound
}
func (r consignmentStockReservations) Release(ctx context.Context, typ, id string, ids []string, reason string, at time.Time) error {
	if err := r.s.step(ctx, "ReleaseHold"); err != nil {
		return err
	}
	for vehicleID, h := range r.s.state.holds {
		if h.HolderType == typ && h.HolderID == id && h.Status == "held" && (ids == nil || len(ids) == 1 && ids[0] == vehicleID) {
			h.Status, h.Reason, h.ClosedAt = "released", reason, &at
		}
	}
	return nil
}

type consignmentStockFacts struct {
	FactRepository
	s *consignmentStockStore
}

func (r consignmentStockFacts) Append(ctx context.Context, facts ...model.Fact) error {
	if err := r.s.step(ctx, "AppendFact"); err != nil {
		return err
	}
	r.s.state.facts = append(r.s.state.facts, facts...)
	return nil
}

func consignmentStockFixture() (*consignmentStockStore, ConsignmentSaleInput, context.Context, time.Time) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	in := ConsignmentSaleInput{UnitID: uuid.NewString(), AgreementID: uuid.NewString(), OrderID: uuid.NewString(), OrderLineID: uuid.NewString(), VehicleID: uuid.NewString(), VIN: "VIN-CONSIGNED-1", SupplierCompanyID: uuid.NewString(), DealerCompanyID: uuid.NewString(), HandoverFactID: uuid.NewString(), Revision: 2, RetailDealID: uuid.NewString(), ActorUserID: uuid.NewString(), At: now}
	wid := uuid.NewString()
	st := &consignmentStockStore{state: consignmentStockState{
		units: map[string]*model.VehicleUnit{in.VehicleID: {ID: in.VehicleID, VIN: in.VIN, OwnerCompanyID: &in.SupplierCompanyID, CustodianCompanyID: &in.DealerCompanyID}},
		places: map[string]*model.Placement{in.VehicleID: {VehicleID: in.VehicleID, WarehouseID: wid}},
		warehouses: map[string]*model.Warehouse{wid: {ID: wid, CompanyID: in.DealerCompanyID, Version: 1}},
		holds: map[string]*model.Reservation{},
	}}
	return st, in, context.WithValue(context.Background(), consignmentStockContextKey{}, "outer-retail-tx"), now
}

func TestConsignmentStockReserveAndRelease(t *testing.T) {
	st, in, ctx, now := consignmentStockFixture()
	s := NewStock(NewDeps(st, func() time.Time { return now }, nil))
	if err := s.ReserveConsignmentSale(ctx, in); err != nil {
		t.Fatal(err)
	}
	h := st.state.holds[in.VehicleID]
	if h == nil || h.CompanyID != in.DealerCompanyID || h.HolderType != "retail-deal" || h.HolderID != in.RetailDealID || h.Status != "held" {
		t.Fatalf("wrong dealer hold: %+v", h)
	}
	if err := s.ReserveConsignmentSale(ctx, in); err != nil || len(st.state.holds) != 1 {
		t.Fatalf("same holder retry: %v, holds=%d", err, len(st.state.holds))
	}
	before := st.state.clone()
	other := in
	other.RetailDealID = uuid.NewString()
	if err := s.ReserveConsignmentSale(ctx, other); !errors.Is(err, apperr.ErrConflict) || !reflect.DeepEqual(st.state, before) {
		t.Fatalf("competing holder changed state: %v", err)
	}
	if err := s.Release(ctx, model.Holder{Type: "retail-deal", ID: other.RetailDealID}, nil, "wrong deal"); err != nil || !reflect.DeepEqual(st.state, before) {
		t.Fatalf("wrong holder release: %v", err)
	}
	if err := s.Release(ctx, model.Holder{Type: "retail-deal", ID: in.RetailDealID}, nil, "cancelled"); err != nil {
		t.Fatal(err)
	}
	if st.state.holds[in.VehicleID].Status != "released" || *st.state.units[in.VehicleID].OwnerCompanyID != in.SupplierCompanyID || *st.state.units[in.VehicleID].CustodianCompanyID != in.DealerCompanyID || st.state.places[in.VehicleID] == nil || len(st.state.facts) != 0 {
		t.Fatal("release changed ownership, custody, placement or history")
	}
}

func TestConsignmentStockPhysicalGuards(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*consignmentStockStore, *ConsignmentSaleInput)
	}{
		{"wrong VIN", func(_ *consignmentStockStore, in *ConsignmentSaleInput) { in.VIN = "OTHER" }},
		{"wrong owner", func(st *consignmentStockStore, in *ConsignmentSaleInput) { x := uuid.NewString(); st.state.units[in.VehicleID].OwnerCompanyID = &x }},
		{"false custody", func(st *consignmentStockStore, in *ConsignmentSaleInput) { x := in.SupplierCompanyID; st.state.units[in.VehicleID].CustodianCompanyID = &x }},
		{"missing placement", func(st *consignmentStockStore, in *ConsignmentSaleInput) { delete(st.state.places, in.VehicleID) }},
		{"foreign warehouse", func(st *consignmentStockStore, in *ConsignmentSaleInput) { st.state.warehouses[st.state.places[in.VehicleID].WarehouseID].CompanyID = in.SupplierCompanyID }},
		{"wrong dealer", func(_ *consignmentStockStore, in *ConsignmentSaleInput) { in.DealerCompanyID = uuid.NewString() }},
		{"empty agreement", func(_ *consignmentStockStore, in *ConsignmentSaleInput) { in.AgreementID = "" }},
		{"no handover fact", func(_ *consignmentStockStore, in *ConsignmentSaleInput) { in.HandoverFactID = "" }},
		{"no revision", func(_ *consignmentStockStore, in *ConsignmentSaleInput) { in.Revision = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, in, ctx, now := consignmentStockFixture()
			tc.change(st, &in)
			before := st.state.clone()
			s := NewStock(NewDeps(st, func() time.Time { return now }, nil))
			if err := s.ReserveConsignmentSale(ctx, in); err == nil || !reflect.DeepEqual(st.state, before) {
				t.Fatalf("invalid reserve changed state: %v", err)
			}
		})
	}
}

func TestConsignmentStockReserveFailureRollback(t *testing.T) {
	for _, fail := range []string{"LockOwned", "LockPlacement", "LockHeldByVehicle", "LockWarehouse", "CreateHold"} {
		t.Run(fail, func(t *testing.T) {
			st, in, ctx, now := consignmentStockFixture()
			before := st.state.clone()
			st.fail = fail
			s := NewStock(NewDeps(st, func() time.Time { return now }, nil))
			if err := s.ReserveConsignmentSale(ctx, in); !errors.Is(err, errConsignmentStockInjected) || !reflect.DeepEqual(st.state, before) {
				t.Fatalf("hold failure did not roll back: %v", err)
			}
		})
	}
}

func TestConsignmentStockDeliveryAndFailureRollback(t *testing.T) {
	st, in, ctx, now := consignmentStockFixture()
	s := NewStock(NewDeps(st, func() time.Time { return now }, nil))
	if err := s.ReserveConsignmentSale(ctx, in); err != nil {
		t.Fatal(err)
	}
	before := st.state.clone()
	other := in
	other.RetailDealID = uuid.NewString()
	if ref, err := s.DeliverConsignmentSale(ctx, other); ref != "" || !errors.Is(err, apperr.ErrConflict) || !reflect.DeepEqual(st.state, before) {
		t.Fatalf("wrong holder delivery: ref=%q err=%v", ref, err)
	}
	for _, fail := range []string{"LockOwned", "LockPlacement", "LockHeldByVehicle", "LockWarehouse", "DeletePlacement", "ClearOwnership", "TouchWarehouse", "FinalizeHold", "AppendFact"} {
		t.Run(fail, func(t *testing.T) {
			st.fail = fail
			if ref, err := s.DeliverConsignmentSale(ctx, in); ref != "" || !errors.Is(err, errConsignmentStockInjected) || !reflect.DeepEqual(st.state, before) {
				t.Fatalf("failure did not roll back: ref=%q err=%v", ref, err)
			}
			st.fail = ""
		})
	}
	ref, err := s.DeliverConsignmentSale(ctx, in)
	if err != nil || uuid.Validate(ref) != nil {
		t.Fatalf("actual delivery: ref=%q err=%v", ref, err)
	}
	if st.state.units[in.VehicleID].OwnerCompanyID != nil || st.state.units[in.VehicleID].CustodianCompanyID != nil || st.state.places[in.VehicleID] != nil || st.state.holds[in.VehicleID].Status != "finalized" || len(st.state.facts) != 1 || st.state.facts[0].ID != ref || st.state.facts[0].FactType != "vehicle.delivered_to_customer" || st.state.facts[0].CompanyID != in.DealerCompanyID {
		t.Fatalf("incomplete delivery: state=%+v", st.state)
	}
	var details map[string]any
	if err := json.Unmarshal(st.state.facts[0].Details, &details); err != nil || details["agreementId"] != in.AgreementID || details["retailDealId"] != in.RetailDealID || details["handoverFactId"] != in.HandoverFactID || details["orderId"] != in.OrderID {
		t.Fatalf("delivery fact lost authority and order links: %v, %+v", err, details)
	}
	if ref, err := s.DeliverConsignmentSale(ctx, in); ref != "" || !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("repeat delivery: ref=%q err=%v", ref, err)
	}
	want := []string{"LockOwned", "LockPlacement", "LockHeldByVehicle", "LockWarehouse", "DeletePlacement", "ClearOwnership", "TouchWarehouse", "FinalizeHold", "AppendFact"}
	if !reflect.DeepEqual(st.calls[len(st.calls)-len(want)-2:len(st.calls)-2], want) {
		t.Fatalf("delivery lock/write order=%v", st.calls)
	}
}

func TestConsignmentStockOrdinaryOwnerBoundary(t *testing.T) {
	for _, companyKind := range []string{"dealer", "distributor"} {
		t.Run(companyKind, func(t *testing.T) {
			st, in, ctx, now := consignmentStockFixture()
			s := NewStock(NewDeps(st, func() time.Time { return now }, nil))
			ordinary := model.Holder{Type: "retail-deal", ID: uuid.NewString()}
			if err := s.Reserve(ctx, in.SupplierCompanyID, ordinary, []string{in.VehicleID}); !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("supplier bypassed dealer custody: %v", err)
			}
			st.state.units[in.VehicleID].CustodianCompanyID = &in.SupplierCompanyID
			if err := s.Reserve(ctx, in.SupplierCompanyID, ordinary, []string{in.VehicleID}); err != nil {
				t.Fatalf("normal %s owner retail failed: %v", companyKind, err)
			}
		})
	}
}
