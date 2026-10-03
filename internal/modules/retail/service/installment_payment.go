package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/google/uuid"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
	"justixauto/internal/pkg/validate"
)

type InstallmentPaymentAllocationInput struct {
	InvoiceID string      `json:"invoiceId"`
	Amount    money.Money `json:"amount"`
}

type InstallmentPaymentInput struct {
	EvidenceInput
	Allocations []InstallmentPaymentAllocationInput `json:"allocations"`
}

type InstallmentPaymentAllocationView struct {
	InvoiceID  string
	EvidenceID string
	Number     int
	Amount     money.Money
}

type InstallmentPaymentView struct {
	Payment        model.InstallmentPayment
	Allocations    []InstallmentPaymentAllocationView
	AllowedActions []string
}

// lockPaymentDeal is the common entry for all monthly money mutations. Never
// call it after an invoice lock; state and branch scope are read under this lock.
func (s *Deal) lockPaymentDeal(ctx context.Context, st Store, p *auth.Principal, id string) (*model.Deal, error) {
	if err := validate.IDs(id); err != nil {
		return nil, err
	}
	d, err := st.Deals().LockDeal(st.Bind(ctx), p.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if !inScope(p, d.BranchID) {
		return nil, apperr.ErrNotFound
	}
	return d, nil
}

// lockEvidenceInvoice discovers immutable identity, then takes the deal first
// for monthly rows. Ordinary invoice workflows keep their existing semantics.
func (s *Deal) lockEvidenceInvoice(ctx context.Context, st Store, p *auth.Principal, id string) (*model.Deal, *model.Invoice, error) {
	if err := validate.IDs(id); err != nil {
		return nil, nil, err
	}
	i, err := st.Deals().Invoice(ctx, p.CompanyID, id)
	if err != nil {
		return nil, nil, err
	}
	if i.Purpose == "monthly-installment" {
		d, err := s.lockPaymentDeal(ctx, st, p, i.DealID)
		if err != nil {
			return nil, nil, err
		}
		rows, err := st.Deals().LockInstallmentInvoices(st.Bind(ctx), p.CompanyID, d.ID, []string{id})
		if err != nil {
			return nil, nil, err
		}
		if len(rows) != 1 {
			return nil, nil, apperr.ErrNotFound
		}
		return d, &rows[0], nil
	}
	i, err = st.Deals().LockInvoice(st.Bind(ctx), p.CompanyID, id)
	if err != nil {
		return nil, nil, err
	}
	d, err := s.deal(ctx, st, p, i.DealID, -1)
	return d, i, err
}

func validateInstallmentAllocations(in InstallmentPaymentInput) error {
	var v apperr.Validation
	claimed, valid := in.ClaimedAmount.Parse()
	if !valid || claimed.Sign() <= 0 {
		v.Add("claimedAmount", "must be a positive amount in minor units with a currency code")
	}
	if len(in.Allocations) == 0 || len(in.Allocations) > 1000 {
		v.Add("allocations", "select between 1 and 1000 monthly invoices")
	}
	total := new(big.Int)
	seen := map[string]bool{}
	for n, a := range in.Allocations {
		field := fmt.Sprintf("allocations[%d]", n)
		if err := validate.IDs(a.InvoiceID); err != nil {
			v.Add(field+".invoiceId", "must be a UUID")
		}
		if seen[a.InvoiceID] {
			v.Add(field+".invoiceId", "invoice must appear only once")
		}
		seen[a.InvoiceID] = true
		value, ok := a.Amount.Parse()
		if !ok || value.Sign() <= 0 {
			v.Add(field+".amount", "must be a positive amount in minor units")
		} else {
			total.Add(total, value)
		}
		if a.Amount.Currency != in.ClaimedAmount.Currency {
			v.Add(field+".amount", "must match the claimed currency")
		}
	}
	if valid && total.Cmp(claimed) != 0 {
		v.Add("allocations", "amounts must sum exactly to claimedAmount")
	}
	return v.Err()
}

// SubmitInstallmentPayment reserves exactly the caller's chosen allocations.
// It never changes the saved schedule, delivery state or confirmed balances.
func (s *Deal) SubmitInstallmentPayment(ctx context.Context, p *auth.Principal, dealID string, in InstallmentPaymentInput) (*InstallmentPaymentView, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	if err := validateInstallmentAllocations(in); err != nil {
		return nil, err
	}
	var v apperr.Validation
	paidOn := date(&v, "paidOn", in.PaidOn, s.clock())
	ref := validate.Text(&v, "externalReference", in.ExternalReference, 1, 100)
	attachments := validate.UniqueIDs(&v, "attachmentBindingIds", in.AttachmentIDs)
	if len(attachments) > 10 {
		v.Add("attachmentBindingIds", "at most 10 files")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	var result *InstallmentPaymentView
	err := s.store.InTx(ctx, func(st Store) error {
		ctx := st.Bind(ctx)
		d, err := s.lockPaymentDeal(ctx, st, p, dealID)
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(in.Allocations))
		for _, a := range in.Allocations {
			ids = append(ids, a.InvoiceID)
		}
		sort.Strings(ids)
		rows, err := st.Deals().LockInstallmentInvoices(ctx, p.CompanyID, d.ID, ids)
		if err != nil {
			return err
		}
		if len(rows) != len(ids) {
			return apperr.ErrNotFound
		}
		byID := make(map[string]model.Invoice, len(rows))
		for _, i := range rows {
			if err := s.monthlyPaymentEligibility(ctx, st, d, &i); err != nil {
				return err
			}
			byID[i.ID] = i
		}
		if in.ClaimedAmount.Currency != d.Currency {
			return apperr.FieldError("claimedAmount", "must be in "+d.Currency)
		}
		if err := s.checkInstallmentReference(ctx, st, d, ref); err != nil {
			return err
		}
		for n, a := range in.Allocations {
			i, ok := byID[a.InvoiceID]
			if !ok {
				return apperr.ErrNotFound
			}
			view, err := s.invoiceView(ctx, st, &i)
			if err != nil {
				return err
			}
			if amount(a.Amount.AmountMinor).Cmp(amount(view.Available.AmountMinor)) > 0 {
				return apperr.FieldError(fmt.Sprintf("allocations[%d].amount", n), "exceeds the open amount")
			}
		}
		raw, _ := json.Marshal(attachments)
		parent := model.InstallmentPayment{ID: uuid.NewString(), CompanyID: d.CompanyID, DealID: d.ID,
			InstallmentPlanID: *rows[0].InstallmentPlanID, AmountMinor: in.ClaimedAmount.AmountMinor, Currency: d.Currency,
			PaidOn: *paidOn, ExternalReference: ref, AttachmentIDs: raw, Status: "submitted", SubmittedBy: p.UserID, Version: 1, CreatedAt: s.clock()}
		if err := st.Deals().CreateInstallmentPayment(ctx, &parent); err != nil {
			return err
		}
		view := &InstallmentPaymentView{Payment: parent, Allocations: []InstallmentPaymentAllocationView{}, AllowedActions: []string{}}
		evidenceIDs := make([]string, 0, len(in.Allocations))
		for _, a := range in.Allocations {
			i := byID[a.InvoiceID]
			e := model.Evidence{ID: uuid.NewString(), InvoiceID: i.ID, PaymentGroupID: &parent.ID, AmountMinor: a.Amount.AmountMinor,
				Currency: parent.Currency, PaidOn: parent.PaidOn, ExternalReference: ref, AttachmentIDs: raw,
				Status: "submitted", SubmittedBy: p.UserID, Version: 1, CreatedAt: parent.CreatedAt}
			if err := s.shareEvidenceAttachments(ctx, st, p, attachments, e.ID); err != nil {
				return err
			}
			if err := st.Deals().AddEvidence(ctx, &e); err != nil {
				return err
			}
			evidenceIDs = append(evidenceIDs, e.ID)
			view.Allocations = append(view.Allocations, InstallmentPaymentAllocationView{InvoiceID: i.ID, EvidenceID: e.ID, Number: *i.InstallmentNumber, Amount: a.Amount})
		}
		if err := s.event(ctx, st, p, "deal.payment_submitted", "deal", d.ID, "", map[string]any{
			"paymentGroupId": parent.ID, "invoiceIds": ids, "evidenceIds": evidenceIDs, "amount": in.ClaimedAmount,
		}); err != nil {
			return err
		}
		if p.Has(model.PermPaymentsAccept) {
			view.AllowedActions = []string{"accept", "reject"}
		}
		result = view
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func inconsistentInstallmentPayment() error {
	return apperr.New(apperr.ErrConflict, "payment_group_inconsistent", "the complete installment payment is inconsistent")
}

// Validate the complete receipt without reading invoice views (which themselves
// contain receipt projections). All money is parsed before comparison.
func validateInstallmentPayment(d *model.Deal, parent *model.InstallmentPayment, children []model.Evidence, rows map[string]model.Invoice) error {
	claim, ok := (money.Money{AmountMinor: parent.AmountMinor, Currency: parent.Currency}).Parse()
	if !ok || claim.Sign() <= 0 || parent.CompanyID != d.CompanyID || parent.DealID != d.ID || parent.Currency != d.Currency ||
		len(children) == 0 || len(children) > 1000 || len(children) != len(rows) {
		return inconsistentInstallmentPayment()
	}
	if parent.Status != "submitted" && parent.Status != "accepted" && parent.Status != "rejected" {
		return inconsistentInstallmentPayment()
	}
	if parent.Status == "submitted" && (parent.DecidedBy != nil || parent.DecidedAt != nil || parent.DecisionReason != "") {
		return inconsistentInstallmentPayment()
	}
	var files []string
	if json.Unmarshal(parent.AttachmentIDs, &files) != nil {
		return inconsistentInstallmentPayment()
	}
	seenRows, seenEvidence := map[string]bool{}, map[string]bool{}
	total := new(big.Int)
	for _, e := range children {
		i, exists := rows[e.InvoiceID]
		n, valid := (money.Money{AmountMinor: e.AmountMinor, Currency: e.Currency}).Parse()
		var childFiles []string
		if !exists || seenRows[e.InvoiceID] || seenEvidence[e.ID] || e.ID == "" || e.PaymentGroupID == nil || *e.PaymentGroupID != parent.ID ||
			!valid || n.Sign() <= 0 || n.Cmp(amount(i.AmountMinor)) > 0 || e.Currency != parent.Currency || e.Status != parent.Status ||
			e.ExternalReference != parent.ExternalReference || !e.PaidOn.Equal(parent.PaidOn) || e.SubmittedBy != parent.SubmittedBy ||
			e.DecisionReason != parent.DecisionReason || !reflect.DeepEqual(e.DecidedBy, parent.DecidedBy) || !reflect.DeepEqual(e.DecidedAt, parent.DecidedAt) ||
			json.Unmarshal(e.AttachmentIDs, &childFiles) != nil || !reflect.DeepEqual(files, childFiles) ||
			i.CompanyID != parent.CompanyID || i.DealID != parent.DealID || i.Purpose != "monthly-installment" || i.Currency != parent.Currency ||
			i.InstallmentPlanID == nil || *i.InstallmentPlanID != parent.InstallmentPlanID || i.InstallmentNumber == nil || *i.InstallmentNumber < 1 {
			return inconsistentInstallmentPayment()
		}
		seenRows[e.InvoiceID], seenEvidence[e.ID] = true, true
		total.Add(total, n)
	}
	if total.Cmp(claim) != 0 {
		return inconsistentInstallmentPayment()
	}
	return nil
}

func paymentView(parent *model.InstallmentPayment, children []model.Evidence, rows map[string]model.Invoice) *InstallmentPaymentView {
	v := &InstallmentPaymentView{Payment: *parent, Allocations: []InstallmentPaymentAllocationView{}, AllowedActions: []string{}}
	for _, e := range children {
		i := rows[e.InvoiceID]
		v.Allocations = append(v.Allocations, InstallmentPaymentAllocationView{InvoiceID: i.ID, EvidenceID: e.ID, Number: *i.InstallmentNumber, Amount: money.Money{AmountMinor: e.AmountMinor, Currency: e.Currency}})
	}
	return v
}

func (s *Deal) installmentPaymentView(ctx context.Context, st Store, p *auth.Principal, d *model.Deal, parent *model.InstallmentPayment) (*InstallmentPaymentView, error) {
	if parent.CompanyID != p.CompanyID || parent.DealID != d.ID || !inScope(p, d.BranchID) {
		return nil, apperr.ErrNotFound
	}
	children, err := st.Deals().InstallmentPaymentEvidence(ctx, parent.ID)
	if err != nil {
		return nil, err
	}
	rows := make(map[string]model.Invoice, len(children))
	eligible := true
	for _, e := range children {
		i, err := st.Deals().Invoice(ctx, p.CompanyID, e.InvoiceID)
		if err != nil {
			return nil, err
		}
		rows[i.ID] = *i
		if err := s.monthlyPaymentEligibility(ctx, st, d, i); err != nil {
			if !errors.Is(err, apperr.ErrConflict) {
				return nil, err
			}
			eligible = false
		}
	}
	if err := validateInstallmentPayment(d, parent, children, rows); err != nil {
		return nil, err
	}
	v := paymentView(parent, children, rows)
	if eligible && parent.Status == "submitted" && p.Has(model.PermPaymentsAccept) {
		v.AllowedActions = []string{"accept", "reject"}
	}
	return v, nil
}

// DecideInstallmentPayment confirms or rejects every allocation together. The
// shared deal lock serializes these decisions with cancellation and submissions.
func (s *Deal) DecideInstallmentPayment(ctx context.Context, p *auth.Principal, id string, expected int64, accept, confirmation bool, why string) (*InstallmentPaymentView, error) {
	if err := p.Allow(model.PermPaymentsAccept); err != nil {
		return nil, err
	}
	if err := validate.IDs(id); err != nil {
		return nil, err
	}
	var validation apperr.Validation
	if accept && !confirmation {
		validation.Add("confirmation", "confirm that the payment was received")
	}
	if !accept {
		why = validate.Reason(&validation, why)
	} else {
		why = ""
	}
	if err := validation.Err(); err != nil {
		return nil, err
	}
	var result *InstallmentPaymentView
	err := s.store.InTx(ctx, func(st Store) error {
		ctx := st.Bind(ctx)
		identity, err := st.Deals().InstallmentPayment(ctx, p.CompanyID, id)
		if err != nil {
			return err
		}
		d, err := s.lockPaymentDeal(ctx, st, p, identity.DealID)
		if err != nil {
			return err
		}
		children, err := st.Deals().InstallmentPaymentEvidence(ctx, id)
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(children))
		for _, e := range children {
			ids = append(ids, e.InvoiceID)
		}
		sort.Strings(ids)
		locked, err := st.Deals().LockInstallmentInvoices(ctx, p.CompanyID, d.ID, ids)
		if err != nil {
			return err
		}
		parent, err := st.Deals().InstallmentPayment(ctx, p.CompanyID, id)
		if err != nil {
			return err
		}
		if parent.Version != expected {
			return apperr.ErrStale
		}
		if parent.Status != "submitted" {
			return apperr.New(apperr.ErrConflict, "payment_group_decided", "the installment payment was already decided")
		}
		children, err = st.Deals().InstallmentPaymentEvidence(ctx, id)
		if err != nil {
			return err
		}
		rows := make(map[string]model.Invoice, len(locked))
		for _, i := range locked {
			rows[i.ID] = i
			if err := s.monthlyPaymentEligibility(ctx, st, d, &i); err != nil {
				return err
			}
		}
		if err := validateInstallmentPayment(d, parent, children, rows); err != nil {
			return err
		}
		for _, i := range locked {
			balance, err := s.invoiceView(ctx, st, &i)
			if err != nil {
				return err
			}
			if new(big.Int).Add(amount(balance.Paid.AmountMinor), amount(balance.Pending.AmountMinor)).Cmp(amount(i.AmountMinor)) > 0 {
				return inconsistentInstallmentPayment()
			}
		}
		now := s.clock()
		parent.Status, parent.DecisionReason, parent.DecidedBy, parent.DecidedAt = "rejected", why, &p.UserID, &now
		if accept {
			parent.Status = "accepted"
		}
		evidenceIDs := make([]string, 0, len(children))
		for n := range children {
			e := &children[n]
			e.Status, e.DecisionReason, e.DecidedBy, e.DecidedAt = parent.Status, why, &p.UserID, &now
			if err := st.Deals().UpdateEvidence(ctx, e, e.Version); err != nil {
				return err
			}
			evidenceIDs = append(evidenceIDs, e.ID)
		}
		if err := st.Deals().UpdateInstallmentPayment(ctx, parent, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "deal.payment_"+parent.Status, "deal", d.ID, why, map[string]any{
			"paymentGroupId": id, "invoiceIds": ids, "evidenceIds": evidenceIDs,
			"amount": money.Money{AmountMinor: parent.AmountMinor, Currency: parent.Currency}, "decision": parent.Status,
		}); err != nil {
			return err
		}
		result = paymentView(parent, children, rows)
		return nil
	})
	return result, err
}

// The deal lock protects this plan-wide guard even for disjoint allocations.
// Legacy records are observed as-is; no grouping or historical rewrite occurs.
func (s *Deal) checkInstallmentReference(ctx context.Context, st Store, d *model.Deal, ref string) error {
	duplicate := apperr.New(apperr.ErrConflict, "payment_reference_used", "the payment reference is already used in this installment plan")
	_, err := st.Deals().InstallmentPaymentByReference(ctx, d.CompanyID, d.ID, ref)
	if err == nil {
		return duplicate
	}
	if !errors.Is(err, apperr.ErrNotFound) {
		return err
	}
	rows, err := st.Deals().Invoices(ctx, d.ID)
	if err != nil {
		return err
	}
	for _, i := range rows {
		if i.Purpose != "monthly-installment" {
			continue
		}
		es, err := st.Deals().Evidence(ctx, i.ID)
		if err != nil {
			return err
		}
		for _, e := range es {
			if e.ExternalReference == ref && e.Status != "rejected" {
				return duplicate
			}
		}
	}
	return nil
}
