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
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

type candidateHTTPRepository struct {
	service.ConsignmentRepository
	rows          []model.ConsignmentUnit
	dealerID      string
	limit, offset int
}

func (r *candidateHTTPRepository) RetailCandidates(_ context.Context, dealerID string, limit, offset int) ([]model.ConsignmentUnit, error) {
	r.dealerID, r.limit, r.offset = dealerID, limit, offset
	if offset >= len(r.rows) {
		return []model.ConsignmentUnit{}, nil
	}
	end := offset + limit
	if end > len(r.rows) {
		end = len(r.rows)
	}
	return r.rows[offset:end], nil
}

type candidateHTTPStore struct {
	service.Store
	candidates service.ConsignmentRepository
}

func (s candidateHTTPStore) Consignments() service.ConsignmentRepository { return s.candidates }

func consignmentCandidateHTTPFixture(t *testing.T) (*echo.Echo, **auth.Principal, *candidateHTTPRepository) {
	t.Helper()
	repo := &candidateHTTPRepository{rows: []model.ConsignmentUnit{{VehicleID: "vehicle-1"}}}
	d := service.NewDeps(candidateHTTPStore{candidates: repo}, nil, nil, nil, nil, nil)
	h := &Handler{consignments: service.NewConsignment(d)}
	p := &auth.Principal{CompanyID: httpDealer, ContextRevision: 3, Permissions: map[string]bool{"retail.read": true}}
	e := httpx.NewServer(nil, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if p != nil {
				auth.Set(c, p)
			}
			return next(c)
		}
	})
	h.Routes(e.Group("/api/v1/commerce"))
	return e, &p, repo
}

func consignmentCandidateHTTPRequest(t *testing.T, e *echo.Echo, query, revision string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/commerce/consignments/retail-candidates"+query, nil)
	if revision != "" {
		req.Header.Set("X-Context-Revision", revision)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	body := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("status=%d body=%s: %v", rec.Code, rec.Body.String(), err)
	}
	return rec.Code, body
}

func TestConsignmentRetailProjection(t *testing.T) {
	t.Run("redacted current-company page", func(t *testing.T) {
		e, _, repo := consignmentCandidateHTTPFixture(t)
		status, body := consignmentCandidateHTTPRequest(t, e, "?limit=1&offset=0&dealerId="+httpSupplier, "3")
		if status != http.StatusOK || repo.dealerID != httpDealer || repo.limit != 1 || repo.offset != 0 {
			t.Fatalf("status=%d dealer=%s page=%d,%d body=%v", status, repo.dealerID, repo.limit, repo.offset, body)
		}
		items, ok := body["items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("items=%v", body["items"])
		}
		item := items[0].(map[string]any)
		if len(item) != 1 || item["vehicleId"] != "vehicle-1" {
			t.Fatalf("non-redacted item=%v", item)
		}
		if body["nextCursor"] != "1" {
			t.Fatalf("cursor=%v", body["nextCursor"])
		}
	})
	for _, tc := range []struct {
		name, revision string
		change         func(**auth.Principal)
		status         int
		code           string
	}{
		{"anonymous", "", func(p **auth.Principal) { *p = nil }, 401, "unauthenticated"},
		{"commerce read only", "", func(p **auth.Principal) { (*p).Permissions = map[string]bool{model.PermRead: true} }, 403, "permission_denied"},
		{"missing company", "", func(p **auth.Principal) { (*p).CompanyID = "" }, 409, "no_active_company"},
		{"stale context", "2", nil, 409, "context_changed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, p, repo := consignmentCandidateHTTPFixture(t)
			if tc.change != nil {
				tc.change(p)
			}
			status, body := consignmentCandidateHTTPRequest(t, e, "", tc.revision)
			if status != tc.status {
				t.Fatalf("status=%d body=%v", status, body)
			}
			detail, ok := body["error"].(map[string]any)
			if !ok || detail["code"] != tc.code {
				t.Fatalf("error=%v", body)
			}
			if repo.dealerID != "" {
				t.Fatalf("unauthorized repository read for %s", tc.name)
			}
		})
	}
}
