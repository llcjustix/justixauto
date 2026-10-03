package service

import (
	"context"
	"testing"
	"time"

	"justixauto/internal/modules/inventory/model"
)

func TestShipmentColorsReceiveDeliveryKeepsExactIncomingPin(t *testing.T) {
	warehouses, vehicles := &colorWarehouseRepo{}, &colorVehicleRepo{}
	receipt, models := colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Blue", "Red"}, []string{"Tan", "Black"}), 2: colorSpec(2, []string{"Green"}, []string{"White"})}, warehouses, vehicles)
	stock := &Stock{receipt.Deps}
	id, err := stock.ReceiveDelivery(context.Background(), Delivery{ToCompanyID: "buyer", ToWarehouseID: colorWarehouseID, ModelID: colorModelID,
		ModelSpecificationVersion: 1, ExteriorColor: "Blue", InteriorColor: "Tan", Quantity: 2, VINs: []string{"1HGCM82633A004352"}, ActorUserID: "supplier", At: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	b := warehouses.created
	if b == nil || id != b.ID || models.lastVersion != 1 || b.SpecVersion != 1 || *b.ExteriorColor != "Blue" || *b.InteriorColor != "Tan" || b.ConfirmedQuantity != 2 || b.IdentifiedCount != 1 || b.UnidentifiedCount != 1 || b.CompanyID != "buyer" {
		t.Fatalf("receipt %+v", b)
	}
	if len(vehicles.created) != 1 || vehicles.created[0].SpecVersion != 1 || *vehicles.created[0].ExteriorColor != "Blue" || *vehicles.created[0].InteriorColor != "Tan" {
		t.Fatalf("VIN %+v", vehicles.created)
	}
}

func TestShipmentColorsReceiveDeliveryRejectsMissingPinAndWrongPalette(t *testing.T) {
	for _, name := range []string{"missing pin", "wrong pinned palette"} {
		t.Run(name, func(t *testing.T) {
			warehouses, vehicles := &colorWarehouseRepo{}, &colorVehicleRepo{}
			receipt, _ := colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Blue"}, []string{"Tan"}), 2: colorSpec(2, []string{"Green"}, []string{"White"})}, warehouses, vehicles)
			d := Delivery{ModelID: colorModelID, ModelSpecificationVersion: 1, ExteriorColor: "Green", InteriorColor: "White", Quantity: 1}
			if name == "missing pin" {
				d.ModelSpecificationVersion = 0
			}
			if _, err := (&Stock{receipt.Deps}).ReceiveDelivery(context.Background(), d); err == nil || warehouses.created != nil || len(vehicles.created) != 0 {
				t.Fatalf("invalid incoming facts persisted: err=%v batch=%+v", err, warehouses.created)
			}
		})
	}
}
