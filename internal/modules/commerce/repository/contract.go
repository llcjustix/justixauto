package repository

import (
	"context"
	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/database"

	"gorm.io/gorm"
)

type ContractRepository struct{ db *gorm.DB }

func (r *ContractRepository) CreateTemplate(ctx context.Context, t *model.ContractTemplate) error {
	return database.Translate(r.db.WithContext(ctx).Create(t).Error)
}
func (r *ContractRepository) Template(ctx context.Context, companyID, id string) (*model.ContractTemplate, error) {
	var t model.ContractTemplate
	if e := r.db.WithContext(ctx).Where("id = ? AND company_id = ?", id, companyID).Take(&t).Error; e != nil {
		return nil, database.Translate(e)
	}
	return &t, nil
}
func (r *ContractRepository) Templates(ctx context.Context, companyID string, limit, offset int) ([]model.ContractTemplate, error) {
	limit, offset = database.Page(limit, offset)
	out := []model.ContractTemplate{}
	e := r.db.WithContext(ctx).Where("company_id = ?", companyID).Order("updated_at DESC,id").Limit(limit).Offset(offset).Find(&out).Error
	return out, database.Translate(e)
}
func (r *ContractRepository) UpdateTemplate(ctx context.Context, t *model.ContractTemplate, expected int64) error {
	e := database.UpdateVersioned(r.db.WithContext(ctx), &model.ContractTemplate{}, t.ID, expected, map[string]any{"name": t.Name, "body": t.Body, "updated_at": t.UpdatedAt})
	if e == nil {
		t.Version = expected + 1
	}
	return e
}
func (r *ContractRepository) Create(ctx context.Context, c *model.ContractVersion) error {
	return database.Translate(r.db.WithContext(ctx).Create(c).Error)
}
func (r *ContractRepository) Contract(ctx context.Context, viewer, id string) (*model.ContractVersion, error) {
	var c model.ContractVersion
	e := r.db.WithContext(ctx).Where("id = ? AND (company_id = ? OR (state = 'signed' AND EXISTS (SELECT 1 FROM commerce_orders o WHERE o.id = commerce_contract_versions.order_id AND (o.buyer_company_id = ? OR o.supplier_company_id = ?))))", id, viewer, viewer, viewer).Take(&c).Error
	if e != nil {
		return nil, database.Translate(e)
	}
	return &c, nil
}
func (r *ContractRepository) ForOrder(ctx context.Context, viewer, orderID string, limit, offset int) ([]model.ContractVersion, error) {
	limit, offset = database.Page(limit, offset)
	out := []model.ContractVersion{}
	e := r.db.WithContext(ctx).Where("order_id = ? AND (company_id = ? OR state = 'signed')", orderID, viewer).Order("created_at,id").Limit(limit).Offset(offset).Find(&out).Error
	return out, database.Translate(e)
}
func (r *ContractRepository) Update(ctx context.Context, c *model.ContractVersion, expected int64) error {
	e := database.UpdateVersioned(r.db.WithContext(ctx), &model.ContractVersion{}, c.ID, expected, map[string]any{"template_id": c.TemplateID, "template_version": c.TemplateVersion, "prior_signed_version_id": c.PriorSignedVersionID, "state": c.State, "reference": c.Reference, "signed_on": c.SignedOn, "external_file_ids": c.ExternalFileIDs, "signed_file_ids": c.SignedFileIDs, "content": c.Content, "party_snapshot": c.PartySnapshot, "terms_snapshot": c.TermsSnapshot, "terms_digest": c.TermsDigest, "source_order_revision": c.SourceOrderRevision, "updated_at": c.UpdatedAt})
	if e == nil {
		c.Version = expected + 1
	}
	return e
}
func (r *ContractRepository) HasBaseSigned(ctx context.Context, orderID string) (bool, error) {
	var n int64
	e := r.db.WithContext(ctx).Model(&model.ContractVersion{}).Where("order_id = ? AND kind = 'contract' AND state = 'signed'", orderID).Count(&n).Error
	return n > 0, database.Translate(e)
}
