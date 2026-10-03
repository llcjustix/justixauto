package service

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/money"
)

type colorsOffers struct {
	OfferRepository
	offer   *model.Offer
	version *model.OfferVersion
}

func (r colorsOffers) VisibleByPublishedVersion(context.Context, string, string) (*model.Offer, error) {
	return r.offer, nil
}
func (r colorsOffers) Version(context.Context, string, string) (*model.OfferVersion, error) {
	return r.version, nil
}

func TestOfferColorsVersionPinsWithoutRequiringBuyerChoice(t *testing.T) {
	catalog, terms := colorsFixture()
	terms.Lines[0].ModelSpecificationVersion = "1"
	s, _, p := colorsDeal(catalog)
	o := NewOffer(s.Deps)
	version, err := o.newVersion(context.Background(), p, uuid.NewString(), OfferInput{Terms: terms, Audience: model.Audience{Mode: "all-active"}})
	if err != nil {
		t.Fatal(err)
	}
	saved, _ := version.Decode()
	l := saved.Lines[0]
	if l.ModelSpecificationVersion != "1" || l.ExteriorColor != "" || l.InteriorColor != "" {
		t.Fatalf("offer selected a default color: %+v", l)
	}
	terms.Lines[0].ExteriorColor = "Red"
	if _, err := o.newVersion(context.Background(), p, uuid.NewString(), OfferInput{Terms: terms, Audience: model.Audience{Mode: "all-active"}}); err == nil {
		t.Fatal("offer accepted color from current instead of pinned palette")
	}
}

func TestSelectOrderLinesSplitIdentityPriceAndWholeSchedule(t *testing.T) {
	_, offered := colorsFixture()
	source := offered.Lines[0]
	offered.PaymentSchedule = []model.Installment{{ID: uuid.NewString(), Amount: money.Money{AmountMinor: "2500000", Currency: "USD"}, DueDate: "2026-11-01"}}
	in := []OfferOrderLine{
		{OfferLineID: source.LineID, Quantity: 1, ExteriorColor: "White", InteriorColor: "Black"},
		{OfferLineID: source.LineID, Quantity: 1, ExteriorColor: "Blue", InteriorColor: "Tan"},
	}
	var v apperr.Validation
	terms, err := selectOrderLines(&v, offered, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(terms.Lines) != 2 || terms.Lines[0].LineID == terms.Lines[1].LineID || len(terms.PaymentSchedule) != 1 {
		t.Fatalf("split identity/schedule: %+v", terms)
	}
	for i, l := range terms.Lines {
		if uuid.Validate(l.LineID) != nil || l.LineID == source.LineID || l.OfferLineID != source.LineID || l.UnitPrice != source.UnitPrice || l.ExteriorColor != in[i].ExteriorColor || l.InteriorColor != in[i].InteriorColor {
			t.Fatalf("split %d lost price, identity or pair: %+v", i, l)
		}
	}
	for _, name := range []string{"partial quantity", "more than offer", "missing other row"} {
		t.Run(name, func(t *testing.T) {
			input := append([]OfferOrderLine(nil), in...)
			quote := offered
			switch name {
			case "partial quantity":
				input = input[:1]
			case "more than offer":
				input[1].Quantity = 3
			case "missing other row":
				quote.Lines = append(append([]model.Line(nil), offered.Lines...), model.Line{LineID: uuid.NewString(), Quantity: 1})
			}
			var v apperr.Validation
			got, err := selectOrderLines(&v, quote, input)
			if err != nil || len(got.PaymentSchedule) != 0 {
				t.Fatalf("not-whole schedule kept: %+v, %v", got, err)
			}
		})
	}
}

func TestSelectOrderLinesRejectsUnknownAndChangedOfferedFacts(t *testing.T) {
	_, offered := colorsFixture()
	ol := &offered.Lines[0]
	ol.ModelSpecificationVersion, ol.ExteriorColor, ol.InteriorColor = "1", "Blue", "Tan"
	for _, name := range []string{"unknown source", "version", "body", "interior"} {
		t.Run(name, func(t *testing.T) {
			in := OfferOrderLine{OfferLineID: ol.LineID, Quantity: 1}
			switch name {
			case "unknown source":
				in.OfferLineID = uuid.NewString()
			case "version":
				in.ModelSpecificationVersion = "2"
			case "body":
				in.ExteriorColor = "White"
			case "interior":
				in.InteriorColor = "Black"
			}
			var v apperr.Validation
			if _, err := selectOrderLines(&v, offered, []OfferOrderLine{in}); err == nil {
				t.Fatal("unknown or conflicting offered facts accepted")
			}
		})
	}
}

func TestOfferColorsOrderUsesPinnedPaletteAndLegacyResolvesWithoutRewrite(t *testing.T) {
	for _, name := range []string{"pinned split", "legacy", "known colors"} {
		t.Run(name, func(t *testing.T) {
			catalog, offered := colorsFixture()
			s, st, p := colorsDeal(catalog)
			source := &offered.Lines[0]
			input := []OfferOrderLine{{OfferLineID: source.LineID, Quantity: 2}}
			if name != "legacy" {
				source.ModelSpecificationVersion = "1"
			}
			if name == "pinned split" {
				input = []OfferOrderLine{
					{OfferLineID: source.LineID, Quantity: 1, ExteriorColor: "White", InteriorColor: "Cream"},
					{OfferLineID: source.LineID, Quantity: 1, ExteriorColor: "Blue", InteriorColor: "Tan"},
				}
			} else if name == "known colors" {
				source.ExteriorColor, source.InteriorColor = "Blue", "Tan"
			}
			raw, _ := json.Marshal(offered)
			stored := &model.OfferVersion{ID: uuid.NewString(), Terms: raw}
			st.offers = colorsOffers{offer: &model.Offer{ID: uuid.NewString(), SupplierCompanyID: uuid.NewString()}, version: stored}
			o, err := s.OrderFromOffer(context.Background(), p, DirectOrderInput{OfferVersionID: stored.ID, Lines: input})
			if err != nil {
				t.Fatal(err)
			}
			ls := o.DecodeTerms().Lines
			if len(ls) != len(input) || !bytes.Equal(stored.Terms, raw) {
				t.Fatal("source offer rewritten or rows merged")
			}
			for i, l := range ls {
				wantVersion, wantBody, wantInterior := "1", input[i].ExteriorColor, input[i].InteriorColor
				if name == "legacy" {
					wantVersion, wantBody, wantInterior = "2", "Red", "Gray"
				}
				if name == "known colors" {
					wantBody, wantInterior = "Blue", "Tan"
				}
				if l.ModelSpecificationVersion != wantVersion || l.ExteriorColor != wantBody || l.InteriorColor != wantInterior || l.OfferLineID != source.LineID || l.UnitPrice != source.UnitPrice {
					t.Fatalf("saved line %d: %+v", i, l)
				}
			}
		})
	}
}

func TestOfferColorsMissingChoiceAndStaleLegacyPinFailBeforeWrite(t *testing.T) {
	for _, name := range []string{"ambiguous", "stale legacy pin", "missing exact version"} {
		t.Run(name, func(t *testing.T) {
			catalog, offered := colorsFixture()
			s, st, p := colorsDeal(catalog)
			input := OfferOrderLine{OfferLineID: offered.Lines[0].LineID, Quantity: 1}
			switch name {
			case "ambiguous":
				offered.Lines[0].ModelSpecificationVersion = "1"
			case "stale legacy pin":
				input.ModelSpecificationVersion, input.ExteriorColor, input.InteriorColor = "1", "Blue", "Tan"
			case "missing exact version":
				offered.Lines[0].ModelSpecificationVersion = "9"
			}
			raw, _ := json.Marshal(offered)
			stored := &model.OfferVersion{ID: uuid.NewString(), Terms: raw}
			st.offers = colorsOffers{offer: &model.Offer{ID: uuid.NewString(), SupplierCompanyID: uuid.NewString()}, version: stored}
			if _, err := s.OrderFromOffer(context.Background(), p, DirectOrderInput{OfferVersionID: stored.ID, Lines: []OfferOrderLine{input}}); err == nil || st.deals.order != nil {
				t.Fatal("invalid offered selection persisted")
			}
		})
	}
}
