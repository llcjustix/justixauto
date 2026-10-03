package handler

import (
	"encoding/json"
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/modules/retail/service"
)

func TestDealColorsSnapshotProjectionAndLegacyNull(t *testing.T) {
	snapshot := &model.VehicleSnapshot{VehicleID: "vehicle", VIN: "VIN-1", ModelID: "model", ModelSpecificationVersion: "7", ExteriorColor: "Graphite", InteriorColor: "Sand"}
	withSnapshot := toDeal(false)(&service.DealView{Deal: model.Deal{VehicleSnapshot: snapshot}})
	raw, err := json.Marshal(withSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	vehicle, ok := got["vehicleSnapshot"].(map[string]any)
	if !ok || vehicle["vin"] != "VIN-1" || vehicle["modelSpecificationVersion"] != "7" || vehicle["exteriorColor"] != "Graphite" || vehicle["interiorColor"] != "Sand" {
		t.Fatalf("vehicleSnapshot = %#v", got["vehicleSnapshot"])
	}

	legacyRaw, err := json.Marshal(toDeal(false)(&service.DealView{Deal: model.Deal{}}))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(legacyRaw, &got); err != nil {
		t.Fatal(err)
	}
	if got["vehicleSnapshot"] != nil {
		t.Fatalf("legacy vehicleSnapshot = %#v, want null", got["vehicleSnapshot"])
	}
}
