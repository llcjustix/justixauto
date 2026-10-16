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

const (
	contractCompany = "10000000-0000-0000-0000-000000000001"
	contractBranch  = "10000000-0000-0000-0000-000000000002"
	contractDealID  = "10000000-0000-0000-0000-000000000003"
	contractDraftID = "10000000-0000-0000-0000-000000000004"
	contractActor   = "10000000-0000-0000-0000-000000000005"
	contractFile    = "10000000-0000-0000-0000-000000000006"
)

type contractRepo struct {
	ContractRepository
	rows      map[string]model.ContractVersion
	templates map[string]model.ContractTemplate
	failSign  bool
}

func (r *contractRepo) Template(_ context.Context, company, id string) (*model.ContractTemplate, error) {
	v, ok := r.templates[id]
	if !ok || v.CompanyID != company {
		return nil, apperr.ErrNotFound
	}
	return &v, nil
}

func (r *contractRepo) CreateContract(_ context.Context, v *model.ContractVersion) error {
	r.rows[v.ID] = *v
	return nil
}

func (r *contractRepo) UpdateDraft(_ context.Context, v *model.ContractVersion, expected int64) error {
	old, ok := r.rows[v.ID]
	if !ok || old.State != "draft" || old.Revision != expected {
		return apperr.ErrStale
	}
	v.Revision = expected + 1
	r.rows[v.ID] = *v
	return nil
}

func (r *contractRepo) Contract(_ context.Context, company, deal, id string) (*model.ContractVersion, error) {
	v, ok := r.rows[id]
	if !ok || v.CompanyID != company || v.DealID != deal {
		return nil, apperr.ErrNotFound
	}
	return &v, nil
}

func (r *contractRepo) Contracts(_ context.Context, company, deal string) ([]model.ContractVersion, error) {
	out := []model.ContractVersion{}
	for _, v := range r.rows {
		if v.CompanyID == company && v.DealID == deal {
			out = append(out, v)
		}
	}
	return out, nil
}

func (r *contractRepo) SignDraft(_ context.Context, v *model.ContractVersion, expected int64) error {
	if r.failSign {
		return apperr.ErrStale
	}
	old, ok := r.rows[v.ID]
	if !ok || old.CompanyID != v.CompanyID || old.DealID != v.DealID || old.State != "draft" || old.Revision != expected {
		return apperr.ErrStale
	}
	v.Revision = expected + 1
	r.rows[v.ID] = *v
	return nil
}

type contractDeals struct {
	DealRepository
	deal       *model.Deal
	failUpdate bool
	plan       *model.InstallmentPlan
	invoices   []model.Invoice
}

func (r *contractDeals) CreateInstallmentPlan(_ context.Context, plan *model.InstallmentPlan) error {
	if r.plan != nil {
		return apperr.ErrConflict
	}
	r.plan = plan
	return nil
}

func (r *contractDeals) CreateInvoice(_ context.Context, invoice *model.Invoice) error {
	r.invoices = append(r.invoices, *invoice)
	return nil
}

func (r *contractDeals) Deal(_ context.Context, company, id string) (*model.Deal, error) {
	if r.deal.CompanyID != company || r.deal.ID != id {
		return nil, apperr.ErrNotFound
	}
	return r.deal, nil
}

func (r *contractDeals) Update(_ context.Context, d *model.Deal, expected int64) error {
	if r.failUpdate || d.Version != expected {
		return apperr.ErrStale
	}
	d.Version = expected + 1
	return nil
}

func (r *contractDeals) InstallmentPlan(context.Context, string) (*model.InstallmentPlan, error) {
	if r.plan == nil {
		return nil, apperr.ErrNotFound
	}
	return r.plan, nil
}

func (r *contractDeals) Invoices(context.Context, string) ([]model.Invoice, error) {
	return r.invoices, nil
}

type contractCRM struct {
	CRMRepository
	customer *model.Customer
}

func (r contractCRM) Customer(_ context.Context, company, id string) (*model.Customer, error) {
	if r.customer.CompanyID != company || r.customer.ID != id {
		return nil, apperr.ErrNotFound
	}
	return r.customer, nil
}

type contractEvents struct {
	EventRepository
	events []model.Event
	fail   bool
}

func (r *contractEvents) Append(_ context.Context, e *model.Event) error {
	if r.fail {
		return errors.New("event failure")
	}
	r.events = append(r.events, *e)
	return nil
}

type contractFiles struct {
	shares          []string
	fail            bool
	sawBoundContext bool
}
type contractBoundKey struct{}

func (f *contractFiles) Share(ctx context.Context, _, file, _, resourceType, resourceID string) error {
	if ctx.Value(contractBoundKey{}) == true {
		f.sawBoundContext = true
	}
	if f.fail {
		return errors.New("share failure")
	}
	f.shares = append(f.shares, file+":"+resourceType+":"+resourceID)
	return nil
}

func (*contractFiles) BindPassport(context.Context, string, string, string, string) error { return nil }

type contractCompanyPort struct{ Company }

func (contractCompanyPort) ContractRequisitesFor(_ context.Context, authorized, company string) (*CompanyRequisites, error) {
	if authorized != contractCompany || company != contractCompany {
		return nil, apperr.ErrNotFound
	}
	return &CompanyRequisites{ID: company, Name: "Seller"}, nil
}

type contractStore struct {
	contracts *contractRepo
	scoring   *scoringRepo
	deals     *contractDeals
	crm       contractCRM
	events    *contractEvents
	files     *contractFiles
}

func (s *contractStore) CRM() CRMRepository            { return s.crm }
func (*contractStore) Listings() ListingRepository     { return nil }
func (s *contractStore) Deals() DealRepository         { return s.deals }
func (s *contractStore) Contracts() ContractRepository { return s.contracts }
func (s *contractStore) Scoring() ScoringRepository    { return s.scoring }
func (s *contractStore) Events() EventRepository       { return s.events }
func (*contractStore) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, contractBoundKey{}, true)
}

func (s *contractStore) InTx(_ context.Context, fn func(Store) error) error {
	type state struct {
		Rows     map[string]model.ContractVersion
		Scoring  []model.DemoScoring
		Deal     model.Deal
		Plan     *model.InstallmentPlan
		Invoices []model.Invoice
		Events   []model.Event
		Shares   []string
		Bound    bool
	}
	raw, err := json.Marshal(state{s.contracts.rows, s.scoring.runs, *s.deals.deal, s.deals.plan, s.deals.invoices, s.events.events, s.files.shares, s.files.sawBoundContext})
	if err != nil {
		return err
	}
	var before state
	if err := json.Unmarshal(raw, &before); err != nil {
		return err
	}
	if err := fn(s); err != nil {
		s.contracts.rows, s.scoring.runs, *s.deals.deal, s.deals.plan, s.deals.invoices, s.events.events, s.files.shares, s.files.sawBoundContext = before.Rows, before.Scoring, before.Deal, before.Plan, before.Invoices, before.Events, before.Shares, before.Bound
		return err
	}
	return nil
}

func contractFixture() (*Contract, *contractStore, *auth.Principal) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	draft := model.ContractVersion{ID: contractDraftID, CompanyID: contractCompany, DealID: contractDealID, Kind: "contract", State: "draft", Content: "content", PartySnapshot: []byte("{}"), TermsSnapshot: []byte(`{"price":{"amountMinor":"","currency":""},"paymentScheme":""}`), FileIDs: []byte("[]"), Revision: 1}
	d := &model.Deal{ID: contractDealID, CompanyID: contractCompany, BranchID: contractBranch, CustomerID: "10000000-0000-0000-0000-000000000007", Version: 7}
	files := &contractFiles{}
	st := &contractStore{contracts: &contractRepo{rows: map[string]model.ContractVersion{draft.ID: draft}, templates: map[string]model.ContractTemplate{}}, scoring: &scoringRepo{}, deals: &contractDeals{deal: d}, crm: contractCRM{customer: &model.Customer{ID: d.CustomerID, CompanyID: contractCompany, DisplayName: "Buyer"}}, events: &contractEvents{}, files: files}
	p := &auth.Principal{UserID: contractActor, CompanyID: contractCompany, Permissions: map[string]bool{model.PermDeals: true, model.PermRead: true}, CommercialAccess: true}
	return NewContract(NewDeps(st, contractCompanyPort{}, nil, files, func() time.Time { return now })), st, p
}

func contractSignInput() SignContractInput {
	return SignContractInput{SignedOn: "2026-10-11", Reference: "SC-1", BindingIDs: []string{contractFile}, DraftRevision: "1"}
}
