package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

type installmentDeals struct {
	DealRepository
	deal          *model.Deal
	invoices      []model.Invoice
	plan          *model.InstallmentPlan
	failInvoice   bool
	failInvoiceAt int
	invoiceCalls  int
	failPlan      error
	failUpdate    error
	updateCalls   int
	evidence      map[string][]model.Evidence
}

func (r *installmentDeals) Deal(_ context.Context, companyID, id string) (*model.Deal, error) {
	if r.deal.CompanyID != companyID || r.deal.ID != id {
		return nil, apperr.ErrNotFound
	}
	return r.deal, nil
}
func (r *installmentDeals) Invoices(context.Context, string) ([]model.Invoice, error) {
	return r.invoices, nil
}
func (r *installmentDeals) InstallmentPlan(context.Context, string) (*model.InstallmentPlan, error) {
	if r.plan == nil {
		return nil, apperr.ErrNotFound
	}
	return r.plan, nil
}
func (r *installmentDeals) CreateInstallmentPlan(_ context.Context, p *model.InstallmentPlan) error {
	if r.failPlan != nil {
		return r.failPlan
	}
	if r.plan != nil {
		return apperr.ErrConflict
	}
	r.plan = p
	return nil
}
func (r *installmentDeals) CreateInvoice(_ context.Context, i *model.Invoice) error {
	r.invoiceCalls++
	if r.failInvoice || r.invoiceCalls == r.failInvoiceAt {
		return errors.New("insert failed")
	}
	for _, existing := range r.invoices {
		if existing.Status == "issued" && existing.Purpose == i.Purpose && i.Purpose != "monthly-installment" {
			return apperr.ErrConflict
		}
	}
	r.invoices = append(r.invoices, *i)
	return nil
}
func (r *installmentDeals) Update(_ context.Context, d *model.Deal, expected int64) error {
	r.updateCalls++
	if r.failUpdate != nil {
		return r.failUpdate
	}
	if d.Version != expected {
		return apperr.ErrStale
	}
	d.Version = expected + 1
	return nil
}

func (r *installmentDeals) Evidence(_ context.Context, invoiceID string) ([]model.Evidence, error) {
	return r.evidence[invoiceID], nil
}

type installmentCRM struct {
	CRMRepository
	customer *model.Customer
}

func (r installmentCRM) Customer(context.Context, string, string) (*model.Customer, error) {
	return r.customer, nil
}

type installmentEvents struct {
	EventRepository
	events   []model.Event
	failType string
}

func (r *installmentEvents) Append(_ context.Context, e *model.Event) error {
	if s := r.failType; s != "" && s == e.EventType {
		return errors.New("event append failed")
	}
	r.events = append(r.events, *e)
	return nil
}

type installmentStore struct {
	deals  *installmentDeals
	crm    installmentCRM
	events *installmentEvents
}

func (s *installmentStore) CRM() CRMRepository                       { return s.crm }
func (s *installmentStore) Listings() ListingRepository              { return nil }
func (s *installmentStore) Deals() DealRepository                    { return s.deals }
func (s *installmentStore) Events() EventRepository                  { return s.events }
func (s *installmentStore) Bind(ctx context.Context) context.Context { return ctx }
func (s *installmentStore) InTx(ctx context.Context, fn func(Store) error) error {
	// Deep copy persisted facts, including contract bytes, draft rows and invoice
	// pointer fields. Failure controls/call counters are instrumentation, not data.
	type state struct {
		Deal     model.Deal
		Plan     *model.InstallmentPlan
		Invoices []model.Invoice
		Events   []model.Event
		Evidence map[string][]model.Evidence
	}
	var old state
	raw, err := json.Marshal(state{*s.deals.deal, s.deals.plan, s.deals.invoices, s.events.events, s.deals.evidence})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &old); err != nil {
		return err
	}
	if err := fn(s); err != nil {
		*s.deals.deal = old.Deal
		s.deals.plan, s.deals.invoices, s.events.events, s.deals.evidence = old.Plan, old.Invoices, old.Events, old.Evidence
		return err
	}
	return nil
}

func installmentFixture(failInvoice bool) (*Deal, *installmentStore, *auth.Principal) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	dealID, companyID, branchID, customerID, invoiceID, actorID := "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000006"
	signed := now
	d := &model.Deal{ID: dealID, CompanyID: companyID, BranchID: branchID, CustomerID: customerID, PaymentScheme: "own-installment", PriceMinor: "3400000", Currency: "USD", Status: "delivered", Version: 7, ContractSignedOn: &signed, ContractReference: "SC-1", ContractFileIDs: []byte("[]")}
	r := &installmentDeals{deal: d, invoices: []model.Invoice{{ID: invoiceID, DealID: dealID, CompanyID: companyID, Purpose: "first-installment", AmountMinor: "800000", Currency: "USD", Status: "issued"}}, failInvoice: failInvoice}
	st := &installmentStore{deals: r, crm: installmentCRM{customer: &model.Customer{ID: customerID, DisplayName: "Buyer"}}, events: &installmentEvents{}}
	s := NewDeal(NewDeps(st, nil, nil, nil, func() time.Time { return now }), nil)
	p := &auth.Principal{UserID: actorID, CompanyID: companyID, Permissions: map[string]bool{model.PermDeals: true}}
	return s, st, p
}
