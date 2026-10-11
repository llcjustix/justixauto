package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

func TestCommercialAccessBlocksNewCommerceCommitmentsBeforeWrites(t *testing.T) {
	p := &auth.Principal{CompanyID: "company"}
	deal := &Deal{}
	for name, call := range map[string]func() error{
		"RFQ":   func() error { _, err := deal.CreateRFQ(context.Background(), p, RFQInput{}); return err },
		"quote": func() error { _, err := deal.Quote(context.Background(), p, "", 0, model.Terms{}); return err },
		"order": func() error { _, err := deal.OrderDirect(context.Background(), p, DirectOrderInput{}); return err },
		"offer": func() error { _, err := (&Offer{}).Create(context.Background(), p, OfferInput{}); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !commercialCommitmentError(err) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func commercialCommitmentError(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == "company_verification_required"
}
