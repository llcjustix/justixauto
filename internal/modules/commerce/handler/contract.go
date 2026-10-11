package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

type contractTemplateDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Revision string `json:"revision"`
}

func toContractTemplate(v *model.ContractTemplate) contractTemplateDTO {
	return contractTemplateDTO{ID: v.ID, Name: v.Name, Content: v.Body, Revision: httpx.Revision(v.Version)}
}

type contractDTO struct {
	ID                   string              `json:"id"`
	OrderID              string              `json:"orderId"`
	RecordingCompanyID   string              `json:"recordingCompanyId"`
	State                model.ContractState `json:"state"`
	Kind                 model.ContractKind  `json:"kind"`
	TemplateID           *string             `json:"templateId"`
	TemplateRevision     *string             `json:"templateRevision"`
	PriorSignedVersionID *string             `json:"priorSignedVersionId"`
	Content              string              `json:"content"`
	FileIDs              []string            `json:"fileIds"`
	PartySnapshot        json.RawMessage     `json:"partySnapshot"`
	TermsSnapshot        json.RawMessage     `json:"termsSnapshot"`
	VehicleSnapshot      json.RawMessage     `json:"vehicleSnapshot"`
	AcceptedTermsDigest  string              `json:"acceptedTermsDigest"`
	SourceOrderRevision  string              `json:"sourceOrderRevision"`
	SignedOn             *string             `json:"signedOn"`
	Reference            string              `json:"reference"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
	ContractRevision     string              `json:"contractRevision"`
	OrderRevision        string              `json:"orderRevision"`
}

func rawObject(raw []byte, field string) (json.RawMessage, error) {
	if len(raw) == 0 || !json.Valid(raw) || string(raw) == "null" {
		return nil, apperr.New(apperr.ErrConflict, "contract_snapshot_invalid", "stored contract "+field+" is invalid")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, apperr.New(apperr.ErrConflict, "contract_snapshot_invalid", "stored contract "+field+" is invalid")
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, apperr.New(apperr.ErrConflict, "contract_snapshot_invalid", "stored contract "+field+" is invalid")
	}
	return append(json.RawMessage(nil), raw...), nil
}

func contractFileIDs(v *model.ContractVersion) ([]string, error) {
	all := []string{}
	for _, raw := range [][]byte{v.ExternalFileIDs, v.SignedFileIDs} {
		if len(raw) == 0 || !json.Valid(raw) || string(raw) == "null" {
			return nil, apperr.New(apperr.ErrConflict, "contract_files_invalid", "stored contract files are invalid")
		}
		var ids []string
		if err := json.Unmarshal(raw, &ids); err != nil || ids == nil {
			return nil, apperr.New(apperr.ErrConflict, "contract_files_invalid", "stored contract files are invalid")
		}
		for _, id := range ids {
			if id != "" && !contains(all, id) {
				all = append(all, id)
			}
		}
	}
	return all, nil
}

func contains(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func toContract(v *model.ContractVersion, orderRevision int64) (contractDTO, error) {
	party, err := rawObject(v.PartySnapshot, "party snapshot")
	if err != nil {
		return contractDTO{}, err
	}
	terms, err := rawObject(v.TermsSnapshot, "terms snapshot")
	if err != nil {
		return contractDTO{}, err
	}
	var snapshot struct {
		Vehicles json.RawMessage `json:"vehicles"`
	}
	if err := json.Unmarshal(v.PartySnapshot, &snapshot); err != nil || len(snapshot.Vehicles) == 0 || !json.Valid(snapshot.Vehicles) {
		return contractDTO{}, apperr.New(apperr.ErrConflict, "contract_snapshot_invalid", "stored contract vehicle snapshot is invalid")
	}
	var vehicles any
	if json.Unmarshal(snapshot.Vehicles, &vehicles) != nil {
		return contractDTO{}, apperr.New(apperr.ErrConflict, "contract_snapshot_invalid", "stored contract vehicle snapshot is invalid")
	}
	if _, ok := vehicles.([]any); !ok {
		return contractDTO{}, apperr.New(apperr.ErrConflict, "contract_snapshot_invalid", "stored contract vehicle snapshot is invalid")
	}
	files, err := contractFileIDs(v)
	if err != nil {
		return contractDTO{}, err
	}
	var templateRevision *string
	if v.TemplateVersion != nil {
		value := httpx.Revision(*v.TemplateVersion)
		templateRevision = &value
	}
	var signedOn *string
	if v.SignedOn != nil {
		value := v.SignedOn.Format(time.DateOnly)
		signedOn = &value
	}
	return contractDTO{ID: v.ID, OrderID: v.OrderID, RecordingCompanyID: v.CompanyID, State: v.State, Kind: v.Kind,
		TemplateID: v.TemplateID, TemplateRevision: templateRevision, PriorSignedVersionID: v.PriorSignedVersionID,
		Content: v.Content, FileIDs: files, PartySnapshot: party, TermsSnapshot: terms, VehicleSnapshot: append(json.RawMessage(nil), snapshot.Vehicles...),
		AcceptedTermsDigest: v.TermsDigest, SourceOrderRevision: httpx.Revision(v.SourceOrderRevision), SignedOn: signedOn, Reference: v.Reference,
		CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, ContractRevision: httpx.Revision(v.Version), OrderRevision: httpx.Revision(orderRevision)}, nil
}

type templateRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
type draftRequest struct {
	TemplateID      *string   `json:"templateId"`
	Content         *string   `json:"content"`
	BindingIDs      *[]string `json:"bindingIds"`
	RefreshSnapshot bool      `json:"refreshSnapshot"`
}
type draftEditRequest struct {
	DraftRevision   int64     `json:"draftRevision"`
	TemplateID      *string   `json:"templateId"`
	Content         *string   `json:"content"`
	BindingIDs      *[]string `json:"bindingIds"`
	RefreshSnapshot bool      `json:"refreshSnapshot"`
}
type signRequest struct {
	DraftRevision int64    `json:"draftRevision"`
	BindingIDs    []string `json:"bindingIds"`
	SignedOn      string   `json:"signedOn"`
	Reference     string   `json:"reference"`
}
type amendmentRequest struct {
	DraftRevision        int64    `json:"draftRevision"`
	PriorSignedVersionID string   `json:"priorSignedVersionId"`
	BindingIDs           []string `json:"bindingIds"`
	SignedOn             string   `json:"signedOn"`
	Reference            string   `json:"reference"`
}

func contractDraftInput(in draftRequest) service.DraftInput {
	return service.DraftInput{TemplateID: in.TemplateID, Content: in.Content, ExternalFileIDs: in.BindingIDs, RefreshSnapshot: in.RefreshSnapshot}
}
func contractEditInput(in draftEditRequest) service.DraftInput {
	return service.DraftInput{TemplateID: in.TemplateID, Content: in.Content, ExternalFileIDs: in.BindingIDs, RefreshSnapshot: in.RefreshSnapshot}
}
func contractSignInput(signedOn, reference string, ids []string) (service.SignInput, error) {
	parsed, err := time.Parse(time.DateOnly, signedOn)
	if err != nil {
		return service.SignInput{}, apperr.FieldError("signedOn", "must be an ISO date")
	}
	return service.SignInput{SignedOn: parsed, Reference: reference, SignedFileIDs: ids}, nil
}

func (h *Handler) contractRoutes(c *echo.Group) {
	c.GET("/contract-templates", h.listContractTemplates, auth.Require(model.PermRead))
	c.GET("/contract-templates/:templateID", h.getContractTemplate, auth.Require(model.PermRead))
	c.POST("/contract-templates", h.createContractTemplate, auth.Require(model.PermTrade))
	c.PATCH("/contract-templates/:templateID", h.editContractTemplate, auth.Require(model.PermTrade))
	c.GET("/orders/:id/contracts", h.listContracts, auth.Require(model.PermRead))
	c.GET("/orders/:id/contracts/:contractID", h.getContract, auth.Require(model.PermRead))
	c.POST("/orders/:id/contracts/preview", h.previewContract, auth.Require(model.PermRead))
	c.POST("/orders/:id/contracts", h.createContractDraft, auth.Require(model.PermTrade))
	c.PATCH("/orders/:id/contracts/:contractID", h.editContractDraft, auth.Require(model.PermTrade))
	c.POST("/orders/:id/contracts/:contractID/sign", h.signContract, auth.Require(model.PermTrade))
	c.POST("/orders/:id/contracts/:contractID/amendments", h.amendContract, auth.Require(model.PermTrade))
}

func (h *Handler) listContractTemplates(c *echo.Context) error {
	limit, offset, err := paging(c)
	if err != nil {
		return err
	}
	rows, err := h.contracts.Templates(c.Request().Context(), auth.Get(c), limit, offset)
	if err != nil {
		return err
	}
	return httpx.List(c, mapSlice(rows, toContractTemplate), nil)
}
func (h *Handler) getContractTemplate(c *echo.Context) error {
	v, err := h.contracts.TemplateDetail(c.Request().Context(), auth.Get(c), c.Param("templateID"))
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toContractTemplate(v), v.Version)
}
func (h *Handler) createContractTemplate(c *echo.Context) error {
	var in templateRequest
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.contracts.TemplateCreate(c.Request().Context(), auth.Get(c), service.TemplateInput{Name: in.Name, Body: in.Content})
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toContractTemplate(v), v.Version)
}
func (h *Handler) editContractTemplate(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in templateRequest
	if err = httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.contracts.TemplateEdit(c.Request().Context(), auth.Get(c), c.Param("templateID"), expected, service.TemplateInput{Name: in.Name, Body: in.Content})
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toContractTemplate(v), v.Version)
}

func (h *Handler) orderRevision(c *echo.Context, orderID string) (int64, error) {
	o, err := h.contracts.OrderPartiesFor(c.Request().Context(), auth.Get(c).CompanyID, orderID)
	if err != nil {
		return 0, err
	}
	return o.Version, nil
}
func (h *Handler) contractResponse(c *echo.Context, status int, v *model.ContractVersion) error {
	orderRevision, err := h.orderRevision(c, v.OrderID)
	if err != nil {
		return err
	}
	out, err := toContract(v, orderRevision)
	if err != nil {
		return err
	}
	return httpx.Data(c, status, out, v.Version)
}
func (h *Handler) listContracts(c *echo.Context) error {
	limit, offset, err := paging(c)
	if err != nil {
		return err
	}
	rows, err := h.contracts.List(c.Request().Context(), auth.Get(c), c.Param("id"), limit, offset)
	if err != nil {
		return err
	}
	revision, err := h.orderRevision(c, c.Param("id"))
	if err != nil {
		return err
	}
	out := make([]contractDTO, 0, len(rows))
	for i := range rows {
		item, err := toContract(&rows[i], revision)
		if err != nil {
			return err
		}
		out = append(out, item)
	}
	return httpx.List(c, out, nil)
}
func (h *Handler) getContract(c *echo.Context) error {
	v, err := h.contracts.Detail(c.Request().Context(), auth.Get(c), c.Param("contractID"))
	if err != nil {
		return err
	}
	if v.OrderID != c.Param("id") {
		return apperr.ErrNotFound
	}
	return h.contractResponse(c, http.StatusOK, v)
}
func (h *Handler) previewContract(c *echo.Context) error {
	var in draftRequest
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.contracts.Preview(c.Request().Context(), auth.Get(c), c.Param("id"), contractDraftInput(in))
	if err != nil {
		return err
	}
	party, err := rawObject(v.PartySnapshot, "party snapshot")
	if err != nil {
		return err
	}
	terms, err := rawObject(v.TermsSnapshot, "terms snapshot")
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, map[string]any{"content": v.Content, "templateId": v.TemplateID, "templateRevision": revisionPointer(v.TemplateVersion), "partySnapshot": party, "termsSnapshot": terms, "acceptedTermsDigest": v.TermsDigest, "orderRevision": httpx.Revision(v.SourceOrderRevision)}, v.SourceOrderRevision)
}
func revisionPointer(v *int64) *string {
	if v == nil {
		return nil
	}
	s := httpx.Revision(*v)
	return &s
}
func (h *Handler) createContractDraft(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in draftRequest
	if err = httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.contracts.DraftCreate(c.Request().Context(), auth.Get(c), c.Param("id"), expected, contractDraftInput(in))
	if err != nil {
		return err
	}
	return h.contractResponse(c, http.StatusCreated, v)
}
func (h *Handler) editContractDraft(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in draftEditRequest
	if err = httpx.Bind(c, &in); err != nil {
		return err
	}
	v, err := h.contracts.DraftEdit(c.Request().Context(), auth.Get(c), c.Param("contractID"), expected, in.DraftRevision, contractEditInput(in))
	if err != nil {
		return err
	}
	if v.OrderID != c.Param("id") {
		return apperr.ErrNotFound
	}
	return h.contractResponse(c, http.StatusOK, v)
}
func (h *Handler) signContract(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in signRequest
	if err = httpx.Bind(c, &in); err != nil {
		return err
	}
	sign, err := contractSignInput(in.SignedOn, in.Reference, in.BindingIDs)
	if err != nil {
		return err
	}
	v, err := h.contracts.Sign(c.Request().Context(), auth.Get(c), c.Param("contractID"), expected, in.DraftRevision, sign)
	if err != nil {
		return err
	}
	if v.OrderID != c.Param("id") {
		return apperr.ErrNotFound
	}
	return h.contractResponse(c, http.StatusOK, v)
}
func (h *Handler) amendContract(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in amendmentRequest
	if err = httpx.Bind(c, &in); err != nil {
		return err
	}
	sign, err := contractSignInput(in.SignedOn, in.Reference, in.BindingIDs)
	if err != nil {
		return err
	}
	v, err := h.contracts.Amend(c.Request().Context(), auth.Get(c), c.Param("contractID"), in.PriorSignedVersionID, expected, in.DraftRevision, sign)
	if err != nil {
		return err
	}
	if v.OrderID != c.Param("id") {
		return apperr.ErrNotFound
	}
	return h.contractResponse(c, http.StatusCreated, v)
}
