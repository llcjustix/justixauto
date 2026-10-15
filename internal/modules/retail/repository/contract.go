package repository

import (
	"context"

	"gorm.io/gorm"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/database"
)

type ContractRepository struct{ db *gorm.DB }

func (r *ContractRepository) CreateTemplate(ctx context.Context, v *model.ContractTemplate) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(v).Error)
}

func (r *ContractRepository) Template(ctx context.Context, company, id string) (*model.ContractTemplate, error) {
	var v model.ContractTemplate
	if err := database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Where("id = ? AND company_id = ?", id, company).Take(&v).Error); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *ContractRepository) Templates(ctx context.Context, company string) ([]model.ContractTemplate, error) {
	var v []model.ContractTemplate
	err := database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Where("company_id = ?", company).Order("updated_at DESC, id").Find(&v).Error)
	return v, err
}

func (r *ContractRepository) UpdateTemplate(ctx context.Context, v *model.ContractTemplate, expected int64) error {
	return database.UpdateVersioned(database.Conn(ctx, r.db).WithContext(ctx), &model.ContractTemplate{}, v.ID, expected, map[string]any{"name": v.Name, "body": v.Body, "updated_at": v.UpdatedAt})
}

func (r *ContractRepository) CreateContract(ctx context.Context, v *model.ContractVersion) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(v).Error)
}

func (r *ContractRepository) UpdateDraft(ctx context.Context, v *model.ContractVersion, expected int64) error {
	db := database.Conn(ctx, r.db).WithContext(ctx)
	res := db.Model(&model.ContractVersion{}).Where("id = ? AND company_id = ? AND deal_id = ? AND state = ? AND revision = ?", v.ID, v.CompanyID, v.DealID, "draft", expected).Updates(map[string]any{
		"template_id": v.TemplateID, "template_version": v.TemplateVersion, "content": v.Content,
		"party_snapshot": v.PartySnapshot, "terms_snapshot": v.TermsSnapshot, "file_ids": v.FileIDs,
		"revision": expected + 1,
	})
	if res.Error != nil {
		return database.Translate(res.Error)
	}
	if res.RowsAffected != 1 {
		return apperr.ErrStale
	}
	v.Revision = expected + 1
	return nil
}

func (r *ContractRepository) SignDraft(ctx context.Context, v *model.ContractVersion, expected int64) error {
	db := database.Conn(ctx, r.db).WithContext(ctx)
	res := db.Model(&model.ContractVersion{}).Where("id = ? AND company_id = ? AND deal_id = ? AND state = ? AND revision = ?", v.ID, v.CompanyID, v.DealID, "draft", expected).Updates(map[string]any{"kind": v.Kind, "state": v.State, "prior_signed_version_id": v.PriorSignedVersionID, "reference": v.Reference, "signed_on": v.SignedOn, "file_ids": v.FileIDs, "revision": expected + 1})
	if res.Error != nil {
		return database.Translate(res.Error)
	}
	if res.RowsAffected != 1 {
		return apperr.ErrStale
	}
	v.Revision = expected + 1
	return nil
}

func (r *ContractRepository) Contract(ctx context.Context, company, deal, id string) (*model.ContractVersion, error) {
	var v model.ContractVersion
	err := database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Where("id = ? AND company_id = ? AND deal_id = ?", id, company, deal).Take(&v).Error)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *ContractRepository) Contracts(ctx context.Context, company, deal string) ([]model.ContractVersion, error) {
	var v []model.ContractVersion
	err := database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Where("company_id = ? AND deal_id = ?", company, deal).Order("created_at, id").Find(&v).Error)
	return v, err
}
