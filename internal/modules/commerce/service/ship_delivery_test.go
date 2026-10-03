package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
)

// boundStore is a Store that only binds a context.
type boundStore struct{ Store }

func (boundStore) Bind(ctx context.Context) context.Context { return ctx }

// transferStock records the hand-over deliverOnShipment asks for.
type transferStock struct {
	Stock
	err         error
	toCompany   string
	toWarehouse string
	vehicles    []string
}

func (s *transferStock) Transfer(_ context.Context, _ string, vehicleIDs []string, toCompanyID, toWarehouseID, _ string, _ time.Time) error {
	s.toCompany, s.toWarehouse, s.vehicles = toCompanyID, toWarehouseID, vehicleIDs
	return s.err
}

func shippedOrder(quantity int64) *model.Order {
	terms, _ := json.Marshal(model.Terms{Lines: []model.Line{{LineID: "l-1", ModelID: "m-1", Quantity: jsonx.Quantity(quantity)}}})
	warehouse := "w-buyer"
	return &model.Order{ID: "o-1", BuyerCompanyID: "c-buyer", SupplierCompanyID: "c-supplier", Terms: terms, ReceivingWarehouseID: &warehouse, Status: model.OrderFulfilling}
}

func TestDeliverOnShipmentPutsVehiclesInBuyerWarehouse(t *testing.T) {
	allocs := []model.Allocation{
		{LineID: "l-1", VehicleID: "v-1", Status: "allocated"},
		{LineID: "l-1", VehicleID: "v-2", Status: "allocated"},
	}
	stock := &transferStock{}
	s := &Fulfilment{Deps{stock: stock}}
	p := &auth.Principal{CompanyID: "c-supplier", UserID: "u-1"}

	// One of two ordered vehicles: delivered, the order stays in fulfilment.
	o, sh := shippedOrder(2), &model.Shipment{Status: "in-transit"}
	if err := s.deliverOnShipment(context.Background(), boundStore{}, p, o, sh, allocs, nil, []string{"v-1"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if stock.toCompany != "c-buyer" || stock.toWarehouse != "w-buyer" || len(stock.vehicles) != 1 {
		t.Fatalf("handed over to %s/%s: %v", stock.toCompany, stock.toWarehouse, stock.vehicles)
	}
	if sh.Status != "received" || o.Status != model.OrderFulfilling {
		t.Fatalf("partial shipment: shipment %s, order %s", sh.Status, o.Status)
	}

	// Every ordered vehicle shipped: the order is completed without a receipt.
	o, sh = shippedOrder(2), &model.Shipment{Status: "in-transit"}
	if err := s.deliverOnShipment(context.Background(), boundStore{}, p, o, sh, allocs, nil, []string{"v-1", "v-2"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if sh.Status != "received" || o.Status != model.OrderCompleted {
		t.Fatalf("full shipment: shipment %s, order %s", sh.Status, o.Status)
	}
}

func TestPlanShipmentLinesKeepsWithinTheOrder(t *testing.T) {
	o := shippedOrder(10)
	// 4 of 10 are already shipped: 6 more fit, 7 do not.
	plan, err := planShipmentLines(o, map[string]int{"l-1": 4}, []ShipmentLineInput{{OrderLineID: "l-1", Quantity: 6, VINs: []string{"LGXC16DF0P0000001"}}})
	if err != nil || len(plan) != 1 || plan[0].quantity != 6 || plan[0].line.ModelID != "m-1" || len(plan[0].vins) != 1 {
		t.Fatalf("plan %+v, err %v", plan, err)
	}
	for name, in := range map[string]ShipmentLineInput{
		"too many":      {OrderLineID: "l-1", Quantity: 7},
		"zero":          {OrderLineID: "l-1", Quantity: 0},
		"unknown line":  {OrderLineID: "l-9", Quantity: 1},
		"too many VINs": {OrderLineID: "l-1", Quantity: 1, VINs: []string{"A", "B"}},
	} {
		if _, err := planShipmentLines(o, map[string]int{"l-1": 4}, []ShipmentLineInput{in}); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestDeliverOnShipmentCountsQuantityShipments(t *testing.T) {
	// 1 of 2 went out by quantity before; shipping the allocated one completes the order.
	s := &Fulfilment{Deps{stock: &transferStock{}}}
	o, sh := shippedOrder(2), &model.Shipment{Status: "in-transit"}
	allocs := []model.Allocation{{LineID: "l-1", VehicleID: "v-1", Status: "allocated"}}
	if err := s.deliverOnShipment(context.Background(), boundStore{}, &auth.Principal{}, o, sh, allocs, map[string]int{"l-1": 1}, []string{"v-1"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if o.Status != model.OrderCompleted {
		t.Fatalf("order %s", o.Status)
	}
}

func TestDeliverOnShipmentBlockedByFullWarehouse(t *testing.T) {
	stock := &transferStock{err: apperr.New(apperr.ErrConflict, "capacity_exceeded", "not enough free space in the receiving warehouse")}
	s := &Fulfilment{Deps{stock: stock}}
	o, sh := shippedOrder(1), &model.Shipment{Status: "in-transit"}
	allocs := []model.Allocation{{LineID: "l-1", VehicleID: "v-1", Status: "allocated"}}
	err := s.deliverOnShipment(context.Background(), boundStore{}, &auth.Principal{}, o, sh, allocs, nil, []string{"v-1"}, time.Now())
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "buyer_warehouse_full" {
		t.Fatalf("want buyer_warehouse_full, got %v", err)
	}
	if sh.Status != "in-transit" || o.Status != model.OrderFulfilling {
		t.Fatalf("a blocked shipment changed state: shipment %s, order %s", sh.Status, o.Status)
	}
}
