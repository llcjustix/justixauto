package service

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

type colorsCatalog struct {
	m     *Model
	calls int
}

func (c *colorsCatalog) Model(_ context.Context, id string) (*Model, error) {
	c.calls++
	if c.m == nil || c.m.ID != id {
		return nil, apperr.ErrNotFound
	}
	return c.m, nil
}

func colorsFixture() (*colorsCatalog, model.Terms) {
	c := &colorsCatalog{m: &Model{ID: uuid.NewString(), CurrentSpecificationVersion: "2", Specifications: []ModelSpecification{
		{Version: "1", ExteriorColors: []string{"White", "Blue"}, InteriorColors: []string{"Black", "Cream", "Tan"}},
		{Version: "2", ExteriorColors: []string{"Red"}, InteriorColors: []string{"Gray"}},
	}}}
	terms := model.Terms{Route: "local", Lines: []model.Line{{LineID: uuid.NewString(), ModelID: c.m.ID, Quantity: 2,
		UnitPrice: money.Money{AmountMinor: "1250000", Currency: "USD"}}}}
	return c, terms
}

func TestTermsColorsPinnedSelectionAndSingletonCompatibility(t *testing.T) {
	catalog, in := colorsFixture()
	d := Deps{catalog: catalog}
	var v apperr.Validation
	legacy := d.validateTermsSelections(context.Background(), &v, in, true, nil)
	if err := v.Err(); err != nil {
		t.Fatal(err)
	}
	if l := legacy.Lines[0]; l.ModelSpecificationVersion != "2" || l.ExteriorColor != "Red" || l.InteriorColor != "Gray" {
		t.Fatalf("singleton selection: %+v", l)
	}
	in.Lines[0].ModelSpecificationVersion = "1"
	in.Lines[0].ExteriorColor, in.Lines[0].InteriorColor = " blue ", "tan"
	v = apperr.Validation{}
	pinned := d.validateTermsSelections(context.Background(), &v, in, true, nil)
	if err := v.Err(); err != nil {
		t.Fatal(err)
	}
	if l := pinned.Lines[0]; l.ModelSpecificationVersion != "1" || l.ExteriorColor != "Blue" || l.InteriorColor != "Tan" {
		t.Fatalf("historical selection used current palette: %+v", l)
	}
}

func TestTermsColorsRejectsAmbiguousAndWrongVersionSelections(t *testing.T) {
	for _, name := range []string{"missing body", "missing interior", "wrong body", "wrong interior", "missing version"} {
		t.Run(name, func(t *testing.T) {
			catalog, in := colorsFixture()
			l := &in.Lines[0]
			l.ModelSpecificationVersion, l.ExteriorColor, l.InteriorColor = "1", "Blue", "Tan"
			switch name {
			case "missing body":
				l.ExteriorColor = ""
			case "missing interior":
				l.InteriorColor = ""
			case "wrong body":
				l.ExteriorColor = "Red"
			case "wrong interior":
				l.InteriorColor = "Gray"
			case "missing version":
				l.ModelSpecificationVersion = "9"
			}
			var v apperr.Validation
			Deps{catalog: catalog}.validateTermsSelections(context.Background(), &v, in, true, nil)
			if v.Err() == nil {
				t.Fatal("invalid selection accepted")
			}
		})
	}
}

func TestTermsColorsRFQOptionalSelectionsArePinned(t *testing.T) {
	catalog, in := colorsFixture()
	s := &Deal{Deps: Deps{catalog: catalog}}
	var v apperr.Validation
	ls := s.validateRFQLines(context.Background(), &v, []model.RFQLine{{ModelID: in.Lines[0].ModelID, Quantity: 1,
		ModelSpecificationVersion: "1", ExteriorColor: "Blue"}})
	if v.Err() != nil || ls[0].ModelSpecificationVersion != "1" || ls[0].ExteriorColor != "Blue" || ls[0].InteriorColor != "" {
		t.Fatalf("RFQ optional facts: %+v, %v", ls, v.Err())
	}
	v = apperr.Validation{}
	ls = s.validateRFQLines(context.Background(), &v, []model.RFQLine{{ModelID: in.Lines[0].ModelID, Quantity: 1}})
	if v.Err() != nil || ls[0].ModelSpecificationVersion != "2" {
		t.Fatalf("unpinned RFQ: %+v, %v", ls, v.Err())
	}
}

// Small service fakes exercise persisted terms and transaction guards without
// a database or HTTP fixture. Unexpected repository calls panic via embeds.
type colorsDeals struct {
	DealRepository
	rfq      *model.RFQ
	order    *model.Order
	quotes   []model.Quotation
	addendum *model.Addendum
}

func (r *colorsDeals) RFQ(context.Context, string, string) (*model.RFQ, error) { return r.rfq, nil }
func (r *colorsDeals) UpdateRFQ(context.Context, *model.RFQ, int64) error      { return nil }
func (r *colorsDeals) AddQuotation(_ context.Context, q *model.Quotation) error {
	r.quotes = append(r.quotes, *q)
	return nil
}
func (r *colorsDeals) Quotations(context.Context, string) ([]model.Quotation, error) {
	return r.quotes, nil
}
func (r *colorsDeals) CreateOrder(_ context.Context, o *model.Order) error { r.order = o; return nil }
func (r *colorsDeals) Order(context.Context, string, string) (*model.Order, error) {
	return r.order, nil
}
func (r *colorsDeals) UpdateOrder(context.Context, *model.Order, int64) error { return nil }
func (r *colorsDeals) AddAddendum(_ context.Context, a *model.Addendum) error {
	r.addendum = a
	return nil
}

type colorsPartners struct {
	PartnershipRepository
	active bool
}

func (r colorsPartners) ActiveBetween(context.Context, string, string) (bool, error) {
	return r.active, nil
}

type colorsEvents struct{ EventRepository }

func (colorsEvents) Append(context.Context, *model.Event) error { return nil }

type colorsStore struct {
	Store
	deals  *colorsDeals
	offers OfferRepository
	active bool
}

func (s *colorsStore) InTx(ctx context.Context, fn func(Store) error) error { return fn(s) }
func (s *colorsStore) Deals() DealRepository                                { return s.deals }
func (s *colorsStore) Offers() OfferRepository                              { return s.offers }
func (s *colorsStore) Partnerships() PartnershipRepository                  { return colorsPartners{active: s.active} }
func (s *colorsStore) Events() EventRepository                              { return colorsEvents{} }
func colorsDeal(catalog Catalog) (*Deal, *colorsStore, *auth.Principal) {
	p := &auth.Principal{CompanyID: uuid.NewString(), UserID: uuid.NewString()}
	st := &colorsStore{deals: &colorsDeals{}, active: true}
	d := NewDeps(st, nil, catalog, nil, nil, func() time.Time { return time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC) })
	return NewDeal(d, nil), st, p
}

func TestOrderDirectColorsPersistsPairAndPartnerGuard(t *testing.T) {
	catalog, terms := colorsFixture()
	s, st, p := colorsDeal(catalog)
	in := DirectOrderInput{SupplierCompanyID: uuid.NewString(), Terms: &terms}
	o, err := s.OrderDirect(context.Background(), p, in)
	if err != nil {
		t.Fatal(err)
	}
	l := o.DecodeTerms().Lines[0]
	if l.ModelSpecificationVersion != "2" || l.ExteriorColor != "Red" || l.InteriorColor != "Gray" || l.UnitPrice != terms.Lines[0].UnitPrice {
		t.Fatalf("saved direct line: %+v", l)
	}
	st.active = false
	if _, err := s.OrderDirect(context.Background(), p, in); err == nil {
		t.Fatal("inactive partnership accepted")
	}
}

func TestQuotationColorsNewQuotePinsButAcceptedLegacyBytesStayExact(t *testing.T) {
	catalog, terms := colorsFixture()
	s, st, supplier := colorsDeal(catalog)
	buyer := &auth.Principal{CompanyID: uuid.NewString(), UserID: uuid.NewString()}
	st.deals.rfq = &model.RFQ{ID: uuid.NewString(), BuyerCompanyID: buyer.CompanyID, SupplierCompanyID: supplier.CompanyID, Status: model.RFQSent, Version: 3}
	if _, err := s.Quote(context.Background(), supplier, st.deals.rfq.ID, 3, terms); err != nil {
		t.Fatal(err)
	}
	var saved model.Terms
	q := st.deals.quotes[0]
	if err := json.Unmarshal(q.Terms, &saved); err != nil {
		t.Fatal(err)
	}
	if l := saved.Lines[0]; l.ModelSpecificationVersion != "2" || l.ExteriorColor != "Red" || l.InteriorColor != "Gray" || q.Digest != model.Digest(q.Terms) {
		t.Fatalf("new quote: %+v", q)
	}
	// Deliberately noncanonical old JSON: acceptance must not re-encode it.
	raw := []byte(`{ "route":"local", "lines":[{"lineId":"old","modelId":"unknown","quantity":"1","unitPrice":{"amountMinor":"10","currency":"USD"}}] }`)
	st.deals.quotes = []model.Quotation{{ID: uuid.NewString(), Terms: raw, Digest: model.Digest(raw)}}
	q = st.deals.quotes[0]
	catalog.calls = 0
	o, err := s.Accept(context.Background(), buyer, st.deals.rfq.ID, 3, q.ID, q.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(o.Terms, raw) || st.deals.quotes[0].Digest != q.Digest || catalog.calls != 0 {
		t.Fatalf("legacy acceptance changed bytes/digest or queried catalog: %s", o.Terms)
	}
}

func TestQuotationColorsAmbiguousOmissionRejectedBeforeWrite(t *testing.T) {
	catalog, in := colorsFixture()
	in.Lines[0].ModelSpecificationVersion = "1"
	s, st, p := colorsDeal(catalog)
	if _, err := s.Quote(context.Background(), p, uuid.NewString(), 1, in); err == nil || len(st.deals.quotes) != 0 {
		t.Fatal("ambiguous quotation reached persistence")
	}
}

func TestAddendumColorsUnrelatedEditPreservesKnownAndUnknownFacts(t *testing.T) {
	for _, name := range []string{"known", "unknown", "partial legacy"} {
		t.Run(name, func(t *testing.T) {
			catalog, original := colorsFixture()
			if name == "known" {
				original.Lines[0].ModelSpecificationVersion = "1"
				original.Lines[0].ExteriorColor, original.Lines[0].InteriorColor = "Blue", "Tan"
			} else if name == "partial legacy" {
				original.Lines[0].ExteriorColor = "White"
			}
			original.Lines[0].OfferLineID = uuid.NewString()
			raw, _ := json.Marshal(original)
			s, st, p := colorsDeal(catalog)
			st.deals.order = &model.Order{ID: uuid.NewString(), Terms: raw, Status: model.OrderAccepted, Version: 7}
			in := original
			in.Lines = append([]model.Line(nil), original.Lines...)
			in.Lines[0].ModelSpecificationVersion, in.Lines[0].ExteriorColor, in.Lines[0].InteriorColor, in.Lines[0].OfferLineID = "", "", "", ""
			in.DeliveryTerms = "Ship next week"
			in.Lines[0].UnitPrice.AmountMinor = "1400000"
			if _, err := s.ProposeAddendum(context.Background(), p, st.deals.order.ID, 7, in, "Change delivery and price"); err != nil {
				t.Fatal(err)
			}
			var saved model.Terms
			if err := json.Unmarshal(st.deals.addendum.Terms, &saved); err != nil {
				t.Fatal(err)
			}
			l, old := saved.Lines[0], original.Lines[0]
			if l.ModelSpecificationVersion != old.ModelSpecificationVersion || l.ExteriorColor != old.ExteriorColor || l.InteriorColor != old.InteriorColor || l.OfferLineID != old.OfferLineID || catalog.calls != 0 {
				t.Fatalf("history repinned: old %+v, new %+v, catalog calls %d", old, l, catalog.calls)
			}
			if !bytes.Equal(st.deals.order.Terms, raw) || l.UnitPrice.AmountMinor != "1400000" {
				t.Fatal("proposal mutated order or lost price edit")
			}
		})
	}
}
