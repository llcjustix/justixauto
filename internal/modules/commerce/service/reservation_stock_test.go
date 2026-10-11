package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

const discoveryDealer = "11111111-1111-4111-8111-111111111111"
const discoverySupplier = "22222222-2222-4222-8222-222222222222"
const discoveryOutsider = "33333333-3333-4333-8333-333333333333"
const discoveryModel = "44444444-4444-4444-8444-444444444444"

type discoveryDirectory struct {
	companies map[string]*Company
	errors map[string]error
	calls []string
}

func (d *discoveryDirectory) Company(_ context.Context, id string) (*Company, error) {
	d.calls = append(d.calls, id)
	if err := d.errors[id]; err != nil { return nil, err }
	if c := d.companies[id]; c != nil { return c, nil }
	return nil, apperr.ErrNotFound
}

type discoveryPartnerships struct {
	PartnershipRepository
	active bool
	err error
	calls int
	a, b string
}

func (r *discoveryPartnerships) ActiveBetween(_ context.Context, a, b string) (bool, error) {
	r.calls++; r.a, r.b = a, b
	return r.active, r.err
}

type discoveryStore struct {
	Store
	partnerships PartnershipRepository
}

func (s discoveryStore) Partnerships() PartnershipRepository { return s.partnerships }
func (s discoveryStore) InTx(context.Context, func(Store) error) error { panic("discovery wrote transaction") }
func (s discoveryStore) Events() EventRepository { panic("discovery wrote event") }
func (s discoveryStore) Reservations() ReservationRepository { panic("discovery wrote reservation") }
func (s discoveryStore) Supply() SupplyRepository { panic("discovery wrote claim") }

type discoveryStock struct {
	calls int
	filter ReservationStockFilter
	rows []ReservationStockVehicle
	err error
}

func (s *discoveryStock) ReservableStock(_ context.Context, f ReservationStockFilter) ([]ReservationStockVehicle, error) {
	s.calls++; s.filter = f
	return s.rows, s.err
}

func discoveryFixture() (*ReservationStockService, *discoveryDirectory, *discoveryPartnerships, *discoveryStock, *auth.Principal) {
	dealer, distributor := "dealer", "distributor"
	d := &discoveryDirectory{companies: map[string]*Company{
		discoveryDealer: {ID: discoveryDealer, Name: "Dealer", Kind: "seller", Active: true, UserType: &dealer},
		discoverySupplier: {ID: discoverySupplier, Name: "Distributor", Kind: "seller", Active: true, UserType: &distributor},
		discoveryOutsider: {ID: discoveryOutsider, Name: "Outsider", Kind: "seller", Active: true, UserType: &distributor},
	}}
	r := &discoveryPartnerships{active: true}
	stock := &discoveryStock{rows: []ReservationStockVehicle{{ID: "supplier-vehicle", VIN: "VIN-SUPPLIER", ModelID: discoveryModel, ModelSpecificationVersion: "7", ExteriorColor: "white", InteriorColor: "black"}}}
	s := NewReservationStockService(Deps{store: discoveryStore{partnerships: r}, directory: d}, stock)
	p := &auth.Principal{CompanyID: discoveryDealer, Permissions: map[string]bool{model.PermRead: true}, CommercialAccess: false}
	return s, d, r, stock, p
}

func TestReservationStockDiscoveryAuthorized(t *testing.T) {
	s, d, partner, stock, p := discoveryFixture()
	rows, err := s.List(context.Background(), p, ReservationStockFilter{SupplierCompanyID: discoverySupplier, ModelID: discoveryModel, Limit: 1, Offset: 9})
	if err != nil { t.Fatal(err) }
	if len(rows) != 1 || rows[0] != (ReservationStockView{VehicleID: "supplier-vehicle", VIN: "VIN-SUPPLIER", SupplierCompanyID: discoverySupplier, SupplierCompanyName: "Distributor", ModelID: discoveryModel, ModelSpecificationVersion: "7", ExteriorColor: "white", InteriorColor: "black"}) { t.Fatalf("rows=%+v", rows) }
	if stock.calls != 1 || stock.filter != (ReservationStockFilter{SupplierCompanyID: discoverySupplier, ModelID: discoveryModel, Limit: 1, Offset: 9}) || partner.calls != 1 || partner.a != discoveryDealer || partner.b != discoverySupplier || len(d.calls) != 2 { t.Fatalf("stock=%+v partner=%+v directory=%+v", stock, partner, d.calls) }
	// Read access is available even without commercial verification or trade permission.
	stock.rows = nil
	rows, err = s.List(context.Background(), p, ReservationStockFilter{SupplierCompanyID: discoverySupplier})
	if err != nil || rows == nil || len(rows) != 0 || stock.filter.Limit != 50 { t.Fatalf("empty=%v filter=%+v err=%v", rows, stock.filter, err) }
	stock.err = errors.New("inventory failed")
	if _, err = s.List(context.Background(), p, ReservationStockFilter{SupplierCompanyID: discoverySupplier}); err != stock.err { t.Fatalf("inventory error=%v", err) }
}

func TestReservationStockDiscoveryDeniedScopes(t *testing.T) {
	sNil, _, _, stockNil, _ := discoveryFixture()
	if _, err := sNil.List(context.Background(), nil, ReservationStockFilter{SupplierCompanyID: discoverySupplier}); !errors.Is(err, apperr.ErrForbidden) || stockNil.calls != 0 { t.Fatalf("nil principal error=%v calls=%d", err, stockNil.calls) }
	for _, tc := range []struct {
		name string
		change func(*discoveryDirectory, *discoveryPartnerships, *discoveryStock, *auth.Principal, *ReservationStockFilter)
		kind error
		code string
		field string
	}{
		{"nil principal", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, p *auth.Principal, _ *ReservationStockFilter) { p.Permissions = nil }, apperr.ErrForbidden, "permission_denied", ""},
		{"trade only", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, p *auth.Principal, _ *ReservationStockFilter) { p.Permissions = map[string]bool{model.PermTrade: true} }, apperr.ErrForbidden, "permission_denied", ""},
		{"no company", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, p *auth.Principal, _ *ReservationStockFilter) { p.CompanyID = "" }, apperr.ErrConflict, "no_active_company", ""},
		{"unknown caller", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { delete(d.companies, discoveryDealer) }, nil, "", "companyId"},
		{"inactive caller", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { d.companies[discoveryDealer].Active = false }, apperr.ErrConflict, "company_not_trading", ""},
		{"nonseller caller", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { d.companies[discoveryDealer].Kind = "other" }, apperr.ErrConflict, "company_not_trading", ""},
		{"wrong stored caller type", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { kind := "distributor"; d.companies[discoveryDealer].UserType = &kind }, apperr.ErrForbidden, "dealer_required", ""},
		{"missing stored caller type", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { d.companies[discoveryDealer].UserType = nil }, apperr.ErrForbidden, "dealer_required", ""},
		{"missing supplier", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, f *ReservationStockFilter) { f.SupplierCompanyID = "" }, nil, "", "supplierCompanyId"},
		{"bad supplier", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, f *ReservationStockFilter) { f.SupplierCompanyID = "bad" }, nil, "", "supplierCompanyId"},
		{"self supplier", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, f *ReservationStockFilter) { f.SupplierCompanyID = discoveryDealer }, nil, "", "supplierCompanyId"},
		{"bad model", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, f *ReservationStockFilter) { f.ModelID = "bad" }, nil, "", "modelId"},
		{"negative limit", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, f *ReservationStockFilter) { f.Limit = -1 }, nil, "", "limit"},
		{"oversize limit", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, f *ReservationStockFilter) { f.Limit = 101 }, nil, "", "limit"},
		{"negative offset", func(_ *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, f *ReservationStockFilter) { f.Offset = -1 }, nil, "", "offset"},
		{"unknown supplier", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { delete(d.companies, discoverySupplier) }, apperr.ErrNotFound, "", ""},
		{"inactive supplier", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { d.companies[discoverySupplier].Active = false }, apperr.ErrNotFound, "", ""},
		{"nonseller supplier", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { d.companies[discoverySupplier].Kind = "other" }, apperr.ErrNotFound, "", ""},
		{"wrong supplier type", func(d *discoveryDirectory, _ *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { kind := "dealer"; d.companies[discoverySupplier].UserType = &kind }, apperr.ErrNotFound, "", ""},
		{"no partner", func(_ *discoveryDirectory, r *discoveryPartnerships, _ *discoveryStock, _ *auth.Principal, _ *ReservationStockFilter) { r.active = false }, apperr.ErrNotFound, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, d, r, stock, p := discoveryFixture()
			f := ReservationStockFilter{SupplierCompanyID: discoverySupplier}
			tc.change(d, r, stock, p, &f)
			_, err := s.List(context.Background(), p, f)
			if tc.field != "" {
				var v *apperr.ValidationError
				if !errors.As(err, &v) || v.Fields[tc.field] == "" { t.Fatalf("field error=%v", err) }
			} else if !errors.Is(err, tc.kind) { t.Fatalf("error=%v want kind=%v", err, tc.kind) }
			if tc.code != "" {
				var coded *apperr.Error
				if !errors.As(err, &coded) || coded.Code != tc.code { t.Fatalf("error=%v want code=%s", err, tc.code) }
			}
			if stock.calls != 0 { t.Fatalf("unauthorized stock calls=%d", stock.calls) }
		})
	}
}

func TestReservationStockDiscoveryDependencyErrors(t *testing.T) {
	for _, tc := range []struct { name string; set func(*discoveryDirectory, *discoveryPartnerships); expected error }{
		{"caller directory", func(d *discoveryDirectory, _ *discoveryPartnerships) { d.errors = map[string]error{discoveryDealer: errors.New("directory failed")} }, nil},
		{"supplier directory", func(d *discoveryDirectory, _ *discoveryPartnerships) { d.errors = map[string]error{discoverySupplier: errors.New("directory failed")} }, nil},
		{"partnership", func(_ *discoveryDirectory, r *discoveryPartnerships) { r.err = errors.New("partnership failed") }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, d, r, stock, p := discoveryFixture()
			tc.set(d, r)
			_, err := s.List(context.Background(), p, ReservationStockFilter{SupplierCompanyID: discoverySupplier})
			if err == nil || errors.Is(err, apperr.ErrNotFound) || stock.calls != 0 { t.Fatalf("error=%v calls=%d", err, stock.calls) }
		})
	}
}
