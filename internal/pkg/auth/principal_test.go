package auth

import (
	"testing"

	"justixauto/internal/pkg/apperr"
)

func TestAllowNewCommercialOperation(t *testing.T) {
	if err := (&Principal{CommercialAccess: true}).AllowNewCommercialOperation(); err != nil {
		t.Fatalf("verified typed seller should be permitted: %v", err)
	}
	err := (&Principal{}).AllowNewCommercialOperation()
	if err == nil {
		t.Fatal("seller without commercial capability must be denied")
	}
	if got, ok := err.(*apperr.Error); !ok || got.Code != "company_verification_required" {
		t.Fatalf("unexpected denial: %#v", err)
	}
}
