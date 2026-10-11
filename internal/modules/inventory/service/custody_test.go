package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
)

type custodyTestStore struct {
	Store
	units      map[string]*model.VehicleUnit
	places     map[string]*model.Placement
	warehouses map[string]*model.Warehouse
	holds      map[string]*model.Reservation
	facts      []model.Fact
}

func (s *custodyTestStore) InTx(_ context.Context, fn func(Store) error) error { return fn(s) }
func (s *custodyTestStore) Vehicles() VehicleRepository                        { return custodyVehicles{s: s} }
func (s *custodyTestStore) Warehouses() WarehouseRepository                    { return custodyWarehouses{s: s} }
func (s *custodyTestStore) Reservations() ReservationRepository                { return custodyReservations{s: s} }
func (s *custodyTestStore) Facts() FactRepository                              { return custodyFacts{s: s} }

type custodyVehicles struct {
	VehicleRepository
	s *custodyTestStore
}

func (r custodyVehicles) LockOwned(_ context.Context, id string) (*model.VehicleUnit, error) {
	if x := r.s.units[id]; x != nil {
		return x, nil
	}
	return nil, apperr.ErrNotFound
}
func (r custodyVehicles) LockPlacement(_ context.Context, id string) (*model.Placement, error) {
	if x := r.s.places[id]; x != nil {
		return x, nil
	}
	return nil, apperr.ErrNotFound
}
func (r custodyVehicles) SetCustodian(_ context.Context, id, c string) error {
	x := r.s.units[id]
	x.CustodianCompanyID = &c
	x.Version++
	return nil
}
func (r custodyVehicles) MovePlacement(_ context.Context, id, w string, at time.Time) error {
	x := r.s.places[id]
	x.WarehouseID = w
	x.PlacedAt = at
	return nil
}
func (r custodyVehicles) CountPlaced(_ context.Context, w string, ids []string) (int, error) {
	n := 0
	for _, id := range ids {
		if r.s.places[id].WarehouseID == w {
			n++
		}
	}
	return n, nil
}

type custodyWarehouses struct {
	WarehouseRepository
	s *custodyTestStore
}

func (r custodyWarehouses) Lock(_ context.Context, c, id string) (*model.Warehouse, error) {
	x := r.s.warehouses[id]
	if x == nil || x.CompanyID != c {
		return nil, apperr.ErrNotFound
	}
	return x, nil
}
func (r custodyWarehouses) Occupied(_ context.Context, ids []string) (map[string]int, error) {
	out := map[string]int{}
	for _, id := range ids {
		for _, p := range r.s.places {
			if p.WarehouseID == id {
				out[id]++
			}
		}
	}
	return out, nil
}
func (r custodyWarehouses) Touch(_ context.Context, w *model.Warehouse, _ time.Time) error {
	w.Version++
	return nil
}

type custodyReservations struct {
	ReservationRepository
	s *custodyTestStore
}

func (r custodyReservations) LockHeldByVehicle(_ context.Context, id string) (*model.Reservation, error) {
	x := r.s.holds[id]
	if x == nil || x.Status != "held" {
		return nil, apperr.ErrNotFound
	}
	return x, nil
}
func (r custodyReservations) Finalize(_ context.Context, typ, id string, ids []string, _ time.Time) error {
	for _, vid := range ids {
		x := r.s.holds[vid]
		if x == nil || x.HolderType != typ || x.HolderID != id {
			return apperr.ErrConflict
		}
		x.Status = "finalized"
	}
	return nil
}

type custodyFacts struct {
	FactRepository
	s *custodyTestStore
}

func (r custodyFacts) Append(_ context.Context, facts ...model.Fact) error {
	r.s.facts = append(r.s.facts, facts...)
	return nil
}

func TestCustody(t *testing.T) {
	owner, dealer, order, agreement, operation, actor, source, destination, alternate := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	vehicle := uuid.NewString()
	at := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	newStore := func() *custodyTestStore {
		return &custodyTestStore{units: map[string]*model.VehicleUnit{vehicle: {ID: vehicle, OwnerCompanyID: &owner, CustodianCompanyID: &owner}}, places: map[string]*model.Placement{vehicle: {VehicleID: vehicle, WarehouseID: source}}, warehouses: map[string]*model.Warehouse{source: {ID: source, CompanyID: owner, Capacity: 4}, destination: {ID: destination, CompanyID: dealer, Capacity: 4}, alternate: {ID: alternate, CompanyID: dealer, Capacity: 4}}, holds: map[string]*model.Reservation{vehicle: {VehicleID: vehicle, HolderType: "commerce-order", HolderID: order, Status: "held"}}}
	}
	input := func(purpose string) CustodyInput {
		return CustodyInput{Purpose: purpose, OwnerCompanyID: owner, SourceCustodianCompanyID: owner, DestinationCompanyID: dealer, DestinationWarehouseID: destination, ExpectedHolder: model.Holder{Type: "commerce-order", ID: order}, AgreementID: agreement, OperationRef: operation, ActorUserID: actor, VehicleIDs: []string{vehicle}, At: at}
	}
	t.Run("consignment finalizes only matching hold and records link", func(t *testing.T) {
		st := newStore()
		s := NewStock(Deps{store: st, now: func() time.Time { return at }})
		out, err := s.Custody(context.Background(), input(CustodyConsignment))
		if err != nil {
			t.Fatal(err)
		}
		if *st.units[vehicle].OwnerCompanyID != owner || *st.units[vehicle].CustodianCompanyID != dealer || st.places[vehicle].WarehouseID != destination || st.holds[vehicle].Status != "finalized" {
			t.Fatalf("state=%+v placement=%+v hold=%+v", st.units[vehicle], st.places[vehicle], st.holds[vehicle])
		}
		if len(out.FactIDs) != 1 || len(st.facts) != 1 || st.facts[0].FactType != "vehicle.custody_consignment" {
			t.Fatalf("facts=%+v out=%+v", st.facts, out)
		}
		if _, err = s.Custody(context.Background(), input(CustodyConsignment)); !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("repeat=%v", err)
		}
	})
	t.Run("return uses current dealer placement and rejects competing hold", func(t *testing.T) {
		st := newStore()
		s := NewStock(Deps{store: st, now: func() time.Time { return at }})
		if _, err := s.Custody(context.Background(), input(CustodyConsignment)); err != nil {
			t.Fatal(err)
		}
		st.places[vehicle].WarehouseID = alternate
		ret := input(CustodyReturn)
		ret.SourceCustodianCompanyID = dealer
		ret.DestinationCompanyID = owner
		ret.DestinationWarehouseID = source
		ret.ExpectedHolder = model.Holder{}
		if _, err := s.Custody(context.Background(), ret); err != nil {
			t.Fatal(err)
		}
		if *st.units[vehicle].CustodianCompanyID != owner || st.places[vehicle].WarehouseID != source {
			t.Fatal("return did not use current dealer placement")
		}
		st = newStore()
		s = NewStock(Deps{store: st, now: func() time.Time { return at }})
		if _, err := s.Custody(context.Background(), input(CustodyConsignment)); err != nil {
			t.Fatal(err)
		}
		st.holds[vehicle] = &model.Reservation{VehicleID: vehicle, Status: "held"}
		ret = input(CustodyReturn)
		ret.SourceCustodianCompanyID = dealer
		ret.DestinationCompanyID = owner
		ret.DestinationWarehouseID = source
		ret.ExpectedHolder = model.Holder{}
		if _, err := s.Custody(context.Background(), ret); !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("competing hold=%v", err)
		}
	})
	t.Run("capacity failure leaves physical state untouched", func(t *testing.T) {
		st := newStore()
		st.warehouses[destination].Capacity = 0
		s := NewStock(Deps{store: st, now: func() time.Time { return at }})
		if _, err := s.Custody(context.Background(), input(CustodyConsignment)); !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("capacity=%v", err)
		}
		if *st.units[vehicle].CustodianCompanyID != owner || st.places[vehicle].WarehouseID != source || st.holds[vehicle].Status != "held" || len(st.facts) != 0 {
			t.Fatal("capacity failure mutated state")
		}
	})
}
