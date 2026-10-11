package app

import (
	"context"
	"testing"

	"justixauto/internal/modules/identity"
	identitymodel "justixauto/internal/modules/identity/model"
)

type workflowProfiles03 struct{ profiles map[string]*identity.Profile }

func (f workflowProfiles03) CompanyProfile(_ context.Context, id string) (*identity.Profile, error) {
	return f.profiles[id], nil
}

func TestWorkflowPort03(t *testing.T) {
	dealer := identitymodel.SellerUserTypeDealer
	profiles := workflowProfiles03{profiles: map[string]*identity.Profile{
		"seller": {ID: "seller", Name: "Seller", Kind: identitymodel.KindSeller, UserType: &dealer, Verification: identitymodel.VerificationVerified, Access: identitymodel.AccessActive},
		"bank":   {ID: "bank", Name: "Bank", Kind: identitymodel.KindBank, Verification: identitymodel.VerificationUnverified, Access: identitymodel.AccessActive},
	}}
	d := directory{companies: profiles}

	seller, err := d.Company(context.Background(), "seller")
	if err != nil {
		t.Fatal(err)
	}
	if seller.UserType == nil || *seller.UserType != "dealer" || seller.Verification != "verified" || seller.Kind != "seller" || !seller.Active {
		t.Fatalf("seller mapping = %+v", seller)
	}
	bank, err := d.Company(context.Background(), "bank")
	if err != nil {
		t.Fatal(err)
	}
	if bank.Kind != "bank" || bank.Verification != "unverified" || bank.UserType != nil || !bank.Active {
		t.Fatalf("provider mapping = %+v", bank)
	}
}
