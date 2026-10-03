package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/retail/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

// submitInstallmentPayment records one externally identified receipt across the
// caller-selected monthly invoices. Eligibility remains in the service.
func (h *Handler) submitInstallmentPayment(c *echo.Context) error {
	var in service.InstallmentPaymentInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.deals.SubmitInstallmentPayment(c.Request().Context(), auth.Get(c), c.Param("id"), in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toInstallmentPayment(v), v.Payment.Version)
}

// decideInstallmentPayment accepts or rejects every allocation in a receipt.
func (h *Handler) decideInstallmentPayment(accept bool) echo.HandlerFunc {
	return func(c *echo.Context) error {
		expected, err := httpx.IfMatch(c)
		if err != nil {
			return err
		}
		var in decideEvidenceRequest
		if err := httpx.Bind(c, &in); err != nil {
			return err
		}
		v, err := h.deals.DecideInstallmentPayment(c.Request().Context(), auth.Get(c), c.Param("id"), expected, accept, in.Confirmation, in.Reason)
		if err != nil {
			return err
		}
		return httpx.Data(c, http.StatusOK, toInstallmentPayment(v), v.Payment.Version)
	}
}
