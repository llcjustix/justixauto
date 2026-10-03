package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

const filterID = "11111111-1111-4111-8111-111111111111"
const filterWarehouseID = "22222222-2222-4222-8222-222222222222"

type filterVehicleRepository struct {
	VehicleRepository
	filter model.VehicleFilter
}

func (r *filterVehicleRepository) List(_ context.Context, f model.VehicleFilter) ([]model.VehicleRow, error) {
	r.filter = f
	return nil, nil
}

type filterStore struct {
	Store
	vehicles VehicleRepository
}

func (s filterStore) Vehicles() VehicleRepository { return s.vehicles }

func TestVehicleFilterForcesCompanyAndNormalizesVINSearch(t *testing.T) {
	repo := &filterVehicleRepository{}
	s := NewVehicle(Deps{store: filterStore{vehicles: repo}})
	p := &auth.Principal{CompanyID: "caller-company"}
	_, err := s.List(context.Background(), p, model.VehicleFilter{
		CompanyID: "client-company", ModelID: filterID, WarehouseID: filterWarehouseID,
		Placement: "warehouse", Search: "  ab%_\\z  ", Eligible: true, Limit: 25, Offset: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.filter.CompanyID != p.CompanyID || repo.filter.Search != "AB%_\\Z" {
		t.Fatalf("filter = %+v", repo.filter)
	}
	if !repo.filter.Eligible || repo.filter.ModelID != filterID || repo.filter.Offset != 100 {
		t.Fatalf("filter = %+v", repo.filter)
	}
}

func TestVehicleFilterRejectsInvalidFields(t *testing.T) {
	s := NewVehicle(Deps{store: filterStore{vehicles: &filterVehicleRepository{}}})
	_, err := s.List(context.Background(), &auth.Principal{CompanyID: "caller"}, model.VehicleFilter{
		Placement: "yard", WarehouseID: "bad", ModelID: "bad", Eligible: true, Limit: 101, Offset: -1,
	})
	var validation *apperr.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v", err)
	}
	for _, field := range []string{"placement", "warehouseId", "modelId", "limit", "offset"} {
		if _, ok := validation.Fields[field]; !ok {
			t.Errorf("missing %s validation: %#v", field, validation.Fields)
		}
	}
}

// Sale creation lists every eligible warehouse vehicle; order allocation narrows by model.
func TestVehicleFilterAllowsEligibleLookupAcrossModels(t *testing.T) {
	repo := &filterVehicleRepository{}
	s := NewVehicle(Deps{store: filterStore{vehicles: repo}})
	_, err := s.List(context.Background(), &auth.Principal{CompanyID: "caller"}, model.VehicleFilter{Placement: "warehouse", Eligible: true})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.filter.Eligible || repo.filter.ModelID != "" || repo.filter.CompanyID != "caller" {
		t.Fatalf("filter = %+v", repo.filter)
	}
}
