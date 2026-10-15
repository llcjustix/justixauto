package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

// ReservationAllocationItem selects an accepted future source line. Claim IDs
// and lot IDs are resolved from the stored reservation, never from the caller.
type ReservationAllocationItem struct {
	LineID    string `json:"lineId"`
	VehicleID string `json:"vehicleId"`
}

type ReservationAllocationView struct {
	ReservationID string
	Revision      int64
	Allocations   []model.SupplyClaimAllocation
}

func (s *Reservation) Allocations(ctx context.Context, p *auth.Principal, id string) (*ReservationAllocationView, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	if uuid.Validate(id) != nil {
		return nil, apperr.ErrNotFound
	}
	r, _, err := s.store.Reservations().Get(ctx, p.CompanyID, id)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.Supply().ClaimAllocations(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ReservationAllocationView{ReservationID: id, Revision: r.Version, Allocations: rows}, nil
}

// Allocate holds exact supplier VINs and consumes existing held claims in one
// commerce transaction. The reservation row, then sorted lots, then inventory
// VINs are locked in that order. No new capacity claim is created here.
func (s *Reservation) Allocate(ctx context.Context, p *auth.Principal, id string, expected int64, items []ReservationAllocationItem) (*ReservationAllocationView, error) {
	if err := s.distributor(ctx, p); err != nil {
		return nil, err
	}
	if len(items) == 0 || len(items) > 1000 {
		return nil, apperr.FieldError("items", "list 1-1000 VINs")
	}
	var out *ReservationAllocationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, _, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if r.Status != model.ReservationActive || r.OrderID != nil || r.ExpiresAt == nil || !r.ExpiresAt.After(s.clock()) {
			return apperr.New(apperr.ErrConflict, "reservation_not_allocatable", "reservation is not active or has moved to an order")
		}
		snap, err := decodeReservationSnapshot(r.CurrentTerms)
		if err != nil {
			return err
		}
		if snap == nil {
			return apperr.New(apperr.ErrConflict, "reservation_terms_missing", "accepted source is unavailable")
		}
		plan, err := s.prepareReservationAllocationLocked(ctx, st, r, snap, items)
		if err != nil {
			return err
		}
		if err := s.persistReservationAllocationsLocked(ctx, st, r, plan); err != nil {
			return err
		}
		r.UpdatedAt = s.clock()
		if err := st.Reservations().Update(ctx, r, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.vins_allocated", "reservation", r.ID, "", map[string]any{"vehicleIds": plan.vehicleIDs}); err != nil {
			return err
		}
		rows, err := st.Supply().ClaimAllocations(ctx, id)
		if err != nil {
			return err
		}
		out = &ReservationAllocationView{ReservationID: id, Revision: r.Version, Allocations: rows}
		return nil
	})
	return out, err
}

func sameLotSource(lot *model.SupplyLot, supplier string, line ReservationSnapshotLine) bool {
	return lot != nil && lot.SupplierCompanyID == supplier && lot.ModelID == line.ModelID && lot.ModelSpecificationVersion == line.ModelSpecificationVersion && lot.ExteriorColor == line.ExteriorColor && lot.InteriorColor == line.InteriorColor
}

// TransferObligations is a transaction-bound primitive for the conversion
// service. The caller creates the order and source-line mapping, holds the
// reservation lock, adds returned order allocations, and commits all together.
// No independent transaction or public transfer endpoint is exposed.
func (s *Reservation) TransferObligations(ctx context.Context, st Store, r *model.Reservation, orderID string, sourceToOrderLine map[string]string) ([]model.Allocation, error) {
	if r.Status != model.ReservationActive || r.OrderID != nil || uuid.Validate(orderID) != nil {
		return nil, apperr.New(apperr.ErrConflict, "reservation_not_transferable", "reservation cannot be transferred")
	}
	snap, err := decodeReservationSnapshot(r.CurrentTerms)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, apperr.New(apperr.ErrConflict, "reservation_terms_missing", "accepted source is unavailable")
	}
	lotIDs := []string{}
	for _, line := range snap.Lines {
		if uuid.Validate(sourceToOrderLine[line.LineID]) != nil {
			return nil, apperr.FieldError("sourceToOrderLine", "every accepted line needs an order line")
		}
		if line.SupplyLotID != nil {
			lotIDs = append(lotIDs, *line.SupplyLotID)
		}
	}
	slices.Sort(lotIDs)
	lotIDs = slices.Compact(lotIDs)
	for _, id := range lotIDs {
		if _, err := st.Supply().LockLot(ctx, id); err != nil {
			return nil, err
		}
	}
	associations, err := st.Supply().ClaimAllocations(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	current := map[string]model.SupplyClaimAllocation{}
	for _, a := range associations {
		if a.Current {
			current[a.ClaimID] = a
		}
	}
	transfer := reservationTransfer{vinIDs: []string{}, allocations: []model.Allocation{}}
	for _, line := range snap.Lines {
		if line.VehicleID != nil {
			transfer.vinIDs = append(transfer.vinIDs, *line.VehicleID)
			v, err := s.reservationStock.Vehicle(st.Bind(ctx), r.SupplierCompanyID, *line.VehicleID)
			if err != nil {
				return nil, err
			}
			transfer.allocations = append(transfer.allocations, model.Allocation{ID: uuid.NewString(), OrderID: orderID, LineID: sourceToOrderLine[line.LineID], VehicleID: *line.VehicleID, VIN: v.VIN, Status: "allocated", CreatedAt: s.clock()})
		}
	}
	for _, lotID := range lotIDs {
		if err := s.transferReservationLotLocked(ctx, st, r, orderID, sourceToOrderLine, current, lotID, &transfer); err != nil {
			return nil, err
		}
	}
	slices.Sort(transfer.vinIDs)
	if len(transfer.vinIDs) > 0 {
		if err := s.reservationStock.ReassignReservationToOrder(st.Bind(ctx), r.ID, orderID, transfer.vinIDs); err != nil {
			return nil, err
		}
	}
	return transfer.allocations, nil
}

// materializeOrderClaims is called by the P18b ordinary order allocation path
// inside its existing Store.InTx, after supplier/order checks and before commit.
// It leaves stock reservation and order revision/event to that caller.
func (d Deps) materializeOrderClaims(ctx context.Context, st Store, order *model.Order, add []model.Allocation) error {
	r, _, err := st.Reservations().ByOrderForUpdate(ctx, order.ID)
	if errors.Is(err, apperr.ErrNotFound) && order.Source != "reservation" {
		return nil
	}
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.New(apperr.ErrConflict, "reservation_source_missing", "order source reservation is missing")
		}
		return err
	}
	if order.Source != "reservation" || r.Status != model.ReservationConverted || r.OrderID == nil || *r.OrderID != order.ID || r.SupplierCompanyID != order.SupplierCompanyID || r.BuyerCompanyID != order.BuyerCompanyID {
		return apperr.New(apperr.ErrConflict, "reservation_source_changed", "order source reservation does not match")
	}
	provenance := order.DecodeTerms().Reservation
	if provenance == nil || provenance.ReservationID != r.ID || provenance.BookingDigest != r.CurrentDigest || provenance.ContinuationDigest != r.ContinuationAcceptedDigest || provenance.ContinuationRevision != r.ContinuationAcceptedRevision {
		return apperr.New(apperr.ErrConflict, "reservation_source_changed", "order continuation provenance does not match")
	}
	snap, err := decodeReservationSnapshot(r.CurrentTerms)
	if err != nil {
		return err
	}
	if snap == nil {
		return apperr.New(apperr.ErrConflict, "reservation_terms_missing", "accepted source is unavailable")
	}
	orderLines := map[string]model.Line{}
	for _, line := range order.DecodeTerms().Lines {
		orderLines[line.LineID] = line
	}
	lotIDs := []string{}
	lines := map[string]ReservationSnapshotLine{}
	for _, line := range snap.Lines {
		lines[line.LineID] = line
		if line.SupplyLotID != nil {
			lotIDs = append(lotIDs, *line.SupplyLotID)
		}
	}
	slices.Sort(lotIDs)
	lotIDs = slices.Compact(lotIDs)
	claimsByLot, err := lockReservationClaimLots(ctx, st, lotIDs)
	if err != nil {
		return err
	}
	associations, err := st.Supply().ClaimAllocations(ctx, r.ID)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, a := range associations {
		if a.Current {
			active[a.ClaimID] = true
		}
	}
	sources := orderClaimSources{lines: lines, lotIDs: lotIDs, claimsByLot: claimsByLot, active: active}
	for _, a := range add {
		if err := d.materializeOrderAllocationLocked(ctx, st, r, order, &a, orderLines, &sources); err != nil {
			return err
		}
	}
	return nil
}

// materializeReceiptClaims consumes the converted order's existing future
// units. A claim's immutable reference is the persisted commerce shipment line;
// that line carries the inventory receipt batch. Its ordinal identifies a unit
// within the line without inventing a VIN before inventory Identify runs.
func (d Deps) materializeReceiptClaims(ctx context.Context, st Store, order *model.Order, rows []model.ShipmentLine, at time.Time) error {
	if order.Source != "reservation" {
		return nil
	}
	r, _, err := st.Reservations().ByOrderForUpdate(ctx, order.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		return apperr.New(apperr.ErrConflict, "reservation_source_missing", "order source reservation is missing")
	}
	if err != nil {
		return err
	}
	if r.Status != model.ReservationConverted || r.OrderID == nil || *r.OrderID != order.ID || r.SupplierCompanyID != order.SupplierCompanyID || r.BuyerCompanyID != order.BuyerCompanyID {
		return apperr.New(apperr.ErrConflict, "reservation_source_changed", "order source reservation does not match")
	}
	terms := order.DecodeTerms()
	provenance := terms.Reservation
	if provenance == nil || provenance.ReservationID != r.ID || provenance.BookingDigest != r.CurrentDigest || provenance.ContinuationDigest != r.ContinuationAcceptedDigest || provenance.ContinuationRevision != r.ContinuationAcceptedRevision || provenance.Mode != "purchase" {
		return apperr.New(apperr.ErrConflict, "reservation_source_changed", "order continuation provenance does not match")
	}
	snap, err := decodeReservationSnapshot(r.CurrentTerms)
	if err != nil {
		return err
	}
	if snap == nil {
		return apperr.New(apperr.ErrConflict, "reservation_terms_missing", "accepted source is unavailable")
	}
	sources := map[string]ReservationSnapshotLine{}
	lotIDs := []string{}
	for _, line := range snap.Lines {
		sources[line.LineID] = line
		if line.SupplyLotID != nil {
			lotIDs = append(lotIDs, *line.SupplyLotID)
		}
	}
	slices.Sort(lotIDs)
	lotIDs = slices.Compact(lotIDs)
	claimsByLot, err := lockReservationClaimLots(ctx, st, lotIDs)
	if err != nil {
		return err
	}
	orderLines := map[string]model.Line{}
	for _, line := range terms.Lines {
		orderLines[line.LineID] = line
	}
	for _, row := range rows {
		if err := materializeReceiptRowLocked(ctx, st, r, order, &row, orderLines, sources, claimsByLot, at); err != nil {
			return err
		}
	}
	return nil
}

// releaseOrderClaims runs only after existing order cancellation blockers have
// allowed the transition. Only residual held claims reopen future capacity.
func (d Deps) releaseOrderClaims(ctx context.Context, st Store, order *model.Order) error {
	r, _, err := st.Reservations().ByOrderForUpdate(ctx, order.ID)
	if err != nil {
		return err
	}
	if r.Status != model.ReservationConverted || r.OrderID == nil || *r.OrderID != order.ID || r.BuyerCompanyID != order.BuyerCompanyID || r.SupplierCompanyID != order.SupplierCompanyID {
		return apperr.New(apperr.ErrConflict, "reservation_source_changed", "order source reservation does not match")
	}
	provenance := order.DecodeTerms().Reservation
	if provenance == nil || provenance.ReservationID != r.ID || provenance.BookingDigest != r.CurrentDigest || provenance.ContinuationDigest != r.ContinuationAcceptedDigest {
		return apperr.New(apperr.ErrConflict, "reservation_source_changed", "order continuation provenance does not match")
	}
	snap, err := decodeReservationSnapshot(r.CurrentTerms)
	if err != nil {
		return err
	}
	if snap == nil {
		return apperr.New(apperr.ErrConflict, "reservation_terms_missing", "accepted source is unavailable")
	}
	lotIDs := []string{}
	for _, line := range snap.Lines {
		if line.SupplyLotID != nil {
			lotIDs = append(lotIDs, *line.SupplyLotID)
		}
	}
	slices.Sort(lotIDs)
	lotIDs = slices.Compact(lotIDs)
	for _, lotID := range lotIDs {
		if _, err := st.Supply().LockLot(ctx, lotID); err != nil {
			return err
		}
	}
	for _, lotID := range lotIDs {
		if err := releaseHeldOrderClaims(ctx, st, order.ID, r.ID, lotID); err != nil {
			return err
		}
	}
	associations, err := st.Supply().ClaimAllocations(ctx, r.ID)
	if err != nil {
		return err
	}
	for _, association := range associations {
		if association.Current && association.HolderType == "commerce-order" && association.HolderID == order.ID {
			if err := st.Supply().RetireClaimAllocation(ctx, association.ClaimID, association.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// A rejected physical VIN loses its current holder association, while its
// materialized future claim remains consumed and available capacity unchanged.
func (d Deps) retireRejectedOrderClaims(ctx context.Context, st Store, order *model.Order, vehicleIDs []string) error {
	r, _, err := st.Reservations().ByOrderForUpdate(ctx, order.ID)
	if err != nil {
		return err
	}
	if r.Status != model.ReservationConverted || r.OrderID == nil || *r.OrderID != order.ID {
		return apperr.New(apperr.ErrConflict, "reservation_source_changed", "order source reservation does not match")
	}
	associations, err := st.Supply().ClaimAllocations(ctx, r.ID)
	if err != nil {
		return err
	}
	for _, id := range vehicleIDs {
		for _, a := range associations {
			if a.Current && a.VehicleID == id && a.HolderType == "commerce-order" && a.HolderID == order.ID {
				if err := st.Supply().RetireClaimAllocation(ctx, a.ClaimID, a.ID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func vehicleMatchesLine(v *StockVehicle, line ReservationSnapshotLine) bool {
	return v.ModelID == line.ModelID && v.ModelSpecificationVersion == line.ModelSpecificationVersion && strings.EqualFold(v.ExteriorColor, line.ExteriorColor) && strings.EqualFold(v.InteriorColor, line.InteriorColor)
}

type reservationAllocationSources struct {
	lines               map[string]ReservationSnapshotLine
	initialVIN, usedVIN map[string]bool
	claimsByLot         map[string][]model.SupplyClaim
}

type reservationAllocationPlan struct {
	claims       []model.SupplyClaim
	associations []model.SupplyClaimAllocation
	vehicleIDs   []string
}

func (s *Reservation) prepareReservationAllocationLocked(ctx context.Context, st Store, r *model.Reservation, snap *ReservationSnapshot, items []ReservationAllocationItem) (*reservationAllocationPlan, error) {
	lines := map[string]ReservationSnapshotLine{}
	lotIDs := []string{}
	initialVIN := map[string]bool{}
	for _, l := range snap.Lines {
		lines[l.LineID] = l
		if l.SupplyLotID != nil {
			lotIDs = append(lotIDs, *l.SupplyLotID)
		}
		if l.VehicleID != nil {
			initialVIN[*l.VehicleID] = true
		}
	}
	slices.Sort(lotIDs)
	lotIDs = slices.Compact(lotIDs)
	for _, lotID := range lotIDs {
		if _, err := st.Supply().LockLot(ctx, lotID); err != nil {
			return nil, err
		}
	}
	claimsByLot := map[string][]model.SupplyClaim{}
	for _, lotID := range lotIDs {
		claims, err := st.Supply().ClaimsForUpdate(ctx, lotID)
		if err != nil {
			return nil, err
		}
		claimsByLot[lotID] = claims
	}
	allAssociations, err := st.Supply().ClaimAllocations(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	usedVIN := map[string]bool{}
	for _, a := range allAssociations {
		if a.Current {
			usedVIN[a.VehicleID] = true
		}
	}
	selected := make([]model.SupplyClaim, 0, len(items))
	associations := make([]model.SupplyClaimAllocation, 0, len(items))
	vehicleIDs := make([]string, 0, len(items))
	sources := reservationAllocationSources{lines: lines, initialVIN: initialVIN, usedVIN: usedVIN, claimsByLot: claimsByLot}
	for _, item := range items {
		claim, association, err := s.selectReservationAllocationLocked(ctx, st, r, &sources, item)
		if err != nil {
			return nil, err
		}
		selected = append(selected, claim)
		associations = append(associations, association)
		vehicleIDs = append(vehicleIDs, item.VehicleID)
		usedVIN[item.VehicleID] = true
	}
	return &reservationAllocationPlan{claims: selected, associations: associations, vehicleIDs: vehicleIDs}, nil
}

func (s *Reservation) selectReservationAllocationLocked(ctx context.Context, st Store, r *model.Reservation, sources *reservationAllocationSources, item ReservationAllocationItem) (model.SupplyClaim, model.SupplyClaimAllocation, error) {
	if uuid.Validate(item.LineID) != nil || uuid.Validate(item.VehicleID) != nil {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, apperr.FieldError("items", "line and vehicle IDs must be valid")
	}
	line, ok := sources.lines[item.LineID]
	if !ok || line.SupplyLotID == nil {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, apperr.FieldError("items.lineId", "must name an accepted future source line")
	}
	if sources.usedVIN[item.VehicleID] || sources.initialVIN[item.VehicleID] {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, apperr.New(apperr.ErrConflict, "vin_already_allocated", "VIN already belongs to this reservation")
	}
	lot, err := st.Supply().Lot(ctx, *line.SupplyLotID)
	if err != nil {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, err
	}
	if !sameLotSource(lot, r.SupplierCompanyID, line) {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, apperr.New(apperr.ErrConflict, "supply_source_changed", "future source has changed")
	}
	v, err := s.reservationStock.Vehicle(st.Bind(ctx), r.SupplierCompanyID, item.VehicleID)
	if err != nil {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, apperr.FieldError("items.vehicleId", "VIN is not owned by this supplier")
	}
	if v.ModelID != line.ModelID || v.ModelSpecificationVersion != line.ModelSpecificationVersion || !strings.EqualFold(v.ExteriorColor, line.ExteriorColor) || !strings.EqualFold(v.InteriorColor, line.InteriorColor) {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, apperr.FieldError("items.vehicleId", "VIN does not match the accepted model, specification and colors")
	}
	claims := sources.claimsByLot[lot.ID]
	chosen := -1
	for j := range claims {
		c := claims[j]
		if c.State == "held" && c.ReservationID != nil && *c.ReservationID == r.ID && c.ReservationLineID != nil && *c.ReservationLineID == line.LineID && c.OrderID == nil {
			chosen = j
			break
		}
	}
	if chosen < 0 {
		return model.SupplyClaim{}, model.SupplyClaimAllocation{}, apperr.New(apperr.ErrConflict, "future_obligation_exhausted", "no unallocated future unit remains on the source line")
	}
	claim := claims[chosen]
	association := model.SupplyClaimAllocation{ID: uuid.NewString(), ClaimID: claim.ID, ReservationLineID: line.LineID, VehicleID: item.VehicleID, HolderType: "commerce-reservation", HolderID: r.ID, Current: true, CreatedAt: s.clock()}
	ordinal := int32(1)
	claim.State = "materialized"
	claim.MaterializationKind = "vin-allocation"
	claim.MaterializationRef = &association.ID
	claim.MaterializationUnitOrdinal = &ordinal
	now := s.clock()
	claim.MaterializedAt = &now
	claims[chosen] = claim
	sources.claimsByLot[lot.ID] = claims
	return claim, association, nil
}

func (s *Reservation) persistReservationAllocationsLocked(ctx context.Context, st Store, r *model.Reservation, plan *reservationAllocationPlan) error {
	slices.Sort(plan.vehicleIDs)
	if err := s.reservationStock.ReserveReservation(st.Bind(ctx), r.SupplierCompanyID, r.ID, plan.vehicleIDs); err != nil {
		return err
	}
	for i := range plan.claims {
		if err := st.Supply().UpdateClaim(ctx, &plan.claims[i]); err != nil {
			return err
		}
		if err := st.Supply().AppendClaimAllocation(ctx, plan.associations[i], ""); err != nil {
			return err
		}
	}
	return nil
}

type reservationTransfer struct {
	vinIDs      []string
	allocations []model.Allocation
}

func (s *Reservation) transferReservationLotLocked(ctx context.Context, st Store, r *model.Reservation, orderID string, sourceToOrderLine map[string]string, current map[string]model.SupplyClaimAllocation, lotID string, transfer *reservationTransfer) error {
	claims, err := st.Supply().ClaimsForUpdate(ctx, lotID)
	if err != nil {
		return err
	}
	for i := range claims {
		c := &claims[i]
		if c.ReservationID == nil || *c.ReservationID != r.ID || c.OrderID != nil {
			continue
		}
		if c.ReservationLineID == nil || sourceToOrderLine[*c.ReservationLineID] == "" {
			return apperr.New(apperr.ErrConflict, "source_line_missing", "future claim has no mapped order line")
		}
		if c.State != "held" && c.State != "materialized" {
			return apperr.New(apperr.ErrConflict, "claim_not_transferable", "future claim cannot be transferred")
		}
		c.OrderID = &orderID
		lineID := sourceToOrderLine[*c.ReservationLineID]
		c.OrderLineID = &lineID
		if err := st.Supply().UpdateClaim(ctx, c); err != nil {
			return err
		}
		if c.State != "materialized" {
			continue
		}
		a, ok := current[c.ID]
		if !ok || a.HolderType != "commerce-reservation" || a.HolderID != r.ID {
			return apperr.New(apperr.ErrConflict, "claim_allocation_missing", "materialized claim has no current reservation VIN")
		}
		v, err := s.reservationStock.Vehicle(st.Bind(ctx), r.SupplierCompanyID, a.VehicleID)
		if err != nil {
			return err
		}
		allocation := model.Allocation{ID: uuid.NewString(), OrderID: orderID, LineID: lineID, VehicleID: a.VehicleID, VIN: v.VIN, Status: "allocated", CreatedAt: s.clock()}
		transfer.allocations = append(transfer.allocations, allocation)
		transfer.vinIDs = append(transfer.vinIDs, a.VehicleID)
		successor := model.SupplyClaimAllocation{ID: allocation.ID, ClaimID: c.ID, ReservationLineID: *c.ReservationLineID, VehicleID: a.VehicleID, HolderType: "commerce-order", HolderID: orderID, OrderLineID: &lineID, Current: true, CreatedAt: s.clock()}
		if err := st.Supply().AppendClaimAllocation(ctx, successor, a.ID); err != nil {
			return err
		}
	}
	return nil
}

type orderClaimSources struct {
	lines       map[string]ReservationSnapshotLine
	lotIDs      []string
	claimsByLot map[string][]model.SupplyClaim
	active      map[string]bool
}

func lockReservationClaimLots(ctx context.Context, st Store, lotIDs []string) (map[string][]model.SupplyClaim, error) {
	for _, id := range lotIDs {
		if _, err := st.Supply().LockLot(ctx, id); err != nil {
			return nil, err
		}
	}
	claimsByLot := map[string][]model.SupplyClaim{}
	for _, id := range lotIDs {
		claims, err := st.Supply().ClaimsForUpdate(ctx, id)
		if err != nil {
			return nil, err
		}
		claimsByLot[id] = claims
	}
	return claimsByLot, nil
}

func (d Deps) materializeOrderAllocationLocked(ctx context.Context, st Store, r *model.Reservation, order *model.Order, a *model.Allocation, orderLines map[string]model.Line, sources *orderClaimSources) error {
	orderLine, ok := orderLines[a.LineID]
	if !ok {
		return apperr.New(apperr.ErrConflict, "source_line_missing", "allocation has no saved order line")
	}
	v, err := d.stock.Vehicle(st.Bind(ctx), order.SupplierCompanyID, a.VehicleID)
	if err != nil {
		return err
	}
	chosenLot, chosenIndex, err := findOrderAllocationClaim(ctx, st, order.ID, a.LineID, &orderLine, v, r.SupplierCompanyID, sources)
	if err != nil {
		return err
	}
	claim := sources.claimsByLot[chosenLot][chosenIndex]
	association := model.SupplyClaimAllocation{ID: a.ID, ClaimID: claim.ID, ReservationLineID: *claim.ReservationLineID, VehicleID: a.VehicleID, HolderType: "commerce-order", HolderID: order.ID, OrderLineID: &a.LineID, Current: true, CreatedAt: d.clock()}
	if claim.State == "held" {
		ordinal := int32(1)
		claim.State = "materialized"
		claim.MaterializationKind = "vin-allocation"
		claim.MaterializationRef = &association.ID
		claim.MaterializationUnitOrdinal = &ordinal
		now := d.clock()
		claim.MaterializedAt = &now
		if err := st.Supply().UpdateClaim(ctx, &claim); err != nil {
			return err
		}
	}
	if err := st.Supply().AppendClaimAllocation(ctx, association, ""); err != nil {
		return err
	}
	sources.active[claim.ID] = true
	sources.claimsByLot[chosenLot][chosenIndex] = claim
	return nil
}

func findOrderAllocationClaim(ctx context.Context, st Store, orderID, lineID string, orderLine *model.Line, v *StockVehicle, supplierID string, sources *orderClaimSources) (string, int, error) {
	// Reuse a previously consumed claim before consuming a held unit. Its
	// first materialization reference and ordinal remain immutable.
	for pass := 0; pass < 2; pass++ {
		for _, lotID := range sources.lotIDs {
			chosen, err := findOrderAllocationClaimInLot(ctx, st, orderID, lineID, orderLine, v, supplierID, sources, lotID, pass)
			if err != nil {
				return "", -1, err
			}
			if chosen >= 0 {
				return lotID, chosen, nil
			}
		}
	}
	return "", -1, apperr.New(apperr.ErrConflict, "future_obligation_exhausted", "no matching order-owned future unit remains")
}

func findOrderAllocationClaimInLot(ctx context.Context, st Store, orderID, lineID string, orderLine *model.Line, v *StockVehicle, supplierID string, sources *orderClaimSources, lotID string, pass int) (int, error) {
	for i, c := range sources.claimsByLot[lotID] {
		if c.OrderID == nil || *c.OrderID != orderID || c.OrderLineID == nil || *c.OrderLineID != lineID || c.ReservationLineID == nil {
			continue
		}
		line, ok := sources.lines[*c.ReservationLineID]
		if !ok || orderLine.OfferLineID != line.LineID || line.SupplyLotID == nil || *line.SupplyLotID != lotID || !vehicleMatchesLine(v, line) {
			continue
		}
		lot, err := st.Supply().Lot(ctx, lotID)
		if err != nil {
			return -1, err
		}
		if !sameLotSource(lot, supplierID, line) {
			return -1, apperr.New(apperr.ErrConflict, "supply_source_changed", "future source has changed")
		}
		if pass == 0 && c.State == "materialized" && c.MaterializationKind == "vin-allocation" && !sources.active[c.ID] || pass == 1 && c.State == "held" {
			return i, nil
		}
	}
	return -1, nil
}

func materializeReceiptRowLocked(ctx context.Context, st Store, r *model.Reservation, order *model.Order, row *model.ShipmentLine, orderLines map[string]model.Line, sources map[string]ReservationSnapshotLine, claimsByLot map[string][]model.SupplyClaim, at time.Time) error {
	line, ok := orderLines[row.LineID]
	if !ok || row.OrderID != order.ID || uuid.Validate(row.ID) != nil || uuid.Validate(row.ReceiptBatchID) != nil || row.Quantity < 1 {
		return apperr.New(apperr.ErrConflict, "receipt_source_missing", "receipt has no valid saved order line or batch")
	}
	source, ok := sources[line.OfferLineID]
	if !ok || source.SupplyLotID == nil || source.LineID != line.OfferLineID || source.ModelID != line.ModelID || source.ModelSpecificationVersion != line.ModelSpecificationVersion || source.ExteriorColor != line.ExteriorColor || source.InteriorColor != line.InteriorColor || source.Quantity != line.Quantity {
		return apperr.New(apperr.ErrConflict, "receipt_source_changed", "receipt line does not match the accepted future source")
	}
	lot, err := st.Supply().Lot(ctx, *source.SupplyLotID)
	if err != nil {
		return err
	}
	if !sameLotSource(lot, r.SupplierCompanyID, source) {
		return apperr.New(apperr.ErrConflict, "supply_source_changed", "future source has changed")
	}
	claims := claimsByLot[lot.ID]
	for unit := 1; unit <= row.Quantity; unit++ {
		chosen := -1
		for i, claim := range claims {
			if claim.State == "held" && claim.ReservationID != nil && *claim.ReservationID == r.ID && claim.ReservationLineID != nil && *claim.ReservationLineID == source.LineID && claim.OrderID != nil && *claim.OrderID == order.ID && claim.OrderLineID != nil && *claim.OrderLineID == row.LineID {
				chosen = i
				break
			}
		}
		if chosen < 0 {
			return apperr.New(apperr.ErrConflict, "future_obligation_exhausted", "no matching order-owned future unit remains")
		}
		claim := claims[chosen]
		ordinal := int32(unit)
		claim.State = "materialized"
		claim.MaterializationKind = "quantity-receipt"
		claim.MaterializationRef = &row.ID
		claim.MaterializationUnitOrdinal = &ordinal
		claim.MaterializedAt = &at
		if err := st.Supply().UpdateClaim(ctx, &claim); err != nil {
			return err
		}
		claims[chosen] = claim
	}
	claimsByLot[lot.ID] = claims
	return nil
}

func releaseHeldOrderClaims(ctx context.Context, st Store, orderID, reservationID, lotID string) error {
	claims, err := st.Supply().ClaimsForUpdate(ctx, lotID)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if claim.OrderID == nil || *claim.OrderID != orderID || claim.ReservationID == nil || *claim.ReservationID != reservationID {
			continue
		}
		if claim.State == "held" {
			claim.State = "released"
			if err := st.Supply().UpdateClaim(ctx, &claim); err != nil {
				return err
			}
		}
	}
	return nil
}
