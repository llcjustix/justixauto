package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
	"justixauto/internal/pkg/jsonx"
)

// supplyWireDTO keeps quantity field names explicit instead of exposing model
// fields directly in the HTTP contract.
type supplyWireDTO struct {
	ID                        string         `json:"id"`
	SupplierCompanyID         string         `json:"supplierCompanyId"`
	ModelID                   string         `json:"modelId"`
	ModelSpecificationVersion string         `json:"modelSpecificationVersion"`
	ExteriorColor             string         `json:"exteriorColor"`
	InteriorColor             string         `json:"interiorColor"`
	TotalQuantity             jsonx.Quantity `json:"totalQuantity"`
	AvailableQuantity         jsonx.Quantity `json:"availableQuantity"`
	HeldQuantity              jsonx.Quantity `json:"heldQuantity"`
	MaterializedQuantity      jsonx.Quantity `json:"materializedQuantity"`
	ExpectedArrival           *time.Time     `json:"expectedArrival"`
	Revision                  string         `json:"revision"`
	UpdatedAt                 time.Time      `json:"updatedAt"`
}

func toSupply(v *service.SupplyView) supplyWireDTO {
	lot, c := v.Lot, v.Capacity
	return supplyWireDTO{
		ID: lot.ID, SupplierCompanyID: lot.SupplierCompanyID, ModelID: lot.ModelID, ModelSpecificationVersion: lot.ModelSpecificationVersion,
		ExteriorColor: lot.ExteriorColor, InteriorColor: lot.InteriorColor, TotalQuantity: jsonx.Quantity(lot.TotalQuantity), AvailableQuantity: jsonx.Quantity(c.Available), HeldQuantity: jsonx.Quantity(c.Held), MaterializedQuantity: jsonx.Quantity(c.Materialized), ExpectedArrival: lot.ExpectedArrival, Revision: httpx.Revision(lot.Version), UpdatedAt: lot.UpdatedAt,
	}
}

func (h *Handler) listSupply(c *echo.Context) error {
	limit, err := httpx.IntQuery(c, "limit")
	if err != nil {
		return err
	}
	offset, err := httpx.IntQuery(c, "offset")
	if err != nil {
		return err
	}
	views, err := h.supply.List(c.Request().Context(), auth.Get(c), limit, offset)
	if err != nil {
		return err
	}
	return httpx.List(c, mapSlice(views, toSupply), nil)
}

func (h *Handler) getSupply(c *echo.Context) error {
	v, err := h.supply.Get(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toSupply(v), v.Lot.Version)
}

func (h *Handler) createSupply(c *echo.Context) error {
	var in service.SupplyInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.supply.Create(c.Request().Context(), auth.Get(c), in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toSupply(v), v.Lot.Version)
}

func (h *Handler) updateSupply(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in service.SupplyInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.supply.Update(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toSupply(v), v.Lot.Version)
}
