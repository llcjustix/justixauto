package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/database"
)

type ConsignmentReturnRepository struct{ db *gorm.DB }

func (r *ConsignmentReturnRepository) Create(ctx context.Context, row *model.ConsignmentReturn) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(row).Error)
}

func (r *ConsignmentReturnRepository) Get(ctx context.Context, id string) (*model.ConsignmentReturn, error) {
	var row model.ConsignmentReturn
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("id = ?", id).Take(&row).Error
	return &row, database.Translate(err)
}

func (r *ConsignmentReturnRepository) List(ctx context.Context, companyID, orderID string, limit, offset int) ([]model.ConsignmentReturn, error) {
	rows := []model.ConsignmentReturn{}
	q := database.Conn(ctx, r.db).WithContext(ctx).
		Where("supplier_company_id = ? OR dealer_company_id = ?", companyID, companyID)
	if orderID != "" {
		q = q.Where("order_id = ?", orderID)
	}
	err := q.Order("requested_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, database.Translate(err)
}

func (r *ConsignmentReturnRepository) Lock(ctx context.Context, id string) (*model.ConsignmentReturn, error) {
	var row model.ConsignmentReturn
	err := database.Conn(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&row).Error
	return &row, database.Translate(err)
}

func (r *ConsignmentReturnRepository) Decide(ctx context.Context, row *model.ConsignmentReturn, expected int64) error {
	err := database.UpdateVersioned(database.Conn(ctx, r.db).WithContext(ctx), &model.ConsignmentReturn{}, row.ID, expected, map[string]any{
		"status": row.Status, "decided_by": row.DecidedBy, "decided_at": row.DecidedAt,
		"decision_reason": row.DecisionReason, "moved_at": row.MovedAt,
		"movement_fact_id": row.MovementFactID, "movement_operation_ref": row.MovementOperationRef,
		"updated_at": row.UpdatedAt,
	})
	if err == nil {
		row.Version = expected + 1
	}
	return err
}
