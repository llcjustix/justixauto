package service

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

// Contract records externally signed company-to-company order documents. It
// never changes accepted order economics: the order write only advances its
// optimistic revision to serialize documentary history.
type Contract struct {
	Deps
	parties ContractParties
	orders  *ActualOrderParties
}

func NewContract(d Deps, parties ContractParties) *Contract {
	return &Contract{Deps: d, parties: parties, orders: NewActualOrderParties(d)}
}

// ActualOrderParties is constructed independently during composition, before
// the private-requisites adapter is built. It centralizes the Party recheck.
type ActualOrderParties struct{ store Store }

func NewActualOrderParties(d Deps) *ActualOrderParties { return &ActualOrderParties{store: d.store} }

func (s *ActualOrderParties) OrderPartiesFor(ctx context.Context, viewerCompanyID, orderID string) (*model.Order, error) {
	o, err := s.store.Deals().Order(ctx, viewerCompanyID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Party(viewerCompanyID) == "" {
		return nil, apperr.New(apperr.ErrForbidden, "wrong_party", "the company is not an order party")
	}
	return o, nil
}

type (
	TemplateInput struct{ Name, Body string }
	DraftInput    struct {
		TemplateID      *string
		Content         *string
		ExternalFileIDs *[]string
		RefreshSnapshot bool
	}
)

type SignInput struct {
	Reference     string
	SignedOn      time.Time
	SignedFileIDs []string
}

// ContractPreview is an unsaved, authorization-checked rendering source for a
// prospective draft. Its JSON snapshots remain raw so later DTO mapping can
// retain unknown accepted-term fields without base64 encoding them.
type ContractPreview struct {
	Content             string
	TemplateID          *string
	TemplateVersion     *int64
	PartySnapshot       []byte
	TermsSnapshot       []byte
	TermsDigest         string
	SourceOrderRevision int64
}

// OrderPartiesFor is the narrow source for adapters which must prove an order
// viewer is a real party before asking identity for private requisites.
func (s *Contract) OrderPartiesFor(ctx context.Context, viewerCompanyID, orderID string) (*model.Order, error) {
	return s.orders.OrderPartiesFor(ctx, viewerCompanyID, orderID)
}

func (s *Contract) TemplateCreate(ctx context.Context, p *auth.Principal, in TemplateInput) (*model.ContractTemplate, error) {
	if err := p.Allow(model.PermTrade); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" || in.Body == "" {
		return nil, apperr.FieldError("template", "name and body are required")
	}
	now := s.clock()
	t := &model.ContractTemplate{ID: uuid.NewString(), CompanyID: p.CompanyID, Name: in.Name, Body: in.Body, Version: 1, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now}
	return t, s.store.InTx(ctx, func(st Store) error { return st.Contracts().CreateTemplate(ctx, t) })
}

func (s *Contract) TemplateEdit(ctx context.Context, p *auth.Principal, id string, expected int64, in TemplateInput) (*model.ContractTemplate, error) {
	if err := p.Allow(model.PermTrade); err != nil {
		return nil, err
	}
	var out *model.ContractTemplate
	err := s.store.InTx(ctx, func(st Store) error {
		t, e := st.Contracts().Template(ctx, p.CompanyID, id)
		if e != nil {
			return e
		}
		if strings.TrimSpace(in.Name) == "" || in.Body == "" {
			return apperr.FieldError("template", "name and body are required")
		}
		t.Name, t.Body, t.UpdatedAt = in.Name, in.Body, s.clock()
		if e := st.Contracts().UpdateTemplate(ctx, t, expected); e != nil {
			return e
		}
		out = t
		return nil
	})
	return out, err
}

func (s *Contract) Templates(ctx context.Context, p *auth.Principal, limit, offset int) ([]model.ContractTemplate, error) {
	if e := p.Allow(model.PermRead); e != nil {
		return nil, e
	}
	return s.store.Contracts().Templates(ctx, p.CompanyID, limit, offset)
}

func (s *Contract) TemplateDetail(ctx context.Context, p *auth.Principal, id string) (*model.ContractTemplate, error) {
	if e := p.Allow(model.PermRead); e != nil {
		return nil, e
	}
	return s.store.Contracts().Template(ctx, p.CompanyID, id)
}

func orderParties(ctx context.Context, st Store, viewerCompanyID, orderID string) (*model.Order, error) {
	o, err := st.Deals().Order(ctx, viewerCompanyID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Party(viewerCompanyID) == "" {
		return nil, apperr.New(apperr.ErrForbidden, "wrong_party", "the company is not an order party")
	}
	return o, nil
}

func (s *Contract) capture(ctx context.Context, st Store, p *auth.Principal, orderID string) (*model.Order, []byte, []byte, string, error) {
	o, e := orderParties(ctx, st, p.CompanyID, orderID)
	if e != nil {
		return nil, nil, nil, "", e
	}
	if !json.Valid(o.Terms) {
		return nil, nil, nil, "", apperr.New(apperr.ErrConflict, "order_terms_invalid", "stored order terms are invalid")
	}
	var buyer, supplier ContractRequisites
	if s.parties == nil {
		return nil, nil, nil, "", apperr.New(apperr.ErrConflict, "contract_parties_unwired", "contract party projection is not configured")
	}
	bound := st.Bind(ctx)
	buyer, supplier, e = s.parties.ForOrder(bound, p.CompanyID, orderID)
	if e != nil {
		return nil, nil, nil, "", e
	}
	allocs, e := st.Fulfilment().Allocations(ctx, orderID)
	if e != nil {
		return nil, nil, nil, "", e
	}
	vs := make([]model.ContractVehicleSnapshot, 0, len(allocs))
	for _, a := range allocs {
		if a.Counts() {
			vs = append(vs, model.ContractVehicleSnapshot{ID: a.VehicleID, LineID: a.LineID, VIN: a.VIN, Status: a.Status})
		}
	}
	snap := model.ContractSnapshot{Parties: model.ContractPartySnapshot{Buyer: requisite(buyer), Supplier: requisite(supplier)}, Vehicles: vs}
	parties, e := json.Marshal(snap)
	if e != nil {
		return nil, nil, nil, "", e
	}
	terms := bytes.Clone(o.Terms)
	return o, parties, terms, model.Digest(terms), nil
}

// Preview captures current authoritative parties, terms, and allocated VINs
// without creating a contract, granting a file, or advancing an order revision.
func (s *Contract) Preview(ctx context.Context, p *auth.Principal, orderID string, in DraftInput) (*ContractPreview, error) {
	if e := p.Allow(model.PermRead); e != nil {
		return nil, e
	}
	o, party, terms, digest, e := s.capture(ctx, s.store, p, orderID)
	if e != nil {
		return nil, e
	}
	out := &ContractPreview{PartySnapshot: party, TermsSnapshot: terms, TermsDigest: digest, SourceOrderRevision: o.Version}
	if in.TemplateID != nil {
		t, e := s.store.Contracts().Template(ctx, p.CompanyID, *in.TemplateID)
		if e != nil {
			return nil, e
		}
		out.TemplateID, out.TemplateVersion = &t.ID, &t.Version
		out.Content = s.rendered(t, party, terms)
	}
	if in.Content != nil {
		out.Content = *in.Content
	}
	return out, nil
}

func requisite(r ContractRequisites) model.ContractRequisiteSnapshot {
	return model.ContractRequisiteSnapshot{ID: r.ID, Name: r.Name, LegalName: r.LegalName, RegistrationNumber: r.RegistrationNumber, TaxID: r.TaxID, BankRequisites: r.BankRequisites, Address: r.Address, Phone: r.Phone, Email: r.Email}
}

func ids(raw []string) ([]byte, error) {
	if raw == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(raw)
}

func validDraft(c *model.ContractVersion) bool {
	return strings.TrimSpace(c.Content) != "" || string(c.ExternalFileIDs) != "[]"
}

func (s *Contract) rendered(t *model.ContractTemplate, snap, terms []byte) string {
	var v model.ContractSnapshot
	_ = json.Unmarshal(snap, &v)
	x := t.Body
	values := map[string]string{"buyer.id": v.Parties.Buyer.ID, "buyer.name": v.Parties.Buyer.Name, "buyer.legalName": v.Parties.Buyer.LegalName, "buyer.registrationNumber": v.Parties.Buyer.RegistrationNumber, "buyer.taxId": v.Parties.Buyer.TaxID, "buyer.bankRequisites": v.Parties.Buyer.BankRequisites, "buyer.address": v.Parties.Buyer.Address, "buyer.phone": v.Parties.Buyer.Phone, "buyer.email": v.Parties.Buyer.Email, "supplier.id": v.Parties.Supplier.ID, "supplier.name": v.Parties.Supplier.Name, "supplier.legalName": v.Parties.Supplier.LegalName, "supplier.registrationNumber": v.Parties.Supplier.RegistrationNumber, "supplier.taxId": v.Parties.Supplier.TaxID, "supplier.bankRequisites": v.Parties.Supplier.BankRequisites, "supplier.address": v.Parties.Supplier.Address, "supplier.phone": v.Parties.Supplier.Phone, "supplier.email": v.Parties.Supplier.Email}
	var known map[string]json.RawMessage
	if json.Unmarshal(terms, &known) == nil {
		for _, key := range []string{"route", "deliveryTerms", "warrantyTerms", "serviceTerms", "lines", "paymentSchedule"} {
			if raw, ok := known[key]; ok {
				var text string
				if json.Unmarshal(raw, &text) == nil {
					values["order."+key] = text
				} else {
					values["order."+key] = string(raw)
				}
			}
		}
	}
	for k, val := range values {
		x = strings.ReplaceAll(x, "{{"+k+"}}", val)
	}
	return x
}

func (s *Contract) DraftCreate(ctx context.Context, p *auth.Principal, orderID string, expectedOrder int64, in DraftInput) (*model.ContractVersion, error) {
	return s.draftCreate(ctx, p, orderID, expectedOrder, in, model.ContractInitial, nil)
}

func (s *Contract) draftCreate(ctx context.Context, p *auth.Principal, orderID string, expectedOrder int64, in DraftInput, kind model.ContractKind, prior *string) (*model.ContractVersion, error) {
	if e := p.Allow(model.PermTrade); e != nil {
		return nil, e
	}
	var out *model.ContractVersion
	e := s.store.InTx(ctx, func(st Store) error {
		o, party, terms, digest, e := s.capture(ctx, st, p, orderID)
		if e != nil {
			return e
		}
		if o.Version != expectedOrder {
			return apperr.ErrStale
		}
		files, e := ids(uniqueIDs(valueIDs(in.ExternalFileIDs)))
		if e != nil {
			return e
		}
		now := s.clock()
		c := &model.ContractVersion{ID: uuid.NewString(), OrderID: orderID, CompanyID: p.CompanyID, PriorSignedVersionID: prior, Kind: kind, State: model.ContractDraft, ExternalFileIDs: files, SignedFileIDs: []byte("[]"), PartySnapshot: party, TermsSnapshot: terms, TermsDigest: digest, SourceOrderRevision: o.Version, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now, Version: 1}
		if e := validatePriorContract(ctx, st, p.CompanyID, orderID, prior); e != nil {
			return e
		}
		if in.TemplateID != nil {
			t, e := st.Contracts().Template(ctx, p.CompanyID, *in.TemplateID)
			if e != nil {
				return e
			}
			c.TemplateID = &t.ID
			c.TemplateVersion = &t.Version
			c.Content = s.rendered(t, party, terms)
		}
		if in.Content != nil {
			c.Content = *in.Content
		}
		if !validDraft(c) {
			return apperr.FieldError("draft", "content or external files are required")
		}
		if e := st.Contracts().Create(ctx, c); e != nil {
			return e
		}
		for _, fileID := range decodeIDs(c.ExternalFileIDs) {
			if e := s.files.Share(st.Bind(ctx), p.CompanyID, fileID, p.CompanyID, "contract", c.ID); e != nil {
				return e
			}
		}
		o.UpdatedAt = now
		if e := st.Deals().UpdateOrder(ctx, o, expectedOrder); e != nil {
			return e
		}
		if e := s.event(ctx, st, p, "contract.draft_created", "contract", c.ID, "", nil); e != nil {
			return e
		}
		out = c
		return nil
	})
	return out, e
}

func validatePriorContract(ctx context.Context, st Store, companyID, orderID string, prior *string) error {
	if prior == nil {
		return nil
	}
	old, err := st.Contracts().Contract(ctx, companyID, *prior)
	if err != nil {
		return err
	}
	if old.OrderID != orderID || old.State != model.ContractSigned {
		return apperr.New(apperr.ErrConflict, "prior_contract_invalid", "prior contract must be signed for this order")
	}
	return nil
}

func valueIDs(v *[]string) []string {
	if v == nil {
		return nil
	}
	return *v
}

func (s *Contract) DraftEdit(ctx context.Context, p *auth.Principal, id string, expectedOrder, expectedDraft int64, in DraftInput) (*model.ContractVersion, error) {
	if e := p.Allow(model.PermTrade); e != nil {
		return nil, e
	}
	var out *model.ContractVersion
	e := s.store.InTx(ctx, func(st Store) error {
		c, e := st.Contracts().Contract(ctx, p.CompanyID, id)
		if e != nil {
			return e
		}
		if c.CompanyID != p.CompanyID || c.State != model.ContractDraft {
			return apperr.New(apperr.ErrForbidden, "contract_private", "draft is private")
		}
		o, e := orderParties(ctx, st, p.CompanyID, c.OrderID)
		if e != nil {
			return e
		}
		if o.Version != expectedOrder {
			return apperr.ErrStale
		}
		if in.RefreshSnapshot {
			o, c.PartySnapshot, c.TermsSnapshot, c.TermsDigest, e = s.capture(ctx, st, p, c.OrderID)
			if e != nil {
				return e
			}
			c.SourceOrderRevision = o.Version
		}
		if e := s.editDraftTemplate(ctx, st, p.CompanyID, c, in); e != nil {
			return e
		}
		if in.Content != nil {
			c.Content = *in.Content
		}
		if e := s.editDraftFiles(ctx, st, p.CompanyID, c, in.ExternalFileIDs); e != nil {
			return e
		}
		if !validDraft(c) {
			return apperr.FieldError("draft", "content or external files are required")
		}
		now := s.clock()
		c.UpdatedAt = now
		if e := st.Contracts().Update(ctx, c, expectedDraft); e != nil {
			return e
		}
		o.UpdatedAt = now
		if e := st.Deals().UpdateOrder(ctx, o, expectedOrder); e != nil {
			return e
		}
		if e := s.event(ctx, st, p, "contract.draft_edited", "contract", c.ID, "", nil); e != nil {
			return e
		}
		out = c
		return nil
	})
	return out, e
}

func (s *Contract) editDraftTemplate(ctx context.Context, st Store, companyID string, c *model.ContractVersion, in DraftInput) error {
	if in.TemplateID == nil {
		return nil
	}
	if *in.TemplateID == "" {
		c.TemplateID = nil
		c.TemplateVersion = nil
		return nil
	}
	t, err := st.Contracts().Template(ctx, companyID, *in.TemplateID)
	if err != nil {
		return err
	}
	c.TemplateID = &t.ID
	c.TemplateVersion = &t.Version
	if in.Content == nil {
		c.Content = s.rendered(t, c.PartySnapshot, c.TermsSnapshot)
	}
	return nil
}

func (s *Contract) editDraftFiles(ctx context.Context, st Store, companyID string, c *model.ContractVersion, fileIDs *[]string) error {
	if fileIDs == nil {
		return nil
	}
	previous := decodeIDs(c.ExternalFileIDs)
	var err error
	c.ExternalFileIDs, err = ids(uniqueIDs(*fileIDs))
	if err != nil {
		return err
	}
	for _, fileID := range addedIDs(previous, decodeIDs(c.ExternalFileIDs)) {
		if err := s.files.Share(st.Bind(ctx), companyID, fileID, companyID, "contract", c.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Contract) Sign(ctx context.Context, p *auth.Principal, id string, expectedOrder, expectedDraft int64, in SignInput) (*model.ContractVersion, error) {
	if e := p.Allow(model.PermTrade); e != nil {
		return nil, e
	}
	var out *model.ContractVersion
	e := s.store.InTx(ctx, func(st Store) error {
		c, e := st.Contracts().Contract(ctx, p.CompanyID, id)
		if e != nil {
			return e
		}
		if c.CompanyID != p.CompanyID || c.State != model.ContractDraft {
			return apperr.New(apperr.ErrConflict, "contract_not_draft", "contract is not a private draft")
		}
		o, e := orderParties(ctx, st, p.CompanyID, c.OrderID)
		if e != nil {
			return e
		}
		if c.Kind == model.ContractInitial {
			yes, e := st.Contracts().HasBaseSigned(ctx, c.OrderID)
			if e != nil {
				return e
			}
			if yes {
				return apperr.New(apperr.ErrConflict, "contract_amendment_required", "initial contract already signed")
			}
		}
		return s.sign(ctx, st, p, c, o, expectedOrder, expectedDraft, in, &out)
	})
	return out, e
}

func (s *Contract) sign(ctx context.Context, st Store, p *auth.Principal, c *model.ContractVersion, o *model.Order, expectedOrder, expectedDraft int64, in SignInput, out **model.ContractVersion) error {
	if o.Version != expectedOrder {
		return apperr.ErrStale
	}
	if !bytes.Equal(c.TermsSnapshot, o.Terms) || c.TermsDigest != model.Digest(o.Terms) {
		return apperr.New(apperr.ErrConflict, "contract_snapshot_stale", "refresh the draft before signing")
	}
	if len(in.SignedFileIDs) == 0 {
		return apperr.FieldError("signedFiles", "at least one signed scan is required")
	}
	allFiles := uniqueIDs(append(decodeIDs(c.ExternalFileIDs), in.SignedFileIDs...))
	scan, err := ids(allFiles)
	if err != nil {
		return err
	}
	other := o.BuyerCompanyID
	if other == p.CompanyID {
		other = o.SupplierCompanyID
	}
	bound := st.Bind(ctx)
	for _, f := range allFiles {
		if err := s.files.Share(bound, p.CompanyID, f, other, "contract", c.ID); err != nil {
			return err
		}
	}
	now := s.clock()
	c.State, c.Reference, c.SignedOn, c.SignedFileIDs, c.UpdatedAt = model.ContractSigned, in.Reference, &in.SignedOn, scan, now
	if err := st.Contracts().Update(ctx, c, expectedDraft); err != nil {
		return err
	}
	o.UpdatedAt = now
	if err := st.Deals().UpdateOrder(ctx, o, expectedOrder); err != nil {
		return err
	}
	if err := s.event(ctx, st, p, "contract.signed", "contract", c.ID, "", nil); err != nil {
		return err
	}
	*out = c
	return nil
}
func decodeIDs(raw []byte) []string { var x []string; _ = json.Unmarshal(raw, &x); return x }
func uniqueIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func addedIDs(previous, current []string) []string {
	seen := make(map[string]bool, len(previous))
	for _, id := range previous {
		seen[id] = true
	}
	added := make([]string, 0, len(current))
	for _, id := range current {
		if !seen[id] {
			added = append(added, id)
		}
	}
	return added
}

// Amend consumes an existing private draft into a signed amendment. The
// immutable signed parent is validated for state, order, and actor visibility;
// no parent revision is accepted or mutated.
func (s *Contract) Amend(ctx context.Context, p *auth.Principal, id, priorID string, expectedOrder, expectedDraft int64, in SignInput) (*model.ContractVersion, error) {
	if e := p.Allow(model.PermTrade); e != nil {
		return nil, e
	}
	var out *model.ContractVersion
	e := s.store.InTx(ctx, func(st Store) error {
		c, e := st.Contracts().Contract(ctx, p.CompanyID, id)
		if e != nil {
			return e
		}
		if c.CompanyID != p.CompanyID || c.State != model.ContractDraft {
			return apperr.New(apperr.ErrConflict, "contract_not_draft", "contract is not a private draft")
		}
		o, e := orderParties(ctx, st, p.CompanyID, c.OrderID)
		if e != nil {
			return e
		}
		if o.Version != expectedOrder {
			return apperr.ErrStale
		}
		prior, e := st.Contracts().Contract(ctx, p.CompanyID, priorID)
		if e != nil {
			return e
		}
		if prior.OrderID != c.OrderID || prior.State != model.ContractSigned {
			return apperr.New(apperr.ErrConflict, "prior_contract_invalid", "prior contract must be signed for this order")
		}
		c.Kind, c.PriorSignedVersionID = model.ContractAmendment, &priorID
		return s.sign(ctx, st, p, c, o, expectedOrder, expectedDraft, in, &out)
	})
	return out, e
}

func (s *Contract) Detail(ctx context.Context, p *auth.Principal, id string) (*model.ContractVersion, error) {
	if e := p.Allow(model.PermRead); e != nil {
		return nil, e
	}
	c, e := s.store.Contracts().Contract(ctx, p.CompanyID, id)
	if e != nil {
		return nil, e
	}
	if _, e = s.OrderPartiesFor(ctx, p.CompanyID, c.OrderID); e != nil {
		return nil, e
	}
	return c, nil
}

func (s *Contract) List(ctx context.Context, p *auth.Principal, orderID string, limit, offset int) ([]model.ContractVersion, error) {
	if e := p.Allow(model.PermRead); e != nil {
		return nil, e
	}
	if _, e := s.OrderPartiesFor(ctx, p.CompanyID, orderID); e != nil {
		return nil, e
	}
	return s.store.Contracts().ForOrder(ctx, p.CompanyID, orderID, limit, offset)
}
