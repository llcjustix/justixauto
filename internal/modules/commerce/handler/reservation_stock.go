package handler

import (
	"strconv"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/database"
	"justixauto/internal/pkg/httpx"
)

type reservationStockDTO struct {
	VehicleID                 string `json:"vehicleId"`
	VIN                       string `json:"vin"`
	SupplierCompanyID         string `json:"supplierCompanyId"`
	SupplierCompanyName       string `json:"supplierCompanyName"`
	ModelID                   string `json:"modelId"`
	ModelSpecificationVersion string `json:"modelSpecificationVersion"`
	ExteriorColor             string `json:"exteriorColor"`
	InteriorColor             string `json:"interiorColor"`
}

func (h *Handler) listReservationStock(c *echo.Context) error {
	limit, err := httpx.IntQuery(c, "limit")
	if err != nil {
		return err
	}
	offset, err := httpx.IntQuery(c, "offset")
	if err != nil {
		return err
	}
	items, err := h.reservationStock.List(c.Request().Context(), auth.Get(c), service.ReservationStockFilter{
		SupplierCompanyID: c.QueryParam("supplierCompanyId"), ModelID: c.QueryParam("modelId"), Limit: limit, Offset: offset,
	})
	if err != nil {
		return err
	}
	pageLimit, _ := database.Page(limit, offset)
	var next *string
	if len(items) == pageLimit {
		value := strconv.Itoa(offset + pageLimit)
		next = &value
	}
	dtos := make([]reservationStockDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, reservationStockDTO{
			VehicleID: item.VehicleID, VIN: item.VIN,
			SupplierCompanyID: item.SupplierCompanyID, SupplierCompanyName: item.SupplierCompanyName,
			ModelID: item.ModelID, ModelSpecificationVersion: item.ModelSpecificationVersion,
			ExteriorColor: item.ExteriorColor, InteriorColor: item.InteriorColor,
		})
	}
	return httpx.List(c, dtos, next)
}
