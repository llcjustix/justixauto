package repository

import (
	"context"

	"gorm.io/gorm"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/database"
)

type ScoringRepository struct{ db *gorm.DB }

func (r *ScoringRepository) Create(ctx context.Context, v *model.DemoScoring) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(v).Error)
}

func (r *ScoringRepository) Current(ctx context.Context, companyID, dealID string) (*model.DemoScoring, error) {
	var v model.DemoScoring
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("company_id = ? AND deal_id = ?", companyID, dealID).Order("completed_at DESC, id DESC").Take(&v).Error
	if err != nil {
		return nil, database.Translate(err)
	}
	return &v, nil
}
