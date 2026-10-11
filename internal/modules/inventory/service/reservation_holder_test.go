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

type holderTestStore struct {
	Store
	owner, custodian string
	holds            map[string]*model.Reservation
}

func (s *holderTestStore) InTx(_ context.Context, fn func(Store) error) error { return fn(s) }
func (s *holderTestStore) Vehicles() VehicleRepository                        { return holderTestVehicles{s: s} }
func (s *holderTestStore) Reservations() ReservationRepository                { return holderTestReservations{s: s} }

type holderTestVehicles struct {
	VehicleRepository
	s *holderTestStore
}

func (r holderTestVehicles) LockOwned(_ context.Context, id string) (*model.VehicleUnit, error) {
	if id != "vin" {
		return nil, apperr.ErrNotFound
	}
	custodian := r.s.custodian
	if custodian == "" {
		custodian = r.s.owner
	}
	return &model.VehicleUnit{ID: id, OwnerCompanyID: &r.s.owner, CustodianCompanyID: &custodian}, nil
}

type holderTestReservations struct {
	ReservationRepository
	s *holderTestStore
}

func (r holderTestReservations) HeldByVehicle(_ context.Context, id string) (*model.Reservation, error) {
	if x := r.s.holds[id]; x != nil {
		return x, nil
	}
	return nil, apperr.ErrNotFound
}
func (r holderTestReservations) Create(_ context.Context, x *model.Reservation) error {
	r.s.holds[x.VehicleID] = x
	return nil
}
func (r holderTestReservations) Reassign(_ context.Context, from, to model.Holder, ids []string) error {
	for _, id := range ids {
		x := r.s.holds[id]
		if x == nil || x.HolderType != from.Type || x.HolderID != from.ID {
			return apperr.ErrConflict
		}
		x.HolderType, x.HolderID = to.Type, to.ID
	}
	return nil
}
func (r holderTestReservations) Release(_ context.Context, holderType, holderID string, ids []string, _ string, _ time.Time) error {
	for _, id := range ids {
		if x := r.s.holds[id]; x != nil && x.HolderType == holderType && x.HolderID == holderID {
			delete(r.s.holds, id)
		}
	}
	return nil
}
func TestReservationHolderNamespace(t *testing.T) {
	st := &holderTestStore{owner: uuid.NewString(), holds: map[string]*model.Reservation{}}
	s := NewStock(NewDeps(st, func() time.Time { return time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC) }, nil))
	reservation := model.Holder{Type: "commerce-reservation", ID: uuid.NewString()}
	order := model.Holder{Type: "commerce-order", ID: reservation.ID}
	if err := s.Reserve(context.Background(), st.owner, reservation, []string{"vin"}); err != nil {
		t.Fatal(err)
	}
	original := st.holds["vin"]
	if err := s.Reassign(context.Background(), reservation, order, []string{"vin"}); err != nil {
		t.Fatal(err)
	}
	if got := st.holds["vin"]; got != original || got.HolderType != "commerce-order" || got.HolderID != order.ID {
		t.Fatalf("reassignment replaced or changed hold=%+v", got)
	}
	if err := s.Reassign(context.Background(), reservation, order, []string{"vin"}); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("stale reassignment=%v", err)
	}
	if err := s.Reassign(context.Background(), order, reservation, []string{"vin"}); err != nil {
		t.Fatal(err)
	}
	if x := st.holds["vin"]; x == nil || x.HolderType != "commerce-reservation" || x.HolderID != reservation.ID {
		t.Fatalf("holder=%+v", x)
	}
	if err := s.Reserve(context.Background(), st.owner, order, []string{"vin"}); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("order reused reservation hold: %v", err)
	}
	if err := s.Reserve(context.Background(), st.owner, reservation, []string{"vin"}); err != nil {
		t.Fatalf("same holder retry: %v", err)
	}
	if len(st.holds) != 1 {
		t.Fatalf("hold count=%d", len(st.holds))
	}
	if err := s.Release(context.Background(), reservation, []string{"vin"}, "released"); err != nil {
		t.Fatal(err)
	}
	if len(st.holds) != 0 {
		t.Fatal("reservation release did not clear hold")
	}
	st.custodian = uuid.NewString()
	if err := s.Reserve(context.Background(), st.owner, reservation, []string{"vin"}); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("owner reserved dealer-custodied vehicle: %v", err)
	}
}
