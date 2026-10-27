package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
	"justixauto/internal/pkg/money"
)

type reservationEvidenceDTO struct {
	ID                string      `json:"id"`
	Amount            money.Money `json:"amount"`
	PaidOn            string      `json:"paidOn"`
	ExternalReference string      `json:"externalReference"`
	AttachmentIDs     []string    `json:"attachmentIds"`
	Status            string      `json:"status"`
	DecisionReason    string      `json:"decisionReason"`
	SubmittedBy       string      `json:"submittedBy"`
	DecidedBy         *string     `json:"decidedBy,omitempty"`
	Revision          string      `json:"revision"`
	CreatedAt         time.Time   `json:"createdAt"`
	DecidedAt         *time.Time  `json:"decidedAt,omitempty"`
}
type reservationRefundDTO struct {
	Status            string       `json:"status"`
	RequestedBy       string       `json:"requestedBy"`
	RequestedAt       time.Time    `json:"requestedAt"`
	RequestReason     string       `json:"requestReason"`
	Amount            *money.Money `json:"amount,omitempty"`
	ExternalReference *string      `json:"externalReference,omitempty"`
	AttachmentIDs     *[]string    `json:"attachmentIds,omitempty"`
	RefundedOn        *string      `json:"refundedOn,omitempty"`
	ConfirmedBy       *string      `json:"confirmedBy,omitempty"`
	ConfirmedAt       *time.Time   `json:"confirmedAt,omitempty"`
}
type reservationAccountingDTO struct {
	ReservationID    string                   `json:"reservationId"`
	AgreedAdvance    money.Money              `json:"agreedAdvance"`
	Accepted         money.Money              `json:"accepted"`
	Pending          money.Money              `json:"pending"`
	Refunded         money.Money              `json:"refunded"`
	RefundObligation money.Money              `json:"refundObligation"`
	RefundStatus     string                   `json:"refundStatus"`
	Refund           *reservationRefundDTO    `json:"refund,omitempty"`
	Evidence         []reservationEvidenceDTO `json:"evidence"`
	CreditOrderID    *string                  `json:"creditOrderId,omitempty"`
	CreditInvoiceID  *string                  `json:"creditInvoiceId,omitempty"`
	AllowedActions   []string                 `json:"allowedActions"`
	Revision         string                   `json:"revision"`
}

func toReservationAccounting(p *auth.Principal, v *service.ReservationMoneyView) reservationAccountingDTO {
	d := reservationAccountingDTO{ReservationID: v.ReservationID, AgreedAdvance: v.AgreedAdvance, Accepted: v.Accepted, Pending: v.Pending, Refunded: v.Refunded, RefundObligation: v.RefundObligation, RefundStatus: v.RefundStatus, Evidence: []reservationEvidenceDTO{}, AllowedActions: []string{}, Revision: httpx.Revision(v.Revision)}
	for _, e := range v.Evidence {
		d.Evidence = append(d.Evidence, reservationEvidenceDTO{ID: e.ID, Amount: money.Money{AmountMinor: e.AmountMinor, Currency: e.Currency}, PaidOn: e.PaidOn.Format(time.DateOnly), ExternalReference: e.ExternalReference, AttachmentIDs: ids(e.AttachmentIDs), Status: e.Status, DecisionReason: e.DecisionReason, SubmittedBy: e.SubmittedBy, DecidedBy: e.DecidedBy, Revision: httpx.Revision(e.Version), CreatedAt: e.CreatedAt, DecidedAt: e.DecidedAt})
	}
	if r := v.Refund; r != nil {
		d.Refund = &reservationRefundDTO{Status: r.Status, RequestedBy: r.RequestedBy, RequestedAt: r.RequestedAt, RequestReason: r.RequestReason}
		if r.Status == "confirmed" {
			amount := money.Money{AmountMinor: r.AmountMinor, Currency: r.Currency}
			attachments := ids(r.AttachmentIDs)
			d.Refund.Amount, d.Refund.ExternalReference, d.Refund.AttachmentIDs = &amount, &r.ExternalReference, &attachments
			if r.RefundedOn != nil {
				refundedOn := r.RefundedOn.Format(time.DateOnly)
				d.Refund.RefundedOn = &refundedOn
			}
			d.Refund.ConfirmedBy, d.Refund.ConfirmedAt = r.ConfirmedBy, r.ConfirmedAt
		}
	}
	if v.Credit != nil {
		d.CreditOrderID = &v.Credit.OrderID
		d.CreditInvoiceID = v.Credit.InvoiceID
	}
	if p.CompanyID == v.SupplierCompanyID && p.Has(model.PermPaymentsAccept) {
		for _, e := range v.Evidence {
			if e.Status == "submitted" {
				d.AllowedActions = append(d.AllowedActions, "accept-payment", "reject-payment")
				break
			}
		}
		if v.RefundStatus == "requested" && v.PendingCount == 0 && v.Accepted.AmountMinor != "0" {
			d.AllowedActions = append(d.AllowedActions, "confirm-refund")
		}
	}
	if p.CompanyID == v.BuyerCompanyID && p.Has(model.PermTrade) && v.Credit == nil && v.RefundStatus == "none" && v.OrderID == nil {
		open := amount(v.AgreedAdvance.AmountMinor)
		open.Sub(open, amount(v.Accepted.AmountMinor))
		open.Sub(open, amount(v.Pending.AmountMinor))
		if open.Sign() > 0 {
			d.AllowedActions = append(d.AllowedActions, "submit-payment")
		}
	}
	if (p.CompanyID == v.BuyerCompanyID || p.CompanyID == v.SupplierCompanyID) && p.Has(model.PermTrade) && v.Credit == nil && v.RefundStatus == "none" && v.OrderID == nil && (v.Accepted.AmountMinor != "0" || v.PendingCount > 0) {
		d.AllowedActions = append(d.AllowedActions, "request-refund")
	}
	return d
}

// reservationAccounting get reservation accounting.
//
//	@Summary	Get reservation accounting
//	@Tags		commerce/reservations
//	@Produce	json
//	@Param		id			path		string	true	"reservation ID"
//	@Success	200			{object}	httpx.DataEnvelope[handler.reservationAccountingDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/accounting [get]
func (h *Handler) reservationAccounting(c *echo.Context) error {
	v, err := h.reservationPayments.Get(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservationAccounting(auth.Get(c), v), v.Revision)
}

// submitReservationPayment submit reservation payment evidence.
//
//	@Summary	Submit reservation payment evidence
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string					true	"reservation ID"
//	@Param		If-Match					header		string					true	"reservation revision"
//	@Param		body						body		service.EvidenceInput	true	"request"
//	@Success	201							{object}	httpx.DataEnvelope[handler.reservationAccountingDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/payment-evidence [post]
func (h *Handler) submitReservationPayment(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in service.EvidenceInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservationPayments.Submit(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toReservationAccounting(auth.Get(c), v), v.Revision)
}

// decideReservationPayment accept or reject reservation payment evidence.
//
//	@Summary	Accept or reject reservation payment evidence
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string																true	"payment evidence ID"
//	@Param		If-Match					header		string																true	"payment evidence revision"
//	@Param		body						body		object{reservationRevision=string,confirmation=bool,reason=string}	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationAccountingDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservation-payment-evidence/{id}/accept [post]
//	@Router		/commerce/reservation-payment-evidence/{id}/reject [post]
func (h *Handler) decideReservationPayment(accept bool) echo.HandlerFunc {
	return func(c *echo.Context) error {
		expected, err := httpx.IfMatch(c)
		if err != nil {
			return err
		}
		var in struct {
			ReservationRevision string `json:"reservationRevision"`
			Confirmation        bool   `json:"confirmation"`
			Reason              string `json:"reason"`
		}
		if err := httpx.Bind(c, &in); err != nil {
			return err
		}
		reservationRevision, err := strconv.ParseInt(in.ReservationRevision, 10, 64)
		if err != nil || reservationRevision < 1 {
			return apperr.FieldError("reservationRevision", "must be a revision number")
		}
		v, err := h.reservationPayments.Decide(c.Request().Context(), auth.Get(c), c.Param("id"), expected, reservationRevision, accept, in.Confirmation, in.Reason)
		if err != nil {
			return err
		}
		return httpx.Data(c, http.StatusOK, toReservationAccounting(auth.Get(c), v), v.Revision)
	}
}

// requestReservationRefund request reservation refund.
//
//	@Summary	Request reservation refund
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string					true	"reservation ID"
//	@Param		If-Match					header		string					true	"reservation revision"
//	@Param		body						body		object{reason=string}	true	"request"
//	@Success	201							{object}	httpx.DataEnvelope[handler.reservationAccountingDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/refund-requests [post]
func (h *Handler) requestReservationRefund(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservationPayments.RequestRefund(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in.Reason)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toReservationAccounting(auth.Get(c), v), v.Revision)
}

// confirmReservationRefund confirm reservation refund.
//
//	@Summary	Confirm reservation refund
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string										true	"reservation ID"
//	@Param		If-Match					header		string										true	"reservation revision"
//	@Param		body						body		service.EvidenceInput{confirmation=bool}	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationAccountingDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/refund-confirmations [post]
func (h *Handler) confirmReservationRefund(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in struct {
		service.EvidenceInput
		Confirmation bool `json:"confirmation"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservationPayments.ConfirmRefund(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in.EvidenceInput, in.Confirmation)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservationAccounting(auth.Get(c), v), v.Revision)
}
