package service

import (
	"context"
	"testing"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
)

type offerLine = struct {
	OfferLineID string         `json:"offerLineId"`
	Quantity    jsonx.Quantity `json:"quantity"`
}

func TestSelectOrderLinesIgnoresOfferQuantity(t *testing.T) {
	// An offer is a promotion, not stock: ordering more than it lists is fine.
	offered := model.Terms{Lines: []model.Line{{LineID: "l-1", ModelID: "m-1", Quantity: 1}}}
	var v apperr.Validation
	terms, err := selectOrderLines(&v, offered, []offerLine{{OfferLineID: "l-1", Quantity: 50}})
	if err != nil {
		t.Fatalf("ordering 50 of an offer listing 1: %v", err)
	}
	if len(terms.Lines) != 1 || terms.Lines[0].Quantity != 50 {
		t.Fatalf("lines = %+v, want one line of 50", terms.Lines)
	}
}

func TestSelectOrderLinesBoundsQuantity(t *testing.T) {
	offered := model.Terms{Lines: []model.Line{{LineID: "l-1", ModelID: "m-1", Quantity: 1}}}
	for _, q := range []jsonx.Quantity{0, maxLineQuantity + 1} {
		var v apperr.Validation
		if _, err := selectOrderLines(&v, offered, []offerLine{{OfferLineID: "l-1", Quantity: q}}); err == nil {
			t.Fatalf("quantity %d accepted", q)
		}
	}
}

func TestOrderDirectValidatesSupplierAndTerms(t *testing.T) {
	s := &Deal{}
	p := &auth.Principal{CompanyID: "11111111-1111-4111-8111-111111111111"}
	cases := map[string]DirectOrderInput{
		"no supplier": {Terms: &model.Terms{}},
		"own company": {SupplierCompanyID: p.CompanyID},
		"no terms":    {SupplierCompanyID: "22222222-2222-4222-8222-222222222222"},
	}
	for name, in := range cases {
		if _, err := s.OrderDirect(context.Background(), p, in); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
