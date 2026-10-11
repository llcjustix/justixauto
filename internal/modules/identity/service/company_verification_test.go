package service

import (
	"testing"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
)

func TestVerificationTransitions(t *testing.T) {
	if !canSubmitVerification(model.VerificationUnverified) || !canSubmitVerification(model.VerificationReturned) {
		t.Fatal("unverified and returned companies must be able to submit")
	}
	if canSubmitVerification(model.VerificationSubmitted) || canSubmitVerification(model.VerificationVerified) {
		t.Fatal("submitted and verified companies must not submit again")
	}
	if !canReviewVerification(model.VerificationSubmitted, model.VerificationReturned) || !canReviewVerification(model.VerificationSubmitted, model.VerificationVerified) {
		t.Fatal("submitted verification must be reviewable")
	}
	if canReviewVerification(model.VerificationUnverified, model.VerificationVerified) || canReviewVerification(model.VerificationSubmitted, model.VerificationSubmitted) {
		t.Fatal("only submitted verification may be returned or verified")
	}
}

func TestSellerUserTypeValidation(t *testing.T) {
	if !model.SellerUserTypeDealer.Valid() || !model.SellerUserTypeDistributor.Valid() {
		t.Fatal("known seller types must validate")
	}
	if model.SellerUserType("provider").Valid() {
		t.Fatal("provider is a company kind, not a seller user type")
	}
}

func TestSellerOnboardingTypeMayBeDeferred(t *testing.T) {
	var v apperr.Validation
	if got := optionalSellerUserType(&v, ""); got != nil || v.Err() != nil {
		t.Fatalf("an omitted seller type must be accepted and stored as nil, got %#v / %v", got, v.Err())
	}
	if got := optionalSellerUserType(&v, model.SellerUserTypeDealer); got == nil || *got != model.SellerUserTypeDealer {
		t.Fatalf("a chosen seller type must be retained, got %#v", got)
	}
	if got := optionalSellerUserType(&v, "provider"); got != nil || v.Err() == nil {
		t.Fatalf("an invalid non-empty seller type must be rejected, got %#v / %v", got, v.Err())
	}
}
