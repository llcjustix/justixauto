package handler

import (
	"justixauto/internal/modules/retail/model"
	"justixauto/internal/modules/retail/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
	"net/http"

	"github.com/labstack/echo/v5"
)

func (h *Handler) listContractTemplates(c *echo.Context) error {
	v, e := h.contracts.Templates(c.Request().Context(), auth.Get(c))
	if e != nil {
		return e
	}
	return httpx.List(c, mapSlice(v, toContractTemplate), nil)
}
func (h *Handler) createContractTemplate(c *echo.Context) error {
	var in service.TemplateInput
	if e := httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.CreateTemplate(c.Request().Context(), auth.Get(c), in)
	if e != nil {
		return e
	}
	return httpx.Data(c, http.StatusCreated, toContractTemplate(v), v.Version)
}
func (h *Handler) updateContractTemplate(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.TemplateInput
	if e = httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.UpdateTemplate(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if e != nil {
		return e
	}
	return httpx.Data(c, http.StatusOK, toContractTemplate(v), v.Version)
}
func (h *Handler) listContracts(c *echo.Context) error {
	v, e := h.contracts.Contracts(c.Request().Context(), auth.Get(c), c.Param("id"))
	if e != nil {
		return e
	}
	deal, e := h.deals.Get(c.Request().Context(), auth.Get(c), c.Param("id"))
	if e != nil {
		return e
	}
	items := make([]contractDTO, 0, len(v))
	for i := range v {
		item, err := toContract(&v[i], deal.Deal.Version)
		if err != nil {
			return err
		}
		items = append(items, item)
	}
	return httpx.List(c, items, nil)
}
func (h *Handler) getContract(c *echo.Context) error {
	v, e := h.contracts.Contract(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("contractID"))
	if e != nil {
		return e
	}
	deal, e := h.deals.Get(c.Request().Context(), auth.Get(c), c.Param("id"))
	if e != nil {
		return e
	}
	out, e := toContract(v, deal.Deal.Version)
	if e != nil {
		return e
	}
	return httpx.Data(c, http.StatusOK, out, v.Revision)
}

func contractResponse(c *echo.Context, status int, v *model.ContractVersion) error {
	out, err := toContract(v, v.DealRevision)
	if err != nil {
		return err
	}
	return httpx.Data(c, status, out, v.Revision)
}
func (h *Handler) draftContract(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.DraftContractInput
	if e = httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.Draft(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusCreated, v)
}
func (h *Handler) editContractDraft(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.EditDraftContractInput
	if e = httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.EditDraft(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("contractID"), expected, in)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusOK, v)
}
func (h *Handler) signContract(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.SignContractInput
	if e = httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.Sign(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("contractID"), expected, in, nil)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusCreated, v)
}
func (h *Handler) amendContract(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in struct {
		service.SignContractInput
		PriorSignedVersionID string `json:"priorSignedVersionId"`
	}
	if e = httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.Sign(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("contractID"), expected, in.SignContractInput, &in.PriorSignedVersionID)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusCreated, v)
}
