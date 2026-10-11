package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

func TestCommercialAccessBlocksNewRetailOperationsBeforeWrites(t *testing.T) {
	p := &auth.Principal{CompanyID: "company", Permissions: map[string]bool{model.PermDeals: true}}
	if _, err := (&Deal{}).Create(context.Background(), p, DealInput{}); !commercialAccessError(err) {
		t.Fatalf("Create deal error = %v", err)
	}
	if _, err := (&Listing{}).Create(context.Background(), p, ListingInput{}); !commercialAccessError(err) {
		t.Fatalf("Create listing error = %v", err)
	}
	if _, err := (&Listing{}).SetPublished(context.Background(), p, "", 0, true); !commercialAccessError(err) {
		t.Fatalf("Publish listing error = %v", err)
	}
}

func TestCommercialAccessDoesNotReplaceRetailPermission(t *testing.T) {
	_, err := (&Deal{}).Create(context.Background(), &auth.Principal{CompanyID: "company"}, DealInput{})
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "permission_denied" {
		t.Fatalf("Create deal error = %v", err)
	}
}

func commercialAccessError(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == "company_verification_required"
}
