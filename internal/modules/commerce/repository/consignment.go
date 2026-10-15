package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/database"
)

type ConsignmentRepository struct{ db *gorm.DB }

func (r *ConsignmentRepository) CreateAgreement(ctx context.Context, a *model.ConsignmentAgreement) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(a).Error)
}

func (r *ConsignmentRepository) AgreementByOrder(ctx context.Context, orderID string) (*model.ConsignmentAgreement, error) {
	var a model.ConsignmentAgreement
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("order_id = ?", orderID).Take(&a).Error
	return &a, database.Translate(err)
}

func (r *ConsignmentRepository) Agreement(ctx context.Context, id string) (*model.ConsignmentAgreement, error) {
	var a model.ConsignmentAgreement
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("id = ?", id).Take(&a).Error
	return &a, database.Translate(err)
}

func (r *ConsignmentRepository) CreateUnit(ctx context.Context, u *model.ConsignmentUnit) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(u).Error)
}

func (r *ConsignmentRepository) Unit(ctx context.Context, agreementID, vehicleID string) (*model.ConsignmentUnit, error) {
	var u model.ConsignmentUnit
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("agreement_id = ? AND vehicle_id = ?", agreementID, vehicleID).Take(&u).Error
	return &u, database.Translate(err)
}

func (r *ConsignmentRepository) LockUnitByVehicle(ctx context.Context, vehicleID string) (*model.ConsignmentUnit, error) {
	var u model.ConsignmentUnit
	err := database.Conn(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("vehicle_id = ? AND status IN ?", vehicleID, []model.ConsignmentUnitStatus{model.ConsignmentPrepared, model.ConsignmentHeld}).Take(&u).Error
	return &u, database.Translate(err)
}

func (r *ConsignmentRepository) UpdateUnit(ctx context.Context, u *model.ConsignmentUnit, expected int64) error {
	err := database.UpdateVersioned(database.Conn(ctx, r.db).WithContext(ctx), &model.ConsignmentUnit{}, u.ID, expected, map[string]any{
		"status": u.Status, "handover_fact_id": u.HandoverFactID, "handover_operation_ref": u.HandoverOperationRef,
		"handed_over_at": u.HandedOverAt, "handed_over_by": u.HandedOverBy,
		"current_retail_deal_id": u.CurrentRetailDealID, "pending_return_id": u.PendingReturnID,
		"retail_delivery_ref": u.RetailDeliveryRef, "sold_at": u.SoldAt, "sold_by": u.SoldBy,
		"returned_at": u.ReturnedAt, "return_fact_id": u.ReturnFactID, "updated_at": u.UpdatedAt,
	})
	if err == nil {
		u.Version = expected + 1
	}
	return err
}

func (r *ConsignmentRepository) Units(ctx context.Context, agreementID string) ([]model.ConsignmentUnit, error) {
	rows := []model.ConsignmentUnit{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("agreement_id = ?", agreementID).Order("id").Find(&rows).Error
	return rows, database.Translate(err)
}

// RetailCandidates returns only units that can be discovered by their current
// dealer. The retail service remains the mutation authority for a later sale.
func (r *ConsignmentRepository) RetailCandidates(ctx context.Context, dealerID string, limit, offset int) ([]model.ConsignmentUnit, error) {
	rows := []model.ConsignmentUnit{}
	err := database.Conn(ctx, r.db).WithContext(ctx).
		Where("dealer_company_id = ? AND status = ? AND handed_over_at IS NOT NULL AND handover_fact_id IS NOT NULL AND handover_operation_ref IS NOT NULL AND handed_over_by IS NOT NULL AND current_retail_deal_id IS NULL AND pending_return_id IS NULL", dealerID, model.ConsignmentHeld).
		Order("id").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, database.Translate(err)
}
