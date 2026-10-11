package service

import (
	"cmp"
	"context"
	"errors"
	"github.com/google/uuid"
	"maps"
	"slices"
	"testing"
	"time"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

type conversionConsignments struct {
	ConsignmentRepository
	s *conversionStore
}

func (r conversionConsignments) CreateAgreement(_ context.Context, a *model.ConsignmentAgreement) error {
	if r.s.failAgreement {
		return errors.New("agreement write failed")
	}
	for _, old := range r.s.agreements {
		if old.OrderID == a.OrderID || old.ReservationID == a.ReservationID {
			return apperr.ErrConflict
		}
	}
	r.s.agreements[a.ID] = *a
	return nil
}
func (r conversionConsignments) AgreementByOrder(_ context.Context, orderID string) (*model.ConsignmentAgreement, error) {
	for _, a := range r.s.agreements {
		if a.OrderID == orderID {
			return &a, nil
		}
	}
	return nil, apperr.ErrNotFound
}
func (r conversionConsignments) Agreement(_ context.Context, id string) (*model.ConsignmentAgreement, error) {
	a, ok := r.s.agreements[id]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	return &a, nil
}
func (r conversionConsignments) CreateUnit(_ context.Context, u *model.ConsignmentUnit) error {
	if r.s.failUnit {
		return errors.New("unit write failed")
	}
	for _, old := range r.s.units {
		if old.AgreementID == u.AgreementID && old.VehicleID == u.VehicleID {
			return apperr.ErrConflict
		}
		if old.VehicleID == u.VehicleID && (old.Status == model.ConsignmentPrepared || old.Status == model.ConsignmentHeld) {
			return apperr.ErrConflict
		}
	}
	r.s.units[u.ID] = *u
	return nil
}
func (r conversionConsignments) Unit(_ context.Context, agreementID, vehicleID string) (*model.ConsignmentUnit, error) {
	for _, u := range r.s.units {
		if u.AgreementID == agreementID && u.VehicleID == vehicleID {
			return &u, nil
		}
	}
	return nil, apperr.ErrNotFound
}
func (r conversionConsignments) LockUnitByVehicle(_ context.Context, vehicleID string) (*model.ConsignmentUnit, error) {
	for _, u := range r.s.units {
		if u.VehicleID == vehicleID && (u.Status == model.ConsignmentPrepared || u.Status == model.ConsignmentHeld) {
			return &u, nil
		}
	}
	return nil, apperr.ErrNotFound
}
func (r conversionConsignments) UpdateUnit(_ context.Context, u *model.ConsignmentUnit, expected int64) error {
	old, ok := r.s.units[u.ID]
	if !ok {
		return apperr.ErrNotFound
	}
	if old.Version != expected {
		return apperr.ErrStale
	}
	u.Version = expected + 1
	r.s.units[u.ID] = *u
	return nil
}
func (r conversionConsignments) Units(_ context.Context, agreementID string) ([]model.ConsignmentUnit, error) {
	out := []model.ConsignmentUnit{}
	for _, u := range r.s.units {
		if u.AgreementID == agreementID {
			out = append(out, u)
		}
	}
	return out, nil
}
func (r conversionConsignments) RetailCandidates(_ context.Context, dealerID string, limit, offset int) ([]model.ConsignmentUnit, error) {
	out := []model.ConsignmentUnit{}
	for _, u := range r.s.units {
		if u.DealerCompanyID == dealerID && u.Status == model.ConsignmentHeld && u.HandedOverAt != nil && u.HandoverFactID != nil && u.HandoverOperationRef != nil && u.HandedOverBy != nil && u.CurrentRetailDealID == nil && u.PendingReturnID == nil {
			out = append(out, u)
		}
	}
	slices.SortFunc(out, func(a, b model.ConsignmentUnit) int { return cmp.Compare(a.ID, b.ID) })
	if offset >= len(out) {
		return []model.ConsignmentUnit{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func convertConsignment(t *testing.T, s *Reservation, st *conversionStore, buyer, supplier *auth.Principal, id, line string) *model.Order {
	t.Helper()
	ctx := context.Background()
	proposal := conversionProposal(st, line, false)
	proposal.Mode = "consignment"
	v, err := s.ProposeContinuation(ctx, buyer, id, st.reservations[id].Version, proposal)
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.AcceptContinuation(ctx, supplier, id, v.Reservation.Version, v.Reservation.ContinuationProposedDigest, v.Reservation.ContinuationProposedRevision)
	if err != nil {
		t.Fatal(err)
	}
	converted, err := s.Convert(ctx, buyer, id, v.Reservation.Version, v.Reservation.ContinuationAcceptedDigest, v.Reservation.ContinuationAcceptedRevision)
	if err != nil {
		t.Fatal(err)
	}
	return converted.Order
}

func TestConsignmentAgreementFutureOnlyAndParties(t *testing.T) {
	ctx := context.Background()
	s, st, buyer, supplier, id, line, ids := conversionFixture(t)
	order := convertConsignment(t, s, st, buyer, supplier, id, line)
	if len(st.agreements) != 1 || len(st.units) != 0 {
		t.Fatalf("future-only agreement=%d units=%d", len(st.agreements), len(st.units))
	}
	a, err := st.Consignments().AgreementByOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	p := order.DecodeTerms().Reservation
	if a.SupplierCompanyID != supplier.CompanyID || a.DealerCompanyID != buyer.CompanyID || a.ReservationID != id || a.ContinuationDigest != p.ContinuationDigest || a.ContinuationRevision != p.ContinuationRevision || a.BookingDigest != p.BookingDigest {
		t.Fatalf("agreement lost exact source: %+v", a)
	}
	view, err := NewConsignment(s.Deps).Get(ctx, buyer, order.ID)
	if err != nil || len(view.Units) != 0 {
		t.Fatalf("party view=%+v err=%v", view, err)
	}
	outsider := &auth.Principal{CompanyID: uuid.NewString(), Permissions: map[string]bool{model.PermRead: true}}
	if _, err := NewConsignment(s.Deps).Get(ctx, outsider, order.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("foreign view=%v", err)
	}
	if _, err := NewConsignment(s.Deps).ActiveForVehicle(ctx, buyer.CompanyID, ids[0]); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("unhanded future VIN active: %v", err)
	}
	if _, err := s.Convert(ctx, buyer, id, st.reservations[id].Version, p.ContinuationDigest, p.ContinuationRevision); err == nil || len(st.agreements) != 1 {
		t.Fatal("duplicate conversion created agreement")
	}
}

func TestConsignmentAgreementPreparedUnitAndHandover(t *testing.T) {
	ctx := context.Background()
	s, st, buyer, supplier, id, line, ids := conversionFixture(t)
	if _, err := s.Allocate(ctx, supplier, id, st.reservations[id].Version, []ReservationAllocationItem{{LineID: line, VehicleID: ids[0]}}); err != nil {
		t.Fatal(err)
	}
	order := convertConsignment(t, s, st, buyer, supplier, id, line)
	a, err := st.Consignments().AgreementByOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	u, err := st.Consignments().Unit(ctx, a.ID, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if u.Status != model.ConsignmentPrepared || u.HandedOverAt != nil || u.OrderID != order.ID || u.VIN != ids[0] {
		t.Fatalf("prepared unit=%+v", u)
	}
	if _, err := NewConsignment(s.Deps).ActiveForVehicle(ctx, buyer.CompanyID, ids[0]); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("prepared VIN active: %v", err)
	}
	if _, err := NewConsignment(s.Deps).PrepareUnitLocked(ctx, st, a, st.allocations[0]); err == nil {
		t.Fatal("duplicate VIN prepared")
	}
	if _, err := NewConsignment(s.Deps).PrepareUnitLocked(ctx, st, a, model.Allocation{ID: uuid.NewString(), OrderID: order.ID, LineID: u.OrderLineID, VehicleID: ids[1], VIN: ids[1], Status: "allocated"}); err == nil {
		t.Fatal("unallocated VIN prepared")
	}
	if _, err := NewConsignment(s.Deps).MarkHandedOverLocked(ctx, st, a.ID, ids[0], u.Version, uuid.NewString(), uuid.NewString(), supplier.UserID, time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	active, err := NewConsignment(s.Deps).ActiveForVehicle(ctx, buyer.CompanyID, ids[0])
	if err != nil || active.Status != model.ConsignmentHeld || active.HandoverFactID == nil {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if _, err := NewConsignment(s.Deps).ActiveForVehicle(ctx, supplier.CompanyID, ids[0]); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("wrong dealer active: %v", err)
	}
	if _, err := NewConsignment(s.Deps).MarkHandedOverLocked(ctx, st, a.ID, ids[0], u.Version, uuid.NewString(), uuid.NewString(), supplier.UserID, time.Now()); err == nil {
		t.Fatal("stale/repeated handover")
	}
	if order.DecodeTerms().Reservation.ContinuationDigest != a.ContinuationDigest || len(st.credits) != 0 {
		t.Fatal("handover changed accepted settlement")
	}
}

func TestConsignmentAgreementConversionRollback(t *testing.T) {
	ctx := context.Background()
	for _, failure := range []string{"agreement", "unit", "source-cas", "event"} {
		t.Run(failure, func(t *testing.T) {
			s, st, buyer, supplier, id, line, ids := conversionFixture(t)
			if _, err := s.Allocate(ctx, supplier, id, st.reservations[id].Version, []ReservationAllocationItem{{LineID: line, VehicleID: ids[0]}}); err != nil {
				t.Fatal(err)
			}
			proposal := conversionProposal(st, line, false)
			proposal.Mode = "consignment"
			v, err := s.ProposeContinuation(ctx, buyer, id, st.reservations[id].Version, proposal)
			if err != nil {
				t.Fatal(err)
			}
			v, err = s.AcceptContinuation(ctx, supplier, id, v.Reservation.Version, v.Reservation.ContinuationProposedDigest, v.Reservation.ContinuationProposedRevision)
			if err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "agreement":
				st.failAgreement = true
			case "unit":
				st.failUnit = true
			case "source-cas":
				st.failUpdate = true
			case "event":
				st.failEvent = true
			}
			if _, err := s.Convert(ctx, buyer, id, v.Reservation.Version, v.Reservation.ContinuationAcceptedDigest, v.Reservation.ContinuationAcceptedRevision); err == nil {
				t.Fatal("failed conversion succeeded")
			}
			if len(st.orders) != 0 || len(st.allocations) != 0 || len(st.agreements) != 0 || len(st.units) != 0 || st.reservations[id].OrderID != nil || st.holds[ids[0]] != id {
				t.Fatalf("partial conversion: orders=%d allocations=%d agreements=%d units=%d holder=%s", len(st.orders), len(st.allocations), len(st.agreements), len(st.units), st.holds[ids[0]])
			}
		})
	}
}

func TestConsignmentAgreementPerVINStateAndObligation(t *testing.T) {
	ctx := context.Background()
	s, st, buyer, supplier, id, line, ids := conversionFixture(t)
	if _, err := s.Allocate(ctx, supplier, id, st.reservations[id].Version, []ReservationAllocationItem{{LineID: line, VehicleID: ids[0]}, {LineID: line, VehicleID: ids[1]}}); err != nil {
		t.Fatal(err)
	}
	order := convertConsignment(t, s, st, buyer, supplier, id, line)
	a, err := st.Consignments().AgreementByOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	c := NewConsignment(s.Deps)
	at := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	for _, vehicleID := range ids[:2] {
		u, err := st.Consignments().Unit(ctx, a.ID, vehicleID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.MarkHandedOverLocked(ctx, st, a.ID, vehicleID, u.Version, uuid.NewString(), uuid.NewString(), supplier.UserID, at); err != nil {
			t.Fatal(err)
		}
	}
	first, err := c.LockUnitForDecision(ctx, st, buyer.CompanyID, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	returnID := uuid.NewString()
	first.PendingReturnID = &returnID
	first.UpdatedAt = at
	if err := st.Consignments().UpdateUnit(ctx, first, first.Version); err != nil {
		t.Fatal(err)
	}
	other, err := c.LockUnitForDecision(ctx, st, buyer.CompanyID, ids[1])
	if err != nil || other.PendingReturnID != nil {
		t.Fatalf("other VIN blocked: %+v %v", other, err)
	}
	first.PendingReturnID = nil
	first.Status = model.ConsignmentSold
	dealID := uuid.NewString()
	first.CurrentRetailDealID = &dealID
	first.RetailDeliveryRef = uuid.NewString()
	first.SoldAt = &at
	first.SoldBy = &buyer.UserID
	if err := st.Consignments().UpdateUnit(ctx, first, first.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ActiveForVehicle(ctx, buyer.CompanyID, ids[0]); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("sold VIN active: %v", err)
	}
	other.Status = model.ConsignmentReturned
	other.ReturnedAt = &at
	factID := uuid.NewString()
	other.ReturnFactID = &factID
	if err := st.Consignments().UpdateUnit(ctx, other, other.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ActiveForVehicle(ctx, buyer.CompanyID, ids[1]); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("returned VIN active: %v", err)
	}
	if a.OrderID != order.ID || order.DecodeTerms().Reservation.ContinuationDigest != a.ContinuationDigest || len(st.credits) != 0 {
		t.Fatal("unit transitions lost accepted order obligation")
	}
}

func TestConsignmentRetailCandidates(t *testing.T) {
	ctx := context.Background()
	s, st, dealer, supplier, _, _, _ := conversionFixture(t)
	dealer.Permissions = map[string]bool{"retail.read": true}
	at := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	agreement := uuid.NewString()
	operation := uuid.NewString()
	actor := supplier.UserID
	unit := func(status model.ConsignmentUnitStatus) model.ConsignmentUnit {
		fact := uuid.NewString()
		return model.ConsignmentUnit{ID: uuid.NewString(), AgreementID: agreement, OrderID: uuid.NewString(), OrderLineID: uuid.NewString(), VehicleID: uuid.NewString(), VIN: uuid.NewString(), SupplierCompanyID: supplier.CompanyID, DealerCompanyID: dealer.CompanyID, Status: status, HandoverFactID: &fact, HandoverOperationRef: &operation, HandedOverBy: &actor, HandedOverAt: &at, Version: 1}
	}
	first := unit(model.ConsignmentHeld)
	second := unit(model.ConsignmentHeld)
	st.units[first.ID] = first
	st.units[second.ID] = second
	prepared := unit(model.ConsignmentPrepared)
	st.units[prepared.ID] = prepared
	unhanded := unit(model.ConsignmentHeld)
	unhanded.HandoverFactID = nil
	st.units[unhanded.ID] = unhanded
	reserved := unit(model.ConsignmentHeld)
	deal := uuid.NewString()
	reserved.CurrentRetailDealID = &deal
	st.units[reserved.ID] = reserved
	pending := unit(model.ConsignmentHeld)
	returnID := uuid.NewString()
	pending.PendingReturnID = &returnID
	st.units[pending.ID] = pending
	sold := unit(model.ConsignmentSold)
	st.units[sold.ID] = sold
	returned := unit(model.ConsignmentReturned)
	st.units[returned.ID] = returned
	before := maps.Clone(st.units)
	c := NewConsignment(s.Deps)
	items, err := c.RetailCandidates(ctx, dealer, 1, 0)
	if err != nil || len(items) != 1 {
		t.Fatalf("first page=%+v err=%v", items, err)
	}
	next, err := c.RetailCandidates(ctx, dealer, 1, 1)
	if err != nil || len(next) != 1 || next[0].VehicleID == items[0].VehicleID {
		t.Fatalf("next page=%+v err=%v", next, err)
	}
	if !maps.EqualFunc(before, st.units, func(a, b model.ConsignmentUnit) bool {
		return a.ID == b.ID && a.CurrentRetailDealID == b.CurrentRetailDealID && a.PendingReturnID == b.PendingReturnID
	}) {
		t.Fatal("candidate read mutated authority")
	}
	supplier.Permissions = map[string]bool{"retail.read": true}
	items, err = c.RetailCandidates(ctx, supplier, 50, 0)
	if err != nil || len(items) != 0 {
		t.Fatalf("supplier candidates=%+v err=%v", items, err)
	}
	without := *dealer
	without.Permissions = map[string]bool{}
	if _, err := c.RetailCandidates(ctx, &without, 50, 0); !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("missing retail.read=%v", err)
	}
}

type handoverTestStore struct {
	*conversionStore
	shipments                                                                            map[string]model.Shipment
	physical                                                                             map[string]string
	invoices                                                                             []model.Invoice
	invoiceEvidence                                                                      map[string][]model.Evidence
	failShipment, failStatus, failShipmentUpdate, failCustody, failTransfer, failDeliver bool
	failAgreementLookup, failUnitUpdate, failCapacity                                    bool
	custodyCalls, transferCalls, deliverCalls                                            int
}
type handoverTxKey struct{}

func (s *handoverTestStore) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, handoverTxKey{}, true)
}
func (s *handoverTestStore) InTx(ctx context.Context, fn func(Store) error) error {
	shipments, physical := maps.Clone(s.shipments), maps.Clone(s.physical)
	custody, transfer, deliver := s.custodyCalls, s.transferCalls, s.deliverCalls
	err := s.conversionStore.InTx(ctx, func(Store) error { return fn(s) })
	if err != nil {
		s.shipments, s.physical = shipments, physical
		s.custodyCalls, s.transferCalls, s.deliverCalls = custody, transfer, deliver
	}
	return err
}
func (s *handoverTestStore) Fulfilment() FulfilmentRepository {
	return handoverFulfilment{conversionFulfilment: conversionFulfilment{s: s.conversionStore}, s: s}
}
func (s *handoverTestStore) Invoices() InvoiceRepository {
	return handoverInvoices{InvoiceRepository: conversionInvoices{}, s: s}
}
func (s *handoverTestStore) Consignments() ConsignmentRepository {
	return handoverConsignments{conversionConsignments: conversionConsignments{s: s.conversionStore}, s: s}
}

type handoverConsignments struct {
	conversionConsignments
	s *handoverTestStore
}

func (r handoverConsignments) AgreementByOrder(ctx context.Context, id string) (*model.ConsignmentAgreement, error) {
	if r.s.failAgreementLookup {
		return nil, errors.New("agreement read failed")
	}
	return r.conversionConsignments.AgreementByOrder(ctx, id)
}
func (r handoverConsignments) UpdateUnit(ctx context.Context, u *model.ConsignmentUnit, expected int64) error {
	if r.s.failUnitUpdate {
		return errors.New("unit CAS failed")
	}
	return r.conversionConsignments.UpdateUnit(ctx, u, expected)
}

type handoverFulfilment struct {
	conversionFulfilment
	s *handoverTestStore
}

func (r handoverFulfilment) CreateShipment(_ context.Context, sh *model.Shipment) error {
	if r.s.failShipment {
		return errors.New("shipment failed")
	}
	r.s.shipments[sh.ID] = *sh
	return nil
}
func (r handoverFulfilment) Shipment(_ context.Context, id string) (*model.Shipment, error) {
	sh, ok := r.s.shipments[id]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	return &sh, nil
}
func (r handoverFulfilment) UpdateShipment(_ context.Context, sh *model.Shipment, expected int64) error {
	if r.s.failShipmentUpdate {
		return errors.New("shipment CAS failed")
	}
	old, ok := r.s.shipments[sh.ID]
	if !ok || old.Version != expected {
		return apperr.ErrStale
	}
	sh.Version++
	r.s.shipments[sh.ID] = *sh
	return nil
}
func (r handoverFulfilment) SetAllocationStatus(_ context.Context, orderID string, ids []string, status string, shipmentID *string) error {
	if r.s.failStatus {
		return errors.New("allocation status failed")
	}
	for i := range r.s.allocations {
		a := &r.s.allocations[i]
		if a.OrderID == orderID && slices.Contains(ids, a.VehicleID) {
			a.Status = status
			a.ShipmentID = shipmentID
		}
	}
	return nil
}
func (r handoverFulfilment) Milestones(context.Context, string) ([]model.Milestone, error) {
	return nil, nil
}
func (r handoverFulfilment) AddShipmentLines(context.Context, []model.ShipmentLine) error { return nil }

type handoverInvoices struct {
	InvoiceRepository
	s *handoverTestStore
}

func (r handoverInvoices) ForOrder(_ context.Context, orderID string) ([]model.Invoice, error) {
	var out []model.Invoice
	for _, i := range r.s.invoices {
		if i.OrderID == orderID {
			out = append(out, i)
		}
	}
	return out, nil
}
func (r handoverInvoices) Evidence(_ context.Context, id string) ([]model.Evidence, error) {
	return slices.Clone(r.s.invoiceEvidence[id]), nil
}

type handoverStock struct {
	conversionStock
	s *handoverTestStore
}
type handoverCatalog struct{ line model.Line }

func (c handoverCatalog) Model(_ context.Context, id string) (*Model, error) {
	if id != c.line.ModelID {
		return nil, apperr.ErrNotFound
	}
	return &Model{ID: id, CurrentSpecificationVersion: c.line.ModelSpecificationVersion, Specifications: []ModelSpecification{{Version: c.line.ModelSpecificationVersion, ExteriorColors: []string{c.line.ExteriorColor}, InteriorColors: []string{c.line.InteriorColor}}}}, nil
}
func (x handoverStock) Custody(ctx context.Context, m CustodyMove) (string, error) {
	if ctx.Value(handoverTxKey{}) != true {
		return "", errors.New("custody without bound transaction")
	}
	if x.s.failCustody {
		return "", errors.New("custody failed")
	}
	if x.s.failCapacity {
		return "", apperr.New(apperr.ErrConflict, "capacity_exceeded", "destination full")
	}
	if x.s.holds[m.VehicleID] != m.OrderID || x.s.physical[m.VehicleID] != m.SourceCustodianCompanyID || m.OwnerCompanyID != m.SourceCustodianCompanyID {
		return "", apperr.ErrConflict
	}
	x.s.physical[m.VehicleID] = m.DestinationCompanyID
	delete(x.s.holds, m.VehicleID)
	x.s.custodyCalls++
	return uuid.NewString(), nil
}
func (x handoverStock) Transfer(ctx context.Context, orderID string, ids []string, toCompanyID, toWarehouseID, actorID string, at time.Time) error {
	if ctx.Value(handoverTxKey{}) != true {
		return errors.New("transfer without bound transaction")
	}
	if x.s.failTransfer {
		return errors.New("transfer failed")
	}
	for _, id := range ids {
		if x.s.holds[id] != orderID {
			return apperr.ErrConflict
		}
		x.s.physical[id] = toCompanyID
		delete(x.s.holds, id)
	}
	x.s.transferCalls++
	return nil
}
func (x handoverStock) Deliver(ctx context.Context, d Delivery) (string, error) {
	if ctx.Value(handoverTxKey{}) != true {
		return "", errors.New("delivery without bound transaction")
	}
	if x.s.failDeliver {
		return "", errors.New("delivery failed")
	}
	x.s.deliverCalls++
	return uuid.NewString(), nil
}

func handoverConsignmentFixture(t *testing.T, preallocate bool) (*Fulfilment, *handoverTestStore, *auth.Principal, *auth.Principal, *model.Order, []string) {
	t.Helper()
	ctx := context.Background()
	reservation, base, buyer, supplier, id, line, ids := conversionFixture(t)
	if preallocate {
		if _, err := reservation.Allocate(ctx, supplier, id, base.reservations[id].Version, []ReservationAllocationItem{{LineID: line, VehicleID: ids[0]}}); err != nil {
			t.Fatal(err)
		}
	}
	order := convertConsignment(t, reservation, base, buyer, supplier, id, line)
	warehouse := uuid.NewString()
	order.ReceivingWarehouseID = &warehouse
	base.orders[order.ID] = *order
	st := &handoverTestStore{conversionStore: base, shipments: map[string]model.Shipment{}, physical: map[string]string{}, invoiceEvidence: map[string][]model.Evidence{}}
	for _, id := range ids {
		st.physical[id] = supplier.CompanyID
	}
	stock := handoverStock{conversionStock: conversionStock{reservationTestStock: reservationTestStock{store: base.reservationTestStore, vehicles: reservation.reservationStock.(conversionStock).vehicles, owners: reservation.reservationStock.(conversionStock).owners}}, s: st}
	f := NewFulfilment(reservation.Deps)
	f.Deps.store = st
	f.Deps.stock = stock
	return f, st, buyer, supplier, order, ids
}

func TestConsignmentHandoverAllocatedShipAndLaterReceipt(t *testing.T) {
	ctx := context.Background()
	for _, immediate := range []bool{true, false} {
		t.Run(map[bool]string{true: "immediate", false: "later-receipt"}[immediate], func(t *testing.T) {
			f, st, buyer, supplier, o, ids := handoverConsignmentFixture(t, true)
			if !immediate {
				o.ReceivingWarehouseID = nil
				st.orders[o.ID] = *o
			}
			sh, err := f.Ship(ctx, supplier, o.ID, o.Version, ShipmentInput{VehicleIDs: []string{ids[0]}, Route: "local"})
			if err != nil {
				t.Fatal(err)
			}
			if !immediate {
				if st.custodyCalls != 0 || sh.Status != "in-transit" {
					t.Fatalf("premature custody=%d shipment=%s", st.custodyCalls, sh.Status)
				}
				_, err = f.DecideReceipt(ctx, buyer, sh.ID, sh.Version, ReceiptDecision{Decision: "accept", VehicleIDs: []string{ids[0]}, WarehouseID: uuid.NewString()})
				if err != nil {
					t.Fatal(err)
				}
			}
			a, err := st.Consignments().AgreementByOrder(ctx, o.ID)
			if err != nil {
				t.Fatal(err)
			}
			u, err := st.Consignments().Unit(ctx, a.ID, ids[0])
			if err != nil {
				t.Fatal(err)
			}
			if st.custodyCalls != 1 || st.transferCalls != 0 || u.Status != model.ConsignmentHeld || u.HandoverFactID == nil || u.HandoverOperationRef == nil || st.physical[ids[0]] != buyer.CompanyID || st.holds[ids[0]] != "" || st.orders[o.ID].Status != model.OrderFulfilling {
				t.Fatalf("custody=%d transfer=%d unit=%+v physical=%s order=%s", st.custodyCalls, st.transferCalls, u, st.physical[ids[0]], st.orders[o.ID].Status)
			}
			if _, err := f.Ship(ctx, supplier, o.ID, st.orders[o.ID].Version, ShipmentInput{VehicleIDs: []string{ids[0]}, Route: "local"}); err == nil {
				t.Fatal("repeated handover accepted")
			}
		})
	}
}

func TestConsignmentHandoverFutureVINAndQuantityGuard(t *testing.T) {
	ctx := context.Background()
	f, st, _, supplier, o, ids := handoverConsignmentFixture(t, false)
	if _, err := f.Ship(ctx, supplier, o.ID, o.Version, ShipmentInput{Lines: []ShipmentLineInput{{OrderLineID: o.DecodeTerms().Lines[0].LineID, Quantity: 1, VINs: []string{"VIN"}}}, Route: "local"}); err == nil || st.deliverCalls != 0 || len(st.shipments) != 0 {
		t.Fatalf("quantity consignment accepted: %v", err)
	}
	updated, err := f.Allocate(ctx, supplier, o.ID, o.Version, []AllocationItem{{OrderLineID: o.DecodeTerms().Lines[0].LineID, VehicleID: ids[0]}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.Consignments().AgreementByOrder(ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	u, err := st.Consignments().Unit(ctx, a.ID, ids[0])
	if err != nil || u.Status != model.ConsignmentPrepared {
		t.Fatalf("later prepared=%+v err=%v", u, err)
	}
	if _, err := f.Ship(ctx, supplier, o.ID, updated.Version, ShipmentInput{VehicleIDs: []string{ids[0]}, Route: "local"}); err != nil {
		t.Fatal(err)
	}
	if st.custodyCalls != 1 {
		t.Fatalf("later custody=%d", st.custodyCalls)
	}
}

func TestConsignmentHandoverRollback(t *testing.T) {
	for _, failure := range []string{"custody", "capacity", "agreement", "unit-update", "shipment", "status", "order", "event"} {
		t.Run(failure, func(t *testing.T) {
			f, st, _, supplier, o, ids := handoverConsignmentFixture(t, true)
			switch failure {
			case "custody":
				st.failCustody = true
			case "capacity":
				st.failCapacity = true
			case "agreement":
				st.failAgreementLookup = true
			case "unit-update":
				st.failUnitUpdate = true
			case "shipment":
				st.failShipment = true
			case "status":
				st.failStatus = true
			case "order":
				st.failOrder = true
			case "event":
				st.failEvent = true
			}
			if _, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{VehicleIDs: []string{ids[0]}, Route: "local"}); err == nil {
				t.Fatal("injected failure succeeded")
			}
			st.failAgreementLookup = false
			a, _ := st.Consignments().AgreementByOrder(context.Background(), o.ID)
			u, _ := st.Consignments().Unit(context.Background(), a.ID, ids[0])
			if u.Status != model.ConsignmentPrepared || st.physical[ids[0]] != supplier.CompanyID || st.holds[ids[0]] != o.ID || len(st.shipments) != 0 || st.custodyCalls != 0 || st.orders[o.ID].Version != o.Version {
				t.Fatalf("partial failure=%s unit=%+v physical=%s shipments=%d", failure, u, st.physical[ids[0]], len(st.shipments))
			}
		})
	}
}

func TestConsignmentHandoverReceiptRollback(t *testing.T) {
	ctx := context.Background()
	f, st, buyer, supplier, o, ids := handoverConsignmentFixture(t, true)
	o.ReceivingWarehouseID = nil
	st.orders[o.ID] = *o
	sh, err := f.Ship(ctx, supplier, o.ID, o.Version, ShipmentInput{VehicleIDs: []string{ids[0]}, Route: "local"})
	if err != nil {
		t.Fatal(err)
	}
	st.failShipmentUpdate = true
	if _, err := f.DecideReceipt(ctx, buyer, sh.ID, sh.Version, ReceiptDecision{Decision: "accept", VehicleIDs: []string{ids[0]}, WarehouseID: uuid.NewString()}); err == nil {
		t.Fatal("shipment CAS failure accepted")
	}
	a, _ := st.Consignments().AgreementByOrder(ctx, o.ID)
	u, _ := st.Consignments().Unit(ctx, a.ID, ids[0])
	if u.Status != model.ConsignmentPrepared || st.physical[ids[0]] != supplier.CompanyID || st.holds[ids[0]] != o.ID || st.custodyCalls != 0 || st.shipments[sh.ID].Status != "in-transit" {
		t.Fatal("receipt failure left physical custody or authority changed")
	}
}

func purchaseHandoverFixture(t *testing.T, partial bool, minimum string) (*Fulfilment, *handoverTestStore, *auth.Principal, *model.Order, string) {
	t.Helper()
	ctx := context.Background()
	r, base, buyer, supplier, id, line, ids := conversionFixture(t)
	row := base.reservations[id]
	snapshot, err := decodeReservationSnapshot(row.CurrentTerms)
	if err != nil {
		t.Fatal(err)
	}
	advance := money.Money{AmountMinor: "2000", Currency: "USD"}
	snapshot.Terms.Advance = &advance
	due := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	snapshot.Terms.AdvanceDueAt = &due
	row.CurrentDigest, row.CurrentTerms = reservationDigest(snapshot)
	row.AdvanceDueAt = &due
	base.reservations[id] = row
	base.evidence = []model.ReservationEvidence{{ID: uuid.NewString(), ReservationID: id, TermsDigest: row.CurrentDigest, TermsRevision: row.CurrentRevision, AmountMinor: "2000", Currency: "USD", Status: "accepted"}}
	if _, err := r.Allocate(ctx, supplier, id, row.Version, []ReservationAllocationItem{{LineID: line, VehicleID: ids[0]}}); err != nil {
		t.Fatal(err)
	}
	proposal := conversionProposal(base, line, partial)
	if minimum != "" {
		proposal.MinimumPaid = &money.Money{AmountMinor: minimum, Currency: "USD"}
		proposal.HandoverConditions = "agreed partial purchase release"
	}
	v, err := r.ProposeContinuation(ctx, buyer, id, base.reservations[id].Version, proposal)
	if err != nil {
		t.Fatal(err)
	}
	v, err = r.AcceptContinuation(ctx, supplier, id, v.Reservation.Version, v.Reservation.ContinuationProposedDigest, v.Reservation.ContinuationProposedRevision)
	if err != nil {
		t.Fatal(err)
	}
	converted, err := r.Convert(ctx, buyer, id, v.Reservation.Version, v.Reservation.ContinuationAcceptedDigest, v.Reservation.ContinuationAcceptedRevision)
	if err != nil {
		t.Fatal(err)
	}
	o := converted.Order
	warehouse := uuid.NewString()
	o.ReceivingWarehouseID = &warehouse
	base.orders[o.ID] = *o
	st := &handoverTestStore{conversionStore: base, shipments: map[string]model.Shipment{}, physical: map[string]string{ids[0]: supplier.CompanyID}, invoiceEvidence: map[string][]model.Evidence{}}
	stock := handoverStock{conversionStock: conversionStock{reservationTestStock: reservationTestStock{store: base.reservationTestStore, vehicles: r.reservationStock.(conversionStock).vehicles, owners: r.reservationStock.(conversionStock).owners}}, s: st}
	f := NewFulfilment(r.Deps)
	f.Deps.store = st
	f.Deps.stock = stock
	f.Deps.catalog = handoverCatalog{line: o.DecodeTerms().Lines[0]}
	return f, st, supplier, o, ids[0]
}

func TestConsignmentPurchaseHandoverAcceptedMoney(t *testing.T) {
	for _, tc := range []struct {
		name    string
		partial bool
		minimum string
		allowed bool
	}{{"no-agreement", false, "", false}, {"minimum-met", true, "2000", true}, {"minimum-unmet", true, "3000", false}, {"no-default-minimum", true, "", true}} {
		t.Run(tc.name, func(t *testing.T) {
			f, st, supplier, o, id := purchaseHandoverFixture(t, tc.partial, tc.minimum)
			_, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{VehicleIDs: []string{id}, Route: "local"})
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v err=%v", tc.allowed, err)
			}
			if tc.allowed {
				if st.transferCalls != 1 || st.custodyCalls != 0 || st.physical[id] != o.BuyerCompanyID {
					t.Fatalf("purchase movement transfer=%d custody=%d", st.transferCalls, st.custodyCalls)
				}
			} else if st.transferCalls != 0 || len(st.shipments) != 0 {
				t.Fatal("unpaid purchase moved")
			}
		})
	}
	// The invoice view includes assigned advance once, ordinary accepted money,
	// and no submitted claim; full settlement permits the ordinary transfer.
	f, st, supplier, o, id := purchaseHandoverFixture(t, false, "")
	invoiceID := uuid.NewString()
	credit := st.credits[o.DecodeTerms().Reservation.ReservationID]
	credit.InvoiceID = &invoiceID
	st.credits[credit.ReservationID] = credit
	st.invoices = []model.Invoice{{ID: invoiceID, OrderID: o.ID, SupplierCompanyID: o.SupplierCompanyID, BuyerCompanyID: o.BuyerCompanyID, TotalMinor: "10000", Currency: "USD", Status: "issued"}}
	st.invoiceEvidence[invoiceID] = []model.Evidence{{AmountMinor: "8000", Currency: "USD", Status: "accepted"}, {AmountMinor: "1", Currency: "USD", Status: "rejected"}}
	if _, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{VehicleIDs: []string{id}, Route: "local"}); err != nil {
		t.Fatal(err)
	}
	if st.transferCalls != 1 {
		t.Fatalf("full-paid transfer=%d", st.transferCalls)
	}
}

func TestConsignmentPurchaseHandoverQuantityAndPending(t *testing.T) {
	f, st, supplier, o, id := purchaseHandoverFixture(t, false, "")
	line := o.DecodeTerms().Lines[0].LineID
	if _, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{Lines: []ShipmentLineInput{{OrderLineID: line, Quantity: 1}}, Route: "local"}); err == nil || st.deliverCalls != 0 {
		t.Fatalf("unpaid quantity delivered: %v", err)
	}
	// A submitted invoice claim is not accepted money.
	inv := model.Invoice{ID: uuid.NewString(), OrderID: o.ID, SupplierCompanyID: o.SupplierCompanyID, BuyerCompanyID: o.BuyerCompanyID, TotalMinor: "10000", Currency: "USD", Status: "issued"}
	st.invoices = []model.Invoice{inv}
	credit := st.credits[o.DecodeTerms().Reservation.ReservationID]
	credit.InvoiceID = &inv.ID
	st.credits[credit.ReservationID] = credit
	st.invoiceEvidence[inv.ID] = []model.Evidence{{AmountMinor: "8000", Currency: "USD", Status: "submitted"}}
	if _, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{VehicleIDs: []string{id}, Route: "local"}); err == nil || st.transferCalls != 0 {
		t.Fatalf("pending claim handed over: %v", err)
	}
	// Explicit partial purchase agreement enables the existing quantity path.
	f, st, supplier, o, _ = purchaseHandoverFixture(t, true, "2000")
	line = o.DecodeTerms().Lines[0].LineID
	if _, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{Lines: []ShipmentLineInput{{OrderLineID: line, Quantity: 1}}, Route: "local"}); err != nil {
		t.Fatal(err)
	}
	if st.deliverCalls != 1 {
		t.Fatalf("quantity deliveries=%d", st.deliverCalls)
	}
	f, st, supplier, o, _ = purchaseHandoverFixture(t, true, "2000")
	st.failDeliver = true
	line = o.DecodeTerms().Lines[0].LineID
	if _, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{Lines: []ShipmentLineInput{{OrderLineID: line, Quantity: 1}}, Route: "local"}); err == nil || st.deliverCalls != 0 || len(st.shipments) != 0 {
		t.Fatalf("failed quantity delivery changed state: %v", err)
	}
	f, st, supplier, o, id = purchaseHandoverFixture(t, true, "2000")
	st.failTransfer = true
	if _, err := f.Ship(context.Background(), supplier, o.ID, o.Version, ShipmentInput{VehicleIDs: []string{id}, Route: "local"}); err == nil || st.transferCalls != 0 || st.physical[id] != supplier.CompanyID {
		t.Fatalf("failed purchase transfer changed state: %v", err)
	}
}
