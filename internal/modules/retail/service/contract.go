package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
	"justixauto/internal/pkg/validate"

	"github.com/google/uuid"
)

type Contract struct{ Deps }

func NewContract(d Deps) *Contract          { return &Contract{d} }
func contracts(st Store) ContractRepository { return st.Contracts() }
func (s *Contract) contractDeal(ctx context.Context, st Store, p *auth.Principal, id string, expected int64) (*model.Deal, error) {
	if err := validate.IDs(id); err != nil {
		return nil, err
	}
	d, err := st.Deals().Deal(ctx, p.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if !inScope(p, d.BranchID) {
		return nil, apperr.ErrNotFound
	}
	if expected >= 0 && d.Version != expected {
		return nil, apperr.ErrStale
	}
	return d, nil
}

type TemplateInput struct {
	Name string `json:"name"`
	Body string `json:"body"`
}
type DraftContractInput struct {
	TemplateID string   `json:"templateId"`
	Body       string   `json:"body"`
	BindingIDs []string `json:"bindingIds"`
}

// EditDraftContractInput distinguishes omitted fields from explicit empties:
// omitted fields are preserved, while an explicit empty body/template/file set
// clears that value when the resulting draft remains valid.
type EditDraftContractInput struct {
	TemplateID    *string   `json:"templateId"`
	Body          *string   `json:"body"`
	BindingIDs    *[]string `json:"bindingIds"`
	DraftRevision string    `json:"draftRevision"`
}
type SignContractInput struct {
	SignedOn      string   `json:"signedOn"`
	Reference     string   `json:"reference"`
	BindingIDs    []string `json:"bindingIds"`
	DraftRevision string   `json:"draftRevision"`
}

func contractText(v *apperr.Validation, field, value string) string {
	return validate.Text(v, field, value, 1, 20000)
}
func (s *Contract) CreateTemplate(ctx context.Context, p *auth.Principal, in TemplateInput) (*model.ContractTemplate, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	var v apperr.Validation
	t := &model.ContractTemplate{ID: uuid.NewString(), CompanyID: p.CompanyID, Name: validate.Text(&v, "name", in.Name, 1, 200), Body: contractText(&v, "body", in.Body), Version: 1, CreatedBy: p.UserID, CreatedAt: s.clock(), UpdatedAt: s.clock()}
	if err := v.Err(); err != nil {
		return nil, err
	}
	if err := contracts(s.store).CreateTemplate(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
func (s *Contract) UpdateTemplate(ctx context.Context, p *auth.Principal, id string, expected int64, in TemplateInput) (*model.ContractTemplate, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	t, err := contracts(s.store).Template(ctx, p.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if t.Version != expected {
		return nil, apperr.ErrStale
	}
	var v apperr.Validation
	t.Name = validate.Text(&v, "name", in.Name, 1, 200)
	t.Body = contractText(&v, "body", in.Body)
	if err := v.Err(); err != nil {
		return nil, err
	}
	t.UpdatedAt = s.clock()
	if err := contracts(s.store).UpdateTemplate(ctx, t, expected); err != nil {
		return nil, err
	}
	t.Version = expected + 1
	return t, nil
}
func (s *Contract) Templates(ctx context.Context, p *auth.Principal) ([]model.ContractTemplate, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	return contracts(s.store).Templates(ctx, p.CompanyID)
}
func vehicleVIN(d *model.Deal) string {
	if d.VehicleSnapshot == nil {
		return ""
	}
	return d.VehicleSnapshot.VIN
}
func renderContract(body string, seller *CompanyRequisites, c *model.Customer, d *model.Deal) string {
	vehicle := d.VehicleSnapshot
	value := func(v string) string { return v }
	var modelID, vehicleID, modelVersion, exterior, interior string
	if vehicle != nil {
		vehicleID, modelID, modelVersion, exterior, interior = vehicle.VehicleID, vehicle.ModelID, vehicle.ModelSpecificationVersion, vehicle.ExteriorColor, vehicle.InteriorColor
	}
	dateValue := func(v *time.Time) string {
		if v == nil {
			return ""
		}
		return v.Format(time.DateOnly)
	}
	return strings.NewReplacer(
		"{{seller.name}}", value(seller.Name), "{{seller.legalName}}", value(seller.LegalName), "{{seller.registrationNumber}}", value(seller.RegistrationNumber), "{{seller.taxId}}", value(seller.TaxID), "{{seller.bankRequisites}}", value(seller.BankRequisites), "{{seller.address}}", value(seller.Address), "{{seller.phone}}", value(seller.Phone), "{{seller.email}}", value(seller.Email),
		"{{customer.name}}", value(c.DisplayName), "{{customer.phone}}", value(c.Phone), "{{customer.passportSeries}}", value(c.PassportSeries), "{{customer.passportNumber}}", value(c.PassportNumber), "{{customer.passportIssuer}}", value(c.PassportIssuer), "{{customer.passportIssuedOn}}", dateValue(c.PassportIssuedOn), "{{customer.passportExpiresOn}}", dateValue(c.PassportExpiresOn), "{{customer.birthDate}}", dateValue(c.BirthDate), "{{customer.personalId}}", value(c.PersonalID), "{{customer.registeredAddress}}", value(c.RegisteredAddress),
		"{{vehicle.id}}", vehicleID, "{{vehicle.vin}}", vehicleVIN(d), "{{vehicle.modelId}}", modelID, "{{vehicle.modelSpecificationVersion}}", modelVersion, "{{vehicle.exteriorColor}}", exterior, "{{vehicle.interiorColor}}", interior,
		"{{price.amountMinor}}", d.PriceMinor, "{{price.currency}}", d.Currency, "{{payment.scheme}}", d.PaymentScheme,
	).Replace(body)
}
func snapshotDate(v *time.Time) *string {
	if v == nil {
		return nil
	}
	out := v.Format(time.DateOnly)
	return &out
}
func (s *Contract) termsSnapshot(ctx context.Context, st Store, d *model.Deal) (*model.ContractTermsSnapshot, error) {
	out := &model.ContractTermsSnapshot{Price: d.Price(), PaymentScheme: d.PaymentScheme}
	if d.PaymentScheme != "own-installment" {
		return out, nil
	}
	if d.InstallmentDraft != nil {
		x := d.InstallmentDraft
		rows := make([]model.ContractInstallmentObligation, 0, len(x.Rows))
		for _, row := range x.Rows {
			n, due := row.Number, row.DueDate
			rows = append(rows, model.ContractInstallmentObligation{Number: &n, Purpose: "monthly-installment", Amount: row.Amount, DueDate: &due})
		}
		out.Installment = &model.ContractInstallmentSnapshot{Source: "draft", PolicyID: x.PolicyID, PolicyVersion: x.PolicyVersion, Price: x.Price, DownPayment: x.DownPayment, TermMonths: x.TermMonths, FirstDueDate: x.FirstDueDate, ScheduledTotal: x.ScheduledTotal, RegularPayment: x.RegularPayment, Obligations: rows}
		return out, nil
	}
	plan, err := st.Deals().InstallmentPlan(ctx, d.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		out.Legacy = true
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	invoices, err := st.Deals().Invoices(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	rows := []model.ContractInstallmentObligation{}
	for _, invoice := range invoices {
		if invoice.ID != plan.DownPaymentInvoiceID && (invoice.InstallmentPlanID == nil || *invoice.InstallmentPlanID != plan.ID) {
			continue
		}
		rows = append(rows, model.ContractInstallmentObligation{InvoiceID: invoice.ID, Number: invoice.InstallmentNumber, Purpose: invoice.Purpose, Amount: money.Money{AmountMinor: invoice.AmountMinor, Currency: invoice.Currency}, DueDate: snapshotDate(invoice.DueDate)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Number == nil {
			return true
		}
		if rows[j].Number == nil {
			return false
		}
		return *rows[i].Number < *rows[j].Number
	})
	out.Installment = &model.ContractInstallmentSnapshot{Source: "plan", Price: d.Price(), DownPayment: money.Money{AmountMinor: plan.DownPaymentMinor, Currency: plan.Currency}, ScheduledTotal: money.Money{AmountMinor: plan.ContractTotalMinor, Currency: plan.Currency}, PlanID: plan.ID, DownPaymentInvoiceID: plan.DownPaymentInvoiceID, Obligations: rows}
	return out, nil
}
func (s *Contract) snapshot(ctx context.Context, st Store, p *auth.Principal, d *model.Deal, body string) (string, []byte, []byte, error) {
	seller, err := s.company.ContractRequisitesFor(st.Bind(ctx), p.CompanyID, d.CompanyID)
	if err != nil {
		return "", nil, nil, err
	}
	c, err := st.CRM().Customer(ctx, p.CompanyID, d.CustomerID)
	if err != nil {
		return "", nil, nil, err
	}
	party, err := json.Marshal(model.ContractPartySnapshot{Seller: model.ContractSellerSnapshot{ID: seller.ID, Name: seller.Name, LegalName: seller.LegalName, RegistrationNumber: seller.RegistrationNumber, TaxID: seller.TaxID, BankRequisites: seller.BankRequisites, Address: seller.Address, Phone: seller.Phone, Email: seller.Email}, Customer: model.ContractCustomerSnapshot{ID: c.ID, Name: c.DisplayName, Phone: c.Phone, PassportSeries: c.PassportSeries, PassportNumber: c.PassportNumber, PassportIssuer: c.PassportIssuer, PassportIssuedOn: snapshotDate(c.PassportIssuedOn), PassportExpiresOn: snapshotDate(c.PassportExpiresOn), BirthDate: snapshotDate(c.BirthDate), PersonalID: c.PersonalID, RegisteredAddress: c.RegisteredAddress}, Vehicle: d.VehicleSnapshot})
	if err != nil {
		return "", nil, nil, err
	}
	termsValue, err := s.termsSnapshot(ctx, st, d)
	if err != nil {
		return "", nil, nil, err
	}
	terms, err := json.Marshal(termsValue)
	if err != nil {
		return "", nil, nil, err
	}
	return renderContract(body, seller, c, d), party, terms, nil
}
func (s *Contract) Draft(ctx context.Context, p *auth.Principal, dealID string, expected int64, in DraftContractInput) (*model.ContractVersion, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	var out *model.ContractVersion
	err := s.store.InTx(ctx, func(st Store) error {
		d, err := s.contractDeal(ctx, st, p, dealID, expected)
		if err != nil {
			return err
		}
		body := in.Body
		var tid *string
		var tv *int64
		if in.TemplateID != "" {
			t, err := contracts(st).Template(ctx, p.CompanyID, in.TemplateID)
			if err != nil {
				return err
			}
			if body == "" {
				body = t.Body
			}
			tid = &t.ID
			version := t.Version
			tv = &version
		}
		var v apperr.Validation
		if body != "" {
			body = contractText(&v, "body", body)
		}
		files := validate.UniqueIDs(&v, "bindingIds", in.BindingIDs)
		if len(files) > 10 {
			v.Add("bindingIds", "at most 10 files")
		}
		if body == "" && tid == nil && len(files) == 0 {
			v.Add("body", "requires content, templateId, or bindingIds")
		}
		if err := v.Err(); err != nil {
			return err
		}
		content, party, terms, err := s.snapshot(ctx, st, p, d, body)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(files)
		if err != nil {
			return err
		}
		out = &model.ContractVersion{ID: uuid.NewString(), DealID: d.ID, CompanyID: p.CompanyID, TemplateID: tid, TemplateVersion: tv, Kind: "contract", State: "draft", Content: content, PartySnapshot: party, TermsSnapshot: terms, FileIDs: raw, CreatedBy: p.UserID, CreatedAt: s.clock(), Revision: 1}
		if err := contracts(st).CreateContract(ctx, out); err != nil {
			return err
		}
		for _, f := range files {
			if err := s.shareContractFile(ctx, st, p, f, out.ID); err != nil {
				return err
			}
		}
		d.UpdatedAt = s.clock()
		if err := st.Deals().Update(ctx, d, expected); err != nil {
			return err
		}
		out.DealRevision = d.Version
		return nil
	})
	return out, err
}

func draftFiles(v *model.ContractVersion) ([]string, error) {
	var files []string
	if len(v.FileIDs) == 0 {
		return []string{}, nil
	}
	if err := json.Unmarshal(v.FileIDs, &files); err != nil {
		return nil, err
	}
	return files, nil
}
func (s *Contract) shareContractFile(ctx context.Context, st Store, p *auth.Principal, fileID, contractID string) error {
	if err := s.files.Share(st.Bind(ctx), p.CompanyID, fileID, p.CompanyID, "retail.contract", contractID); err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.FieldError("bindingIds", "contains files that are not yours")
		}
		return err
	}
	return nil
}
func canonicalRevision(field, raw string) (int64, error) {
	if raw == "" {
		return 0, apperr.FieldError(field, "is required")
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 || strconv.FormatInt(n, 10) != raw {
		return 0, apperr.FieldError(field, "must be a canonical decimal revision")
	}
	return n, nil
}
func (s *Contract) EditDraft(ctx context.Context, p *auth.Principal, dealID, id string, expected int64, in EditDraftContractInput) (*model.ContractVersion, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	draftRevision, err := canonicalRevision("draftRevision", in.DraftRevision)
	if err != nil {
		return nil, err
	}
	var out *model.ContractVersion
	err = s.store.InTx(ctx, func(st Store) error {
		d, err := s.contractDeal(ctx, st, p, dealID, expected)
		if err != nil {
			return err
		}
		draft, err := contracts(st).Contract(ctx, p.CompanyID, d.ID, id)
		if err != nil {
			return err
		}
		if draft.State != "draft" {
			return apperr.New(apperr.ErrConflict, "contract_already_signed", "contract version is no longer a draft")
		}
		if draft.Revision != draftRevision {
			return apperr.ErrStale
		}
		updated := *draft
		body, bodyChanged := updated.Content, false
		if in.TemplateID != nil {
			if *in.TemplateID == "" {
				updated.TemplateID, updated.TemplateVersion = nil, nil
			} else {
				t, err := contracts(st).Template(ctx, p.CompanyID, *in.TemplateID)
				if err != nil {
					return err
				}
				updated.TemplateID = &t.ID
				version := t.Version
				updated.TemplateVersion = &version
				body = t.Body
				bodyChanged = true
			}
		}
		if in.Body != nil {
			body = *in.Body
			bodyChanged = true
		}
		files, err := draftFiles(draft)
		if err != nil {
			return err
		}
		var v apperr.Validation
		if in.BindingIDs != nil {
			files = validate.UniqueIDs(&v, "bindingIds", *in.BindingIDs)
		}
		if len(files) > 10 {
			v.Add("bindingIds", "at most 10 files")
		}
		if bodyChanged && body != "" {
			body = contractText(&v, "body", body)
		}
		if body == "" && updated.TemplateID == nil && len(files) == 0 {
			v.Add("body", "requires content, templateId, or bindingIds")
		}
		if err := v.Err(); err != nil {
			return err
		}
		if bodyChanged {
			content, party, terms, err := s.snapshot(ctx, st, p, d, body)
			if err != nil {
				return err
			}
			updated.Content, updated.PartySnapshot, updated.TermsSnapshot = content, party, terms
		}
		raw, err := json.Marshal(files)
		if err != nil {
			return err
		}
		updated.FileIDs = raw
		if err := contracts(st).UpdateDraft(ctx, &updated, draftRevision); err != nil {
			return err
		}
		for _, f := range files {
			if err := s.shareContractFile(ctx, st, p, f, updated.ID); err != nil {
				return err
			}
		}
		d.UpdatedAt = s.clock()
		if err := st.Deals().Update(ctx, d, expected); err != nil {
			return err
		}
		updated.DealRevision, out = d.Version, &updated
		return nil
	})
	return out, err
}
func (s *Contract) Sign(ctx context.Context, p *auth.Principal, dealID, id string, expected int64, in SignContractInput, prior *string) (*model.ContractVersion, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	draftRevision, err := canonicalRevision("draftRevision", in.DraftRevision)
	if err != nil {
		return nil, err
	}
	var out *model.ContractVersion
	err = s.store.InTx(ctx, func(st Store) error {
		d, err := s.contractDeal(ctx, st, p, dealID, expected)
		if err != nil {
			return err
		}
		draft, err := contracts(st).Contract(ctx, p.CompanyID, d.ID, id)
		if err != nil {
			return err
		}
		if draft.State != "draft" {
			return apperr.New(apperr.ErrConflict, "contract_already_signed", "contract version is no longer a draft")
		}
		if draft.Revision != draftRevision {
			return apperr.ErrStale
		}
		out, err = s.signDraft(ctx, st, p, d, draft, draftRevision, in, prior)
		return err
	})
	return out, err
}

// signDraft completes either a reviewed draft or the draft-shaped record made
// for the legacy upload endpoint. It deliberately joins the caller's single
// transaction so the signed record, file bindings, deal compatibility fields,
// installment plan, and audit event succeed or fail together.
func (s *Contract) signDraft(ctx context.Context, st Store, p *auth.Principal, d *model.Deal, draft *model.ContractVersion, draftRevision int64, in SignContractInput, prior *string) (*model.ContractVersion, error) {
	var v apperr.Validation
	signed := date(&v, "signedOn", in.SignedOn, s.clock())
	reference := validate.Text(&v, "reference", in.Reference, 1, 100)
	files := validate.UniqueIDs(&v, "bindingIds", in.BindingIDs)
	if len(files) == 0 {
		v.Add("bindingIds", "requires a signed paper scan")
	}
	if len(files) > 10 {
		v.Add("bindingIds", "at most 10 files")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	if prior != nil {
		old, err := contracts(st).Contract(ctx, p.CompanyID, d.ID, *prior)
		if err != nil || old.State != "signed" {
			return nil, apperr.FieldError("priorSignedVersionId", "must be a signed version")
		}
	} else {
		versions, err := contracts(st).Contracts(ctx, p.CompanyID, d.ID)
		if err != nil {
			return nil, err
		}
		for _, version := range versions {
			if version.State == "signed" {
				return nil, apperr.New(apperr.ErrConflict, "contract_amendment_required", "an existing signed contract requires an explicit amendment")
			}
		}
		ready, err := currentDemoScoring(ctx, st, d)
		if err != nil {
			return nil, err
		}
		if !ready {
			return nil, apperr.New(apperr.ErrConflict, "demo_scoring_required", "complete current demo scoring before signing an own-installment contract")
		}
	}
	currentTerms, err := s.termsSnapshot(ctx, st, d)
	if err != nil {
		return nil, err
	}
	currentTermsJSON, err := json.Marshal(currentTerms)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(draft.TermsSnapshot, currentTermsJSON) {
		return nil, apperr.New(apperr.ErrConflict, "contract_terms_stale", "refresh the draft before signing changed installment terms")
	}
	sourceFiles, err := draftFiles(draft)
	if err != nil {
		return nil, err
	}
	allFiles := append(append([]string{}, sourceFiles...), files...)
	seen := map[string]bool{}
	merged := make([]string, 0, len(allFiles))
	for _, f := range allFiles {
		if !seen[f] {
			seen[f] = true
			merged = append(merged, f)
		}
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	signedDraft := *draft
	signedDraft.Kind = "contract"
	if prior != nil {
		signedDraft.Kind = "amendment"
	}
	signedDraft.State = "signed"
	signedDraft.PriorSignedVersionID = prior
	signedDraft.Reference = reference
	signedDraft.SignedOn = signed
	signedDraft.FileIDs = raw
	if err := contracts(st).SignDraft(ctx, &signedDraft, draftRevision); err != nil {
		return nil, err
	}
	for _, f := range merged {
		if err := s.shareContractFile(ctx, st, p, f, signedDraft.ID); err != nil {
			return nil, err
		}
	}
	d.ContractSignedOn = signed
	d.ContractReference = reference
	d.ContractFileIDs = raw
	if err := (&Deal{Deps: s.Deps}).finalizeInstallmentDraft(ctx, st, p, d); err != nil {
		return nil, err
	}
	d.UpdatedAt = s.clock()
	if err := st.Deals().Update(ctx, d, d.Version); err != nil {
		return nil, err
	}
	if err := s.event(ctx, st, p, "deal.contract_signed", "deal", d.ID, "", nil); err != nil {
		return nil, err
	}
	signedDraft.DealRevision = d.Version
	return &signedDraft, nil
}
func (s *Contract) Contracts(ctx context.Context, p *auth.Principal, dealID string) ([]model.ContractVersion, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	if _, err := s.contractDeal(ctx, s.store, p, dealID, -1); err != nil {
		return nil, err
	}
	return contracts(s.store).Contracts(ctx, p.CompanyID, dealID)
}

func (s *Contract) Contract(ctx context.Context, p *auth.Principal, dealID, id string) (*model.ContractVersion, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	if _, err := s.contractDeal(ctx, s.store, p, dealID, -1); err != nil {
		return nil, err
	}
	return contracts(s.store).Contract(ctx, p.CompanyID, dealID, id)
}
