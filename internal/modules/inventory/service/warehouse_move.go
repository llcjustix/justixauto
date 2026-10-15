package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
)

type UnidentifiedMoveInput struct {
	ReceiptBatchID string         `json:"receiptBatchId"`
	Quantity       jsonx.Quantity `json:"quantity"`
	Revision       jsonx.Quantity `json:"revision"`
}

type WarehouseMoveInput struct {
	ToWarehouseID       string                  `json:"toWarehouseId"`
	ToWarehouseRevision jsonx.Quantity          `json:"toWarehouseRevision"`
	OccurredAt          time.Time               `json:"occurredAt"`
	VehicleIDs          []string                `json:"vehicleIds"`
	Unidentified        []UnidentifiedMoveInput `json:"unidentified"`
}

type WarehouseMoveResult struct {
	SourceWarehouse      WarehouseView
	DestinationWarehouse WarehouseView
	MovedVehicleIDs      []string
	MovedUnidentified    []UnidentifiedMoveInput
	OccurredAt           time.Time
}

func canonicalUUID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed.String() == id
}

func validateWarehouseMoveInput(sourceID string, expected int64, in WarehouseMoveInput, now time.Time) error {
	var v apperr.Validation
	if !canonicalUUID(sourceID) {
		v.Add("sourceWarehouseId", "must be a valid ID")
	}
	if !canonicalUUID(in.ToWarehouseID) || in.ToWarehouseID == sourceID {
		v.Add("toWarehouseId", "must be a different warehouse")
	}
	if expected <= 0 {
		v.Add("If-Match", "must be a revision number")
	}
	if in.ToWarehouseRevision <= 0 {
		v.Add("toWarehouseRevision", "must be a revision number")
	}
	if in.OccurredAt.IsZero() || in.OccurredAt.After(now.Add(5*time.Minute)) {
		v.Add("occurredAt", "required and not in the future")
	}
	if len(in.VehicleIDs)+len(in.Unidentified) == 0 {
		v.Add("selection", "at least one vehicle or unidentified batch is required")
	}
	if len(in.VehicleIDs) > 1000 {
		v.Add("vehicleIds", "must contain at most 1000 IDs")
	}
	if len(in.Unidentified) > 1000 {
		v.Add("unidentified", "must contain at most 1000 batches")
	}
	seenVehicles := map[string]bool{}
	for _, id := range in.VehicleIDs {
		if !canonicalUUID(id) {
			v.Add("vehicleIds", "must contain valid IDs")
		}
		if seenVehicles[id] {
			v.Add("vehicleIds", "must not contain duplicates")
		}
		seenVehicles[id] = true
	}
	total := len(in.VehicleIDs)
	seenBatches := map[string]bool{}
	for _, item := range in.Unidentified {
		if !canonicalUUID(item.ReceiptBatchID) {
			v.Add("unidentified", "must contain valid receipt batch IDs")
		}
		if seenBatches[item.ReceiptBatchID] {
			v.Add("unidentified", "must not contain duplicate receipt batch IDs")
		}
		seenBatches[item.ReceiptBatchID] = true
		switch {
		case item.Quantity <= 0 || item.Quantity > 10000:
			v.Add("unidentified.quantity", "must be a positive whole number no greater than 10000")
		case int(item.Quantity) > 10000-total:
			v.Add("selection", "must move at most 10000 units")
		default:
			total += int(item.Quantity)
		}
		if item.Revision <= 0 {
			v.Add("unidentified.revision", "must be a revision number")
		}
	}
	if total > 10000 {
		v.Add("selection", "must move at most 10000 units")
	}
	return v.Err()
}

func stale(actual int64, expected jsonx.Quantity) bool { return actual != int64(expected) }

// MoveWarehouseStock moves the supplied VINs and receipt-local balances as one
// command. Its lock order deliberately differs from receipt identification:
// units/placements, batches, then warehouses; all move locks are NOWAIT.
func (s *Vehicle) MoveWarehouseStock(ctx context.Context, p *auth.Principal, sourceID string, expected int64, in WarehouseMoveInput) (*WarehouseMoveResult, error) {
	if err := p.Allow(model.PermVehiclesMove); err != nil {
		return nil, err
	}
	if err := validateWarehouseMoveInput(sourceID, expected, in, s.clock()); err != nil {
		return nil, err
	}
	vehicleIDs := append([]string(nil), in.VehicleIDs...)
	slices.Sort(vehicleIDs)
	batchInputs := append([]UnidentifiedMoveInput(nil), in.Unidentified...)
	slices.SortFunc(batchInputs, func(a, b UnidentifiedMoveInput) int { return strings.Compare(a.ReceiptBatchID, b.ReceiptBatchID) })
	result := &WarehouseMoveResult{MovedVehicleIDs: append([]string(nil), in.VehicleIDs...), MovedUnidentified: append([]UnidentifiedMoveInput(nil), in.Unidentified...), OccurredAt: in.OccurredAt.UTC()}
	err := s.store.InTx(ctx, func(st Store) error {
		// Fast-fail a replay before placement validation. The later NOWAIT source
		// lock and revision comparison remain the authoritative revalidation.
		sourcePreflight, err := st.Warehouses().Get(ctx, p.CompanyID, sourceID)
		if err != nil {
			return err
		}
		if sourcePreflight.Version != expected {
			return apperr.ErrStale
		}
		placements, err := lockWarehouseMovePlacements(ctx, st, p, sourceID, vehicleIDs)
		if err != nil {
			return err
		}
		batches, err := lockWarehouseMoveBatches(ctx, st, p, batchInputs)
		if err != nil {
			return err
		}
		source, destination, err := lockWarehouseMoveWarehouses(ctx, st, p, sourceID, expected, in)
		if err != nil {
			return err
		}
		occupied, err := st.Warehouses().Occupied(ctx, []string{source.ID, destination.ID})
		if err != nil {
			return err
		}
		locations, movedUnits, err := lockWarehouseMoveLocations(ctx, st, sourceID, len(vehicleIDs), batchInputs)
		if err != nil {
			return err
		}
		if occupied[destination.ID]+movedUnits > destination.Capacity {
			return apperr.New(apperr.ErrConflict, "capacity_exceeded", "not enough free space in the destination warehouse")
		}
		if err := s.moveWarehouseVehicles(ctx, st, p, sourceID, destination, vehicleIDs, placements, result.OccurredAt); err != nil {
			return err
		}
		if err := s.moveWarehouseBalances(ctx, st, p, sourceID, destination, batchInputs, locations, batches, result.OccurredAt); err != nil {
			return err
		}
		now := s.clock()
		if err := st.Warehouses().Touch(ctx, source, now); err != nil {
			return err
		}
		if err := st.Warehouses().Touch(ctx, destination, now); err != nil {
			return err
		}
		result.SourceWarehouse = WarehouseView{Warehouse: *source, Occupied: occupied[source.ID] - movedUnits}
		result.DestinationWarehouse = WarehouseView{Warehouse: *destination, Occupied: occupied[destination.ID] + movedUnits}
		return nil
	})
	return result, err
}

// lockWarehouseMovePlacements locks sorted units and placements, retaining tenant checks.
func lockWarehouseMovePlacements(ctx context.Context, st Store, p *auth.Principal, sourceID string, vehicleIDs []string) (map[string]*model.Placement, error) {
	placements := make(map[string]*model.Placement, len(vehicleIDs))
	for _, id := range vehicleIDs {
		if _, err := st.Vehicles().LockOwnedNowait(ctx, id); err != nil {
			return nil, err
		}
		if _, err := st.Vehicles().Get(ctx, p.CompanyID, id); err != nil {
			return nil, err
		}
		placement, err := st.Vehicles().LockPlacementNowait(ctx, id)
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.New(apperr.ErrConflict, "placement_changed", "a vehicle is no longer in the source warehouse, reload")
		}
		if err != nil {
			return nil, err
		}
		if placement.WarehouseID != sourceID {
			return nil, apperr.New(apperr.ErrConflict, "placement_changed", "a vehicle is no longer in the source warehouse, reload")
		}
		placements[id] = placement
	}
	return placements, nil
}

// lockWarehouseMoveBatches locks sorted batches and checks their supplied revisions.
func lockWarehouseMoveBatches(ctx context.Context, st Store, p *auth.Principal, batchInputs []UnidentifiedMoveInput) (map[string]*model.ReceiptBatch, error) {
	batches := make(map[string]*model.ReceiptBatch, len(batchInputs))
	for _, item := range batchInputs {
		b, err := st.Warehouses().LockBatchNowait(ctx, p.CompanyID, item.ReceiptBatchID)
		if err != nil {
			return nil, err
		}
		if stale(b.Version, item.Revision) {
			return nil, apperr.ErrStale
		}
		batches[item.ReceiptBatchID] = b
	}
	return batches, nil
}

// lockWarehouseMoveWarehouses locks both warehouses before authoritative revision checks.
func lockWarehouseMoveWarehouses(ctx context.Context, st Store, p *auth.Principal, sourceID string, expected int64, in WarehouseMoveInput) (*model.Warehouse, *model.Warehouse, error) {
	warehouseIDs := []string{sourceID, in.ToWarehouseID}
	slices.Sort(warehouseIDs)
	warehouses := map[string]*model.Warehouse{}
	for _, id := range warehouseIDs {
		w, err := st.Warehouses().LockNowait(ctx, p.CompanyID, id)
		if err != nil {
			return nil, nil, err
		}
		warehouses[id] = w
	}
	source, destination := warehouses[sourceID], warehouses[in.ToWarehouseID]
	if source.Version != expected || stale(destination.Version, in.ToWarehouseRevision) {
		return nil, nil, apperr.ErrStale
	}
	return source, destination, nil
}

// lockWarehouseMoveLocations checks each source balance and totals integer units.
func lockWarehouseMoveLocations(ctx context.Context, st Store, sourceID string, vehicleCount int, batchInputs []UnidentifiedMoveInput) (map[string]*model.ReceiptLocation, int, error) {
	movedUnits := vehicleCount
	locations := make(map[string]*model.ReceiptLocation, len(batchInputs))
	for _, item := range batchInputs {
		location, err := st.Warehouses().LockLocationNowait(ctx, item.ReceiptBatchID, sourceID)
		if errors.Is(err, apperr.ErrNotFound) || (err == nil && location.UnidentifiedCount < int(item.Quantity)) {
			return nil, 0, apperr.New(apperr.ErrConflict, "exceeds_unidentified", "not enough unidentified stock in the source warehouse")
		}
		if err != nil {
			return nil, 0, err
		}
		movedUnits += int(item.Quantity)
		locations[item.ReceiptBatchID] = location
	}
	return locations, movedUnits, nil
}

// moveWarehouseVehicles writes each placement immediately before its movement fact.
func (s *Vehicle) moveWarehouseVehicles(ctx context.Context, st Store, p *auth.Principal, sourceID string, destination *model.Warehouse, vehicleIDs []string, placements map[string]*model.Placement, at time.Time) error {
	for _, id := range vehicleIDs {
		placement := placements[id]
		placement.WarehouseID = destination.ID
		placement.PlacedAt = at
		if err := st.Vehicles().UpsertPlacement(ctx, *placement); err != nil {
			return err
		}
		if err := st.Facts().Append(ctx, s.fact(p, "vehicle.moved", &id, &destination.ID, placement.ReceiptBatchID, at, "", map[string]any{"from": sourceID, "to": destination.ID})); err != nil {
			return err
		}
	}
	return nil
}

// moveWarehouseBalances keeps source, destination, batch and fact writes in order.
func (s *Vehicle) moveWarehouseBalances(ctx context.Context, st Store, p *auth.Principal, sourceID string, destination *model.Warehouse, batchInputs []UnidentifiedMoveInput, locations map[string]*model.ReceiptLocation, batches map[string]*model.ReceiptBatch, at time.Time) error {
	for _, item := range batchInputs {
		location := locations[item.ReceiptBatchID]
		location.UnidentifiedCount -= int(item.Quantity)
		if err := st.Warehouses().UpdateLocation(ctx, location); err != nil {
			return err
		}
		if err := st.Warehouses().AddLocation(ctx, item.ReceiptBatchID, destination.ID, int(item.Quantity)); err != nil {
			return err
		}
		batch := batches[item.ReceiptBatchID]
		if err := st.Warehouses().UpdateBatchCounts(ctx, batch); err != nil {
			return err
		}
		batchID := item.ReceiptBatchID
		if err := st.Facts().Append(ctx, s.fact(p, "receipt.quantity_moved", nil, &destination.ID, &batchID, at, "", map[string]any{"from": sourceID, "to": destination.ID, "quantity": int(item.Quantity)})); err != nil {
			return err
		}
	}
	return nil
}
