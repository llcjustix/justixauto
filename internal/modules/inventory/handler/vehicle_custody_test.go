package handler

import (
	"testing"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/modules/inventory/service"
)

func custodyString(value string) *string { return &value }

func TestVehicleCustodyProjection(t *testing.T) {
	t.Run("owner equals custodian", func(t *testing.T) {
		companyID := custodyString("company")
		got := toVehicle(&model.VehicleRow{VehicleUnit: model.VehicleUnit{OwnerCompanyID: companyID, CustodianCompanyID: companyID}})
		if got.OwnerCompanyID != companyID || got.CustodianCompanyID != companyID {
			t.Fatalf("custody projection = %+v", got)
		}
	})

	t.Run("supplier ownership with dealer custody", func(t *testing.T) {
		ownerID, custodianID := custodyString("supplier"), custodyString("dealer")
		got := toVehicle(&model.VehicleRow{VehicleUnit: model.VehicleUnit{OwnerCompanyID: ownerID, CustodianCompanyID: custodianID}})
		if got.OwnerCompanyID != ownerID || got.CustodianCompanyID != custodianID {
			t.Fatalf("custody projection = %+v", got)
		}
	})

	t.Run("cleared ownership and custody remain null", func(t *testing.T) {
		got := toVehicle(&model.VehicleRow{})
		if got.OwnerCompanyID != nil || got.CustodianCompanyID != nil {
			t.Fatalf("custody projection = %+v", got)
		}
	})

	t.Run("receipt mapper projects custody", func(t *testing.T) {
		ownerID, custodianID := custodyString("supplier"), custodyString("dealer")
		got := receiptBody(&service.ReceiptResult{Vehicles: []model.VehicleUnit{{OwnerCompanyID: ownerID, CustodianCompanyID: custodianID}}})
		if len(got.Vehicles) != 1 || got.Vehicles[0].OwnerCompanyID != ownerID || got.Vehicles[0].CustodianCompanyID != custodianID {
			t.Fatalf("receipt custody projection = %+v", got.Vehicles)
		}
	})
}
