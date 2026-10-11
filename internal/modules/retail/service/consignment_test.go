package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

type retailConsignmentFake struct {
	Consignment
	unit *ConsignmentSale
	associated string
	held, sold bool
	deliveryRef string
	deny, fail string
	pending bool
	steps []string
	bound func(context.Context) bool
}

func (f *retailConsignmentFake) check(ctx context.Context, seller, vehicle string) (*ConsignmentSale, error) {
	if f.bound != nil && !f.bound(ctx) { return nil, errors.New("unbound consignment call") }
	if f.fail == "read" { return nil, errors.New("authority read failed") }
	if f.unit == nil || f.sold { return nil, nil }
	if f.deny != "" || f.pending || f.unit.DealerCompanyID != seller || f.unit.VehicleID != vehicle {
		return nil, apperr.New(apperr.ErrForbidden, "consignment_sale_denied", "VIN has no live sale authority")
	}
	return f.unit, nil
}

func (f *retailConsignmentFake) CheckConsignmentSale(ctx context.Context, seller, vehicle string) (*ConsignmentSale, error) {
	f.steps = append(f.steps, "check")
	u, err := f.check(ctx, seller, vehicle)
	if err != nil || u == nil { return u, err }
	if f.associated != "" { return nil, apperr.New(apperr.ErrConflict, "consignment_retail_reserved", "reserved") }
	return u, nil
}

func (f *retailConsignmentFake) ReserveConsignmentSale(ctx context.Context, seller, vehicle, deal, actor string) (*ConsignmentSale, error) {
	f.steps = append(f.steps, "reserve")
	u, err := f.check(ctx, seller, vehicle)
	if err != nil || u == nil { return u, err }
	if f.associated != "" { return nil, apperr.New(apperr.ErrConflict, "consignment_retail_reserved", "reserved") }
	if f.fail == "reserve" { return nil, errors.New("reserve authority failed") }
	f.associated = deal
	return u, nil
}

func (f *retailConsignmentFake) ConsignmentSaleForDeal(ctx context.Context, seller, vehicle, deal string) (*ConsignmentSale, error) {
	f.steps = append(f.steps, "for-deal")
	u, err := f.check(ctx, seller, vehicle)
	if err != nil { return nil, err }
	if u == nil { return nil, apperr.ErrNotFound }
	if f.associated != deal { return nil, apperr.New(apperr.ErrConflict, "consignment_retail_mismatch", "wrong deal") }
	return u, nil
}

func (f *retailConsignmentFake) HoldConsignmentSale(ctx context.Context, u *ConsignmentSale, deal, actor string, at time.Time) error {
	f.steps = append(f.steps, "hold")
	if f.bound != nil && !f.bound(ctx) { return errors.New("unbound stock hold") }
	if f.fail == "hold" { return errors.New("stock hold failed") }
	if u != f.unit || f.associated != deal || actor == "" || at.IsZero() { return errors.New("wrong hold facts") }
	f.held = true
	return nil
}

func (f *retailConsignmentFake) DeliverConsignmentSale(ctx context.Context, u *ConsignmentSale, deal, actor string, at time.Time) (string, error) {
	f.steps = append(f.steps, "physical-delivery")
	if f.bound != nil && !f.bound(ctx) { return "", errors.New("unbound stock delivery") }
	if f.fail == "delivery" { return "", errors.New("stock delivery failed") }
	if u != f.unit || f.associated != deal || !f.held || actor == "" || at.IsZero() { return "", errors.New("wrong delivery facts") }
	f.held = false
	return "inventory-delivery-fact", nil
}

func (f *retailConsignmentFake) CompleteConsignmentSale(ctx context.Context, seller, vehicle, deal, ref, actor string, at time.Time) error {
	f.steps = append(f.steps, "complete")
	if f.fail == "complete" { return errors.New("commerce completion failed") }
	if _, err := f.ConsignmentSaleForDeal(ctx, seller, vehicle, deal); err != nil { return err }
	if ref != "inventory-delivery-fact" || actor == "" || at.IsZero() { return errors.New("wrong delivery reference") }
	f.sold, f.deliveryRef = true, ref
	return nil
}

func (f *retailConsignmentFake) CancelConsignmentSale(ctx context.Context, seller, vehicle, deal, actor string) error {
	f.steps = append(f.steps, "cancel")
	if f.fail == "cancel" { return errors.New("commerce cancellation failed") }
	if _, err := f.ConsignmentSaleForDeal(ctx, seller, vehicle, deal); err != nil { return err }
	f.associated = ""
	return nil
}

type retailConsignmentListings struct {
	ListingRepository
	row *model.Listing
	fail string
}
func (r *retailConsignmentListings) Create(_ context.Context, l *model.Listing) error {
	if r.fail == "create" { return errors.New("listing insert failed") }
	c := *l; r.row = &c; return nil
}
func (r *retailConsignmentListings) Get(_ context.Context, company, id string) (*model.Listing, error) {
	if r.row == nil || r.row.CompanyID != company || r.row.ID != id { return nil, apperr.ErrNotFound }
	c := *r.row; return &c, nil
}
func (r *retailConsignmentListings) Update(_ context.Context, l *model.Listing, expected int64) error {
	if r.fail == "update" { return errors.New("listing update failed") }
	if r.row == nil || r.row.Version != expected { return apperr.ErrStale }
	c := *l; c.Version++; r.row = &c; return nil
}

type retailConsignmentCreateStore struct {
	*colorStore
	authority *retailConsignmentFake
	listings *retailConsignmentListings
}
func (s *retailConsignmentCreateStore) Listings() ListingRepository { return s.listings }
func (s *retailConsignmentCreateStore) InTx(ctx context.Context, fn func(Store) error) error {
	oldAuthority := *s.authority
	var oldListing *model.Listing
	if s.listings.row != nil { c := *s.listings.row; oldListing = &c }
	err := s.colorStore.InTx(ctx, func(Store) error { return fn(s) })
	if err != nil { *s.authority = oldAuthority; s.listings.row = oldListing }
	return err
}

func retailCreateFixture() (*Deal, *Listing, *retailConsignmentCreateStore, *colorStock, *auth.Principal) {
	d, base, stock, p := colorFixture(nil)
	a := &retailConsignmentFake{unit: &ConsignmentSale{
		UnitID: "unit", AgreementID: "agreement", OrderID: "order", OrderLineID: "line",
		VehicleID: colorVehicle, VIN: stock.vehicle.VIN, SupplierCompanyID: "supplier", DealerCompanyID: p.CompanyID,
		HandoverFactID: "handover", Revision: 1,
	}}
	st := &retailConsignmentCreateStore{colorStore: base, authority: a, listings: &retailConsignmentListings{}}
	d.store, d.consignment = st, a
	stock.vehicle.Owned, stock.vehicle.InWarehouse = false, true
	return d, NewListing(d.Deps), st, stock, p
}

func retailListingInput() ListingInput {
	return ListingInput{VehicleID: colorVehicle, Text: "Car", AskingPrice: money.Money{AmountMinor: "100", Currency: "USD"}}
}

func TestRetailConsignmentListingAndReservation(t *testing.T) {
	d, l, st, stock, p := retailCreateFixture()
	row, err := l.Create(context.Background(), p, retailListingInput())
	if err != nil { t.Fatal(err) }
	if row.Status != "draft" { t.Fatalf("listing status=%s", row.Status) }
	row, err = l.SetPublished(context.Background(), p, row.ID, row.Version, true)
	if err != nil || row.Status != "published" { t.Fatalf("publish=%+v err=%v", row, err) }
	deal, err := d.Create(context.Background(), p, colorInput())
	if err != nil { t.Fatal(err) }
	if st.authority.associated != deal.ID || !st.authority.held || deal.VehicleSnapshot == nil || deal.VehicleSnapshot.VIN != stock.vehicle.VIN {
		t.Fatalf("reserved association/snapshot=%+v %+v", st.authority, deal.VehicleSnapshot)
	}
	if _, err := d.Create(context.Background(), p, colorInput()); err == nil { t.Fatal("second retail hold accepted") }
	if _, err := l.SetPublished(context.Background(), p, row.ID, st.listings.row.Version, false); err != nil { t.Fatal(err) }
}

func TestRetailConsignmentPublicationRechecksLiveAuthority(t *testing.T) {
	_, l, st, _, p := retailCreateFixture()
	row, err := l.Create(context.Background(), p, retailListingInput())
	if err != nil { t.Fatal(err) }
	st.authority.pending = true
	if _, err := l.SetPublished(context.Background(), p, row.ID, row.Version, true); err == nil || st.listings.row.Status != "draft" {
		t.Fatalf("stale authority published: %v %+v", err, st.listings.row)
	}
	st.authority.pending = false
	p.CommercialAccess = false
	if _, err := l.SetPublished(context.Background(), p, row.ID, row.Version, true); err == nil { t.Fatal("unverified publication accepted") }
	if _, err := l.Create(context.Background(), p, retailListingInput()); err == nil { t.Fatal("unverified listing accepted") }
	if _, err := l.Get(context.Background(), p, row.ID); err != nil { t.Fatalf("listing history blocked: %v", err) }
	if _, err := l.SetPublished(context.Background(), p, row.ID, row.Version, false); err != nil { t.Fatalf("existing listing withdrawal blocked: %v", err) }
}

func TestRetailConsignmentAuthorityDenialsAndOrdinaryOwners(t *testing.T) {
	for _, scenario := range []string{"supplier owner bypass", "wrong dealer", "wrong VIN", "wrong agreement", "prepared", "pending return", "read failure", "competing hold", "listing failure"} {
		t.Run(scenario, func(t *testing.T) {
			d, l, st, stock, p := retailCreateFixture()
			switch scenario {
			case "supplier owner bypass": stock.vehicle.Owned = true; st.authority.deny = "supplier"
			case "wrong dealer": st.authority.unit.DealerCompanyID = "other"
			case "wrong VIN": stock.vehicle.VIN = "OTHER-VIN"
			case "wrong agreement": st.authority.deny = "agreement"
			case "prepared": st.authority.deny = "prepared"
			case "pending return": st.authority.pending = true
			case "read failure": st.authority.fail = "read"
			case "competing hold": st.authority.fail = "hold"
			case "listing failure": st.listings.fail = "create"
			}
			if scenario != "competing hold" {
				_, err := l.Create(context.Background(), p, retailListingInput())
				if err == nil { t.Fatal("listing accepted denied VIN") }
			}
			if scenario != "listing failure" {
				_, err := d.Create(context.Background(), p, colorInput())
				if err == nil && scenario != "listing failure" { t.Fatal("deal accepted denied VIN") }
			}
			if len(st.deals.created) != 0 || st.authority.associated != "" || st.authority.held || st.listings.row != nil {
				t.Fatalf("failed operation persisted state: %+v %+v", st.deals.created, st.authority)
			}
		})
	}
	for _, companyType := range []string{"dealer", "distributor"} {
		t.Run(companyType+" ordinary", func(t *testing.T) {
			d, l, st, stock, p := retailCreateFixture()
			st.authority.unit = nil
			stock.vehicle.Owned = true
			if _, err := l.Create(context.Background(), p, retailListingInput()); err != nil { t.Fatal(err) }
			if _, err := d.Create(context.Background(), p, colorInput()); err != nil { t.Fatal(err) }
			if st.authority.associated != "" || st.authority.held { t.Fatal("ordinary owner used consignment hold") }
		})
	}
}

type retailConsignmentCashStock struct { *cashStock; released bool }
func (s *retailConsignmentCashStock) Release(ctx context.Context, deal, reason string) error {
	if ctx.Value(paymentTxKey{}) != s.repo || !s.repo.inTx { return errors.New("unbound release") }
	s.released = true; return nil
}
type retailConsignmentCashStore struct { *cashStore; authority *retailConsignmentFake; stock *retailConsignmentCashStock }
func (s *retailConsignmentCashStore) InTx(ctx context.Context, fn func(Store) error) error {
	oldAuthority, oldReleased := *s.authority, s.stock.released
	err := s.cashStore.InTx(ctx, func(Store) error { return fn(s) })
	if err != nil { *s.authority = oldAuthority; s.stock.released = oldReleased }
	return err
}
func retailCashFixture() (*Deal, *retailConsignmentCashStore, *auth.Principal) {
	d, base, p := cashFixture()
	a := &retailConsignmentFake{unit: &ConsignmentSale{UnitID: "unit", AgreementID: "agreement", OrderID: "accepted-order", OrderLineID: "line", VehicleID: base.repo.deal.VehicleID, VIN: "VIN", SupplierCompanyID: "supplier", DealerCompanyID: p.CompanyID, HandoverFactID: "handover", Revision: 2}, associated: base.repo.deal.ID, held: true}
	a.bound = func(ctx context.Context) bool { return ctx.Value(paymentTxKey{}) == base.repo && base.repo.inTx }
	stock := &retailConsignmentCashStock{cashStock: base.stock}
	st := &retailConsignmentCashStore{cashStore: base, authority: a, stock: stock}
	d.store, d.stock, d.consignment = st, stock, a
	return d, st, p
}

func TestRetailConsignmentDeliveryAndRollback(t *testing.T) {
	for _, fail := range []string{"", "delivery", "complete", "deal", "listing", "event"} {
		t.Run(fail, func(t *testing.T) {
			d, st, p := retailCashFixture()
			cashAccepted(st.cashStore, cashSeed(st.cashStore, "vehicle-payment", "100", "USD"), "100")
			st.authority.fail = fail
			switch fail {
			case "deal": st.repo.failUpdate = errors.New("deal write failed")
			case "listing": st.listings.fail = true
			case "event": st.events.failType = "deal.delivered"
			}
			dealID := st.repo.deal.ID
			at := d.clock()
			out, err := d.Deliver(context.Background(), p, dealID, 7, at)
			if fail == "" {
				if err != nil || out.Status != "delivered" || !st.authority.sold || st.authority.deliveryRef != "inventory-delivery-fact" || st.authority.unit.OrderID != "accepted-order" { t.Fatalf("sale=%+v authority=%+v err=%v", out, st.authority, err) }
				if _, err := d.Deliver(context.Background(), p, dealID, 8, at); err == nil { t.Fatal("repeated delivery accepted") }
			} else if err == nil || st.authority.sold || !st.authority.held || st.repo.deal.Status != "reserved" || st.stock.delivered {
				t.Fatalf("delivery failure not rolled back: err=%v authority=%+v deal=%+v", err, st.authority, st.repo.deal)
			}
		})
	}
}

func TestRetailConsignmentCancellationAndHistory(t *testing.T) {
	d, st, p := retailCashFixture()
	dealID := st.repo.deal.ID
	if _, err := d.Cancel(context.Background(), p, dealID, 7, "customer changed mind"); err != nil { t.Fatal(err) }
	if st.authority.associated != "" || st.authority.sold || !st.authority.held || !st.stock.released || st.repo.deal.Status != "cancelled" || st.authority.unit.OrderID != "accepted-order" { t.Fatalf("cancel=%+v %+v", st.authority, st.repo.deal) }
	if _, err := d.Cancel(context.Background(), p, dealID, 8, "again"); err == nil { t.Fatal("repeat cancellation accepted") }
	if _, err := d.Get(context.Background(), p, dealID); err != nil { t.Fatalf("cancelled history inaccessible: %v", err) }
	for _, fail := range []string{"cancel", "deal"} {
		d, st, p := retailCashFixture()
		if fail == "cancel" { st.authority.fail = "cancel" } else { st.repo.failUpdate = errors.New("deal write failed") }
		if _, err := d.Cancel(context.Background(), p, st.repo.deal.ID, 7, "reason"); err == nil || !st.authority.held || st.authority.associated == "" || st.stock.released || st.repo.deal.Status != "reserved" { t.Fatalf("cancel rollback: %v %+v", err, st.authority) }
	}
	d, st, p = retailCashFixture()
	cashAccepted(st.cashStore, cashSeed(st.cashStore, "vehicle-payment", "100", "USD"), "10")
	if _, err := d.Cancel(context.Background(), p, st.repo.deal.ID, 7, "reason"); err == nil || st.stock.released { t.Fatal("accepted payment cancellation not blocked") }
}

func TestRetailConsignmentExistingDealClassificationAndVerification(t *testing.T) {
	for _, scenario := range []string{"wrong deal", "read failure", "pending return", "no association"} {
		t.Run(scenario, func(t *testing.T) {
			d, st, p := retailCashFixture()
			cashAccepted(st.cashStore, cashSeed(st.cashStore, "vehicle-payment", "100", "USD"), "100")
			switch scenario {
			case "wrong deal": st.authority.associated = "other-deal"
			case "read failure": st.authority.fail = "read"
			case "pending return": st.authority.pending = true
			case "no association": st.authority.associated = ""
			}
			if _, err := d.Deliver(context.Background(), p, st.repo.deal.ID, 7, d.clock()); err == nil || st.stock.delivered || st.authority.sold { t.Fatalf("unsafe fallback: %v", err) }
		})
	}
	d, st, p := retailCashFixture()
	p.CommercialAccess = false
	if _, err := d.Create(context.Background(), p, colorInput()); err == nil { t.Fatal("unverified new deal accepted") }
	if _, err := d.Get(context.Background(), p, st.repo.deal.ID); err != nil { t.Fatalf("existing obligation history blocked: %v", err) }
	if _, err := d.Cancel(context.Background(), p, st.repo.deal.ID, 7, "reason"); err != nil { t.Fatalf("existing obligation service blocked: %v", err) }
}

func TestRetailConsignmentPaymentSchemesKeepDeliveryGates(t *testing.T) {
	for _, scenario := range []string{"cash unpaid", "installment first unpaid", "installment insurance denied", "installment ready", "partner finance"} {
		t.Run(scenario, func(t *testing.T) {
			d, st, p := retailCashFixture()
			deal := st.repo.deal
			switch scenario {
			case "installment first unpaid", "installment insurance denied", "installment ready":
				deal.PaymentScheme = "own-installment"
				now := d.clock(); deal.RegisteredOn = &now
				registration := cashSeed(st.cashStore, "registration", "7", "USD")
				cashAccepted(st.cashStore, registration, "7")
				first := cashSeed(st.cashStore, "first-installment", "20", "USD")
				if scenario != "installment first unpaid" { cashAccepted(st.cashStore, first, "20") }
				if scenario == "installment insurance denied" { d.insurance = cashInsurance(false) }
			case "partner finance": deal.PaymentScheme = "partner-finance"
			}
			out, err := d.Deliver(context.Background(), p, deal.ID, 7, d.clock())
			if scenario == "installment ready" {
				if err != nil || out.Status != "delivered" || !st.authority.sold { t.Fatalf("ready installment not delivered: %+v %v", out, err) }
			} else if err == nil || st.authority.sold || !st.authority.held || deal.Status != "reserved" {
				t.Fatalf("payment gate bypassed: %+v %v", out, err)
			}
		})
	}
}
