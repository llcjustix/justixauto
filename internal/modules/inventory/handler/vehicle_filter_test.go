package handler

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/modules/inventory/service"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

type handlerFilterVehicleRepository struct {
	service.VehicleRepository
	filter model.VehicleFilter
}

func (r *handlerFilterVehicleRepository) List(_ context.Context, f model.VehicleFilter) ([]model.VehicleRow, error) {
	r.filter = f
	return nil, nil
}

type handlerFilterStore struct {
	service.Store
	vehicles service.VehicleRepository
}

func (s handlerFilterStore) Vehicles() service.VehicleRepository { return s.vehicles }

func TestVehicleFilterParsesEligibleQuery(t *testing.T) {
	repo := &handlerFilterVehicleRepository{}
	vehicles := service.NewVehicle(service.NewDeps(handlerFilterStore{vehicles: repo}, nil, nil))
	h := New(nil, nil, nil, vehicles)
	e := echo.New()
	req := httptest.NewRequest("GET", "/?modelId=11111111-1111-4111-8111-111111111111&warehouseId=22222222-2222-4222-8222-222222222222&placement=warehouse&search=ab%25&eligible=true&limit=25&offset=100", nil)
	c := e.NewContext(req, httptest.NewRecorder())
	auth.Set(c, &auth.Principal{CompanyID: "server-company"})
	if err := h.listVehicles(c); err != nil {
		t.Fatal(err)
	}
	if !repo.filter.Eligible || repo.filter.CompanyID != "server-company" || repo.filter.Search != "AB%" || repo.filter.Offset != 100 {
		t.Fatalf("filter = %+v", repo.filter)
	}
}

func TestVehicleFilterRejectsInvalidEligibleQuery(t *testing.T) {
	vehicles := service.NewVehicle(service.NewDeps(handlerFilterStore{vehicles: &handlerFilterVehicleRepository{}}, nil, nil))
	h := New(nil, nil, nil, vehicles)
	c := echo.New().NewContext(httptest.NewRequest("GET", "/?eligible=maybe", nil), httptest.NewRecorder())
	err := h.listVehicles(c)
	var validation *apperr.ValidationError
	if !errors.As(err, &validation) || validation.Fields["eligible"] == "" {
		t.Fatalf("error = %#v", err)
	}
}
