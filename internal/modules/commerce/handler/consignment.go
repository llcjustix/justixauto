package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

// listConsignmentRetailCandidates list consignment vehicles eligible for retail.
//
//	@Summary	List consignment vehicles eligible for retail
//	@Tags		commerce/consignments
//	@Produce	json
//	@Param		limit		query		int	false	"page size"
//	@Param		offset		query		int	false	"offset"
//	@Success	200			{object}	httpx.ListEnvelope[handler.consignmentRetailCandidateDTO]
//	@Failure	401,403,422	{object}	httpx.ErrorBody
//	@Router		/commerce/consignments/retail-candidates [get]
func (h *Handler) listConsignmentRetailCandidates(c *echo.Context) error {
	limit, offset, err := paging(c)
	if err != nil {
		return err
	}
	items, err := h.consignments.RetailCandidates(c.Request().Context(), auth.Get(c), limit, offset)
	if err != nil {
		return err
	}
	pageLimit := limit
	if pageLimit == 0 {
		pageLimit = 50
	}
	var next *string
	if len(items) == pageLimit {
		value := strconv.Itoa(offset + pageLimit)
		next = &value
	}
	dtos := make([]consignmentRetailCandidateDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, toConsignmentRetailCandidate(item))
	}
	return httpx.List(c, dtos, next)
}

// getConsignment get order consignment.
//
//	@Summary	Get order consignment
//	@Tags		commerce/consignments
//	@Produce	json
//	@Param		id			path		string	true	"order ID"
//	@Success	200			{object}	httpx.DataEnvelope[handler.consignmentDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/commerce/consignments/orders/{id} [get]
func (h *Handler) getConsignment(c *echo.Context) error {
	v, err := h.consignments.Get(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toConsignment(v), v.Agreement.Version)
}
