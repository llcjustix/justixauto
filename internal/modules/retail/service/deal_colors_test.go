package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

const (
	colorCompany  = "00000000-0000-0000-0000-000000000001"
	colorBranch   = "00000000-0000-0000-0000-000000000002"
	colorCustomer = "00000000-0000-0000-0000-000000000003"
	colorVehicle  = "00000000-0000-0000-0000-000000000004"
	colorActor    = "00000000-0000-0000-0000-000000000005"
)

type colorDeals struct {
	DealRepository
	created []model.Deal
	deal    *model.Deal
}

func (r *colorDeals) Create(_ context.Context, d *model.Deal) error {
	r.created = append(r.created, *d)
	r.deal = &r.created[len(r.created)-1]
	return nil
}

func (r *colorDeals) Deal(_ context.Context, companyID, id string) (*model.Deal, error) {
	if r.deal == nil || r.deal.CompanyID != companyID || r.deal.ID != id {
		return nil, errors.New("deal not found")
	}
	return r.deal, nil
}

type colorCRM struct{ CRMRepository }

func (colorCRM) Customer(context.Context, string, string) (*model.Customer, error) {
	return &model.Customer{ID: colorCustomer}, nil
}

type colorEvents struct{ EventRepository }

func (colorEvents) Append(context.Context, *model.Event) error { return nil }

type colorStore struct {
	deals  *colorDeals
	crm    colorCRM
	events colorEvents
}

func (s *colorStore) CRM() CRMRepository                       { return s.crm }
func (s *colorStore) Listings() ListingRepository              { return nil }
func (s *colorStore) Deals() DealRepository                    { return s.deals }
func (s *colorStore) Events() EventRepository                  { return s.events }
func (s *colorStore) Bind(ctx context.Context) context.Context { return ctx }
func (s *colorStore) InTx(_ context.Context, fn func(Store) error) error {
	before := append([]model.Deal(nil), s.deals.created...)
	if err := fn(s); err != nil {
		s.deals.created = before
		return err
	}
	return nil
}

type colorCompanyPort struct{ Company }

func (colorCompanyPort) BranchOf(context.Context, string, string) (bool, error) { return true, nil }

type colorStock struct {
	Stock
	vehicle    *Vehicle
	reserveErr error
}

func (s *colorStock) Vehicle(context.Context, string, string) (*Vehicle, error) {
	return s.vehicle, nil
}
func (s *colorStock) Reserve(_ context.Context, _ string, _ string, _ string) error {
	return s.reserveErr
}

func colorFixture(reserveErr error) (*Deal, *colorStore, *colorStock, *auth.Principal) {
	st := &colorStore{deals: &colorDeals{}}
	stock := &colorStock{vehicle: &Vehicle{
		ID: colorVehicle, VIN: "XW8ZZZ61ZHG000001", ModelID: "00000000-0000-0000-0000-000000000006",
		ModelSpecificationVersion: 7, ExteriorColor: "Graphite", InteriorColor: "Sand", Owned: true,
	}, reserveErr: reserveErr}
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	svc := NewDeal(NewDeps(st, colorCompanyPort{}, stock, nil, func() time.Time { return now }), nil)
	p := &auth.Principal{UserID: colorActor, CompanyID: colorCompany, Permissions: map[string]bool{model.PermDeals: true}}
	return svc, st, stock, p
}

func colorInput() DealInput {
	return DealInput{CustomerID: colorCustomer, VehicleID: colorVehicle, BranchID: colorBranch, PaymentScheme: "cash", Price: money.Money{AmountMinor: "100", Currency: "USD"}}
}

func TestSaleColorsCapturesInventoryFactsAtReservation(t *testing.T) {
	svc, st, _, p := colorFixture(nil)
	d, err := svc.Create(context.Background(), p, colorInput())
	if err != nil {
		t.Fatal(err)
	}
	if d.VehicleSnapshot == nil || d.VehicleSnapshot.VIN != "XW8ZZZ61ZHG000001" || d.VehicleSnapshot.ModelSpecificationVersion != "7" || d.VehicleSnapshot.ExteriorColor != "Graphite" || d.VehicleSnapshot.InteriorColor != "Sand" {
		t.Fatalf("snapshot = %#v", d.VehicleSnapshot)
	}
	if len(st.deals.created) != 1 || st.deals.created[0].VehicleSnapshot == nil || st.deals.created[0].VehicleSnapshot.VIN != d.VehicleSnapshot.VIN {
		t.Fatalf("persisted snapshot = %#v", st.deals.created)
	}
}

func TestSaleColorsReservationRollbackDoesNotPersistSnapshot(t *testing.T) {
	svc, st, _, p := colorFixture(errors.New("reservation failed"))
	if _, err := svc.Create(context.Background(), p, colorInput()); err == nil {
		t.Fatal("Create unexpectedly succeeded")
	}
	if len(st.deals.created) != 0 {
		t.Fatalf("rollback left persisted deals: %#v", st.deals.created)
	}
}

func TestSaleColorsSurviveDeliveryWithoutStockRead(t *testing.T) {
	svc, st, stock, p := colorFixture(nil)
	_, err := svc.Create(context.Background(), p, colorInput())
	if err != nil {
		t.Fatal(err)
	}
	st.deals.deal.Status = "delivered"
	stock.vehicle = nil // A delivered vehicle is no longer visible to retail stock.
	view, err := svc.view(context.Background(), p, st.deals.deal, false)
	if err != nil {
		t.Fatal(err)
	}
	if view.Deal.VehicleSnapshot == nil || view.Deal.VehicleSnapshot.VIN != "XW8ZZZ61ZHG000001" {
		t.Fatalf("delivered snapshot = %#v", view.Deal.VehicleSnapshot)
	}
}
