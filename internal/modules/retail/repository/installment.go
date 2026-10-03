package repository

import (
	"context"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/database"
)

func (r *DealRepository) CreateInstallmentPlan(ctx context.Context, p *model.InstallmentPlan) error {
	return database.Translate(r.db.WithContext(ctx).Create(p).Error)
}

func (r *DealRepository) InstallmentPlan(ctx context.Context, dealID string) (*model.InstallmentPlan, error) {
	var p model.InstallmentPlan
	if err := r.db.WithContext(ctx).Where("deal_id = ?", dealID).Take(&p).Error; err != nil {
		return nil, database.Translate(err)
	}
	return &p, nil
}
