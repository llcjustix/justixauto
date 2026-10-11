package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

// contractStore is deliberately copy-on-transaction: a failing command cannot
// leak a mutation through pointers retained by the fake repositories.
type contractStore struct {
	Store
	deals      *contractDeals
	contracts  *contractRepo
	fulfilment *contractFulfilment
	events     int
	failEvent  bool
	files      *contractFiles
}

func (s *contractStore) InTx(_ context.Context, fn func(Store) error) error {
	copy := s.clone()
	if err := fn(copy); err != nil {
		return err
	}
	// Keep the fixture's externally held file fake pointer stable while applying
	// its transaction-local grants only after every later operation succeeded.
	targetFiles := s.files
	*targetFiles = *copy.files
	copy.files = targetFiles
	*s = *copy
	return nil
}

type contractFilesKey struct{}

func (s *contractStore) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, contractFilesKey{}, s.files)
}
func (s *contractStore) Deals() DealRepository            { return s.deals }
func (s *contractStore) Contracts() ContractRepository    { return s.contracts }
func (s *contractStore) Fulfilment() FulfilmentRepository { return s.fulfilment }
func (s *contractStore) Events() EventRepository          { return contractEvents{s} }
func (s *contractStore) clone() *contractStore {
	return &contractStore{deals: s.deals.clone(), contracts: s.contracts.clone(), fulfilment: &contractFulfilment{rows: slices.Clone(s.fulfilment.rows)}, events: s.events, failEvent: s.failEvent, files: s.files.clone()}
}

type contractDeals struct {
	DealRepository
	order *model.Order
	fail  bool
}

func (r *contractDeals) clone() *contractDeals {
	v := *r
	if r.order != nil {
		o := *r.order
		o.Terms = slices.Clone(o.Terms)
		v.order = &o
	}
	return &v
}
func (r *contractDeals) Order(_ context.Context, company, id string) (*model.Order, error) {
	if r.order == nil || r.order.ID != id || r.order.Party(company) == "" {
		return nil, apperr.ErrNotFound
	}
	o := *r.order
	o.Terms = slices.Clone(o.Terms)
	return &o, nil
}
func (r *contractDeals) UpdateOrder(_ context.Context, o *model.Order, expected int64) error {
	if r.fail {
		return errors.New("order update")
	}
	if r.order.Version != expected {
		return apperr.ErrStale
	}
	x := *o
	x.Version = expected + 1
	x.Terms = slices.Clone(o.Terms)
	r.order = &x
	o.Version = x.Version
	return nil
}

type contractFulfilment struct {
	FulfilmentRepository
	rows []model.Allocation
}

func (r *contractFulfilment) Allocations(context.Context, string) ([]model.Allocation, error) {
	return slices.Clone(r.rows), nil
}

type contractRepo struct {
	ContractRepository
	templates map[string]*model.ContractTemplate
	rows      map[string]*model.ContractVersion
	fail      bool
}

func (r *contractRepo) clone() *contractRepo {
	n := &contractRepo{templates: map[string]*model.ContractTemplate{}, rows: map[string]*model.ContractVersion{}, fail: r.fail}
	for k, v := range r.templates {
		x := *v
		n.templates[k] = &x
	}
	for k, v := range r.rows {
		x := *v
		x.ExternalFileIDs = slices.Clone(x.ExternalFileIDs)
		x.SignedFileIDs = slices.Clone(x.SignedFileIDs)
		x.PartySnapshot = slices.Clone(x.PartySnapshot)
		x.TermsSnapshot = slices.Clone(x.TermsSnapshot)
		n.rows[k] = &x
	}
	return n
}
func (r *contractRepo) CreateTemplate(_ context.Context, t *model.ContractTemplate) error {
	r.templates[t.ID] = t
	return nil
}
func (r *contractRepo) Template(_ context.Context, c, id string) (*model.ContractTemplate, error) {
	t := r.templates[id]
	if t == nil || t.CompanyID != c {
		return nil, apperr.ErrNotFound
	}
	x := *t
	return &x, nil
}
func (r *contractRepo) Templates(context.Context, string, int, int) ([]model.ContractTemplate, error) {
	return nil, nil
}
func (r *contractRepo) UpdateTemplate(_ context.Context, t *model.ContractTemplate, e int64) error {
	if t.Version != e {
		return apperr.ErrStale
	}
	t.Version++
	r.templates[t.ID] = t
	return nil
}
func (r *contractRepo) Create(_ context.Context, c *model.ContractVersion) error {
	if r.fail {
		return errors.New("contract create")
	}
	r.rows[c.ID] = c
	return nil
}
func (r *contractRepo) Contract(_ context.Context, viewer, id string) (*model.ContractVersion, error) {
	c := r.rows[id]
	if c == nil || (c.CompanyID != viewer && c.State != model.ContractSigned) {
		return nil, apperr.ErrNotFound
	}
	x := *c
	x.TermsSnapshot = slices.Clone(c.TermsSnapshot)
	x.PartySnapshot = slices.Clone(c.PartySnapshot)
	x.ExternalFileIDs = slices.Clone(c.ExternalFileIDs)
	return &x, nil
}
func (r *contractRepo) ForOrder(_ context.Context, viewer, order string, _, _ int) ([]model.ContractVersion, error) {
	out := []model.ContractVersion{}
	for _, c := range r.rows {
		if c.OrderID == order && (c.CompanyID == viewer || c.State == model.ContractSigned) {
			out = append(out, *c)
		}
	}
	return out, nil
}
func (r *contractRepo) Update(_ context.Context, c *model.ContractVersion, e int64) error {
	if r.fail {
		return errors.New("contract update")
	}
	old := r.rows[c.ID]
	if old == nil {
		return apperr.ErrNotFound
	}
	if old.Version != e {
		return apperr.ErrStale
	}
	c.Version = e + 1
	r.rows[c.ID] = c
	return nil
}
func (r *contractRepo) HasBaseSigned(_ context.Context, order string) (bool, error) {
	for _, c := range r.rows {
		if c.OrderID == order && c.Kind == model.ContractInitial && c.State == model.ContractSigned {
			return true, nil
		}
	}
	return false, nil
}

type contractEvents struct{ s *contractStore }

func (e contractEvents) Append(context.Context, *model.Event) error {
	if e.s.failEvent {
		return errors.New("event")
	}
	e.s.events++
	return nil
}
func (contractEvents) ForResource(context.Context, string, string) ([]model.Event, error) {
	return nil, nil
}

type contractParties struct{}

func (contractParties) ForOrder(context.Context, string, string) (ContractRequisites, ContractRequisites, error) {
	return ContractRequisites{ID: "buyer", Name: "B&<", LegalName: "Buyer LLC", TaxID: "BT"}, ContractRequisites{ID: "supplier", Name: "S", LegalName: "Supplier LLC", TaxID: "ST"}, nil
}

type contractFiles struct {
	Files
	grants    []string
	fail      bool
	failAfter int
}

func (f *contractFiles) clone() *contractFiles {
	return &contractFiles{grants: slices.Clone(f.grants), fail: f.fail, failAfter: f.failAfter}
}
func (f *contractFiles) Share(ctx context.Context, owner, file, to, typ, id string) error {
	target := f
	if tx, ok := ctx.Value(contractFilesKey{}).(*contractFiles); ok {
		target = tx
	}
	if target.fail || (target.failAfter > 0 && len(target.grants) >= target.failAfter) {
		return errors.New("share")
	}
	target.grants = append(target.grants, owner+"/"+file+"/"+to+"/"+typ+"/"+id)
	return nil
}
func contractFixture() (*Contract, *contractStore, *contractFiles) {
	terms := []byte(`{"lines":[{"lineId":"line","unitPrice":{"amountMinor":"100","currency":"USD"}}],"future":{"nested":[12345678901234567890]}}`)
	files := &contractFiles{}
	st := &contractStore{deals: &contractDeals{order: &model.Order{ID: "order", BuyerCompanyID: "buyer", SupplierCompanyID: "supplier", Terms: terms, Version: 1}}, contracts: &contractRepo{templates: map[string]*model.ContractTemplate{}, rows: map[string]*model.ContractVersion{}}, fulfilment: &contractFulfilment{rows: []model.Allocation{{VehicleID: "vehicle", LineID: "line", VIN: "VIN", Status: "allocated"}}}, files: files}
	d := NewDeps(st, nil, nil, nil, files, func() time.Time { return time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC) })
	return NewContract(d, contractParties{}), st, files
}
func contractPrincipal(company string) *auth.Principal {
	return &auth.Principal{CompanyID: company, UserID: "user", Permissions: map[string]bool{model.PermRead: true, model.PermTrade: true}}
}
