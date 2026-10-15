package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/database"
)

type SupplyRepository struct{ db *gorm.DB }

func (r *SupplyRepository) Create(ctx context.Context, lot *model.SupplyLot) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(lot).Error)
}

func (r *SupplyRepository) Lot(ctx context.Context, id string) (*model.SupplyLot, error) {
	var lot model.SupplyLot
	if err := database.Conn(ctx, r.db).WithContext(ctx).Where("id = ?", id).Take(&lot).Error; err != nil {
		return nil, database.Translate(err)
	}
	return &lot, nil
}

func (r *SupplyRepository) LockLot(ctx context.Context, id string) (*model.SupplyLot, error) {
	var lot model.SupplyLot
	if err := database.Conn(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&lot).Error; err != nil {
		return nil, database.Translate(err)
	}
	return &lot, nil
}

func (r *SupplyRepository) ListSupplier(ctx context.Context, supplierID string, limit, offset int) ([]model.SupplyLot, error) {
	limit, offset = database.Page(limit, offset)
	rows := []model.SupplyLot{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("supplier_company_id = ?", supplierID).
		Order("updated_at DESC, id").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, database.Translate(err)
}

func (r *SupplyRepository) ListVisible(ctx context.Context, dealerID string, limit, offset int) ([]model.SupplyLot, error) {
	limit, offset = database.Page(limit, offset)
	rows := []model.SupplyLot{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Where(`EXISTS (SELECT 1 FROM commerce_partnerships p WHERE p.status = 'active' AND
		((p.requester_company_id = commerce_supply_lots.supplier_company_id AND p.recipient_company_id = ?) OR
		 (p.recipient_company_id = commerce_supply_lots.supplier_company_id AND p.requester_company_id = ?)))`, dealerID, dealerID).
		Order("updated_at DESC, id").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, database.Translate(err)
}

func (r *SupplyRepository) Update(ctx context.Context, lot *model.SupplyLot, expected int64) error {
	err := database.UpdateVersioned(database.Conn(ctx, r.db).WithContext(ctx), &model.SupplyLot{}, lot.ID, expected, map[string]any{
		"model_id": lot.ModelID, "model_specification_version": lot.ModelSpecificationVersion,
		"exterior_color": lot.ExteriorColor, "interior_color": lot.InteriorColor,
		"total_quantity": lot.TotalQuantity, "expected_arrival": lot.ExpectedArrival, "updated_at": lot.UpdatedAt,
	})
	if err == nil {
		lot.Version = expected + 1
	}
	return err
}

func (r *SupplyRepository) Capacity(ctx context.Context, lotID string) (model.SupplyCapacity, error) {
	var out model.SupplyCapacity
	err := database.Conn(ctx, r.db).WithContext(ctx).Raw(`SELECT l.total_quantity - count(c.id) FILTER (WHERE c.state IN ('held', 'materialized')) AS available,
		count(c.id) FILTER (WHERE c.state = 'held') AS held,
		count(c.id) FILTER (WHERE c.state = 'materialized') AS materialized
		FROM commerce_supply_lots l LEFT JOIN commerce_supply_claims c ON c.supply_lot_id = l.id WHERE l.id = ? GROUP BY l.id`, lotID).Scan(&out).Error
	return out, database.Translate(err)
}

func (r *SupplyRepository) ClaimsForUpdate(ctx context.Context, lotID string) ([]model.SupplyClaim, error) {
	rows := []model.SupplyClaim{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("supply_lot_id = ?", lotID).Order("id").Find(&rows).Error
	return rows, database.Translate(err)
}

func (r *SupplyRepository) CreateClaims(ctx context.Context, claims []model.SupplyClaim) error {
	if len(claims) == 0 {
		return nil
	}
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(&claims).Error)
}

// UpdateClaim retains the claim ID and forbids changing materialization
// provenance once the unit has been consumed. Order ownership may still move.
func (r *SupplyRepository) UpdateClaim(ctx context.Context, claim *model.SupplyClaim) error {
	result := database.Conn(ctx, r.db).WithContext(ctx).Model(&model.SupplyClaim{}).
		Where(`id = ? AND (state <> 'materialized' OR
			(? = 'materialized' AND materialization_kind = ? AND
			 materialization_ref IS NOT DISTINCT FROM ? AND
			 materialization_unit_ordinal IS NOT DISTINCT FROM ? AND
			 materialized_at IS NOT DISTINCT FROM ?))`,
			claim.ID, claim.State, claim.MaterializationKind, claim.MaterializationRef, claim.MaterializationUnitOrdinal, claim.MaterializedAt).
		Updates(map[string]any{
			"reservation_id": claim.ReservationID, "reservation_line_id": claim.ReservationLineID,
			"order_id": claim.OrderID, "order_line_id": claim.OrderLineID, "state": claim.State,
			"materialization_kind": claim.MaterializationKind, "materialization_ref": claim.MaterializationRef,
			"materialization_unit_ordinal": claim.MaterializationUnitOrdinal, "materialized_at": claim.MaterializedAt,
		})
	if err := database.Translate(result.Error); err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return apperr.New(apperr.ErrConflict, "supply_claim_changed", "supply claim is missing or its materialization cannot be changed")
	}
	return nil
}

// ClaimAllocations includes retired associations so callers can display the
// physical history of a reservation after replacement or conversion.
func (r *SupplyRepository) ClaimAllocations(ctx context.Context, reservationID string) ([]model.SupplyClaimAllocation, error) {
	rows := []model.SupplyClaimAllocation{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Table("commerce_supply_claim_allocations AS a").
		Select("a.*").Joins("JOIN commerce_supply_claims AS c ON c.id = a.claim_id").
		Where("c.reservation_id = ?", reservationID).Order("a.created_at, a.id").Find(&rows).Error
	return rows, database.Translate(err)
}

// AppendClaimAllocation is called only with the enclosing commerce transaction.
// A replacement retires exactly the expected current association before adding
// its successor; the partial unique indexes reject duplicate current owners.
func (r *SupplyRepository) AppendClaimAllocation(ctx context.Context, association model.SupplyClaimAllocation, replaceID string) error {
	db := database.Conn(ctx, r.db).WithContext(ctx)
	if replaceID != "" {
		result := db.Model(&model.SupplyClaimAllocation{}).Where("id = ? AND claim_id = ? AND current = true", replaceID, association.ClaimID).Update("current", false)
		if err := database.Translate(result.Error); err != nil {
			return err
		}
		if result.RowsAffected != 1 {
			return apperr.New(apperr.ErrConflict, "claim_allocation_changed", "current claim allocation changed")
		}
	}
	return database.Translate(db.Create(&association).Error)
}

// RetireClaimAllocation records a physical release/rejection without changing
// the already materialized supply claim or reopening future capacity.
func (r *SupplyRepository) RetireClaimAllocation(ctx context.Context, claimID, associationID string) error {
	result := database.Conn(ctx, r.db).WithContext(ctx).Model(&model.SupplyClaimAllocation{}).
		Where("id = ? AND claim_id = ? AND current = true", associationID, claimID).Update("current", false)
	if err := database.Translate(result.Error); err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return apperr.New(apperr.ErrConflict, "claim_allocation_changed", "current claim allocation changed")
	}
	return nil
}
