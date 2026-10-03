package handler

import (
	"encoding/json"
	"testing"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/jsonx"
)

func TestOrderProgressSeparatesShippedAndAllocated(t *testing.T) {
	terms, _ := json.Marshal(model.Terms{Lines: []model.Line{{LineID: "line-1", Quantity: jsonx.Quantity(10)}}})
	v := &service.OrderView{
		Order: model.Order{ID: "order-1", BuyerCompanyID: "buyer", SupplierCompanyID: "supplier", Terms: terms},
		Allocations: []model.Allocation{
			{LineID: "line-1", Status: "allocated"}, {LineID: "line-1", Status: "allocated"},
			{LineID: "line-1", Status: "shipped"}, {LineID: "line-1", Status: "delivered"},
			{LineID: "line-1", Status: "released"}, {LineID: "line-1", Status: "rejected"},
		},
		ShipmentLines:  []model.ShipmentLine{{LineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", Quantity: 1}},
		ReceiptBatches: []service.ReceiptBatchReference{{OrderLineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", WarehouseID: "warehouse-1", ModelID: "model-1", ShippedQuantity: 1, ConfirmedQuantity: 1, IdentifiedCount: 1, Revision: 4}},
	}

	d := toOrder("buyer")(v)
	p := d.LineProgress[0]
	if p.Shipped != 3 || p.Allocated != 2 || p.Identified != 3 || p.Unidentified != 0 || p.ReceiptQuantityAdjusted {
		t.Fatalf("progress = %+v", p)
	}
	if remaining := 10 - int(p.Shipped) - int(p.Allocated); remaining != 5 {
		t.Fatalf("remaining capacity = %d", remaining)
	}
	if len(d.ReceiptBatches) != 1 || d.ReceiptBatches[0].WarehouseID != "warehouse-1" || d.ReceiptBatches[0].Revision != "4" {
		t.Fatalf("buyer batch references = %+v", d.ReceiptBatches)
	}

	supplierJSON, err := json.Marshal(toOrder("supplier")(v))
	if err != nil {
		t.Fatal(err)
	}
	if string(supplierJSON) == "" || containsJSONKey(supplierJSON, "warehouseId") || containsJSONKey(supplierJSON, "receiptBatches") {
		t.Fatalf("supplier response exposes receipt navigation: %s", supplierJSON)
	}
}

func TestOrderProgressReflectsCurrentBatchIdentification(t *testing.T) {
	base := &service.OrderView{
		Allocations:   []model.Allocation{{LineID: "line-1", Status: "delivered"}},
		ShipmentLines: []model.ShipmentLine{{LineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", Quantity: 3}},
	}
	base.ReceiptBatches = []service.ReceiptBatchReference{{OrderLineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", ShippedQuantity: 3, ConfirmedQuantity: 3, IdentifiedCount: 1, UnidentifiedCount: 2}}
	first := service.OrderLineProgress(base)["line-1"]
	base.ReceiptBatches[0].IdentifiedCount, base.ReceiptBatches[0].UnidentifiedCount = 3, 0
	second := service.OrderLineProgress(base)["line-1"]
	if first.Shipped != 4 || second.Shipped != 4 || first.Identified != 2 || second.Identified != 4 || first.Unidentified != 2 || second.Unidentified != 0 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}

func containsJSONKey(raw []byte, key string) bool {
	var object map[string]any
	_ = json.Unmarshal(raw, &object)
	_, ok := object[key]
	return ok
}

func TestShipmentColorsProgressDTOSeparatesActualAndUnknownOrderedFacts(t *testing.T) {
	terms, _ := json.Marshal(model.Terms{Lines: []model.Line{{LineID: "line", ModelID: "model", Quantity: 2}}})
	v := &service.OrderView{Order: model.Order{BuyerCompanyID: "buyer", SupplierCompanyID: "supplier", Terms: terms}, ReceiptBatches: []service.ReceiptBatchReference{
		{OrderLineID: "line", ReceiptBatchID: "known", ModelID: "model", ModelSpecificationVersion: "1", ExteriorColor: "Blue", InteriorColor: "Tan"},
		{OrderLineID: "line", ReceiptBatchID: "unknown", ModelID: "model"},
	}}
	d := toOrder("buyer")(v)
	if len(d.ReceiptBatches) != 2 || d.ReceiptBatches[0].ModelSpecificationVersion != "1" || d.ReceiptBatches[0].ExteriorColor != "Blue" || d.ReceiptBatches[0].InteriorColor != "Tan" {
		t.Fatalf("actual receipt facts: %+v", d.ReceiptBatches)
	}
	if d.ReceiptBatches[1].ModelSpecificationVersion != "" || d.ReceiptBatches[1].ExteriorColor != "" || d.ReceiptBatches[1].InteriorColor != "" {
		t.Fatal("unknown batch substituted")
	}
	if line := v.Order.DecodeTerms().Lines[0]; line.ModelSpecificationVersion != "" || line.ExteriorColor != "" || line.InteriorColor != "" {
		t.Fatal("legacy ordered facts changed")
	}
	raw, _ := json.Marshal(d.ReceiptBatches[1])
	for _, field := range []string{"modelSpecificationVersion", "exteriorColor", "interiorColor"} {
		if !containsJSONKey(raw, field) {
			t.Fatalf("unknown field %s omitted: %s", field, raw)
		}
	}
	if len(toOrder("supplier")(v).ReceiptBatches) != 0 {
		t.Fatal("buyer receipt navigation exposed to supplier")
	}
}
