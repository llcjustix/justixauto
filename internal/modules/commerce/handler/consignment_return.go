package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

func (h *Handler) listConsignmentReturns(c *echo.Context) error {
	limit, offset, err := paging(c)
	if err != nil { return err }
	p := auth.Get(c)
	rows, err := h.returns.List(c.Request().Context(), p, c.QueryParam("orderId"), limit, offset)
	if err != nil { return err }
	pageLimit := limit
	if pageLimit == 0 { pageLimit = 50 }
	var next *string
	if len(rows) == pageLimit { value := strconv.Itoa(offset + pageLimit); next = &value }
	items := make([]consignmentReturnDTO, 0, len(rows))
	for i := range rows { items = append(items, toConsignmentReturn(&rows[i], p, 0)) }
	return httpx.List(c, items, next)
}

type consignmentReturnDTO struct {
	ID string `json:"id"`
	UnitID string `json:"unitId"`
	AgreementID string `json:"agreementId"`
	OrderID string `json:"orderId"`
	VehicleID string `json:"vehicleId"`
	SupplierCompanyID string `json:"supplierCompanyId"`
	DealerCompanyID string `json:"dealerCompanyId"`
	DestinationWarehouseID string `json:"destinationWarehouseId"`
	Reason string `json:"reason"`
	Status string `json:"status"`
	RequestedBy string `json:"requestedBy"`
	RequestedAt time.Time `json:"requestedAt"`
	DecidedBy *string `json:"decidedBy,omitempty"`
	DecidedAt *time.Time `json:"decidedAt,omitempty"`
	DecisionReason string `json:"decisionReason,omitempty"`
	MovedAt *time.Time `json:"movedAt,omitempty"`
	MovementFactID *string `json:"movementFactId,omitempty"`
	MovementOperationRef *string `json:"movementOperationRef,omitempty"`
	Revision string `json:"revision"`
	UnitRevision string `json:"unitRevision,omitempty"`
	AllowedActions []string `json:"allowedActions"`
}

func toConsignmentReturn(row *model.ConsignmentReturn, p *auth.Principal, unitRevision int64) consignmentReturnDTO {
	d := consignmentReturnDTO{ID: row.ID, UnitID: row.UnitID, AgreementID: row.AgreementID, OrderID: row.OrderID,
		VehicleID: row.VehicleID, SupplierCompanyID: row.SupplierCompanyID, DealerCompanyID: row.DealerCompanyID,
		DestinationWarehouseID: row.DestinationWarehouseID, Reason: row.Reason, Status: string(row.Status),
		RequestedBy: row.RequestedBy, RequestedAt: row.RequestedAt, DecidedBy: row.DecidedBy, DecidedAt: row.DecidedAt,
		DecisionReason: row.DecisionReason, MovedAt: row.MovedAt, MovementFactID: row.MovementFactID,
		MovementOperationRef: row.MovementOperationRef, Revision: httpx.Revision(row.Version), AllowedActions: []string{}}
	if unitRevision > 0 { d.UnitRevision = httpx.Revision(unitRevision) }
	if p != nil && p.CompanyID == row.DealerCompanyID && p.Has(model.PermTrade) && row.Status == model.ConsignmentReturnPending {
		d.AllowedActions = []string{"confirm", "reject"}
	}
	return d
}

func (h *Handler) requestConsignmentReturn(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil { return err }
	var in service.ReturnRequestInput
	if err := httpx.Bind(c, &in); err != nil { return err }
	p := auth.Get(c)
	row, revision, err := h.returns.Request(c.Request().Context(), p, c.Param("vehicleId"), expected, in)
	if err != nil { return err }
	return httpx.Data(c, http.StatusCreated, toConsignmentReturn(row, p, revision), row.Version)
}

func (h *Handler) getConsignmentReturn(c *echo.Context) error {
	p := auth.Get(c)
	row, err := h.returns.Get(c.Request().Context(), p, c.Param("id"))
	if err != nil { return err }
	return httpx.Data(c, http.StatusOK, toConsignmentReturn(row, p, 0), row.Version)
}

func (h *Handler) decideConsignmentReturn(confirm bool) echo.HandlerFunc {
	return func(c *echo.Context) error {
		expected, err := httpx.IfMatch(c)
		if err != nil { return err }
		var in service.ReturnDecisionInput
		if err := httpx.Bind(c, &in); err != nil { return err }
		p := auth.Get(c)
		row, err := h.returns.Decide(c.Request().Context(), p, c.Param("id"), expected, confirm, in)
		if err != nil { return err }
		return httpx.Data(c, http.StatusOK, toConsignmentReturn(row, p, 0), row.Version)
	}
}
