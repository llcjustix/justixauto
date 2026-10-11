package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
)

const stockSupplier = "11111111-1111-4111-8111-111111111111"
const stockModel = "22222222-2222-4222-8222-222222222222"

type reservationStockRepository struct {
	VehicleRepository
	filter model.VehicleFilter
	rows []model.VehicleRow
	err error
	calls int
	marker any
}

func (r *reservationStockRepository) List(ctx context.Context, f model.VehicleFilter) ([]model.VehicleRow, error) {
	r.calls++
	r.filter = f
	r.marker = ctx.Value(stockContextKey{})
	return r.rows, r.err
}

type stockContextKey struct{}

type reservationStockStore struct {
	Store
	vehicles VehicleRepository
}

func (s reservationStockStore) Vehicles() VehicleRepository { return s.vehicles }
func (s reservationStockStore) InTx(context.Context, func(Store) error) error {
	panic("discovery must not start a transaction")
}

func TestReservationStockProjection(t *testing.T) {
	white := "white"
	black := "black"
	repo := &reservationStockRepository{rows: []model.VehicleRow{{VehicleUnit: model.VehicleUnit{
		ID: "vehicle", VIN: "VIN123", ModelID: stockModel, SpecVersion: 7,
		ExteriorColor: &white, InteriorColor: &black,
	}}, {VehicleUnit: model.VehicleUnit{ID: "vehicle2", VIN: "VIN456", ModelID: stockModel, SpecVersion: 2}}}}
	s := NewStock(Deps{store: reservationStockStore{vehicles: repo}})
	ctx := context.WithValue(context.Background(), stockContextKey{}, "retained")
	rows, err := s.ReservableStock(ctx, ReservableStockFilter{SupplierCompanyID: stockSupplier, ModelID: stockModel, Offset: 6})
	if err != nil { t.Fatal(err) }
	if repo.calls != 1 || repo.marker != "retained" || !reflect.DeepEqual(repo.filter, model.VehicleFilter{
		CompanyID: stockSupplier, ModelID: stockModel, Placement: "warehouse", Eligible: true, Limit: 50, Offset: 6,
	}) { t.Fatalf("filter=%+v calls=%d marker=%v", repo.filter, repo.calls, repo.marker) }
	if len(rows) != 2 || rows[0] != (ReservableStockVehicle{ID: "vehicle", VIN: "VIN123", ModelID: stockModel, ModelSpecificationVersion: 7, ExteriorColor: white, InteriorColor: black}) || rows[1].ExteriorColor != "" || rows[1].InteriorColor != "" { t.Fatalf("rows=%+v", rows) }
	rows, err = s.ReservableStock(ctx, ReservableStockFilter{SupplierCompanyID: stockSupplier, Limit: 100})
	if err != nil || len(rows) != 2 || repo.filter.Limit != 100 || repo.filter.ModelID != "" { t.Fatalf("page=%+v filter=%+v err=%v", rows, repo.filter, err) }
	repo.rows = nil
	rows, err = s.ReservableStock(ctx, ReservableStockFilter{SupplierCompanyID: stockSupplier, Limit: 1})
	if err != nil || rows == nil || len(rows) != 0 || repo.filter.Limit != 1 { t.Fatalf("empty=%v filter=%+v err=%v", rows, repo.filter, err) }
	repo.err = errors.New("stock unavailable")
	if _, err = s.ReservableStock(ctx, ReservableStockFilter{SupplierCompanyID: stockSupplier}); err != repo.err { t.Fatalf("dependency error=%v", err) }
}

func TestReservationStockProjectionRejectsInvalidFilters(t *testing.T) {
	repo := &reservationStockRepository{}
	s := NewStock(Deps{store: reservationStockStore{vehicles: repo}})
	for _, tc := range []struct{ name string; filter ReservableStockFilter; field string }{
		{"missing supplier", ReservableStockFilter{}, "supplierCompanyId"},
		{"bad supplier", ReservableStockFilter{SupplierCompanyID: "bad"}, "supplierCompanyId"},
		{"bad model", ReservableStockFilter{SupplierCompanyID: stockSupplier, ModelID: "bad"}, "modelId"},
		{"negative limit", ReservableStockFilter{SupplierCompanyID: stockSupplier, Limit: -1}, "limit"},
		{"oversize limit", ReservableStockFilter{SupplierCompanyID: stockSupplier, Limit: 101}, "limit"},
		{"negative offset", ReservableStockFilter{SupplierCompanyID: stockSupplier, Offset: -1}, "offset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.ReservableStock(context.Background(), tc.filter)
			var v *apperr.ValidationError
			if !errors.As(err, &v) || v.Fields[tc.field] == "" { t.Fatalf("error=%v", err) }
		})
	}
	if repo.calls != 0 { t.Fatalf("invalid filters reached inventory: %d", repo.calls) }
}
