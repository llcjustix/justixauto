package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

func TestCommercialAccessBlocksNewReceiptBeforeWrites(t *testing.T) {
	warehouses, vehicles := &colorWarehouseRepo{}, &colorVehicleRepo{}
	receipt, _ := colorService(nil, warehouses, vehicles)
	_, err := receipt.Receive(context.Background(), colorPrincipalWithoutCommercialAccess(), colorWarehouseID, 1, ReceiptInput{})
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "company_verification_required" {
		t.Fatalf("Receive error = %v", err)
	}
	if warehouses.created != nil || len(vehicles.created) != 0 {
		t.Fatal("unverified receipt reached a write")
	}
}

func colorPrincipalWithoutCommercialAccess() *auth.Principal {
	return &auth.Principal{CompanyID: "company", UserID: "user"}
}
