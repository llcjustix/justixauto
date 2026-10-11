package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"justixauto/internal/modules/commerce"
	"justixauto/internal/modules/inventory"
)

type adapterMarker struct{}

type fakeInventoryReservationStock struct {
	filter inventory.ReservableStockFilter
	marker any
	rows []inventory.ReservableStockVehicle
	err error
}

func (f *fakeInventoryReservationStock) ReservableStock(ctx context.Context, filter inventory.ReservableStockFilter) ([]inventory.ReservableStockVehicle, error) {
	f.filter, f.marker = filter, ctx.Value(adapterMarker{})
	return f.rows, f.err
}

func TestReservationStockAdapter(t *testing.T) {
	f := &fakeInventoryReservationStock{rows: []inventory.ReservableStockVehicle{{ID: "vehicle", VIN: "VIN", ModelID: "model", ModelSpecificationVersion: 12, ExteriorColor: "white", InteriorColor: "black"}}}
	a := commerceReservationStockDiscovery{inventory: f}
	filter := commerce.ReservationStockFilter{SupplierCompanyID: "supplier", ModelID: "model", Limit: 25, Offset: 10}
	rows, err := a.ReservableStock(context.WithValue(context.Background(), adapterMarker{}, "retained"), filter)
	if err != nil { t.Fatal(err) }
	if !reflect.DeepEqual(f.filter, inventory.ReservableStockFilter{SupplierCompanyID: "supplier", ModelID: "model", Limit: 25, Offset: 10}) || f.marker != "retained" { t.Fatalf("filter=%+v marker=%v", f.filter, f.marker) }
	if len(rows) != 1 || rows[0] != (commerce.ReservationStockVehicle{ID: "vehicle", VIN: "VIN", ModelID: "model", ModelSpecificationVersion: "12", ExteriorColor: "white", InteriorColor: "black"}) { t.Fatalf("rows=%+v", rows) }
	f.rows = nil
	rows, err = a.ReservableStock(context.Background(), filter)
	if err != nil || rows == nil || len(rows) != 0 { t.Fatalf("empty=%v err=%v", rows, err) }
	f.err = errors.New("inventory failed")
	if _, err = a.ReservableStock(context.Background(), filter); err != f.err { t.Fatalf("error=%v", err) }
}
