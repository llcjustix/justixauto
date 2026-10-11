package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
)

// ReservationRepository is the persistence for vehicle holds.
type ReservationRepository struct{ db *gorm.DB }

// HeldByVehicle returns the active hold on a vehicle, if any.
func (r *ReservationRepository) HeldByVehicle(ctx context.Context, vehicleID string) (*model.Reservation, error) {
	var res model.Reservation
	err := conn(ctx, r.db).Where("vehicle_id = ? AND status = 'held'", vehicleID).Take(&res).Error
	if err != nil {
		return nil, translate(err)
	}
	return &res, nil
}

// LockHeldByVehicle locks the active hold for physical custody changes.
func (r *ReservationRepository) LockHeldByVehicle(ctx context.Context, vehicleID string) (*model.Reservation, error) {
	var res model.Reservation
	err := conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).Where("vehicle_id = ? AND status = 'held'", vehicleID).Take(&res).Error
	if err != nil {
		return nil, translate(err)
	}
	return &res, nil
}

// HeldFor returns the holder's active hold on a vehicle, if any.
func (r *ReservationRepository) HeldFor(ctx context.Context, holderType, holderID, vehicleID string) (*model.Reservation, error) {
	var res model.Reservation
	err := conn(ctx, r.db).Where("holder_type = ? AND holder_id = ? AND vehicle_id = ? AND status = 'held'", holderType, holderID, vehicleID).Take(&res).Error
	if err != nil {
		return nil, translate(err)
	}
	return &res, nil
}

func (r *ReservationRepository) Create(ctx context.Context, res *model.Reservation) error {
	return translate(conn(ctx, r.db).Create(res).Error)
}

// Reassign preserves the reservation row (including its ID and creation time)
// while moving its active holder.  A short update protects against a stale or
// partially released source hold; callers run this through their transaction.
func (r *ReservationRepository) Reassign(ctx context.Context, from, to model.Holder, vehicleIDs []string) error {
	if len(vehicleIDs) == 0 {
		return nil
	}
	result := conn(ctx, r.db).Model(&model.Reservation{}).
		Where("holder_type = ? AND holder_id = ? AND status = 'held' AND vehicle_id IN ?", from.Type, from.ID, vehicleIDs).
		Updates(map[string]any{"holder_type": to.Type, "holder_id": to.ID})
	if err := translate(result.Error); err != nil {
		return err
	}
	if result.RowsAffected != int64(len(vehicleIDs)) {
		return apperr.New(apperr.ErrConflict, "reservation_hold_changed", "vehicle hold is no longer owned by the source holder")
	}
	return nil
}

// Release marks the holder's held reservations released (nil vehicleIDs = all).
func (r *ReservationRepository) Release(ctx context.Context, holderType, holderID string, vehicleIDs []string, reason string, at time.Time) error {
	q := conn(ctx, r.db).Model(&model.Reservation{}).Where("holder_type = ? AND holder_id = ? AND status = 'held'", holderType, holderID)
	if vehicleIDs != nil {
		q = q.Where("vehicle_id IN ?", vehicleIDs)
	}
	return translate(q.Updates(map[string]any{"status": "released", "reason": reason, "closed_at": at}).Error)
}

// ListHeld returns the vehicle IDs the holder currently holds.
func (r *ReservationRepository) ListHeld(ctx context.Context, holderType, holderID string) ([]string, error) {
	var ids []string
	err := conn(ctx, r.db).Model(&model.Reservation{}).Where("holder_type = ? AND holder_id = ? AND status = 'held'", holderType, holderID).
		Order("vehicle_id").Pluck("vehicle_id", &ids).Error
	return ids, translate(err)
}

// CountHeld counts how many of vehicleIDs the holder currently holds.
func (r *ReservationRepository) CountHeld(ctx context.Context, holderType, holderID string, vehicleIDs []string) (int, error) {
	var n int64
	err := conn(ctx, r.db).Model(&model.Reservation{}).
		Where("holder_type = ? AND holder_id = ? AND status = 'held' AND vehicle_id IN ?", holderType, holderID, vehicleIDs).Count(&n).Error
	return int(n), translate(err)
}

// Finalize marks the holder's held reservations on vehicleIDs finalized.
func (r *ReservationRepository) Finalize(ctx context.Context, holderType, holderID string, vehicleIDs []string, at time.Time) error {
	err := conn(ctx, r.db).Model(&model.Reservation{}).
		Where("holder_type = ? AND holder_id = ? AND status = 'held' AND vehicle_id IN ?", holderType, holderID, vehicleIDs).
		Updates(map[string]any{"status": "finalized", "closed_at": at}).Error
	return translate(err)
}

// FinalizeOne finalizes a single reservation by ID.
func (r *ReservationRepository) FinalizeOne(ctx context.Context, id string, at time.Time) error {
	err := conn(ctx, r.db).Model(&model.Reservation{}).Where("id = ?", id).
		Updates(map[string]any{"status": "finalized", "closed_at": at}).Error
	return translate(err)
}
