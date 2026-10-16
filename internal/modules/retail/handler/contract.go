package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/modules/retail/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

// listContractTemplates list retail contract templates.
//
//	@Summary	List retail contract templates
//	@Tags		retail/contracts
//	@Produce	json
//	@Success	200		{object}	httpx.ListEnvelope[handler.contractTemplateDTO]
//	@Failure	401,403	{object}	httpx.ErrorBody
//	@Router		/retail/contract-templates [get]
func (h *Handler) listContractTemplates(c *echo.Context) error {
	v, e := h.contracts.Templates(c.Request().Context(), auth.Get(c))
	if e != nil {
		return e
	}
	return httpx.List(c, mapSlice(v, toContractTemplate), nil)
}

// createContractTemplate create retail contract template.
//
//	@Summary	Create retail contract template
//	@Tags		retail/contracts
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		body			body		service.TemplateInput	true	"request"
//	@Success	201				{object}	httpx.DataEnvelope[handler.contractTemplateDTO]
//	@Failure	401,403,409,422	{object}	httpx.ErrorBody
//	@Router		/retail/contract-templates [post]
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

// updateContractTemplate update retail contract template.
//
//	@Summary	Update retail contract template
//	@Tags		retail/contracts
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string					true	"contract template ID"
//	@Param		If-Match					header		string					true	"template revision"
//	@Param		body						body		service.TemplateInput	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.contractTemplateDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/retail/contract-templates/{id} [patch]
func (h *Handler) updateContractTemplate(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.TemplateInput
	if e := httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.UpdateTemplate(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if e != nil {
		return e
	}
	return httpx.Data(c, http.StatusOK, toContractTemplate(v), v.Version)
}

// listContracts list deal contracts.
//
//	@Summary	List deal contracts
//	@Tags		retail/contracts
//	@Produce	json
//	@Param		id			path		string	true	"deal ID"
//	@Success	200			{object}	httpx.ListEnvelope[handler.contractDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/retail/deals/{id}/contracts [get]
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

// getContract get deal contract.
//
//	@Summary	Get deal contract
//	@Tags		retail/contracts
//	@Produce	json
//	@Param		id			path		string	true	"deal ID"
//	@Param		contractID	path		string	true	"contract ID"
//	@Success	200			{object}	httpx.DataEnvelope[handler.contractDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/retail/deals/{id}/contracts/{contractID} [get]
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

// draftContract create deal contract draft.
//
//	@Summary	Create deal contract draft
//	@Tags		retail/contracts
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string						true	"deal ID"
//	@Param		If-Match					header		string						true	"deal revision"
//	@Param		body						body		service.DraftContractInput	true	"request"
//	@Success	201							{object}	httpx.DataEnvelope[handler.contractDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/retail/deals/{id}/contracts/drafts [post]
func (h *Handler) draftContract(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.DraftContractInput
	if e := httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.Draft(c.Request().Context(), auth.Get(c), c.Param("id"), expected, in)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusCreated, v)
}

// editContractDraft edit deal contract draft.
//
//	@Summary	Edit deal contract draft
//	@Tags		retail/contracts
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string							true	"deal ID"
//	@Param		contractID					path		string							true	"contract ID"
//	@Param		If-Match					header		string							true	"deal revision"
//	@Param		body						body		service.EditDraftContractInput	true	"request"
//	@Success	200							{object}	httpx.DataEnvelope[handler.contractDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/retail/deals/{id}/contracts/{contractID} [patch]
func (h *Handler) editContractDraft(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.EditDraftContractInput
	if e := httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.EditDraft(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("contractID"), expected, in)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusOK, v)
}

// signContract sign deal contract.
//
//	@Summary	Sign deal contract
//	@Tags		retail/contracts
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string						true	"deal ID"
//	@Param		contractID					path		string						true	"contract ID"
//	@Param		If-Match					header		string						true	"deal revision"
//	@Param		body						body		service.SignContractInput	true	"request"
//	@Success	201							{object}	httpx.DataEnvelope[handler.contractDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/retail/deals/{id}/contracts/{contractID}/sign [post]
func (h *Handler) signContract(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in service.SignContractInput
	if e := httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.Sign(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("contractID"), expected, in, nil)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusCreated, v)
}

// amendContract record deal contract amendment.
//
//	@Summary	Record deal contract amendment
//	@Tags		retail/contracts
//	@Accept		json
//	@Produce	json
//	@Security	CSRF
//	@Param		id							path		string													true	"deal ID"
//	@Param		contractID					path		string													true	"contract ID"
//	@Param		If-Match					header		string													true	"deal revision"
//	@Param		body						body		service.SignContractInput{priorSignedVersionId=string}	true	"request"
//	@Success	201							{object}	httpx.DataEnvelope[handler.contractDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/retail/deals/{id}/contracts/{contractID}/amendments [post]
func (h *Handler) amendContract(c *echo.Context) error {
	expected, e := httpx.IfMatch(c)
	if e != nil {
		return e
	}
	var in struct {
		service.SignContractInput
		PriorSignedVersionID string `json:"priorSignedVersionId"`
	}
	if e := httpx.Bind(c, &in); e != nil {
		return e
	}
	v, e := h.contracts.Sign(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("contractID"), expected, in.SignContractInput, &in.PriorSignedVersionID)
	if e != nil {
		return e
	}
	return contractResponse(c, http.StatusCreated, v)
}
