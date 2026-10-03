package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

// Cash invoice creation and checklist reads do not lock an evidence invoice.
// Keep the shared payment fixture's strict monthly assertions unchanged.
type cashDeals struct {
	*paymentDeals
	failRead error
}

func (r *cashDeals) Evidence(_ context.Context, id string) ([]model.Evidence, error) {
	if r.failRead != nil {
		return nil, r.failRead
	}
	var out []model.Evidence
	for _, e := range r.evidence {
		if e.InvoiceID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

type cashStock struct {
	Stock
	repo      *paymentDeals
	delivered bool
	fail      bool
	dealID    string
	vehicleID string
	actorID   string
	at        time.Time
}

func (s *cashStock) Deliver(ctx context.Context, dealID, vehicleID, actorID string, at time.Time) error {
	if ctx.Value(paymentTxKey{}) != s.repo || !s.repo.inTx {
		panic("stock delivery outside bound transaction")
	}
	s.delivered, s.dealID, s.vehicleID, s.actorID, s.at = true, dealID, vehicleID, actorID, at
	if s.fail {
		return errors.New("stock failed")
	}
	return nil
}

type cashListings struct {
	ListingRepository
	withdrawn bool
	fail      bool
}

func (r *cashListings) WithdrawForVehicle(context.Context, string, time.Time) error {
	r.withdrawn = true
	if r.fail {
		return errors.New("listing failed")
	}
	return nil
}

type cashCRM struct {
	installmentCRM
	lead model.Lead
	fail bool
}

func (r *cashCRM) Lead(_ context.Context, company, id string) (*model.Lead, error) {
	if r.lead.CompanyID != company || r.lead.ID != id {
		return nil, apperr.ErrNotFound
	}
	l := r.lead
	return &l, nil
}

func (r *cashCRM) UpdateLead(_ context.Context, l *model.Lead, expected int64) error {
	if r.fail {
		return errors.New("lead failed")
	}
	if r.lead.Version != expected {
		return apperr.ErrStale
	}
	r.lead = *l
	r.lead.Version++
	return nil
}

type cashStore struct {
	*paymentStore
	cash     *cashDeals
	stock    *cashStock
	listings *cashListings
	crm      *cashCRM
}

func (s *cashStore) Deals() DealRepository       { return s.cash }
func (s *cashStore) Listings() ListingRepository { return s.listings }
func (s *cashStore) CRM() CRMRepository          { return s.crm }
func (s *cashStore) InTx(ctx context.Context, fn func(Store) error) error {
	delivered, withdrawn, lead := s.stock.delivered, s.listings.withdrawn, s.crm.lead
	// The two shared snapshots cover invoice/plan and payment/evidence state.
	err := s.installmentStore.InTx(ctx, func(Store) error {
		return s.paymentStore.InTx(ctx, func(Store) error { return fn(s) })
	})
	if err != nil {
		s.stock.delivered, s.listings.withdrawn, s.crm.lead = delivered, withdrawn, lead
	}
	return err
}

func cashFixture() (*Deal, *cashStore, *auth.Principal) {
	s, base, p := installmentFixture(false)
	d := base.deals.deal
	d.PaymentScheme, d.Status, d.PriceMinor = "cash", "reserved", "100"
	d.VehicleID = "00000000-0000-0000-0000-000000000010"
	leadID := "00000000-0000-0000-0000-000000000011"
	d.LeadID = &leadID
	base.deals.invoices = nil
	repo := &paymentDeals{installmentDeals: base.deals}
	st := &cashStore{
		paymentStore: &paymentStore{installmentStore: base, repo: repo},
		cash:         &cashDeals{paymentDeals: repo},
		stock:        &cashStock{repo: repo},
		listings:     &cashListings{},
		crm:          &cashCRM{installmentCRM: base.crm, lead: model.Lead{ID: leadID, CompanyID: p.CompanyID, Stage: "negotiation", Version: 1}},
	}
	s.store, s.stock, s.insurance = st, st.stock, paymentInsurance{}
	p.Permissions[model.PermPaymentsAccept] = true
	return s, st, p
}

func cashInput(amount, currency string) InvoiceInput {
	return InvoiceInput{Purpose: "vehicle-payment", Amount: money.Money{AmountMinor: amount, Currency: currency}, RecipientSnapshot: "Seller"}
}

func cashSeed(st *cashStore, purpose, value, currency string) string {
	id := "00000000-0000-0000-0000-000000000012"
	if purpose != "vehicle-payment" {
		id = "00000000-0000-0000-0000-000000000013"
	}
	if purpose == "registration" {
		id = "00000000-0000-0000-0000-000000000014"
	}
	d := st.repo.deal
	st.repo.invoices = append(st.repo.invoices, model.Invoice{ID: id, DealID: d.ID, CompanyID: d.CompanyID, Purpose: purpose, AmountMinor: value, Currency: currency, Status: "issued", Version: 1})
	return id
}

func cashAccepted(st *cashStore, id, value string) {
	st.repo.evidence = append(st.repo.evidence, model.Evidence{InvoiceID: id, AmountMinor: value, Currency: "USD", Status: "accepted"})
}

func cashState(t *testing.T, st *cashStore) string {
	t.Helper()
	raw, err := json.Marshal(struct {
		Deal      *model.Deal
		Invoices  []model.Invoice
		Evidence  []model.Evidence
		Events    []model.Event
		Lead      model.Lead
		Delivered bool
		Withdrawn bool
	}{st.repo.deal, st.repo.invoices, st.repo.evidence, st.events.events, st.crm.lead, st.stock.delivered, st.listings.withdrawn})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func cashUnchanged(t *testing.T, st *cashStore, before string) {
	t.Helper()
	if after := cashState(t, st); after != before {
		t.Fatalf("persisted facts changed on failure\nbefore %s\nafter %s", before, after)
	}
}

func TestCashSaleInvoiceExactPrice(t *testing.T) {
	for _, value := range []string{"100", "9007199254740993", strings.Repeat("9", 38)} {
		t.Run(value, func(t *testing.T) {
			s, st, p := cashFixture()
			st.repo.deal.PriceMinor = value
			v, err := s.IssueInvoice(context.Background(), p, st.repo.deal.ID, 7, cashInput(value, "USD"))
			if err != nil {
				t.Fatal(err)
			}
			if v.Invoice.AmountMinor != value || v.Invoice.Currency != "USD" || len(st.repo.invoices) != 1 || st.repo.deal.Version != 8 || len(st.events.events) != 1 || st.events.events[0].EventType != "deal.invoice_issued" {
				t.Fatalf("invoice/version/audit mismatch: %#v", v)
			}
			before := cashState(t, st)
			if _, err := s.IssueInvoice(context.Background(), p, st.repo.deal.ID, 8, cashInput(value, "USD")); !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("duplicate invoice: %v", err)
			}
			cashUnchanged(t, st, before)
		})
	}
}

func TestCashSaleInvoiceRejectsInvalidAmounts(t *testing.T) {
	for _, value := range []string{"99", "101", "0", "-100", "+100", "0100", "1.00", "", " 100", strings.Repeat("9", 39)} {
		t.Run(value, func(t *testing.T) {
			s, st, p := cashFixture()
			before := cashState(t, st)
			if _, err := s.IssueInvoice(context.Background(), p, st.repo.deal.ID, 7, cashInput(value, "USD")); err == nil {
				t.Fatal("invalid cash amount accepted")
			}
			cashUnchanged(t, st, before)
			if st.repo.invoiceCalls != 0 || st.repo.updateCalls != 0 {
				t.Fatal("rejection reached writes")
			}
		})
	}
	for _, currency := range []string{"EUR", "usd", "", "US"} {
		t.Run("currency-"+currency, func(t *testing.T) {
			s, st, p := cashFixture()
			before := cashState(t, st)
			if _, err := s.IssueInvoice(context.Background(), p, st.repo.deal.ID, 7, cashInput("100", currency)); err == nil {
				t.Fatal("wrong currency accepted")
			}
			cashUnchanged(t, st, before)
		})
	}
}

func TestCashSaleInvoiceGuardsAndRollback(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cashStore, *auth.Principal)
	}{
		{"permission", func(_ *cashStore, p *auth.Principal) { delete(p.Permissions, model.PermDeals) }},
		{"company", func(_ *cashStore, p *auth.Principal) { p.CompanyID = "other" }},
		{"branch", func(_ *cashStore, p *auth.Principal) { p.BranchScope = auth.BranchScope{Mode: "SELECTED"} }},
		{"stale", func(st *cashStore, _ *auth.Principal) { st.repo.deal.Version++ }},
		{"closed", func(st *cashStore, _ *auth.Principal) { st.repo.deal.Status = "delivered" }},
		{"invoice insert", func(st *cashStore, _ *auth.Principal) { st.repo.failInvoice = true }},
		{"deal update", func(st *cashStore, _ *auth.Principal) { st.repo.failUpdate = errors.New("update failed") }},
		{"audit append", func(st *cashStore, _ *auth.Principal) { st.events.failType = "deal.invoice_issued" }},
		{"projection read", func(st *cashStore, _ *auth.Principal) { st.cash.failRead = errors.New("evidence read failed") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, st, p := cashFixture()
			tt.setup(st, p)
			before := cashState(t, st)
			if _, err := s.IssueInvoice(context.Background(), p, st.repo.deal.ID, 7, cashInput("100", "USD")); err == nil {
				t.Fatal("guard/failure did not abort invoice")
			}
			cashUnchanged(t, st, before)
		})
	}
}

func TestCashSaleOptionalRegistrationRetainsRecordingRules(t *testing.T) {
	s, st, p := cashFixture()
	ctx := context.Background()
	before := cashState(t, st)
	if _, err := s.RecordRegistration(ctx, p, st.repo.deal.ID, 7, "2026-09-30", "01A123AA", "REG"); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("unpaid registration: %v", err)
	}
	cashUnchanged(t, st, before)
	in := cashInput("37", "USD")
	in.Purpose = "registration"
	v, err := s.IssueInvoice(ctx, p, st.repo.deal.ID, 7, in)
	if err != nil {
		t.Fatal(err)
	}
	cashAccepted(st, v.Invoice.ID, "37")
	if _, err := s.RecordRegistration(ctx, p, st.repo.deal.ID, 8, "2026-09-30", "01A123AA", "REG"); err != nil {
		t.Fatal(err)
	}
	c, err := s.checklist(ctx, st, st.repo.deal)
	if err != nil || !c.RegistrationOptional || !c.Registered || !c.RegistrationPaid || c.Ready() {
		t.Fatalf("optional registration facts cannot replace vehicle payment: %+v, %v", c, err)
	}
	in.Purpose = "first-installment"
	before = cashState(t, st)
	if _, err := s.IssueInvoice(ctx, p, st.repo.deal.ID, 9, in); err == nil {
		t.Fatal("cash accepted an installment purpose")
	}
	cashUnchanged(t, st, before)
}

func TestCashSaleLegacySettlement(t *testing.T) {
	tests := []struct {
		name   string
		change func(*cashStore)
		paid   bool
	}{
		{"exact", func(*cashStore) {}, true},
		{"underpriced", func(st *cashStore) { st.repo.invoices[0].AmountMinor = "99"; st.repo.evidence[0].AmountMinor = "99" }, false},
		{"overpriced", func(st *cashStore) { st.repo.invoices[0].AmountMinor = "101"; st.repo.evidence[0].AmountMinor = "101" }, false},
		{"foreign invoice", func(st *cashStore) { st.repo.invoices[0].Currency = "EUR" }, false},
		{"foreign accepted evidence", func(st *cashStore) { st.repo.evidence[0].Currency = "EUR" }, false},
		{"wrong company", func(st *cashStore) { st.repo.invoices[0].CompanyID = "other" }, false},
		{"wrong deal", func(st *cashStore) { st.repo.invoices[0].DealID = "other" }, false},
		{"overpayment", func(st *cashStore) { st.repo.evidence[0].AmountMinor = "101" }, false},
		{"partial", func(st *cashStore) { st.repo.evidence[0].AmountMinor = "60" }, false},
		{"multiple accepted partials", func(st *cashStore) { st.repo.evidence[0].AmountMinor = "60"; cashAccepted(st, st.repo.invoices[0].ID, "40") }, true},
		{"submitted only", func(st *cashStore) { st.repo.evidence[0].Status = "submitted" }, false},
		{"rejected only", func(st *cashStore) { st.repo.evidence[0].Status = "rejected" }, false},
		{"nonaccepted ignored", func(st *cashStore) { st.repo.evidence = append(st.repo.evidence, model.Evidence{InvoiceID: st.repo.invoices[0].ID, AmountMinor: "bad", Currency: "EUR", Status: "submitted"}, model.Evidence{InvoiceID: st.repo.invoices[0].ID, AmountMinor: "999", Currency: "EUR", Status: "rejected"}) }, true},
		{"none issued", func(st *cashStore) { st.repo.invoices[0].Status = "voided" }, false},
		{"no evidence", func(st *cashStore) { st.repo.evidence = nil }, false},
		{"duplicate issued", func(st *cashStore) { st.repo.invoices = append(st.repo.invoices, st.repo.invoices[0]) }, false},
		{"voided ignored", func(st *cashStore) { old := st.repo.invoices[0]; old.ID = "old"; old.Status = "voided"; old.AmountMinor = "bad"; st.repo.invoices = append(st.repo.invoices, old) }, true},
		{"registration cannot settle vehicle", func(st *cashStore) { st.repo.invoices[0].Purpose = "registration" }, false},
		{"invalid price currency", func(st *cashStore) { st.repo.deal.Currency = "usd" }, false},
		{"large exact", func(st *cashStore) { value := strings.Repeat("9", 38); st.repo.deal.PriceMinor = value; st.repo.invoices[0].AmountMinor = value; st.repo.evidence[0].AmountMinor = value }, true},
	}
	for _, target := range []string{"price", "invoice", "evidence"} {
		for _, value := range []string{"", "0", "-100", "+100", "0100", "1.00", "abc", strings.Repeat("9", 39)} {
			target, value := target, value
			tests = append(tests, struct {
				name   string
				change func(*cashStore)
				paid   bool
			}{target + "-" + value, func(st *cashStore) {
				switch target {
				case "price":
					st.repo.deal.PriceMinor = value
				case "invoice":
					st.repo.invoices[0].AmountMinor = value
				case "evidence":
					// Also seed the full valid sum: invalid evidence must not be
					// coerced to zero and silently ignored.
					cashAccepted(st, st.repo.invoices[0].ID, value)
				}
			}, false})
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, st, p := cashFixture()
			id := cashSeed(st, "vehicle-payment", "100", "USD")
			cashAccepted(st, id, "100")
			tt.change(st)
			before := cashState(t, st)
			c, err := s.checklist(context.Background(), st, st.repo.deal)
			if err != nil || c.VehiclePayment == nil || *c.VehiclePayment != tt.paid || c.Ready() != tt.paid || !c.RegistrationOptional {
				t.Fatalf("cash readiness: %+v, %v; want paid=%v", c, err, tt.paid)
			}
			cashUnchanged(t, st, before)
			if !tt.paid {
				if _, err := s.Deliver(context.Background(), p, st.repo.deal.ID, 7, s.clock()); !errors.Is(err, apperr.ErrConflict) {
					t.Fatalf("inconsistent/unsettled sale delivered: %v", err)
				}
				cashUnchanged(t, st, before)
			}
		})
	}
}

func TestCashSalePartialPaymentLifecycleAndDelivery(t *testing.T) {
	s, st, p := cashFixture()
	ctx := context.Background()
	v, err := s.IssueInvoice(ctx, p, st.repo.deal.ID, 7, cashInput("100", "USD"))
	if err != nil {
		t.Fatal(err)
	}
	id := v.Invoice.ID
	assertReady := func(want bool) {
		t.Helper()
		c, err := s.checklist(ctx, st, st.repo.deal)
		if err != nil || c.Ready() != want || !c.RegistrationOptional || c.Registered || c.RegistrationPaid {
			t.Fatalf("readiness %+v, %v; want %v", c, err, want)
		}
	}
	v, err = s.SubmitEvidence(ctx, p, id, paymentInput("60", "first"))
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "0", "60", "100", "40")
	assertReady(false)
	before := cashState(t, st)
	if _, err := s.SubmitEvidence(ctx, p, id, paymentInput("41", "too much")); err == nil {
		t.Fatal("pending evidence did not reserve availability")
	}
	cashUnchanged(t, st, before)
	v, err = s.DecideEvidence(ctx, p, v.Evidence[0].ID, 1, true, true, "")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "60", "0", "40", "40")
	assertReady(false)
	if !reflect.DeepEqual(v.AllowedActions, []string{"submit-payment"}) {
		t.Fatalf("remaining balance lost payment action: %v", v.AllowedActions)
	}
	v, err = s.SubmitEvidence(ctx, p, id, paymentInput("40", "second"))
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "60", "40", "40", "0")
	assertReady(false)
	v, err = s.DecideEvidence(ctx, p, v.Evidence[1].ID, 1, false, false, "not received")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "60", "0", "40", "40")
	assertReady(false)
	v, err = s.SubmitEvidence(ctx, p, id, paymentInput("40", "replacement"))
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.DecideEvidence(ctx, p, v.Evidence[2].ID, 1, true, true, "")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "100", "0", "0", "0")
	assertReady(true)
	if len(v.AllowedActions) != 0 || len(st.repo.invoices) != 1 || len(st.repo.locks) != 7 {
		t.Fatalf("settled actions/invoice/locks: %v, %d, %v", v.AllowedActions, len(st.repo.invoices), st.repo.locks)
	}
	before = cashState(t, st)
	if _, err := s.SubmitEvidence(ctx, p, id, paymentInput("1", "overpaid")); err == nil {
		t.Fatal("settled invoice accepted more evidence")
	}
	cashUnchanged(t, st, before)
	at := s.clock().Add(-time.Hour)
	d, err := s.Deliver(ctx, p, st.repo.deal.ID, 8, at)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "delivered" || d.Version != 9 || d.DeliveredAt == nil || !d.DeliveredAt.Equal(at) || !st.stock.delivered || !st.listings.withdrawn || st.crm.lead.Stage != "won" || st.crm.lead.Version != 2 {
		t.Fatalf("delivery facts: %+v", d)
	}
	if st.stock.dealID != d.ID || st.stock.vehicleID != d.VehicleID || st.stock.actorID != p.UserID || !st.stock.at.Equal(at) {
		t.Fatal("stock delivery arguments changed")
	}
	events := st.events.events
	if events[len(events)-2].EventType != "lead.won" || events[len(events)-1].EventType != "deal.delivered" {
		t.Fatal("handover audit missing")
	}
}

func TestCashSaleDeliveryGuardsAndRollback(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cashStore, *auth.Principal)
	}{
		{"contract", func(st *cashStore, _ *auth.Principal) { st.repo.deal.ContractSignedOn = nil }},
		{"company", func(_ *cashStore, p *auth.Principal) { p.CompanyID = "other" }},
		{"branch", func(_ *cashStore, p *auth.Principal) { p.BranchScope = auth.BranchScope{Mode: "SELECTED"} }},
		{"stale", func(st *cashStore, _ *auth.Principal) { st.repo.deal.Version++ }},
		{"closed", func(st *cashStore, _ *auth.Principal) { st.repo.deal.Status = "cancelled" }},
		{"evidence read", func(st *cashStore, _ *auth.Principal) { st.cash.failRead = errors.New("read failed") }},
		{"stock", func(st *cashStore, _ *auth.Principal) { st.stock.fail = true }},
		{"deal update", func(st *cashStore, _ *auth.Principal) { st.repo.failUpdate = errors.New("update failed") }},
		{"listing", func(st *cashStore, _ *auth.Principal) { st.listings.fail = true }},
		{"lead", func(st *cashStore, _ *auth.Principal) { st.crm.fail = true }},
		{"lead audit", func(st *cashStore, _ *auth.Principal) { st.events.failType = "lead.won" }},
		{"delivery audit", func(st *cashStore, _ *auth.Principal) { st.events.failType = "deal.delivered" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, st, p := cashFixture()
			cashAccepted(st, cashSeed(st, "vehicle-payment", "100", "USD"), "100")
			tt.setup(st, p)
			before := cashState(t, st)
			if _, err := s.Deliver(context.Background(), p, st.repo.deal.ID, 7, s.clock()); err == nil {
				t.Fatal("delivery guard/failure was ignored")
			}
			cashUnchanged(t, st, before)
		})
	}
}

type cashInsurance bool

func (a cashInsurance) Approved(context.Context, string, string) (bool, error) { return bool(a), nil }

func TestCashSaleOtherSchemesRemainStrict(t *testing.T) {
	for _, scenario := range []string{"ready", "registration fee", "registration fact", "first installment", "insurance", "partner policy"} {
		t.Run(scenario, func(t *testing.T) {
			s, st, p := cashFixture()
			st.repo.deal.PaymentScheme = "own-installment"
			now := s.clock()
			st.repo.deal.RegisteredOn = &now
			first := cashSeed(st, "first-installment", "20", "USD")
			cashAccepted(st, first, "20")
			registration := cashSeed(st, "registration", "7", "USD")
			cashAccepted(st, registration, "7")
			switch scenario {
			case "registration fee":
				st.repo.evidence[1].Status = "submitted"
			case "registration fact":
				st.repo.deal.RegisteredOn = nil
			case "first installment":
				st.repo.evidence[0].Status = "submitted"
			case "insurance":
				s.insurance = cashInsurance(false)
			case "partner policy":
				st.repo.deal.PaymentScheme = "partner-finance"
			}
			c, err := s.checklist(context.Background(), st, st.repo.deal)
			if err != nil || c.RegistrationOptional || c.Ready() != (scenario == "ready") {
				t.Fatalf("scheme guards: %+v, %v", c, err)
			}
			if scenario != "ready" {
				before := cashState(t, st)
				_, err := s.Deliver(context.Background(), p, st.repo.deal.ID, 7, now)
				want := apperr.ErrConflict
				if scenario == "partner policy" {
					want = apperr.ErrForbidden
				}
				if !errors.Is(err, want) {
					t.Fatalf("handover guard: %v, want %v", err, want)
				}
				cashUnchanged(t, st, before)
			}
		})
	}
}
