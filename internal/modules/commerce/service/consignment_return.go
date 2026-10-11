package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

type ConsignmentReturn struct{ Deps }

func NewConsignmentReturn(d Deps) *ConsignmentReturn { return &ConsignmentReturn{d} }

type ReturnRequestInput struct {
	DestinationWarehouseID string `json:"destinationWarehouseId"`
	Reason string `json:"reason"`
}

type ReturnDecisionInput struct {
	MovedAt time.Time `json:"movedAt"`
	Reason string `json:"reason"`
}

func returnActor(p *auth.Principal, permission string) error {
	if p == nil { return apperr.ErrUnauthenticated }
	if err := p.Allow(permission); err != nil { return err }
	if uuid.Validate(p.CompanyID) != nil || uuid.Validate(p.UserID) != nil { return apperr.ErrForbidden }
	return nil
}

// lockReturnUnit is the same live VIN row locked by retail before any stock
// operation. The agreement and accepted order remain the commercial authority.
func (s *ConsignmentReturn) lockReturnUnit(ctx context.Context, st Store, vehicleID string) (*model.ConsignmentUnit, error) {
	u, err := st.Consignments().LockUnitByVehicle(ctx, vehicleID)
	if err != nil { return nil, err }
	if u == nil || u.VehicleID != vehicleID || u.Status != model.ConsignmentHeld ||
		u.HandedOverAt == nil || u.HandoverFactID == nil || u.HandoverOperationRef == nil || u.HandedOverBy == nil ||
		u.Version < 1 || u.SupplierCompanyID == u.DealerCompanyID || u.VIN == "" || u.OrderLineID == "" {
		return nil, apperr.New(apperr.ErrConflict, "consignment_not_held", "VIN is not held under a live consignment")
	}
	a, err := st.Consignments().Agreement(ctx, u.AgreementID)
	if err != nil { return nil, err }
	if a.OrderID != u.OrderID || a.SupplierCompanyID != u.SupplierCompanyID || a.DealerCompanyID != u.DealerCompanyID ||
		a.ReservationID == "" || a.BookingDigest == "" || a.ContinuationDigest == "" || a.ContinuationRevision < 1 {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "agreement no longer matches the VIN")
	}
	o, err := st.Deals().Order(ctx, u.SupplierCompanyID, u.OrderID)
	if err != nil { return nil, err }
	terms := o.DecodeTerms()
	provenance := terms.Reservation
	if o.Source != "reservation" || o.SupplierCompanyID != u.SupplierCompanyID || o.BuyerCompanyID != u.DealerCompanyID ||
		provenance == nil || provenance.Mode != "consignment" || provenance.ReservationID != a.ReservationID ||
		provenance.BookingDigest != a.BookingDigest || provenance.ContinuationDigest != a.ContinuationDigest ||
		provenance.ContinuationRevision != a.ContinuationRevision {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "accepted order no longer matches the VIN")
	}
	found := false
	for _, line := range terms.Lines { if line.LineID == u.OrderLineID { found = true; break } }
	if !found { return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "VIN has no accepted order line") }
	return u, nil
}

// Request records intent only. The row lock and pending association exclude a
// retail hold on this VIN without affecting another unit of the same agreement.
func (s *ConsignmentReturn) Request(ctx context.Context, p *auth.Principal, vehicleID string, expected int64, in ReturnRequestInput) (*model.ConsignmentReturn, int64, error) {
	if err := returnActor(p, model.PermTrade); err != nil { return nil, 0, err }
	if uuid.Validate(vehicleID) != nil { return nil, 0, apperr.ErrNotFound }
	if uuid.Validate(in.DestinationWarehouseID) != nil { return nil, 0, apperr.FieldError("destinationWarehouseId", "valid warehouse ID required") }
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Reason == "" { return nil, 0, apperr.FieldError("reason", "reason is required") }
	var result *model.ConsignmentReturn
	var unitRevision int64
	err := s.store.InTx(ctx, func(st Store) error {
		u, err := s.lockReturnUnit(ctx, st, vehicleID)
		if err != nil { return err }
		if u.SupplierCompanyID != p.CompanyID { return apperr.ErrNotFound }
		if u.Version != expected { return apperr.ErrStale }
		if u.CurrentRetailDealID != nil || u.PendingReturnID != nil {
			return apperr.New(apperr.ErrConflict, "consignment_return_unavailable", "VIN has a retail or return decision in progress")
		}
		bound := st.Bind(ctx)
		if err := s.stock.OwnsWarehouse(bound, p.CompanyID, in.DestinationWarehouseID); err != nil { return err }
		now, id := s.clock(), uuid.NewString()
		row := &model.ConsignmentReturn{ID: id, UnitID: u.ID, AgreementID: u.AgreementID, OrderID: u.OrderID,
			VehicleID: u.VehicleID, SupplierCompanyID: u.SupplierCompanyID, DealerCompanyID: u.DealerCompanyID,
			DestinationWarehouseID: in.DestinationWarehouseID, Reason: in.Reason, Status: model.ConsignmentReturnPending,
			RequestedBy: p.UserID, RequestedAt: now, Version: 1, UpdatedAt: now}
		if err := st.ConsignmentReturns().Create(ctx, row); err != nil { return err }
		version := u.Version
		u.PendingReturnID = &id
		u.UpdatedAt = now
		if err := st.Consignments().UpdateUnit(ctx, u, version); err != nil { return err }
		if err := s.event(ctx, st, p, "consignment.return_requested", "consignment_return", id, in.Reason,
			map[string]any{"unitId": u.ID, "orderId": u.OrderID, "vehicleId": vehicleID, "destinationWarehouseId": in.DestinationWarehouseID}); err != nil { return err }
		result, unitRevision = row, u.Version
		return nil
	})
	return result, unitRevision, err
}

func (s *ConsignmentReturn) Get(ctx context.Context, p *auth.Principal, id string) (*model.ConsignmentReturn, error) {
	if err := returnActor(p, model.PermRead); err != nil { return nil, err }
	if uuid.Validate(id) != nil { return nil, apperr.ErrNotFound }
	row, err := s.store.ConsignmentReturns().Get(ctx, id)
	if err != nil { return nil, err }
	if row.SupplierCompanyID != p.CompanyID && row.DealerCompanyID != p.CompanyID { return nil, apperr.ErrNotFound }
	return row, nil
}

func (s *ConsignmentReturn) List(ctx context.Context, p *auth.Principal, orderID string, limit, offset int) ([]model.ConsignmentReturn, error) {
	if err := returnActor(p, model.PermRead); err != nil { return nil, err }
	if orderID != "" && uuid.Validate(orderID) != nil { return nil, apperr.FieldError("orderId", "valid order ID required") }
	if limit < 0 || limit > 100 || offset < 0 { return nil, apperr.FieldError("pagination", "invalid limit or offset") }
	if limit == 0 { limit = 50 }
	return s.store.ConsignmentReturns().List(ctx, p.CompanyID, orderID, limit, offset)
}

// Decide binds the physical return and recorded decision in one transaction.
// A failed capacity, custody, hold, CAS or audit write rolls the movement back.
func (s *ConsignmentReturn) Decide(ctx context.Context, p *auth.Principal, id string, expected int64, confirm bool, in ReturnDecisionInput) (*model.ConsignmentReturn, error) {
	if err := returnActor(p, model.PermTrade); err != nil { return nil, err }
	if uuid.Validate(id) != nil { return nil, apperr.ErrNotFound }
	if !confirm {
		in.Reason = strings.TrimSpace(in.Reason)
		if in.Reason == "" { return nil, apperr.FieldError("reason", "rejection reason is required") }
	} else if in.MovedAt.IsZero() { return nil, apperr.FieldError("movedAt", "actual movement time is required") }
	lookup, err := s.store.ConsignmentReturns().Get(ctx, id)
	if err != nil { return nil, err }
	if lookup.DealerCompanyID != p.CompanyID { return nil, apperr.ErrNotFound }
	var result *model.ConsignmentReturn
	err = s.store.InTx(ctx, func(st Store) error {
		u, err := s.lockReturnUnit(ctx, st, lookup.VehicleID)
		if err != nil { return err }
		row, err := st.ConsignmentReturns().Lock(ctx, id)
		if err != nil { return err }
		if row.DealerCompanyID != p.CompanyID || row.SupplierCompanyID != u.SupplierCompanyID || row.AgreementID != u.AgreementID || row.OrderID != u.OrderID || row.UnitID != u.ID || row.VehicleID != u.VehicleID { return apperr.ErrNotFound }
		if row.Version != expected { return apperr.ErrStale }
		if row.Status != model.ConsignmentReturnPending || u.PendingReturnID == nil || *u.PendingReturnID != row.ID || u.CurrentRetailDealID != nil {
			return apperr.New(apperr.ErrConflict, "consignment_return_unavailable", "return request is no longer pending for this VIN")
		}
		now := s.clock()
		if confirm {
			bound := st.Bind(ctx)
			if err := s.stock.OwnsWarehouse(bound, row.SupplierCompanyID, row.DestinationWarehouseID); err != nil { return err }
			factID, err := s.stock.ReturnCustody(bound, ReturnCustodyMove{OwnerCompanyID: row.SupplierCompanyID,
				SourceCustodianCompanyID: row.DealerCompanyID, DestinationWarehouseID: row.DestinationWarehouseID,
				AgreementID: row.AgreementID, RequestID: row.ID, ActorUserID: p.UserID, VehicleID: row.VehicleID, At: in.MovedAt})
			if err != nil { return err }
			if uuid.Validate(factID) != nil { return apperr.New(apperr.ErrConflict, "custody_fact_missing", "inventory returned no valid return fact") }
			row.Status = model.ConsignmentReturnConfirmed
			row.MovedAt, row.MovementFactID, row.MovementOperationRef = &in.MovedAt, &factID, &row.ID
			u.Status, u.ReturnedAt, u.ReturnFactID = model.ConsignmentReturned, &in.MovedAt, &factID
		} else {
			row.Status, row.DecisionReason = model.ConsignmentReturnRejected, in.Reason
		}
		row.DecidedBy, row.DecidedAt, row.UpdatedAt = &p.UserID, &now, now
		u.PendingReturnID, u.UpdatedAt = nil, now
		rowVersion, unitVersion := row.Version, u.Version
		if err := st.ConsignmentReturns().Decide(ctx, row, rowVersion); err != nil { return err }
		if err := st.Consignments().UpdateUnit(ctx, u, unitVersion); err != nil { return err }
		eventType := "consignment.return_rejected"
		if confirm { eventType = "consignment.return_confirmed" }
		if err := s.event(ctx, st, p, eventType, "consignment_return", row.ID, row.DecisionReason,
			map[string]any{"unitId": u.ID, "vehicleId": u.VehicleID, "movementFactId": row.MovementFactID}); err != nil { return err }
		result = row
		return nil
	})
	return result, err
}
