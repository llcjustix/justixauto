package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"slices"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
	"justixauto/internal/pkg/validate"
)

// ReservationPayment owns the reservation's external payment evidence, full
// refund record and exactly one source-order credit. It never moves money.
type ReservationPayment struct{ Deps }

func NewReservationPayment(d Deps) *ReservationPayment { return &ReservationPayment{d} }

type ReservationMoneyView struct {
	ReservationID         string
	BuyerCompanyID        string
	SupplierCompanyID     string
	Revision              int64
	Currency              string
	AgreedAdvance         money.Money
	Accepted              money.Money
	Pending               money.Money
	PendingCount          int
	Refunded              money.Money
	RefundObligation      money.Money
	RefundStatus          string
	EvidenceEverSubmitted bool
	Evidence              []model.ReservationEvidence
	Refund                *model.ReservationRefund
	Credit                *model.ReservationCredit
	OrderID               *string
}

func reservationAdvance(r *model.Reservation) (*big.Int, string, error) {
	snap, err := decodeReservationSnapshot(r.CurrentTerms)
	if err != nil {
		return nil, "", err
	}
	if snap == nil || snap.Terms.UnitPrice == nil {
		return nil, "", apperr.New(apperr.ErrConflict, "reservation_terms_missing", "accepted terms are unavailable")
	}
	price, valid := snap.Terms.UnitPrice.Parse()
	if !valid || price.Sign() <= 0 {
		return nil, "", apperr.New(apperr.ErrConflict, "reservation_terms_invalid", "accepted money is invalid")
	}
	if snap.Terms.Advance == nil {
		return new(big.Int), snap.Terms.UnitPrice.Currency, nil
	}
	advance, valid := snap.Terms.Advance.Parse()
	if !valid || advance.Sign() <= 0 || snap.Terms.Advance.Currency != snap.Terms.UnitPrice.Currency {
		return nil, "", apperr.New(apperr.ErrConflict, "reservation_terms_invalid", "accepted advance is invalid")
	}
	return advance, snap.Terms.Advance.Currency, nil
}

// MoneyLocked requires the caller to hold this reservation's FOR UPDATE row
// lock in st's transaction. P18b uses this after its own reservation lock.
func (s *ReservationPayment) MoneyLocked(ctx context.Context, st Store, r *model.Reservation) (*ReservationMoneyView, error) {
	advance, currency, err := reservationAdvance(r)
	if err != nil {
		return nil, err
	}
	es, err := st.ReservationPayments().Evidence(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	accepted, pending := new(big.Int), new(big.Int)
	pendingCount := 0
	for _, e := range es {
		if e.Currency != currency || e.TermsDigest != r.CurrentDigest || e.TermsRevision != r.CurrentRevision {
			return nil, apperr.New(apperr.ErrConflict, "reservation_evidence_source_changed", "evidence is not bound to accepted terms")
		}
		switch e.Status {
		case "accepted":
			accepted.Add(accepted, amount(e.AmountMinor))
		case "submitted":
			pending.Add(pending, amount(e.AmountMinor))
			pendingCount++
		case "rejected":
		default:
			return nil, apperr.New(apperr.ErrConflict, "reservation_evidence_invalid", "evidence status is invalid")
		}
	}
	if new(big.Int).Add(new(big.Int).Set(accepted), pending).Cmp(advance) > 0 {
		return nil, apperr.New(apperr.ErrConflict, "reservation_advance_exceeded", "recorded evidence exceeds the agreement")
	}
	refund, err := st.ReservationPayments().Refund(ctx, r.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		refund, err = nil, nil
	}
	if err != nil {
		return nil, err
	}
	credit, err := st.ReservationPayments().CreditForReservation(ctx, r.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		credit, err = nil, nil
	}
	if err != nil {
		return nil, err
	}
	if credit != nil && refund != nil {
		return nil, apperr.New(apperr.ErrConflict, "reservation_money_conflict", "source is both credited and refunding")
	}
	refunded := new(big.Int)
	obligation := new(big.Int)
	refundStatus := "none"
	if refund != nil {
		refundStatus = refund.Status
		obligation.Set(accepted)
		if refund.Status == "confirmed" {
			refunded.SetString(refund.AmountMinor, 10)
			if refund.Currency != currency || refunded.Cmp(accepted) != 0 || pendingCount != 0 {
				return nil, apperr.New(apperr.ErrConflict, "reservation_refund_invalid", "confirmed refund does not settle the full advance")
			}
		}
	}
	if err := validateReservationCredit(r, credit, es, accepted, currency, pendingCount); err != nil {
		return nil, err
	}
	return &ReservationMoneyView{
		ReservationID: r.ID, BuyerCompanyID: r.BuyerCompanyID, SupplierCompanyID: r.SupplierCompanyID, Revision: r.Version, Currency: currency,
		AgreedAdvance: money.Of(advance, currency), Accepted: money.Of(accepted, currency), Pending: money.Of(pending, currency), PendingCount: pendingCount,
		Refunded: money.Of(refunded, currency), RefundObligation: money.Of(obligation, currency), RefundStatus: refundStatus,
		EvidenceEverSubmitted: len(es) > 0, Evidence: es, Refund: refund, Credit: credit, OrderID: r.OrderID,
	}, nil
}

// LockMoney is the transaction-bound entry for callers that do not already
// hold the reservation lock. Party checks belong to the invoking action.
func (s *ReservationPayment) LockMoney(ctx context.Context, st Store, id string) (*model.Reservation, *ReservationMoneyView, error) {
	if err := validate.IDs(id); err != nil {
		return nil, nil, err
	}
	r, _, err := st.Reservations().ForUpdate(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	v, err := s.MoneyLocked(ctx, st, r)
	return r, v, err
}

func (s *ReservationPayment) Get(ctx context.Context, p *auth.Principal, id string) (*ReservationMoneyView, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	var out *ReservationMoneyView
	err := s.store.InTx(ctx, func(st Store) error {
		r, v, err := s.LockMoney(ctx, st, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		out = v
		return nil
	})
	return out, err
}

func paymentSourceOpen(r *model.Reservation) error {
	if r.Status != model.ReservationActive || r.OrderID != nil {
		return apperr.New(apperr.ErrConflict, "reservation_money_closed", "reservation no longer accepts payment changes")
	}
	return nil
}

func (s *ReservationPayment) Submit(ctx context.Context, p *auth.Principal, id string, expected int64, in EvidenceInput) (*ReservationMoneyView, error) {
	if err := p.Allow(model.PermTrade); err != nil {
		return nil, err
	}
	claimed, paidOn, ref, attachments, err := parseEvidenceInput(s.clock(), in)
	if err != nil {
		return nil, err
	}
	var out *ReservationMoneyView
	err = s.store.InTx(ctx, func(st Store) error {
		r, v, err := s.LockMoney(ctx, st, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if err := paymentSourceOpen(r); err != nil {
			return err
		}
		if v.Refund != nil || v.Credit != nil {
			return apperr.New(apperr.ErrConflict, "reservation_money_settling", "reservation money is settling")
		}
		if in.ClaimedAmount.Currency != v.Currency {
			return apperr.FieldError("claimedAmount", "must be in "+v.Currency)
		}
		open := new(big.Int).Sub(amount(v.AgreedAdvance.AmountMinor), amount(v.Accepted.AmountMinor))
		open.Sub(open, amount(v.Pending.AmountMinor))
		if claimed.Cmp(open) > 0 {
			return apperr.FieldError("claimedAmount", "exceeds the open advance "+open.String())
		}
		now := s.clock()
		raw, _ := json.Marshal(attachments)
		e := &model.ReservationEvidence{
			ID: uuid.NewString(), ReservationID: r.ID, TermsDigest: r.CurrentDigest, TermsRevision: r.CurrentRevision,
			AmountMinor: claimed.String(), Currency: v.Currency, PaidOn: paidOn, ExternalReference: ref, AttachmentIDs: raw,
			Status: "submitted", SubmittedBy: p.UserID, Version: 1, CreatedAt: now,
		}
		if err := s.shareReservationAttachments(ctx, st, p.CompanyID, r.SupplierCompanyID, "commerce.reservation-evidence", e.ID, attachments); err != nil {
			return err
		}
		if err := st.ReservationPayments().AddEvidence(ctx, e); err != nil {
			return err
		}
		r.UpdatedAt = now
		if err := st.Reservations().Update(ctx, r, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.payment_submitted", "reservation", r.ID, "", map[string]any{"evidenceId": e.ID}); err != nil {
			return err
		}
		out, err = s.MoneyLocked(ctx, st, r)
		return err
	})
	return out, err
}

func (s *ReservationPayment) Decide(ctx context.Context, p *auth.Principal, evidenceID string, expectedEvidence, expectedReservation int64, accept, confirmation bool, why string) (*ReservationMoneyView, error) {
	if err := p.Allow(model.PermPaymentsAccept); err != nil {
		return nil, err
	}
	var fields apperr.Validation
	if accept && !confirmation {
		fields.Add("confirmation", "confirm external payment receipt")
	}
	if !accept {
		why = validate.Reason(&fields, why)
	}
	if err := fields.Err(); err != nil {
		return nil, err
	}
	if err := validate.IDs(evidenceID); err != nil {
		return nil, err
	}
	var out *ReservationMoneyView
	err := s.store.InTx(ctx, func(st Store) error {
		// This first lookup locates the source only. Re-read after its row lock.
		found, err := st.ReservationPayments().GetEvidence(ctx, evidenceID)
		if err != nil {
			return err
		}
		r, _, err := s.LockMoney(ctx, st, found.ReservationID)
		if err != nil {
			return err
		}
		e, err := s.decisionEvidenceLocked(ctx, st, r, p.CompanyID, evidenceID, expectedEvidence, expectedReservation)
		if err != nil {
			return err
		}
		v, err := s.MoneyLocked(ctx, st, r)
		if err != nil {
			return err
		}
		if v.Credit != nil || v.RefundStatus == "confirmed" {
			return apperr.New(apperr.ErrConflict, "reservation_money_closed", "money is already settled")
		}
		now := s.clock()
		e.Status, e.DecisionReason, e.DecidedBy, e.DecidedAt = "rejected", why, &p.UserID, &now
		if accept {
			e.Status = "accepted"
		}
		if err := st.ReservationPayments().DecideEvidence(ctx, e, expectedEvidence); err != nil {
			return err
		}
		r.UpdatedAt = now
		if err := st.Reservations().Update(ctx, r, expectedReservation); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.payment_"+e.Status, "reservation", r.ID, why, map[string]any{"evidenceId": e.ID}); err != nil {
			return err
		}
		out, err = s.MoneyLocked(ctx, st, r)
		return err
	})
	return out, err
}

func (s *ReservationPayment) RequestRefund(ctx context.Context, p *auth.Principal, id string, expected int64, why string) (*ReservationMoneyView, error) {
	if err := p.Allow(model.PermTrade); err != nil {
		return nil, err
	}
	var fields apperr.Validation
	why = validate.Reason(&fields, why)
	if err := fields.Err(); err != nil {
		return nil, err
	}
	var out *ReservationMoneyView
	err := s.store.InTx(ctx, func(st Store) error {
		r, v, err := s.LockMoney(ctx, st, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if err := paymentSourceOpen(r); err != nil {
			return err
		}
		if v.Refund != nil || v.Credit != nil {
			return apperr.New(apperr.ErrConflict, "reservation_money_settling", "refund or credit already exists")
		}
		if v.Accepted.AmountMinor == "0" && v.PendingCount == 0 {
			return apperr.New(apperr.ErrConflict, "no_refund_obligation", "no advance requires refund")
		}
		now := s.clock()
		refund := &model.ReservationRefund{ReservationID: r.ID, Status: "requested", RequestedBy: p.UserID, RequestedAt: now, RequestReason: why}
		if err := st.ReservationPayments().CreateRefund(ctx, refund); err != nil {
			return err
		}
		r.UpdatedAt = now
		if err := st.Reservations().Update(ctx, r, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.refund_requested", "reservation", r.ID, why, nil); err != nil {
			return err
		}
		out, err = s.MoneyLocked(ctx, st, r)
		return err
	})
	return out, err
}

func (s *ReservationPayment) ConfirmRefund(ctx context.Context, p *auth.Principal, id string, expected int64, in EvidenceInput, confirmation bool) (*ReservationMoneyView, error) {
	if err := p.Allow(model.PermPaymentsAccept); err != nil {
		return nil, err
	}
	if !confirmation {
		return nil, apperr.FieldError("confirmation", "confirm actual external refund")
	}
	claimed, refundedOn, ref, attachments, err := parseEvidenceInput(s.clock(), in)
	if err != nil {
		return nil, err
	}
	var out *ReservationMoneyView
	err = s.store.InTx(ctx, func(st Store) error {
		r, v, err := s.LockMoney(ctx, st, id)
		if err != nil {
			return err
		}
		if r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if err := paymentSourceOpen(r); err != nil {
			return err
		}
		if v.Refund == nil || v.Refund.Status != "requested" || v.Credit != nil {
			return apperr.New(apperr.ErrConflict, "refund_not_requested", "full refund is not awaiting confirmation")
		}
		if v.PendingCount != 0 {
			return apperr.New(apperr.ErrConflict, "pending_evidence", "decide every submitted payment before confirming refund")
		}
		if v.Accepted.AmountMinor == "0" {
			return apperr.New(apperr.ErrConflict, "no_refund_obligation", "no accepted advance requires refund")
		}
		if in.ClaimedAmount.Currency != v.Currency || claimed.Cmp(amount(v.Accepted.AmountMinor)) != 0 {
			return apperr.FieldError("claimedAmount", "must equal the full accepted advance in "+v.Currency)
		}
		now := s.clock()
		raw, _ := json.Marshal(attachments)
		v.Refund.Status, v.Refund.AmountMinor, v.Refund.Currency = "confirmed", claimed.String(), v.Currency
		v.Refund.ExternalReference, v.Refund.AttachmentIDs, v.Refund.RefundedOn = ref, raw, &refundedOn
		v.Refund.ConfirmedBy, v.Refund.ConfirmedAt = &p.UserID, &now
		if err := s.shareReservationAttachments(ctx, st, p.CompanyID, r.BuyerCompanyID, "commerce.reservation-refund", r.ID, attachments); err != nil {
			return err
		}
		if err := st.ReservationPayments().ConfirmRefund(ctx, v.Refund); err != nil {
			return err
		}
		r.UpdatedAt = now
		if err := st.Reservations().Update(ctx, r, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.refund_confirmed", "reservation", r.ID, "", map[string]any{"refundAmount": in.ClaimedAmount}); err != nil {
			return err
		}
		out, err = s.MoneyLocked(ctx, st, r)
		return err
	})
	return out, err
}

// ReleaseEligibleLocked is a guard, not a stock release. The caller holds the
// reservation lock and must perform its own final status transition in st.
func (s *ReservationPayment) ReleaseEligibleLocked(ctx context.Context, st Store, r *model.Reservation) (bool, error) {
	v, err := s.MoneyLocked(ctx, st, r)
	if err != nil {
		return false, err
	}
	if r.OrderID != nil || v.Credit != nil || v.PendingCount != 0 {
		return false, nil
	}
	return v.Accepted.AmountMinor == "0" || v.RefundStatus == "confirmed" && v.Refunded.AmountMinor == v.Accepted.AmountMinor, nil
}

// LinkCreditLocked is called by P18b2 in its order/claim/holder transaction,
// while the caller holds the reservation row lock. Zero money creates no row.
func (s *ReservationPayment) LinkCreditLocked(ctx context.Context, st Store, r *model.Reservation, o *model.Order) (*model.ReservationCredit, error) {
	if r.Status != model.ReservationActive || r.OrderID != nil && *r.OrderID != o.ID || o.Source != "reservation" ||
		o.BuyerCompanyID != r.BuyerCompanyID || o.SupplierCompanyID != r.SupplierCompanyID || o.Status == model.OrderCancelled {
		return nil, apperr.New(apperr.ErrConflict, "credit_source_mismatch", "order and reservation source do not match")
	}
	v, err := s.MoneyLocked(ctx, st, r)
	if err != nil {
		return nil, err
	}
	if v.Credit != nil {
		return nil, apperr.New(apperr.ErrConflict, "credit_exists", "reservation advance is already linked")
	}
	if v.Refund != nil || v.PendingCount != 0 {
		return nil, apperr.New(apperr.ErrConflict, "money_unsettled", "refund or pending payment prevents conversion")
	}
	if v.Accepted.AmountMinor == "0" {
		return nil, nil
	}
	orderTotal := model.TermsTotal(o.DecodeTerms())
	if orderTotal.Currency != v.Currency || amount(orderTotal.AmountMinor).Cmp(amount(v.Accepted.AmountMinor)) < 0 {
		return nil, apperr.New(apperr.ErrConflict, "credit_amount_mismatch", "accepted advance exceeds the matching order money")
	}
	ids := make([]string, 0)
	for _, e := range v.Evidence {
		if e.Status == "accepted" {
			ids = append(ids, e.ID)
		}
	}
	raw, _ := json.Marshal(ids)
	credit := &model.ReservationCredit{
		ReservationID: r.ID, OrderID: o.ID, BuyerCompanyID: r.BuyerCompanyID, SupplierCompanyID: r.SupplierCompanyID,
		AmountMinor: v.Accepted.AmountMinor, Currency: v.Currency, EvidenceIDs: raw, CreatedAt: s.clock(),
	}
	if err := st.ReservationPayments().CreateCredit(ctx, credit); err != nil {
		return nil, err
	}
	return credit, nil
}

// CreditForOrderLocked validates the unique converted source and returns nil
// for an ordinary order. The reservation lock precedes credit/invoice reads.
func (s *ReservationPayment) CreditForOrderLocked(ctx context.Context, st Store, o *model.Order) (*model.ReservationCredit, error) {
	r, _, err := st.Reservations().ByOrderForUpdate(ctx, o.ID)
	if errors.Is(err, apperr.ErrNotFound) && o.Source != "reservation" {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if o.Source != "reservation" || r.OrderID == nil || *r.OrderID != o.ID || r.BuyerCompanyID != o.BuyerCompanyID || r.SupplierCompanyID != o.SupplierCompanyID {
		return nil, apperr.New(apperr.ErrConflict, "credit_source_mismatch", "converted source does not match order")
	}
	v, err := s.MoneyLocked(ctx, st, r)
	if err != nil {
		return nil, err
	}
	if v.Accepted.AmountMinor != "0" && v.Credit == nil {
		return nil, apperr.New(apperr.ErrConflict, "credit_missing", "converted accepted money has no credit link")
	}
	if v.Credit != nil && (v.Credit.OrderID != o.ID || v.Credit.BuyerCompanyID != o.BuyerCompanyID || v.Credit.SupplierCompanyID != o.SupplierCompanyID) {
		return nil, apperr.New(apperr.ErrConflict, "credit_source_mismatch", "credit parties or order do not match")
	}
	if v.Credit != nil {
		orderTotal := model.TermsTotal(o.DecodeTerms())
		if orderTotal.Currency != v.Credit.Currency || amount(orderTotal.AmountMinor).Cmp(amount(v.Credit.AmountMinor)) < 0 {
			return nil, apperr.New(apperr.ErrConflict, "credit_amount_mismatch", "order money does not match source credit")
		}
	}
	return v.Credit, nil
}

// AttachCreditInvoiceLocked requires reservation lock first, then checks the
// exact issued order invoice. It does not consume or clone the advance.
func (s *ReservationPayment) AttachCreditInvoiceLocked(ctx context.Context, st Store, credit *model.ReservationCredit, invoice *model.Invoice) error {
	if credit == nil || invoice == nil {
		return apperr.New(apperr.ErrConflict, "credit_invoice_mismatch", "invoice and source credit are required")
	}
	r, _, err := st.Reservations().ForUpdate(ctx, credit.ReservationID)
	if err != nil {
		return err
	}
	v, err := s.MoneyLocked(ctx, st, r)
	if err != nil {
		return err
	}
	if v.Credit == nil || v.Credit.OrderID != credit.OrderID || v.Credit.AmountMinor != credit.AmountMinor || v.Credit.Currency != credit.Currency ||
		invoice.Status != "issued" || invoice.OrderID != credit.OrderID ||
		invoice.SupplierCompanyID != v.Credit.SupplierCompanyID || invoice.BuyerCompanyID != v.Credit.BuyerCompanyID || invoice.Currency != v.Credit.Currency {
		return apperr.New(apperr.ErrConflict, "credit_invoice_mismatch", "issued invoice does not match source credit")
	}
	if v.Credit.InvoiceID != nil {
		return apperr.New(apperr.ErrConflict, "credit_invoice_exists", "credit is already assigned")
	}
	if amount(invoice.TotalMinor).Cmp(amount(credit.AmountMinor)) < 0 {
		return apperr.New(apperr.ErrConflict, "credit_invoice_amount", "invoice is smaller than source credit")
	}
	return st.ReservationPayments().AttachInvoice(ctx, credit.ReservationID, invoice.ID, s.clock())
}

func validateReservationCredit(r *model.Reservation, credit *model.ReservationCredit, es []model.ReservationEvidence, accepted *big.Int, currency string, pendingCount int) error {
	if credit != nil {
		if credit.Currency != currency || amount(credit.AmountMinor).Cmp(accepted) != 0 || pendingCount != 0 ||
			credit.BuyerCompanyID != r.BuyerCompanyID || credit.SupplierCompanyID != r.SupplierCompanyID ||
			r.OrderID != nil && credit.OrderID != *r.OrderID {
			return apperr.New(apperr.ErrConflict, "reservation_credit_invalid", "source credit does not match accepted evidence")
		}
		acceptedIDs := []string{}
		for _, e := range es {
			if e.Status == "accepted" {
				acceptedIDs = append(acceptedIDs, e.ID)
			}
		}
		storedIDs := []string{}
		if err := json.Unmarshal(credit.EvidenceIDs, &storedIDs); err != nil {
			return err
		}
		slices.Sort(acceptedIDs)
		slices.Sort(storedIDs)
		if !slices.Equal(acceptedIDs, storedIDs) {
			return apperr.New(apperr.ErrConflict, "reservation_credit_invalid", "credit provenance does not match accepted evidence")
		}
	}
	return nil
}

func (s *ReservationPayment) shareReservationAttachments(ctx context.Context, st Store, companyID, recipientID, kind, resourceID string, attachments []string) error {
	for _, fileID := range attachments {
		if err := s.files.Share(st.Bind(ctx), companyID, fileID, recipientID, kind, resourceID); err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return apperr.FieldError("attachmentBindingIds", "contains files that are not yours")
			}
			return err
		}
	}
	return nil
}

func (s *ReservationPayment) decisionEvidenceLocked(ctx context.Context, st Store, r *model.Reservation, companyID, evidenceID string, expectedEvidence, expectedReservation int64) (*model.ReservationEvidence, error) {
	if r.SupplierCompanyID != companyID {
		return nil, apperr.ErrNotFound
	}
	if r.Version != expectedReservation {
		return nil, apperr.ErrStale
	}
	if err := paymentSourceOpen(r); err != nil {
		return nil, err
	}
	e, err := st.ReservationPayments().GetEvidence(ctx, evidenceID)
	if err != nil {
		return nil, err
	}
	if e.ReservationID != r.ID || e.TermsDigest != r.CurrentDigest || e.TermsRevision != r.CurrentRevision {
		return nil, apperr.New(apperr.ErrConflict, "evidence_source_changed", "evidence source changed")
	}
	if e.Version != expectedEvidence {
		return nil, apperr.ErrStale
	}
	if e.Status != "submitted" {
		return nil, apperr.New(apperr.ErrConflict, "evidence_decided", "evidence is already decided")
	}
	return e, nil
}
