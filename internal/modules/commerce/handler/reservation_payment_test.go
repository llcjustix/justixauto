package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
	"justixauto/internal/pkg/money"
)

func TestReservationPaymentHTTP_AccountingDTO(t *testing.T) {
	p := &auth.Principal{CompanyID: "supplier", Permissions: map[string]bool{model.PermPaymentsAccept: true}}
	invoiceID := "invoice"
	v := &service.ReservationMoneyView{ReservationID: "reservation", BuyerCompanyID: "buyer", SupplierCompanyID: "supplier", Revision: 4, AgreedAdvance: money.Money{AmountMinor: "100", Currency: "USD"}, Accepted: money.Money{AmountMinor: "20", Currency: "USD"}, Pending: money.Money{AmountMinor: "80", Currency: "USD"}, PendingCount: 1, Refunded: money.Money{AmountMinor: "0", Currency: "USD"}, RefundObligation: money.Money{AmountMinor: "20", Currency: "USD"}, RefundStatus: "requested", Credit: &model.ReservationCredit{OrderID: "order", InvoiceID: &invoiceID}, Evidence: []model.ReservationEvidence{{ID: "pending", AmountMinor: "80", Currency: "USD", Status: "submitted", Version: 1}}}
	d := toReservationAccounting(p, v)
	if d.Revision != "4" || d.CreditOrderID == nil || *d.CreditOrderID != "order" || d.CreditInvoiceID == nil || len(d.AllowedActions) != 2 {
		t.Fatalf("accounting DTO lost protected state: %#v", d)
	}
	p = &auth.Principal{CompanyID: "buyer", Permissions: map[string]bool{model.PermTrade: true, model.PermPaymentsAccept: true}}
	v.Credit = nil
	v.RefundStatus = "none"
	v.Pending.AmountMinor = "30"
	d = toReservationAccounting(p, v)
	if len(d.AllowedActions) != 2 || d.AllowedActions[0] != "submit-payment" || d.AllowedActions[1] != "request-refund" {
		t.Fatalf("buyer actions: %#v", d.AllowedActions)
	}
	v.Pending.AmountMinor = "0"
	v.PendingCount = 0
	v.Accepted.AmountMinor = "100"
	d = toReservationAccounting(p, v)
	if len(d.AllowedActions) != 1 || d.AllowedActions[0] != "request-refund" {
		t.Fatalf("full advance actions: %#v", d.AllowedActions)
	}
}

type accountingStore struct {
	service.Store
	r    model.Reservation
	fail error
}

func (s *accountingStore) InTx(_ context.Context, fn func(service.Store) error) error {
	if s.fail != nil {
		return s.fail
	}
	return fn(s)
}

func (s *accountingStore) Reservations() service.ReservationRepository {
	return accountingReservations{r: s.r}
}

func (s *accountingStore) ReservationPayments() service.ReservationPaymentRepository {
	return accountingRows{}
}

type accountingReservations struct {
	service.ReservationRepository
	r model.Reservation
}

func (s accountingReservations) ForUpdate(_ context.Context, id string) (*model.Reservation, []model.ReservationLine, error) {
	if id != s.r.ID {
		return nil, nil, apperr.ErrNotFound
	}
	r := s.r
	return &r, nil, nil
}

type accountingRows struct {
	service.ReservationPaymentRepository
}

func (accountingRows) Evidence(context.Context, string) ([]model.ReservationEvidence, error) {
	return []model.ReservationEvidence{}, nil
}

func (accountingRows) Refund(context.Context, string) (*model.ReservationRefund, error) {
	return nil, apperr.ErrNotFound
}

func (accountingRows) CreditForReservation(context.Context, string) (*model.ReservationCredit, error) {
	return nil, apperr.ErrNotFound
}

func TestReservationPaymentHTTP_Routes(t *testing.T) {
	buyerID, supplierID, resID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	advance := money.Money{AmountMinor: "20", Currency: "USD"}
	price := money.Money{AmountMinor: "100", Currency: "USD"}
	raw, _ := json.Marshal(service.ReservationSnapshot{Terms: service.ReservationTerms{UnitPrice: &price, Advance: &advance}})
	st := &accountingStore{r: model.Reservation{ID: resID, BuyerCompanyID: buyerID, SupplierCompanyID: supplierID, Status: model.ReservationActive, CurrentTerms: raw, CurrentDigest: "digest", CurrentRevision: 1, Version: 4}}
	p := &auth.Principal{CompanyID: buyerID, ContextRevision: 2, Permissions: map[string]bool{model.PermRead: true, model.PermTrade: true}}
	e := httpx.NewServer(slog.Default())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if p != nil {
				auth.Set(c, p)
			}
			return next(c)
		}
	})
	h := &Handler{reservationPayments: service.NewReservationPayment(service.NewDeps(st, nil, nil, nil, nil, func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }))}
	h.Routes(e.Group("/api/v1/commerce"))
	request := func(method, path, body, ifMatch string) (int, map[string]any, string) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if ifMatch != "" {
			req.Header.Set("If-Match", ifMatch)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		var payload map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &payload)
		return rec.Code, payload, rec.Header().Get("ETag")
	}
	path := "/api/v1/commerce/reservations/" + resID + "/accounting"
	if code, body, tag := request(http.MethodGet, path, "", ""); code != 200 || tag != "\"4\"" || body["revision"] != "4" {
		t.Fatalf("accounting read: %d %#v %s", code, body, tag)
	}
	p = nil
	if code, _, _ := request(http.MethodGet, path, "", ""); code != 401 {
		t.Fatalf("anonymous read=%d", code)
	}
	p = &auth.Principal{CompanyID: buyerID, Permissions: map[string]bool{}}
	if code, _, _ := request(http.MethodGet, path, "", ""); code != 403 {
		t.Fatalf("missing read permission=%d", code)
	}
	p = &auth.Principal{Permissions: map[string]bool{model.PermRead: true}}
	if code, _, _ := request(http.MethodGet, path, "", ""); code != 409 {
		t.Fatalf("missing company=%d", code)
	}
	p = &auth.Principal{CompanyID: uuid.NewString(), Permissions: map[string]bool{model.PermRead: true}}
	if code, _, _ := request(http.MethodGet, path, "", ""); code != 404 {
		t.Fatalf("foreign company=%d", code)
	}
	p = &auth.Principal{CompanyID: buyerID, Permissions: map[string]bool{model.PermRead: true, model.PermTrade: true}}
	post := "/api/v1/commerce/reservations/" + resID + "/payment-evidence"
	if code, _, _ := request(http.MethodPost, post, `{}`, ""); code != 428 {
		t.Fatalf("missing If-Match=%d", code)
	}
	if code, _, _ := request(http.MethodPost, post, `{}`, "abc"); code != 422 {
		t.Fatalf("bad revision=%d", code)
	}
	if code, _, _ := request(http.MethodPost, post, `{bad`, "4"); code != 400 {
		t.Fatalf("bad JSON=%d", code)
	}
	if code, _, _ := request(http.MethodPost, post, `{"claimedAmount":{"amountMinor":"1.5","currency":"USD"}}`, "4"); code != 422 {
		t.Fatalf("fractional money=%d", code)
	}
	decision := "/api/v1/commerce/reservation-payment-evidence/" + uuid.NewString() + "/accept"
	if code, _, _ := request(http.MethodPost, decision, `{"reservationRevision":"4","confirmation":true}`, "1"); code != 403 {
		t.Fatalf("buyer accepted evidence=%d", code)
	}
	p = &auth.Principal{CompanyID: supplierID, Permissions: map[string]bool{model.PermPaymentsAccept: true}}
	if code, _, _ := request(http.MethodPost, decision, `{"reservationRevision":"bad","confirmation":true}`, "1"); code != 422 {
		t.Fatalf("bad reservation revision=%d", code)
	}
	if code, _, _ := request(http.MethodPost, "/api/v1/commerce/reservations/"+resID+"/refund-confirmations", `{}`, ""); code != 428 {
		t.Fatalf("refund revision=%d", code)
	}
	for _, path := range []string{"/api/v1/commerce/reservations/" + resID + "/credits", "/api/v1/commerce/reservations/" + resID + "/release", "/api/v1/commerce/reservations/" + resID + "/expire"} {
		if code, _, _ := request(http.MethodPost, path, `{}`, "4"); code != 404 {
			t.Fatalf("public mutation %s=%d", path, code)
		}
	}
	st.fail = apperr.ErrNotFound
	p.Permissions[model.PermRead] = true
	if code, body, _ := request(http.MethodGet, "/api/v1/commerce/reservations/"+resID+"/accounting", "", ""); code != 404 || body["error"] == nil {
		t.Fatalf("service error envelope: %d %#v", code, body)
	}
}
