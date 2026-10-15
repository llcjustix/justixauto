package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/database"
)

type WarehouseRepository struct{ db *gorm.DB }

func (r *WarehouseRepository) Create(ctx context.Context, w *model.Warehouse) error {
	return translate(conn(ctx, r.db).Create(w).Error)
}

func (r *WarehouseRepository) Get(ctx context.Context, companyID, id string) (*model.Warehouse, error) {
	var w model.Warehouse
	if err := conn(ctx, r.db).Where("id = ? AND company_id = ?", id, companyID).Take(&w).Error; err != nil {
		return nil, translate(err)
	}
	return &w, nil
}

func (r *WarehouseRepository) Lock(ctx context.Context, companyID, id string) (*model.Warehouse, error) {
	var w model.Warehouse
	err := conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND company_id = ?", id, companyID).Take(&w).Error
	if err != nil {
		return nil, translate(err)
	}
	return &w, nil
}

func (r *WarehouseRepository) LockNowait(ctx context.Context, companyID, id string) (*model.Warehouse, error) {
	var w model.Warehouse
	err := conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
		Where("id = ? AND company_id = ?", id, companyID).Take(&w).Error
	if err != nil {
		return nil, moveLockError(err)
	}
	return &w, nil
}

func (r *WarehouseRepository) List(ctx context.Context, f model.WarehouseFilter) ([]model.Warehouse, error) {
	q := conn(ctx, r.db).Where("company_id = ?", f.CompanyID).Order("name, id")
	if f.BranchIDs != nil {
		q = q.Where("(branch_id IS NULL OR branch_id IN ?)", append(f.BranchIDs, "00000000-0000-0000-0000-000000000000"))
	}
	ws := []model.Warehouse{}
	if err := translate(q.Find(&ws).Error); err != nil {
		return nil, err
	}
	return ws, nil
}

func (r *WarehouseRepository) Update(ctx context.Context, w *model.Warehouse, expected int64) error {
	err := database.UpdateVersioned(conn(ctx, r.db), &model.Warehouse{}, w.ID, expected, map[string]any{
		"name": w.Name, "country": w.Country, "country_key": w.CountryKey, "region": w.Region,
		"region_key": w.RegionKey, "city": w.City, "address": w.Address, "capacity": w.Capacity,
		"branch_id": w.BranchID, "updated_at": w.UpdatedAt,
	})
	if err == nil {
		w.Version = expected + 1
	}
	return err
}

func (r *WarehouseRepository) ByBranch(ctx context.Context, companyID, branchID string) (*model.Warehouse, error) {
	var w model.Warehouse
	if err := conn(ctx, r.db).Where("company_id = ? AND branch_id = ?", companyID, branchID).Take(&w).Error; err != nil {
		return nil, translate(err)
	}
	return &w, nil
}

func (r *WarehouseRepository) Touch(ctx context.Context, w *model.Warehouse, at time.Time) error {
	err := conn(ctx, r.db).Model(&model.Warehouse{}).Where("id = ?", w.ID).
		Updates(map[string]any{"version": gorm.Expr("version + 1"), "updated_at": at}).Error
	if err == nil {
		w.Version++
		w.UpdatedAt = at
	}
	return translate(err)
}

// BumpByID bumps a warehouse's version by ID, without a prior read or lock.
func (r *WarehouseRepository) BumpByID(ctx context.Context, id string, at time.Time) error {
	err := conn(ctx, r.db).Model(&model.Warehouse{}).Where("id = ?", id).
		Updates(map[string]any{"version": gorm.Expr("version + 1"), "updated_at": at}).Error
	return translate(err)
}

func (r *WarehouseRepository) Occupied(ctx context.Context, ids []string) (map[string]int, error) {
	out := make(map[string]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		WarehouseID string
		Occupied    int
	}
	err := conn(ctx, r.db).Raw(`
		SELECT warehouse_id, sum(n)::int AS occupied FROM (
			SELECT warehouse_id, count(*) AS n FROM inventory_placements WHERE warehouse_id IN ? GROUP BY warehouse_id
			UNION ALL
			SELECT warehouse_id, sum(unidentified_count) FROM inventory_receipt_locations WHERE warehouse_id IN ? GROUP BY warehouse_id
		) t GROUP BY warehouse_id`, ids, ids).Scan(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	for _, row := range rows {
		out[row.WarehouseID] = row.Occupied
	}
	return out, nil
}

func (r *WarehouseRepository) CreateBatch(ctx context.Context, b *model.ReceiptBatch) error {
	return translate(conn(ctx, r.db).Create(b).Error)
}

func (r *WarehouseRepository) CreateLocation(ctx context.Context, l *model.ReceiptLocation) error {
	return translate(conn(ctx, r.db).Create(l).Error)
}

// Batches reads a bounded, caller-selected set of receipt batches without
// locking. All rows must belong to companyID so callers cannot learn another
// company's warehouse or identification state.
func (r *WarehouseRepository) Batches(ctx context.Context, companyID string, ids []string) ([]model.ReceiptBatch, error) {
	if len(ids) == 0 {
		return []model.ReceiptBatch{}, nil
	}
	bs := []model.ReceiptBatch{}
	err := conn(ctx, r.db).Where("company_id = ? AND id IN ?", companyID, ids).Order("id").Find(&bs).Error
	if err != nil {
		return nil, translate(err)
	}
	if len(bs) != len(ids) {
		return nil, apperr.ErrNotFound
	}
	return bs, nil
}

func (r *WarehouseRepository) LockBatch(ctx context.Context, companyID, id string) (*model.ReceiptBatch, error) {
	var b model.ReceiptBatch
	err := conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND company_id = ?", id, companyID).Take(&b).Error
	if err != nil {
		return nil, translate(err)
	}
	return &b, nil
}

func (r *WarehouseRepository) LockBatchNowait(ctx context.Context, companyID, id string) (*model.ReceiptBatch, error) {
	var b model.ReceiptBatch
	err := conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
		Where("id = ? AND company_id = ?", id, companyID).Take(&b).Error
	if err != nil {
		return nil, moveLockError(err)
	}
	return &b, nil
}

func (r *WarehouseRepository) UpdateBatchCounts(ctx context.Context, b *model.ReceiptBatch) error {
	res := conn(ctx, r.db).Model(&model.ReceiptBatch{}).Where("id = ?", b.ID).Updates(map[string]any{
		"confirmed_quantity": b.ConfirmedQuantity, "identified_count": b.IdentifiedCount, "unidentified_count": b.UnidentifiedCount,
		"exterior_color": b.ExteriorColor, "interior_color": b.InteriorColor, "version": b.Version + 1,
	})
	if res.Error == nil {
		b.Version++
	}
	return translate(res.Error)
}

func (r *WarehouseRepository) Locations(ctx context.Context, companyID, batchID string) ([]model.ReceiptLocation, error) {
	locations := []model.ReceiptLocation{}
	err := conn(ctx, r.db).Table("inventory_receipt_locations AS l").
		Select("l.receipt_batch_id, l.warehouse_id, l.unidentified_count").
		Joins("JOIN inventory_receipt_batches b ON b.id = l.receipt_batch_id").
		Where("b.company_id = ? AND l.receipt_batch_id = ? AND l.unidentified_count > 0", companyID, batchID).
		Order("l.warehouse_id").Scan(&locations).Error
	return locations, translate(err)
}

func (r *WarehouseRepository) LockLocation(ctx context.Context, batchID, warehouseID string) (*model.ReceiptLocation, error) {
	var location model.ReceiptLocation
	err := conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("receipt_batch_id = ? AND warehouse_id = ?", batchID, warehouseID).Take(&location).Error
	if err != nil {
		return nil, translate(err)
	}
	return &location, nil
}

func (r *WarehouseRepository) LockLocationNowait(ctx context.Context, batchID, warehouseID string) (*model.ReceiptLocation, error) {
	var location model.ReceiptLocation
	err := conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
		Where("receipt_batch_id = ? AND warehouse_id = ?", batchID, warehouseID).Take(&location).Error
	if err != nil {
		return nil, moveLockError(err)
	}
	return &location, nil
}

func (r *WarehouseRepository) UpdateLocation(ctx context.Context, l *model.ReceiptLocation) error {
	return translate(conn(ctx, r.db).Model(&model.ReceiptLocation{}).
		Where("receipt_batch_id = ? AND warehouse_id = ?", l.ReceiptBatchID, l.WarehouseID).
		Update("unidentified_count", l.UnidentifiedCount).Error)
}

// AddLocation serializes an absent destination location on its already-locked
// receipt batch and increments an existing balance without replacing it.
func (r *WarehouseRepository) AddLocation(ctx context.Context, batchID, warehouseID string, quantity int) error {
	l := model.ReceiptLocation{ReceiptBatchID: batchID, WarehouseID: warehouseID, UnidentifiedCount: quantity}
	err := conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "receipt_batch_id"}, {Name: "warehouse_id"}},
		DoUpdates: clause.Assignments(map[string]any{"unidentified_count": gorm.Expr("inventory_receipt_locations.unidentified_count + ?", quantity)}),
	}).Create(&l).Error
	return translate(err)
}

func (r *WarehouseRepository) OpenBatches(ctx context.Context, warehouseID string) ([]model.LocalReceiptBatch, error) {
	bs := []model.LocalReceiptBatch{}
	err := conn(ctx, r.db).Table("inventory_receipt_locations AS l").
		Select("b.*, l.unidentified_count AS warehouse_unidentified_count").
		Joins("JOIN inventory_receipt_batches b ON b.id = l.receipt_batch_id").
		Where("l.warehouse_id = ? AND l.unidentified_count > 0", warehouseID).
		Order("b.received_at, b.id").Scan(&bs).Error
	return bs, translate(err)
}
