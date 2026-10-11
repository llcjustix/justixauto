package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

const httpDealer = "11111111-1111-4111-8111-111111111111"
const httpSupplier = "22222222-2222-4222-8222-222222222222"
const httpModel = "33333333-3333-4333-8333-333333333333"

type stockHTTPDirectory struct { dealer, supplier service.Company }
func (d stockHTTPDirectory) Company(_ context.Context, id string) (*service.Company, error) {
	switch id {
	case httpDealer: return &d.dealer, nil
	case httpSupplier: return &d.supplier, nil
	default: return nil, apperr.ErrNotFound
	}
}

type stockHTTPPartnership struct { service.PartnershipRepository; active bool }
func (p stockHTTPPartnership) ActiveBetween(_ context.Context, a, b string) (bool, error) {
	if a != httpDealer || b != httpSupplier { panic("unexpected partnership lookup") }
	return p.active, nil
}

type stockHTTPStore struct { service.Store; partner service.PartnershipRepository }
func (s stockHTTPStore) Partnerships() service.PartnershipRepository { return s.partner }
func (s stockHTTPStore) InTx(context.Context, func(service.Store) error) error { panic("read started transaction") }

type stockHTTPPort struct { calls int; filter service.ReservationStockFilter; rows []service.ReservationStockVehicle }
func (s *stockHTTPPort) ReservableStock(_ context.Context, f service.ReservationStockFilter) ([]service.ReservationStockVehicle, error) {
	s.calls++; s.filter = f
	return s.rows, nil
}

func stockHTTPFixture(t *testing.T) (*echo.Echo, **auth.Principal, *stockHTTPPort, *stockHTTPPartnership) {
	t.Helper()
	dealer, distributor := "dealer", "distributor"
	dir := stockHTTPDirectory{dealer: service.Company{ID: httpDealer, Name: "Dealer", Kind: "seller", Active: true, UserType: &dealer}, supplier: service.Company{ID: httpSupplier, Name: "Distributor", Kind: "seller", Active: true, UserType: &distributor}}
	partner := &stockHTTPPartnership{active: true}
	stock := &stockHTTPPort{rows: []service.ReservationStockVehicle{{ID: "supplier-vin-id", VIN: "SUPPLIER-VIN", ModelID: httpModel, ModelSpecificationVersion: "12", ExteriorColor: "silver", InteriorColor: "tan"}}}
	d := service.NewDeps(stockHTTPStore{partner: partner}, dir, nil, nil, nil, nil)
	h := &Handler{reservationStock: service.NewReservationStockService(d, stock)}
	p := &auth.Principal{CompanyID: httpDealer, ContextRevision: 3, Permissions: map[string]bool{model.PermRead: true}, CommercialAccess: false}
	e := httpx.NewServer(nil, func(next echo.HandlerFunc) echo.HandlerFunc { return func(c *echo.Context) error { if p != nil { auth.Set(c, p) }; return next(c) } })
	h.Routes(e.Group("/api/v1/commerce"))
	return e, &p, stock, partner
}

func stockHTTPRequest(t *testing.T, e *echo.Echo, query, revision string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/reservation-stock"+query, nil)
	if revision != "" { req.Header.Set("X-Context-Revision", revision) }
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil { t.Fatalf("status=%d body=%s: %v", rec.Code, rec.Body.String(), err) }
	return rec.Code, body
}

func TestReservationStockHTTPFieldsAndPage(t *testing.T) {
	e, _, stock, _ := stockHTTPFixture(t)
	status, body := stockHTTPRequest(t, e, "?supplierCompanyId="+httpSupplier+"&modelId="+httpModel+"&limit=1&offset=5&companyId=outsider&placement=any&eligible=false", "3")
	if status != 200 { t.Fatalf("status=%d body=%v", status, body) }
	if _, ok := body["asOf"].(string); !ok { t.Fatalf("asOf=%v", body) }
	if body["nextCursor"] != "6" { t.Fatalf("cursor=%v", body) }
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 { t.Fatalf("items=%v", body["items"]) }
	item := items[0].(map[string]any)
	if len(item) != 8 || item["vehicleId"] != "supplier-vin-id" || item["vin"] != "SUPPLIER-VIN" || item["supplierCompanyId"] != httpSupplier || item["supplierCompanyName"] != "Distributor" || item["modelId"] != httpModel || item["modelSpecificationVersion"] != "12" || item["exteriorColor"] != "silver" || item["interiorColor"] != "tan" { t.Fatalf("unsafe or wrong DTO=%v", item) }
	if stock.calls != 1 || stock.filter != (service.ReservationStockFilter{SupplierCompanyID: httpSupplier, ModelID: httpModel, Limit: 1, Offset: 5}) { t.Fatalf("stock calls=%d filter=%+v", stock.calls, stock.filter) }
	stock.rows = nil
	status, body = stockHTTPRequest(t, e, "?supplierCompanyId="+httpSupplier, "")
	if status != 200 || body["nextCursor"] != nil || stock.filter.Limit != 50 { t.Fatalf("empty status=%d body=%v filter=%+v", status, body, stock.filter) }
	if items, ok = body["items"].([]any); !ok || len(items) != 0 { t.Fatalf("empty items=%v", body["items"]) }
}

func TestReservationStockHTTPErrors(t *testing.T) {
	for _, tc := range []struct { name, query, revision string; change func(**auth.Principal, *stockHTTPPort, *stockHTTPPartnership); status int; code, field string }{
		{"anonymous", "?supplierCompanyId="+httpSupplier, "", func(p **auth.Principal, _ *stockHTTPPort, _ *stockHTTPPartnership) { *p = nil }, 401, "unauthenticated", ""},
		{"trade only", "?supplierCompanyId="+httpSupplier, "", func(p **auth.Principal, _ *stockHTTPPort, _ *stockHTTPPartnership) { (*p).Permissions = map[string]bool{model.PermTrade: true} }, 403, "permission_denied", ""},
		{"missing company", "?supplierCompanyId="+httpSupplier, "", func(p **auth.Principal, _ *stockHTTPPort, _ *stockHTTPPartnership) { (*p).CompanyID = "" }, 409, "no_active_company", ""},
		{"stale context", "?supplierCompanyId="+httpSupplier, "2", nil, 409, "context_changed", ""},
		{"missing supplier", "", "", nil, 422, "validation_failed", "supplierCompanyId"},
		{"bad model", "?supplierCompanyId="+httpSupplier+"&modelId=no", "", nil, 422, "validation_failed", "modelId"},
		{"malformed limit", "?supplierCompanyId="+httpSupplier+"&limit=abc", "", nil, 422, "validation_failed", "limit"},
		{"limit 101", "?supplierCompanyId="+httpSupplier+"&limit=101", "", nil, 422, "validation_failed", "limit"},
		{"negative offset", "?supplierCompanyId="+httpSupplier+"&offset=-1", "", nil, 422, "validation_failed", "offset"},
		{"no partnership", "?supplierCompanyId="+httpSupplier, "", func(_ **auth.Principal, _ *stockHTTPPort, partner *stockHTTPPartnership) { partner.active = false }, 404, "not_found", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, p, stock, partner := stockHTTPFixture(t)
			if tc.change != nil { tc.change(p, stock, partner) }
			status, body := stockHTTPRequest(t, e, tc.query, tc.revision)
			if status != tc.status { t.Fatalf("status=%d want=%d body=%v", status, tc.status, body) }
			detail, ok := body["error"].(map[string]any)
			if !ok || detail["code"] != tc.code { t.Fatalf("error=%v", body) }
			if tc.field != "" { fields, ok := detail["fields"].(map[string]any); if !ok || fields[tc.field] == nil { t.Fatalf("fields=%v", detail) } }
			if stock.calls != 0 { t.Fatalf("unauthorized stock calls=%d", stock.calls) }
		})
	}
}

