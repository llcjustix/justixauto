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
)

type reservationWireDTO struct {
	ID                           string                           `json:"id"`
	Status                       model.ReservationStatus          `json:"status"`
	BuyerCompanyID               string                           `json:"buyerCompanyId"`
	SupplierCompanyID            string                           `json:"supplierCompanyId"`
	Buyer                        reservationPartyDTO              `json:"buyer"`
	Supplier                     reservationPartyDTO              `json:"supplier"`
	Requested                    *service.ReservationSnapshot     `json:"requested"`
	Offered                      *service.ReservationSnapshot     `json:"offered,omitempty"`
	Current                      *service.ReservationSnapshot     `json:"current,omitempty"`
	ContinuationProposed         *service.ReservationContinuation `json:"continuationProposed,omitempty"`
	ContinuationAccepted         *service.ReservationContinuation `json:"continuationAccepted,omitempty"`
	ContinuationProposedDigest   string                           `json:"continuationProposedDigest,omitempty"`
	ContinuationAcceptedDigest   string                           `json:"continuationAcceptedDigest,omitempty"`
	ContinuationProposedRevision string                           `json:"continuationProposedRevision,omitempty"`
	ContinuationAcceptedRevision string                           `json:"continuationAcceptedRevision,omitempty"`
	ContinuationProposedBy       *string                          `json:"continuationProposedBy,omitempty"`
	ContinuationAcceptedBy       *string                          `json:"continuationAcceptedBy,omitempty"`
	RequestedDigest              string                           `json:"requestedDigest"`
	OfferedDigest                string                           `json:"offeredDigest"`
	CurrentDigest                string                           `json:"currentDigest"`
	RequestedRevision            string                           `json:"requestedRevision"`
	OfferedRevision              string                           `json:"offeredRevision,omitempty"`
	CurrentRevision              string                           `json:"currentRevision,omitempty"`
	OrderID                      *string                          `json:"orderId,omitempty"`
	ExpiresAt                    *time.Time                       `json:"expiresAt,omitempty"`
	AdvanceDueAt                 *time.Time                       `json:"advanceDueAt,omitempty"`
	ExtendedBy                   *string                          `json:"extendedBy,omitempty"`
	ExtendedAt                   *time.Time                       `json:"extendedAt,omitempty"`
	ReleasedAt                   *time.Time                       `json:"releasedAt,omitempty"`
	ReleaseReason                string                           `json:"releaseReason,omitempty"`
	CancellationState            string                           `json:"cancellationState,omitempty"`
	CreatedAt                    time.Time                        `json:"createdAt"`
	UpdatedAt                    time.Time                        `json:"updatedAt"`
	Revision                     string                           `json:"revision"`
	AllowedActions               []string                         `json:"allowedActions"`
}
type reservationPartyDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toReservation(v *service.ReservationView) reservationWireDTO {
	r := v.Reservation
	return reservationWireDTO{
		ID: r.ID, Status: r.Status, BuyerCompanyID: r.BuyerCompanyID, SupplierCompanyID: r.SupplierCompanyID,
		Buyer: reservationPartyDTO{ID: v.Buyer.ID, Name: v.Buyer.Name}, Supplier: reservationPartyDTO{ID: v.Supplier.ID, Name: v.Supplier.Name},
		Requested: v.Requested, Offered: v.Offered, Current: v.Current, ContinuationProposed: v.ContinuationProposed, ContinuationAccepted: v.ContinuationAccepted, ContinuationProposedDigest: r.ContinuationProposedDigest, ContinuationAcceptedDigest: r.ContinuationAcceptedDigest, ContinuationProposedRevision: optionalReservationRevision(r.ContinuationProposedRevision), ContinuationAcceptedRevision: optionalReservationRevision(r.ContinuationAcceptedRevision), ContinuationProposedBy: r.ContinuationProposedBy, ContinuationAcceptedBy: r.ContinuationAcceptedBy, RequestedDigest: r.RequestedDigest, OfferedDigest: r.OfferedDigest,
		CurrentDigest: r.CurrentDigest, RequestedRevision: httpx.Revision(r.RequestedRevision), OfferedRevision: optionalReservationRevision(r.OfferedRevision), CurrentRevision: optionalReservationRevision(r.CurrentRevision), OrderID: r.OrderID, ExpiresAt: r.ExpiresAt, AdvanceDueAt: r.AdvanceDueAt, ExtendedBy: r.ExtendedBy, ExtendedAt: r.ExtendedAt, ReleasedAt: r.ReleasedAt, ReleaseReason: r.ReleaseReason, CancellationState: v.CancellationState, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		Revision: httpx.Revision(r.Version), AllowedActions: v.AllowedActions,
	}
}

func optionalReservationRevision(n int64) string {
	if n == 0 {
		return ""
	}
	return httpx.Revision(n)
}

// listReservations list reservations.
//
//	@Summary	List reservations
//	@Tags		commerce/reservations
//	@Produce	json
//	@Param		limit		query		int	false	"page size"
//	@Param		offset		query		int	false	"offset"
//	@Success	200			{object}	httpx.ListEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,422	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations [get]
func (h *Handler) listReservations(c *echo.Context) error {
	limit, err := httpx.IntQuery(c, "limit")
	if err != nil {
		return err
	}
	offset, err := httpx.IntQuery(c, "offset")
	if err != nil {
		return err
	}
	views, err := h.reservations.List(c.Request().Context(), auth.Get(c), limit, offset)
	if err != nil {
		return err
	}
	return httpx.List(c, mapSlice(views, toReservation), nil)
}

// getReservation get reservation.
//
//	@Summary	Get reservation
//	@Tags		commerce/reservations
//	@Produce	json
//	@Param		id			path		string	true	"reservation ID"
//	@Success	200			{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id} [get]
func (h *Handler) getReservation(c *echo.Context) error {
	v, err := h.reservations.Get(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservation(v), v.Reservation.Version)
}

// requestReservation request reservation.
//
//	@Summary	Request reservation
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		body			body		service.ReservationRequest	true	"request"
//	@Success	201				{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,409,422	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations [post]
func (h *Handler) requestReservation(c *echo.Context) error {
	var in service.ReservationRequest
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservations.Request(c.Request().Context(), auth.Get(c), in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toReservation(v), v.Reservation.Version)
}

// confirmReservation confirm reservation terms as supplier.
//
//	@Summary	Confirm reservation terms as supplier
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string							true	"reservation ID"
//	@Param		If-Match					header		string							true	"reservation revision"
//	@Param		body						body		service.ReservationConfirmation	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/supplier-confirmations [post]
func (h *Handler) confirmReservation(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in service.ReservationConfirmation
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservations.Confirm(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservation(v), v.Reservation.Version)
}

type reservationAcceptance struct {
	Digest string `json:"digest"`
}

// acceptReservation accept confirmed reservation terms.
//
//	@Summary	Accept confirmed reservation terms
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string							true	"reservation ID"
//	@Param		If-Match					header		string							true	"reservation revision"
//	@Param		body						body		handler.reservationAcceptance	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/acceptances [post]
func (h *Handler) acceptReservation(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in reservationAcceptance
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservations.Accept(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in.Digest)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservation(v), v.Reservation.Version)
}

// extendReservation extend reservation.
//
//	@Summary	Extend reservation
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string							true	"reservation ID"
//	@Param		If-Match					header		string							true	"reservation revision"
//	@Param		body						body		service.ReservationExtension	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/extensions [post]
func (h *Handler) extendReservation(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in service.ReservationExtension
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservations.Extend(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservation(v), v.Reservation.Version)
}

// cancelReservation cancel reservation.
//
//	@Summary	Cancel reservation
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string					true	"reservation ID"
//	@Param		If-Match					header		string					true	"reservation revision"
//	@Param		body						body		object{reason=string}	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/cancellations [post]
func (h *Handler) cancelReservation(c *echo.Context) error {
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
	v, err := h.reservations.Cancel(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in.Reason)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservation(v), v.Reservation.Version)
}

type reservationAllocationDTO struct {
	ID          string    `json:"id"`
	ClaimID     string    `json:"claimId"`
	LineID      string    `json:"lineId"`
	VehicleID   string    `json:"vehicleId"`
	HolderType  string    `json:"holderType"`
	HolderID    string    `json:"holderId"`
	OrderLineID *string   `json:"orderLineId,omitempty"`
	Current     bool      `json:"current"`
	CreatedAt   time.Time `json:"createdAt"`
}
type reservationAllocationsDTO struct {
	ReservationID string                     `json:"reservationId"`
	Allocations   []reservationAllocationDTO `json:"allocations"`
}

func toReservationAllocations(v *service.ReservationAllocationView) reservationAllocationsDTO {
	rows := make([]reservationAllocationDTO, 0, len(v.Allocations))
	for _, a := range v.Allocations {
		rows = append(rows, reservationAllocationDTO{ID: a.ID, ClaimID: a.ClaimID, LineID: a.ReservationLineID, VehicleID: a.VehicleID, HolderType: a.HolderType, HolderID: a.HolderID, OrderLineID: a.OrderLineID, Current: a.Current, CreatedAt: a.CreatedAt})
	}
	return reservationAllocationsDTO{ReservationID: v.ReservationID, Allocations: rows}
}

// reservationAllocations get reservation vehicle allocations.
//
//	@Summary	Get reservation vehicle allocations
//	@Tags		commerce/reservations
//	@Produce	json
//	@Param		id			path		string	true	"reservation ID"
//	@Success	200			{object}	httpx.DataEnvelope[handler.reservationAllocationsDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/allocations [get]
func (h *Handler) reservationAllocations(c *echo.Context) error {
	v, err := h.reservations.Allocations(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservationAllocations(v), v.Revision)
}

// allocateReservation allocate vehicles to reservation.
//
//	@Summary	Allocate vehicles to reservation
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string												true	"reservation ID"
//	@Param		If-Match					header		string												true	"reservation revision"
//	@Param		body						body		object{items=[]service.ReservationAllocationItem}	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationAllocationsDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/allocations [post]
func (h *Handler) allocateReservation(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in struct {
		Items []service.ReservationAllocationItem `json:"items"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservations.Allocate(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in.Items)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservationAllocations(v), v.Revision)
}

func continuationRevision(raw, field string) (int64, error) {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		return 0, apperr.FieldError(field, "must be a decimal revision")
	}
	return n, nil
}

// proposeReservationContinuation propose reservation continuation.
//
//	@Summary	Propose reservation continuation
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string							true	"reservation ID"
//	@Param		If-Match					header		string							true	"reservation revision"
//	@Param		body						body		service.ReservationContinuation	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/continuation-proposals [post]
func (h *Handler) proposeReservationContinuation(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in service.ReservationContinuation
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.reservations.ProposeContinuation(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservation(v), v.Reservation.Version)
}

// acceptReservationContinuation accept reservation continuation.
//
//	@Summary	Accept reservation continuation
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string											true	"reservation ID"
//	@Param		If-Match					header		string											true	"reservation revision"
//	@Param		body						body		object{digest=string,proposedRevision=string}	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.reservationWireDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/continuation-acceptances [post]
func (h *Handler) acceptReservationContinuation(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in struct {
		Digest           string `json:"digest"`
		ProposedRevision string `json:"proposedRevision"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	revision, err := continuationRevision(in.ProposedRevision, "proposedRevision")
	if err != nil {
		return err
	}
	v, err := h.reservations.AcceptContinuation(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in.Digest, revision)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toReservation(v), v.Reservation.Version)
}

// convertReservation convert reservation to an order.
//
//	@Summary	Convert reservation to an order
//	@Tags		commerce/reservations
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string											true	"reservation ID"
//	@Param		If-Match					header		string											true	"reservation revision"
//	@Param		body						body		object{digest=string,acceptedRevision=string}	true	"request"
//	@Success	201							{object}	object{data=object{reservation=handler.reservationWireDTO,reservationId=string,reservationRevision=string,orderId=string,orderRevision=string,continuationAccepted=service.ReservationContinuation},revision=string}
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/commerce/reservations/{id}/conversions [post]
func (h *Handler) convertReservation(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in struct {
		Digest           string `json:"digest"`
		AcceptedRevision string `json:"acceptedRevision"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	revision, err := continuationRevision(in.AcceptedRevision, "acceptedRevision")
	if err != nil {
		return err
	}
	v, err := h.reservations.Convert(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in.Digest, revision)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, map[string]any{"reservation": toReservation(v.Reservation), "reservationId": v.Reservation.Reservation.ID, "reservationRevision": httpx.Revision(v.Reservation.Reservation.Version), "orderId": v.Order.ID, "orderRevision": httpx.Revision(v.Order.Version), "continuationAccepted": v.Reservation.ContinuationAccepted}, v.Reservation.Reservation.Version)
}
