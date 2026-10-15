package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/validate"
)

const (
	CustodyConsignment = "consignment"
	CustodyReturn      = "return"
)

// CustodyInput is a trusted commerce-to-inventory physical movement. Commerce
// checks agreement authority; inventory records the opaque references and
// enforces the physical state transition.
type CustodyInput struct {
	Purpose                                      string
	OwnerCompanyID, SourceCustodianCompanyID     string
	DestinationCompanyID, DestinationWarehouseID string
	ExpectedHolder                               model.Holder
	AgreementID, OperationRef, ActorUserID       string
	VehicleIDs                                   []string
	At                                           time.Time
}

// CustodyResult gives callers immutable facts to link to their own decision.
type CustodyResult struct{ FactIDs []string }

func validateCustodyInput(in CustodyInput, now time.Time) ([]string, error) {
	var v apperr.Validation
	for field, id := range map[string]string{
		"ownerCompanyId": in.OwnerCompanyID, "sourceCustodianCompanyId": in.SourceCustodianCompanyID,
		"destinationCompanyId": in.DestinationCompanyID, "destinationWarehouseId": in.DestinationWarehouseID,
		"agreementId": in.AgreementID, "operationRef": in.OperationRef, "actorUserId": in.ActorUserID,
	} {
		if uuid.Validate(id) != nil {
			v.Add(field, "must be a valid ID")
		}
	}
	if in.Purpose != CustodyConsignment && in.Purpose != CustodyReturn {
		v.Add("purpose", "must be consignment or return")
	}
	if in.At.IsZero() || in.At.After(now.Add(5*time.Minute)) {
		v.Add("at", "required and not in the future")
	}
	ids := validate.UniqueIDs(&v, "vehicleIds", in.VehicleIDs)
	if len(ids) == 0 {
		v.Add("vehicleIds", "list the vehicles")
	}
	if in.Purpose == CustodyConsignment {
		if in.OwnerCompanyID != in.SourceCustodianCompanyID {
			v.Add("sourceCustodianCompanyId", "must initially be the owner")
		}
		if in.ExpectedHolder.Type != "commerce-order" || uuid.Validate(in.ExpectedHolder.ID) != nil {
			v.Add("expectedHolder", "must be a commerce order holder")
		}
	} else if in.ExpectedHolder.Type != "" || in.ExpectedHolder.ID != "" {
		v.Add("expectedHolder", "is only valid for consignment")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	slices.Sort(ids)
	return ids, nil
}

// Custody moves custody and placement, retaining owner. It is a trusted
// module-root primitive and has no HTTP route.
func (s *Stock) Custody(ctx context.Context, in CustodyInput) (*CustodyResult, error) {
	ids, err := validateCustodyInput(in, s.clock())
	if err != nil {
		return nil, err
	}
	result := &CustodyResult{FactIDs: make([]string, 0, len(ids))}
	err = s.store.InTx(ctx, func(st Store) error {
		sources, err := lockCustodySources(ctx, st, in, ids)
		if err != nil {
			return err
		}
		warehouseIDs, warehouses, err := lockCustodyWarehouses(ctx, st, in, sources)
		if err != nil {
			return err
		}
		to := warehouses[in.DestinationWarehouseID]
		occupied, err := st.Warehouses().Occupied(ctx, []string{to.ID})
		if err != nil {
			return err
		}
		alreadyThere, err := st.Vehicles().CountPlaced(ctx, to.ID, ids)
		if err != nil {
			return err
		}
		if occupied[to.ID]+len(ids)-alreadyThere > to.Capacity {
			return apperr.New(apperr.ErrConflict, "capacity_exceeded", "not enough free space in the receiving warehouse")
		}
		return s.applyCustody(ctx, st, in, ids, warehouseIDs, warehouses, result)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// checkCustodyHold enforces the purpose-specific claim while its row is locked.
func checkCustodyHold(ctx context.Context, st Store, in CustodyInput, id string) error {
	hold, err := st.Reservations().LockHeldByVehicle(ctx, id)
	switch {
	case in.Purpose == CustodyConsignment:
		if errors.Is(err, apperr.ErrNotFound) || (err == nil && (hold.HolderType != in.ExpectedHolder.Type || hold.HolderID != in.ExpectedHolder.ID)) {
			return errUnavailable
		}
		if err != nil {
			return err
		}
	case err == nil:
		return errUnavailable
	case !errors.Is(err, apperr.ErrNotFound):
		return err
	}
	return nil
}

// lockCustodySources locks each sorted unit, placement and claim in that order.
func lockCustodySources(ctx context.Context, st Store, in CustodyInput, ids []string) (map[string]string, error) {
	sources := make(map[string]string, len(ids))
	for _, id := range ids {
		unit, err := st.Vehicles().LockOwned(ctx, id)
		if errors.Is(err, apperr.ErrNotFound) || (err == nil && (unit.OwnerCompanyID == nil || *unit.OwnerCompanyID != in.OwnerCompanyID || unit.CustodianCompanyID == nil || *unit.CustodianCompanyID != in.SourceCustodianCompanyID)) {
			return nil, errUnavailable
		}
		if err != nil {
			return nil, err
		}
		p, err := st.Vehicles().LockPlacement(ctx, id)
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.New(apperr.ErrConflict, "placement_changed", "a vehicle is no longer in the source warehouse")
		}
		if err != nil {
			return nil, err
		}
		sources[id] = p.WarehouseID
		if err := checkCustodyHold(ctx, st, in, id); err != nil {
			return nil, err
		}
	}
	return sources, nil
}

// lockCustodyWarehouses reconciles custodians before locking warehouses by ID.
func lockCustodyWarehouses(ctx context.Context, st Store, in CustodyInput, sources map[string]string) ([]string, map[string]*model.Warehouse, error) {
	owners := map[string]string{in.DestinationWarehouseID: in.DestinationCompanyID}
	for _, wid := range sources {
		if owner, exists := owners[wid]; exists && owner != in.SourceCustodianCompanyID {
			return nil, nil, errUnavailable
		}
		owners[wid] = in.SourceCustodianCompanyID
	}
	warehouseIDs := make([]string, 0, len(owners))
	for wid := range owners {
		warehouseIDs = append(warehouseIDs, wid)
	}
	slices.Sort(warehouseIDs)
	warehouses := make(map[string]*model.Warehouse, len(warehouseIDs))
	for _, wid := range warehouseIDs {
		w, err := st.Warehouses().Lock(ctx, owners[wid], wid)
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, nil, errUnavailable
		}
		if err != nil {
			return nil, nil, err
		}
		warehouses[wid] = w
	}
	return warehouseIDs, warehouses, nil
}

// applyCustody records each move and its result ID before finalizing claims and facts.
func (s *Stock) applyCustody(ctx context.Context, st Store, in CustodyInput, ids, warehouseIDs []string, warehouses map[string]*model.Warehouse, result *CustodyResult) error {
	to := warehouses[in.DestinationWarehouseID]
	now, facts := s.clock(), make([]model.Fact, 0, len(ids))
	for _, id := range ids {
		if err := st.Vehicles().SetCustodian(ctx, id, in.DestinationCompanyID); err != nil {
			return err
		}
		if err := st.Vehicles().MovePlacement(ctx, id, to.ID, in.At.UTC()); err != nil {
			return err
		}
		vid, factID := id, uuid.NewString()
		details, _ := json.Marshal(map[string]any{"purpose": in.Purpose, "ownerCompanyId": in.OwnerCompanyID, "sourceCustodianCompanyId": in.SourceCustodianCompanyID, "destinationCompanyId": in.DestinationCompanyID, "agreementId": in.AgreementID, "operationRef": in.OperationRef, "expectedHolderType": in.ExpectedHolder.Type, "expectedHolderId": in.ExpectedHolder.ID})
		facts = append(facts, model.Fact{ID: factID, CompanyID: in.DestinationCompanyID, FactType: "vehicle.custody_" + in.Purpose, VehicleID: &vid, WarehouseID: &to.ID, ActorUserID: in.ActorUserID, OccurredAt: in.At.UTC(), RecordedAt: now, Details: details})
		result.FactIDs = append(result.FactIDs, factID)
	}
	if in.Purpose == CustodyConsignment {
		if err := st.Reservations().Finalize(ctx, in.ExpectedHolder.Type, in.ExpectedHolder.ID, ids, now); err != nil {
			return err
		}
	}
	for _, wid := range warehouseIDs {
		if err := st.Warehouses().Touch(ctx, warehouses[wid], now); err != nil {
			return err
		}
	}
	return st.Facts().Append(ctx, facts...)
}
