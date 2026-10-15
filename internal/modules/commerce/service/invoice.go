package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
	"justixauto/internal/pkg/validate"
)

// Invoice manages order invoices and payment evidence.
type Invoice struct {
	Deps
	reservationPayments *ReservationPayment
}

// InvoiceView adds exact paid and outstanding amounts.
type InvoiceView struct {
	Invoice     model.Invoice
	Evidence    []model.Evidence
	Paid        money.Money
	Outstanding money.Money
	Pending     money.Money // submitted, not decided yet
}

func amount(s string) *big.Int {
	n, _ := new(big.Int).SetString(s, 10)
	if n == nil {
		return new(big.Int)
	}
	return n
}

func invoiceMinor(raw string) (*big.Int, error) {
	n, ok := new(big.Int).SetString(raw, 10)
	if !ok || n.Sign() <= 0 {
		return nil, apperr.New(apperr.ErrConflict, "invoice_money_invalid", "invoice money is invalid")
	}
	return n, nil
}

// The first invoice lookup only locates its order. The second is authoritative
// after the converted reservation lock, which serializes credit-aware writes.
func (s *Invoice) lockedInvoice(ctx context.Context, st Store, p *auth.Principal, id string) (*model.Invoice, error) {
	i, err := s.invoice(ctx, st, p, id)
	if err != nil {
		return nil, err
	}
	o, err := st.Deals().Order(ctx, p.CompanyID, i.OrderID)
	if err != nil {
		return nil, err
	}
	if _, err = s.reservationPayments.CreditForOrderLocked(ctx, st, o); err != nil {
		return nil, err
	}
	return s.invoice(ctx, st, p, id)
}

func (s *Invoice) view(ctx context.Context, st Store, i *model.Invoice) (*InvoiceView, error) {
	// Lock the source before any accounting read, including ordinary evidence.
	order, err := st.Deals().Order(ctx, i.SupplierCompanyID, i.OrderID)
	if err != nil {
		return nil, err
	}
	credit, err := s.reservationPayments.CreditForOrderLocked(ctx, st, order)
	if err != nil {
		return nil, err
	}
	es, err := st.Invoices().Evidence(ctx, i.ID)
	if err != nil {
		return nil, err
	}
	total, err := invoiceMinor(i.TotalMinor)
	if err != nil {
		return nil, err
	}
	paid, pending := new(big.Int), new(big.Int)
	for _, e := range es {
		if e.Currency != i.Currency {
			return nil, apperr.New(apperr.ErrConflict, "invoice_money_invalid", "evidence currency differs from invoice")
		}
		n, err := invoiceMinor(e.AmountMinor)
		if err != nil {
			return nil, err
		}
		switch e.Status {
		case "accepted":
			paid.Add(paid, n)
		case "submitted":
			pending.Add(pending, n)
		case "rejected":
		default:
			return nil, apperr.New(apperr.ErrConflict, "invoice_evidence_invalid", "evidence status is invalid")
		}
	}
	// The source reservation row is the accounting serialization point. A
	// credit is never synthesized as invoice evidence, and only its assigned
	// issued invoice may include it in paid money.
	if credit != nil && credit.InvoiceID != nil && *credit.InvoiceID == i.ID && i.Status == "issued" {
		if credit.Currency != i.Currency {
			return nil, apperr.New(apperr.ErrConflict, "credit_invoice_mismatch", "source credit currency does not match invoice")
		}
		n, err := invoiceMinor(credit.AmountMinor)
		if err != nil {
			return nil, err
		}
		paid.Add(paid, n)
	}
	if new(big.Int).Add(new(big.Int).Set(paid), pending).Cmp(total) > 0 {
		return nil, apperr.New(apperr.ErrConflict, "invoice_paid_exceeds_total", "paid and pending money exceeds invoice total")
	}
	outstanding := new(big.Int).Sub(total, paid)
	return &InvoiceView{
		Invoice: *i, Evidence: es, Paid: money.Of(paid, i.Currency),
		Outstanding: money.Of(outstanding, i.Currency), Pending: money.Of(pending, i.Currency),
	}, nil
}

// invoiceIssueOrder preserves the initial tenant/party lookup and re-reads
// the order only after locking its converted reservation source.
func (s *Invoice) invoiceIssueOrder(ctx context.Context, st Store, p *auth.Principal, orderID string, expected int64) (*model.Order, *model.ReservationCredit, error) {
	if err := validate.IDs(orderID); err != nil {
		return nil, nil, err
	}
	o, err := st.Deals().Order(ctx, p.CompanyID, orderID)
	if err != nil {
		return nil, nil, err
	}
	if o.Party(p.CompanyID) != "supplier" {
		return nil, nil, apperr.New(apperr.ErrForbidden, "wrong_party", "only the supplier issues invoices")
	}
	// The order read above identifies the source. Re-read after the source
	// lock so terms, revision and invoice association are one decision.
	credit, err := s.reservationPayments.CreditForOrderLocked(ctx, st, o)
	if err != nil {
		return nil, nil, err
	}
	o, err = st.Deals().Order(ctx, p.CompanyID, orderID)
	if err != nil {
		return nil, nil, err
	}
	if o.Version != expected {
		return nil, nil, apperr.ErrStale
	}
	if o.Status == model.AwaitingSupplier || o.Status == model.OrderCancelled {
		return nil, nil, apperr.New(apperr.ErrConflict, "invalid_transition", "invoices are issued for accepted orders")
	}
	return o, credit, nil
}

// issuedOrderInvoice rejects inconsistent duplicate issued rows before any
// credit attachment or invoice creation.
func issuedOrderInvoice(ctx context.Context, st Store, orderID string) (*model.Invoice, error) {
	existing, err := st.Invoices().ForOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	var issued *model.Invoice
	for j := range existing {
		if existing[j].Status == "issued" {
			if issued != nil {
				return nil, apperr.New(apperr.ErrConflict, "invoice_exists", "multiple issued invoices exist")
			}
			issued = &existing[j]
		}
	}
	return issued, nil
}

// attachIssuedInvoiceCredit handles the existing-invoice retry and attachment
// paths. The caller assigns its result only when the original view read occurs.
func (s *Invoice) attachIssuedInvoiceCredit(ctx context.Context, st Store, p *auth.Principal, o *model.Order, expected int64, credit *model.ReservationCredit, issued *model.Invoice, total money.Money) error {
	if credit == nil || issued.SupplierCompanyID != o.SupplierCompanyID || issued.BuyerCompanyID != o.BuyerCompanyID || issued.Currency != total.Currency || issued.TotalMinor != total.AmountMinor {
		return apperr.New(apperr.ErrConflict, "invoice_exists", "the order already has an issued invoice")
	}
	if credit.InvoiceID != nil {
		if *credit.InvoiceID != issued.ID {
			return apperr.New(apperr.ErrConflict, "credit_invoice_exists", "credit belongs to another invoice")
		}
		return nil
	}
	if err := s.reservationPayments.AttachCreditInvoiceLocked(ctx, st, credit, issued); err != nil {
		return err
	}
	o.UpdatedAt = s.clock()
	if err := st.Deals().UpdateOrder(ctx, o, expected); err != nil {
		return err
	}
	if err := s.event(ctx, st, p, "order.invoice_credit_attached", "order", o.ID, "", map[string]any{"invoiceId": issued.ID}); err != nil {
		return err
	}
	return nil
}

// createIssuedInvoice persists a new invoice, its optional source credit,
// order revision and event in their original order under the caller's Store.
func (s *Invoice) createIssuedInvoice(ctx context.Context, st Store, p *auth.Principal, o *model.Order, expected int64, credit *model.ReservationCredit, schedule []model.Installment, total money.Money, dueDate string) (*model.Invoice, error) {
	if len(schedule) == 0 {
		if _, err := time.Parse(time.DateOnly, dueDate); err != nil {
			return nil, apperr.FieldError("dueDate", "the order has no payment schedule: give a due date (YYYY-MM-DD)")
		}
		schedule = []model.Installment{{ID: uuid.NewString(), Amount: total, DueDate: dueDate}}
	}
	raw, _ := json.Marshal(schedule)
	now := s.clock()
	i := &model.Invoice{
		ID: uuid.NewString(), OrderID: o.ID, SupplierCompanyID: o.SupplierCompanyID, BuyerCompanyID: o.BuyerCompanyID,
		TotalMinor: total.AmountMinor, Currency: total.Currency, Schedule: raw, Status: "issued", Version: 1,
		CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now,
	}
	// Lock the converted reservation before credit/invoice operations. This
	// is a no-op for ordinary orders and preserves their existing behavior.
	if err := st.Invoices().Create(ctx, i); err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			return nil, apperr.New(apperr.ErrConflict, "invoice_exists", "the order already has an issued invoice; void it first")
		}
		return nil, err
	}
	if credit != nil {
		if err := s.reservationPayments.AttachCreditInvoiceLocked(ctx, st, credit, i); err != nil {
			return nil, err
		}
	}
	o.UpdatedAt = now
	if err := st.Deals().UpdateOrder(ctx, o, expected); err != nil {
		return nil, err
	}
	if err := s.event(ctx, st, p, "order.invoice_issued", "order", o.ID, "", map[string]any{"invoiceId": i.ID, "total": total}); err != nil {
		return nil, err
	}
	return i, nil
}

// Issue creates the supplier's invoice from the order's current terms. The
// schedule is copied; without one, the whole amount is due on dueDate.
func (s *Invoice) Issue(ctx context.Context, p *auth.Principal, orderID string, expected int64, dueDate string) (*InvoiceView, error) {
	if err := p.Allow(model.PermInvoicesManage); err != nil {
		return nil, err
	}
	var result *InvoiceView
	err := s.store.InTx(ctx, func(st Store) error {
		o, credit, err := s.invoiceIssueOrder(ctx, st, p, orderID, expected)
		if err != nil {
			return err
		}
		terms := o.DecodeTerms()
		total := model.TermsTotal(terms)
		if _, err := invoiceMinor(total.AmountMinor); err != nil {
			return err
		}
		// An issued invoice may predate the association. Attach only the
		// matching unassigned credit; never create a second issued invoice.
		issued, err := issuedOrderInvoice(ctx, st, o.ID)
		if err != nil {
			return err
		}
		if issued != nil {
			if err := s.attachIssuedInvoiceCredit(ctx, st, p, o, expected, credit, issued, total); err != nil {
				return err
			}
			result, err = s.view(ctx, st, issued)
			return err
		}
		i, err := s.createIssuedInvoice(ctx, st, p, o, expected, credit, terms.PaymentSchedule, total, dueDate)
		if err != nil {
			return err
		}
		result, err = s.view(ctx, st, i)
		return err
	})
	return result, err
}

func (s *Invoice) invoice(ctx context.Context, st Store, p *auth.Principal, id string) (*model.Invoice, error) {
	if err := validate.IDs(id); err != nil {
		return nil, err
	}
	return st.Invoices().Get(ctx, p.CompanyID, id)
}

// Void withdraws an invoice that has no accepted payment, e.g. after an
// addendum changed the terms; a new invoice can then be issued.
func (s *Invoice) Void(ctx context.Context, p *auth.Principal, id string, expected int64, why string) (*InvoiceView, error) {
	if err := p.Allow(model.PermInvoicesManage); err != nil {
		return nil, err
	}
	var v apperr.Validation
	why = validate.Reason(&v, why)
	if err := v.Err(); err != nil {
		return nil, err
	}
	var result *InvoiceView
	err := s.store.InTx(ctx, func(st Store) error {
		i, err := s.lockedInvoice(ctx, st, p, id)
		if err != nil {
			return err
		}
		if i.Version != expected {
			return apperr.ErrStale
		}
		if i.SupplierCompanyID != p.CompanyID {
			return apperr.New(apperr.ErrForbidden, "wrong_party", "only the supplier voids invoices")
		}
		view, err := s.view(ctx, st, i)
		if err != nil {
			return err
		}
		if i.Status != "issued" || view.Paid.AmountMinor != "0" || view.Pending.AmountMinor != "0" {
			return apperr.New(apperr.ErrConflict, "invoice_has_payments", "only an issued invoice without payments can be voided")
		}
		i.Status, i.VoidReason, i.UpdatedAt = "void", why, s.clock()
		if err := st.Invoices().Update(ctx, i, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "order.invoice_voided", "order", i.OrderID, why, map[string]any{"invoiceId": i.ID}); err != nil {
			return err
		}
		result, err = s.view(ctx, st, i)
		return err
	})
	return result, err
}

type EvidenceInput struct {
	ClaimedAmount     money.Money `json:"claimedAmount"`
	PaidOn            string      `json:"paidOn"`
	ExternalReference string      `json:"externalReference"`
	AttachmentIDs     []string    `json:"attachmentBindingIds"`
}

// parseEvidenceInput validates the field-level shape of a payment evidence
// submission, preserving the original validation order.
func parseEvidenceInput(now time.Time, in EvidenceInput) (claimed *big.Int, paidOn time.Time, ref string, attachments []string, err error) {
	var v apperr.Validation
	var ok bool
	claimed, ok = in.ClaimedAmount.Parse()
	if !ok || claimed.Sign() == 0 {
		v.Add("claimedAmount", "must be a positive amount in minor units with a currency code")
	}
	paidOn, perr := time.Parse(time.DateOnly, in.PaidOn)
	if perr != nil || paidOn.After(now) {
		v.Add("paidOn", "a date (YYYY-MM-DD), not in the future")
	}
	ref = validate.Text(&v, "externalReference", in.ExternalReference, 1, 100)
	attachments = validate.UniqueIDs(&v, "attachmentBindingIds", in.AttachmentIDs)
	if len(attachments) > 10 {
		v.Add("attachmentBindingIds", "at most 10 files")
	}
	if err := v.Err(); err != nil {
		return nil, time.Time{}, "", nil, err
	}
	return claimed, paidOn, ref, attachments, nil
}

// shareEvidenceFiles makes the buyer's proof attachments readable by the
// payee for this evidence claim.
func (s *Invoice) shareEvidenceFiles(txctx context.Context, buyerCompanyID, supplierCompanyID string, attachments []string, evidenceID string) error {
	for _, f := range attachments {
		if err := s.files.Share(txctx, buyerCompanyID, f, supplierCompanyID, "commerce.payment-evidence", evidenceID); err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return apperr.FieldError("attachmentBindingIds", "contains files that are not yours")
			}
			return err
		}
	}
	return nil
}

// SubmitEvidence records the buyer's claim of an external payment. Claims can
// never exceed what is still outstanding (including undecided claims).
func (s *Invoice) SubmitEvidence(ctx context.Context, p *auth.Principal, invoiceID string, in EvidenceInput) (*InvoiceView, error) {
	claimed, paidOn, ref, attachments, err := parseEvidenceInput(s.clock(), in)
	if err != nil {
		return nil, err
	}
	var result *InvoiceView
	err = s.store.InTx(ctx, func(st Store) error {
		i, err := s.lockedInvoice(ctx, st, p, invoiceID)
		if err != nil {
			return err
		}
		if i.BuyerCompanyID != p.CompanyID {
			return apperr.New(apperr.ErrForbidden, "wrong_party", "only the buyer submits payment evidence")
		}
		if i.Status != "issued" {
			return apperr.New(apperr.ErrConflict, "invoice_void", "the invoice was voided")
		}
		if in.ClaimedAmount.Currency != i.Currency {
			return apperr.FieldError("claimedAmount", "must be in "+i.Currency)
		}
		view, err := s.view(ctx, st, i)
		if err != nil {
			return err
		}
		open := new(big.Int).Sub(amount(view.Outstanding.AmountMinor), amount(view.Pending.AmountMinor))
		if claimed.Cmp(open) > 0 {
			return apperr.FieldError("claimedAmount", "exceeds the open amount "+open.String())
		}
		now := s.clock()
		rawIDs, _ := json.Marshal(attachments)
		e := &model.Evidence{
			ID: uuid.NewString(), InvoiceID: i.ID, AmountMinor: claimed.String(), Currency: i.Currency, PaidOn: paidOn,
			ExternalReference: ref, AttachmentIDs: rawIDs, Status: "submitted", SubmittedBy: p.UserID, Version: 1, CreatedAt: now,
		}
		if err := s.shareEvidenceFiles(st.Bind(ctx), p.CompanyID, i.SupplierCompanyID, attachments, e.ID); err != nil {
			return err
		}
		if err := st.Invoices().AddEvidence(ctx, e); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "order.payment_submitted", "order", i.OrderID, "", map[string]any{"evidenceId": e.ID, "amount": in.ClaimedAmount}); err != nil {
			return err
		}
		result, err = s.view(ctx, st, i)
		return err
	})
	return result, err
}

// requireEvidenceDecision validates the authoritative post-lock evidence read
// before accounting validation and mutation, retaining error precedence.
func requireEvidenceDecision(p *auth.Principal, i *model.Invoice, e *model.Evidence, expected int64) error {
	if i.SupplierCompanyID != p.CompanyID {
		return apperr.New(apperr.ErrForbidden, "wrong_party", "only the payee decides on payment evidence")
	}
	if e.Version != expected {
		return apperr.ErrStale
	}
	if e.Status != "submitted" {
		return apperr.New(apperr.ErrConflict, "evidence_decided", "the evidence was already decided")
	}
	if i.Status != "issued" {
		return apperr.New(apperr.ErrConflict, "invoice_void", "the invoice was voided")
	}
	if e.InvoiceID != i.ID {
		return apperr.New(apperr.ErrConflict, "evidence_source_changed", "evidence source changed")
	}
	return nil
}

// DecideEvidence: the supplier's finance staff accept (with explicit
// confirmation) or reject (with a reason) a payment claim.
func (s *Invoice) DecideEvidence(ctx context.Context, p *auth.Principal, evidenceID string, expected int64, accept, confirmation bool, why string) (*InvoiceView, error) {
	if err := p.Allow(model.PermPaymentsAccept); err != nil {
		return nil, err
	}
	var v apperr.Validation
	if accept && !confirmation {
		v.Add("confirmation", "confirm that the payment was received")
	}
	if !accept {
		why = validate.Reason(&v, why)
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	var result *InvoiceView
	err := s.store.InTx(ctx, func(st Store) error {
		if err := validate.IDs(evidenceID); err != nil {
			return err
		}
		e, err := st.Invoices().GetEvidence(ctx, evidenceID)
		if err != nil {
			return err
		}
		i, err := s.lockedInvoice(ctx, st, p, e.InvoiceID)
		if err != nil {
			return err
		}
		e, err = st.Invoices().GetEvidence(ctx, evidenceID)
		if err != nil {
			return err
		}
		if err := requireEvidenceDecision(p, i, e, expected); err != nil {
			return err
		}
		if _, err := s.view(ctx, st, i); err != nil {
			return err
		}
		now := s.clock()
		e.Status, e.DecisionReason, e.DecidedBy, e.DecidedAt = "rejected", why, &p.UserID, &now
		if accept {
			e.Status = "accepted"
		}
		if err := st.Invoices().UpdateEvidence(ctx, e, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "order.payment_"+e.Status, "order", i.OrderID, why, map[string]any{"evidenceId": e.ID}); err != nil {
			return err
		}
		result, err = s.view(ctx, st, i)
		return err
	})
	return result, err
}

func (s *Invoice) Get(ctx context.Context, p *auth.Principal, id string) (*InvoiceView, error) {
	var out *InvoiceView
	err := s.store.InTx(ctx, func(st Store) error {
		i, err := s.lockedInvoice(ctx, st, p, id)
		if err != nil {
			return err
		}
		out, err = s.view(ctx, st, i)
		return err
	})
	return out, err
}

func (s *Invoice) ForOrder(ctx context.Context, p *auth.Principal, orderID string) ([]InvoiceView, error) {
	if err := validate.IDs(orderID); err != nil {
		return nil, err
	}
	var out []InvoiceView
	err := s.store.InTx(ctx, func(st Store) error {
		o, err := st.Deals().Order(ctx, p.CompanyID, orderID)
		if err != nil {
			return err
		}
		if _, err := s.reservationPayments.CreditForOrderLocked(ctx, st, o); err != nil {
			return err
		}
		is, err := st.Invoices().ForOrder(ctx, orderID)
		if err != nil {
			return err
		}
		out = make([]InvoiceView, 0, len(is))
		for i := range is {
			v, err := s.view(ctx, st, &is[i])
			if err != nil {
				return err
			}
			out = append(out, *v)
		}
		return nil
	})
	return out, err
}

func init() {
	// Orders with accepted payments cannot simply be cancelled.
	cancellationChecks = append(cancellationChecks, func(ctx context.Context, st Store, o *model.Order) (string, error) {
		credit, err := NewReservationPayment(Deps{store: st}).CreditForOrderLocked(ctx, st, o)
		if err != nil {
			return "", err
		}
		if credit != nil {
			return "the order has accepted payments", nil
		}
		is, err := st.Invoices().ForOrder(ctx, o.ID)
		if err != nil {
			return "", err
		}
		for _, i := range is {
			es, err := st.Invoices().Evidence(ctx, i.ID)
			if err != nil {
				return "", err
			}
			for _, e := range es {
				if e.Status == "accepted" {
					return "the order has accepted payments", nil
				}
			}
		}
		return "", nil
	})
}
