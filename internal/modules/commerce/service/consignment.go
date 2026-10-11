package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/database"
)

type Consignment struct{ Deps }

func NewConsignment(d Deps) *Consignment { return &Consignment{d} }

type ConsignmentView struct {
	Agreement model.ConsignmentAgreement
	Units     []model.ConsignmentUnit
}

// SaleAuthority is the trusted physical and settlement identity for one VIN.
// OrderID remains the accepted supplier settlement obligation after retail sale.
type SaleAuthority struct {
	UnitID, AgreementID, OrderID, OrderLineID, VehicleID, VIN string
	SupplierCompanyID, DealerCompanyID, HandoverFactID        string
	Revision                                                  int64
}

// RetailCandidate is deliberately limited to the identifier used to hydrate
// stock through inventory's existing company-scoped vehicle route.
type RetailCandidate struct{ VehicleID string }

// CreateFromConversionLocked joins Reservation.Convert's existing transaction.
// The order is the only mutable commercial terms record; this row pins its
// exact accepted source and creates no second payment agreement.
func (s *Consignment) CreateFromConversionLocked(ctx context.Context, st Store, r *model.Reservation, order *model.Order, allocations []model.Allocation) (*model.ConsignmentAgreement, error) {
	if r == nil || order == nil || order.Source != "reservation" || r.OrderID != nil ||
		r.BuyerCompanyID == r.SupplierCompanyID || r.BuyerCompanyID != order.BuyerCompanyID || r.SupplierCompanyID != order.SupplierCompanyID ||
		r.ContinuationAcceptedDigest == "" || r.ContinuationAcceptedRevision < 1 {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_invalid", "converted consignment source is invalid")
	}
	accepted, err := decodeReservationContinuation(r.ContinuationAccepted)
	if err != nil {
		return nil, err
	}
	if accepted == nil || accepted.Mode != "consignment" {
		return nil, apperr.New(apperr.ErrConflict, "consignment_not_accepted", "consignment requires mutual acceptance")
	}
	digest, _ := reservationDigest(*accepted)
	provenance := order.DecodeTerms().Reservation
	if digest != r.ContinuationAcceptedDigest || digest != r.ContinuationProposedDigest ||
		provenance == nil || provenance.Mode != "consignment" || provenance.ReservationID != r.ID ||
		provenance.BookingDigest != r.CurrentDigest || provenance.ContinuationDigest != digest ||
		provenance.ContinuationRevision != r.ContinuationAcceptedRevision {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "accepted order provenance changed")
	}
	expectedProvenance := &model.ReservationOrderTerms{ReservationID: r.ID, BookingDigest: r.CurrentDigest,
		ContinuationDigest: digest, ContinuationRevision: r.ContinuationAcceptedRevision, Mode: accepted.Mode,
		AllowPartialPurchaseHandover: accepted.AllowPartialPurchaseHandover, MinimumPaid: accepted.MinimumPaid,
		HandoverConditions: accepted.HandoverConditions, HandoverTerms: accepted.HandoverTerms}
	expectedProvenanceDigest, _ := reservationDigest(expectedProvenance)
	actualProvenanceDigest, _ := reservationDigest(provenance)
	if expectedProvenanceDigest != actualProvenanceDigest {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "accepted handover provenance changed")
	}
	expectedTerms := accepted.Terms
	expectedTerms.Reservation = expectedProvenance
	expectedDigest, _ := reservationDigest(expectedTerms)
	actualDigest, _ := reservationDigest(order.DecodeTerms())
	if expectedDigest != actualDigest {
		return nil, apperr.New(apperr.ErrConflict, "consignment_terms_changed", "accepted commercial terms changed")
	}
	now := s.clock()
	a := &model.ConsignmentAgreement{ID: uuid.NewString(), OrderID: order.ID, ReservationID: r.ID,
		SupplierCompanyID: r.SupplierCompanyID, DealerCompanyID: r.BuyerCompanyID,
		BookingDigest: r.CurrentDigest, ContinuationDigest: digest, ContinuationRevision: r.ContinuationAcceptedRevision,
		Version: 1, CreatedAt: now}
	if err := st.Consignments().CreateAgreement(ctx, a); err != nil {
		return nil, err
	}
	for _, allocation := range allocations {
		if _, err := s.PrepareUnitLocked(ctx, st, a, allocation); err != nil {
			return nil, err
		}
	}
	return a, nil
}

// PrepareUnitLocked records a concrete allocated VIN without granting custody
// or retail eligibility. P21a2 calls it for VINs allocated after conversion.
func (s *Consignment) PrepareUnitLocked(ctx context.Context, st Store, a *model.ConsignmentAgreement, allocation model.Allocation) (*model.ConsignmentUnit, error) {
	if a == nil || allocation.OrderID != a.OrderID || allocation.VehicleID == "" || allocation.VIN == "" ||
		allocation.LineID == "" || allocation.Status != "allocated" {
		return nil, apperr.New(apperr.ErrConflict, "consignment_allocation_invalid", "allocated VIN must belong to the accepted order")
	}
	order, err := st.Deals().Order(ctx, a.SupplierCompanyID, a.OrderID)
	if err != nil {
		return nil, err
	}
	provenance := order.DecodeTerms().Reservation
	if order.SupplierCompanyID != a.SupplierCompanyID || order.BuyerCompanyID != a.DealerCompanyID || provenance == nil || provenance.Mode != "consignment" || provenance.ContinuationDigest != a.ContinuationDigest || provenance.ContinuationRevision != a.ContinuationRevision {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "order no longer matches accepted consignment")
	}
	validLine := false
	for _, line := range order.DecodeTerms().Lines {
		if line.LineID == allocation.LineID {
			validLine = true
			break
		}
	}
	if !validLine {
		return nil, apperr.New(apperr.ErrConflict, "consignment_allocation_invalid", "VIN has no accepted order line")
	}
	rows, err := st.Fulfilment().Allocations(ctx, a.OrderID)
	if err != nil {
		return nil, err
	}
	validAllocation := false
	for _, row := range rows {
		if row.ID == allocation.ID && row.VehicleID == allocation.VehicleID && row.VIN == allocation.VIN && row.LineID == allocation.LineID && row.Status == "allocated" {
			validAllocation = true
			break
		}
	}
	if !validAllocation {
		return nil, apperr.New(apperr.ErrConflict, "consignment_allocation_invalid", "VIN is not a current order allocation")
	}
	if _, err := st.Consignments().Unit(ctx, a.ID, allocation.VehicleID); err == nil {
		return nil, apperr.New(apperr.ErrConflict, "consignment_unit_exists", "VIN already has consignment authority")
	} else if !errors.Is(err, apperr.ErrNotFound) {
		return nil, err
	}
	now := s.clock()
	u := &model.ConsignmentUnit{ID: uuid.NewString(), AgreementID: a.ID, OrderID: a.OrderID, OrderLineID: allocation.LineID,
		VehicleID: allocation.VehicleID, VIN: allocation.VIN, SupplierCompanyID: a.SupplierCompanyID, DealerCompanyID: a.DealerCompanyID,
		Status: model.ConsignmentPrepared, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := st.Consignments().CreateUnit(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// MarkHandedOverLocked records inventory's successful immutable fact in the
// same caller transaction. A prepared row alone never authorizes retail.
func (s *Consignment) MarkHandedOverLocked(ctx context.Context, st Store, agreementID, vehicleID string, expected int64, factID, operationRef, actorID string, at time.Time) (*model.ConsignmentUnit, error) {
	if uuid.Validate(factID) != nil || uuid.Validate(operationRef) != nil || uuid.Validate(actorID) != nil || at.IsZero() {
		return nil, apperr.FieldError("handover", "fact, operation, actor and time are required")
	}
	u, err := st.Consignments().LockUnitByVehicle(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	if u.AgreementID != agreementID {
		return nil, apperr.ErrNotFound
	}
	if u.Version != expected {
		return nil, apperr.ErrStale
	}
	if u.Status != model.ConsignmentPrepared || u.HandedOverAt != nil || u.PendingReturnID != nil || u.CurrentRetailDealID != nil {
		return nil, apperr.New(apperr.ErrConflict, "consignment_not_prepared", "VIN cannot be handed over again")
	}
	u.Status = model.ConsignmentHeld
	u.HandoverFactID = &factID
	u.HandoverOperationRef = &operationRef
	u.HandedOverBy = &actorID
	u.HandedOverAt = &at
	u.UpdatedAt = s.clock()
	if err := st.Consignments().UpdateUnit(ctx, u, expected); err != nil {
		return nil, err
	}
	return u, nil
}

// LockUnitForDecision retains the per-VIN row lock until the caller's outer
// transaction commits. P21b/P22 recheck their distinct policy after this call.
func (s *Consignment) LockUnitForDecision(ctx context.Context, st Store, dealerID, vehicleID string) (*model.ConsignmentUnit, error) {
	u, err := st.Consignments().LockUnitByVehicle(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	if u.DealerCompanyID != dealerID || u.Status != model.ConsignmentHeld || u.HandedOverAt == nil {
		return nil, apperr.ErrNotFound
	}
	return u, nil
}

func (s *Consignment) Get(ctx context.Context, p *auth.Principal, orderID string) (*ConsignmentView, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	if uuid.Validate(orderID) != nil {
		return nil, apperr.ErrNotFound
	}
	if _, err := s.store.Deals().Order(ctx, p.CompanyID, orderID); err != nil {
		return nil, err
	}
	a, err := s.store.Consignments().AgreementByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.Consignments().Units(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	return &ConsignmentView{Agreement: *a, Units: rows}, nil
}

// ActiveForVehicle is a discovery projection. Mutations must use
// LockUnitForDecision in their own bound transaction and recheck state.
func (s *Consignment) ActiveForVehicle(ctx context.Context, dealerID, vehicleID string) (*model.ConsignmentUnit, error) {
	if uuid.Validate(dealerID) != nil || uuid.Validate(vehicleID) != nil {
		return nil, apperr.ErrNotFound
	}
	var out *model.ConsignmentUnit
	err := s.store.InTx(ctx, func(st Store) error {
		u, err := s.LockUnitForDecision(ctx, st, dealerID, vehicleID)
		if err != nil {
			return err
		}
		out = u
		return nil
	})
	return out, err
}

// RetailCandidates discovers current dealer-held consignment VINs. It creates
// no hold and does not replace the locked sale authority used by retail writes.
func (s *Consignment) RetailCandidates(ctx context.Context, p *auth.Principal, limit, offset int) ([]RetailCandidate, error) {
	if p == nil {
		return nil, apperr.ErrUnauthenticated
	}
	if err := p.Allow("retail.read"); err != nil {
		return nil, err
	}
	if p.CompanyID == "" {
		return nil, apperr.New(apperr.ErrConflict, "no_active_company", "select a company first")
	}
	if limit < 0 || limit > 100 || offset < 0 {
		return nil, apperr.FieldError("pagination", "limit must be 0-100 and offset must be zero or greater")
	}
	limit, offset = database.Page(limit, offset)
	units, err := s.store.Consignments().RetailCandidates(ctx, p.CompanyID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]RetailCandidate, 0, len(units))
	for _, u := range units {
		out = append(out, RetailCandidate{VehicleID: u.VehicleID})
	}
	return out, nil
}

// lockSaleUnit first locks the live VIN row without a dealer filter. A missing
// row is the only ordinary-stock fallback; a live row belonging to another
// dealer, or one not yet physically held, is a denied sale. The caller must
// retain the ambient outer transaction through any stock operation.
func (s *Consignment) lockSaleUnit(ctx context.Context, st Store, sellerID, vehicleID string) (*model.ConsignmentUnit, error) {
	u, err := st.Consignments().LockUnitByVehicle(ctx, vehicleID)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, apperr.New(apperr.ErrConflict, "consignment_authority_missing", "live consignment lookup returned no unit")
	}
	if u.DealerCompanyID != sellerID || u.Status != model.ConsignmentHeld || u.HandedOverAt == nil ||
		u.HandoverFactID == nil || u.HandoverOperationRef == nil || u.HandedOverBy == nil ||
		u.PendingReturnID != nil || u.SupplierCompanyID == sellerID || u.VehicleID != vehicleID || u.VIN == "" || u.OrderID == "" ||
		u.AgreementID == "" || u.OrderLineID == "" || u.Version < 1 {
		return nil, apperr.New(apperr.ErrForbidden, "consignment_sale_denied", "VIN has no live sale authority for this seller")
	}
	a, err := st.Consignments().Agreement(ctx, u.AgreementID)
	if err != nil {
		return nil, err
	}
	if a.OrderID != u.OrderID || a.SupplierCompanyID != u.SupplierCompanyID || a.DealerCompanyID != u.DealerCompanyID ||
		a.ReservationID == "" || a.BookingDigest == "" || a.ContinuationDigest == "" || a.ContinuationRevision < 1 {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "consignment agreement no longer matches the VIN")
	}
	o, err := st.Deals().Order(ctx, u.SupplierCompanyID, u.OrderID)
	if err != nil {
		return nil, err
	}
	terms := o.DecodeTerms()
	p := terms.Reservation
	if o.Source != "reservation" || o.SupplierCompanyID != u.SupplierCompanyID || o.BuyerCompanyID != u.DealerCompanyID ||
		p == nil || p.Mode != "consignment" || p.ReservationID != a.ReservationID ||
		p.BookingDigest != a.BookingDigest || p.ContinuationDigest != a.ContinuationDigest ||
		p.ContinuationRevision != a.ContinuationRevision {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "accepted order no longer matches the VIN")
	}
	lineFound := false
	for _, line := range terms.Lines {
		if line.LineID == u.OrderLineID {
			lineFound = true
			break
		}
	}
	if !lineFound {
		return nil, apperr.New(apperr.ErrConflict, "consignment_source_changed", "VIN has no accepted order line")
	}
	return u, nil
}

func saleAuthority(u *model.ConsignmentUnit) *SaleAuthority {
	return &SaleAuthority{UnitID: u.ID, AgreementID: u.AgreementID, OrderID: u.OrderID,
		OrderLineID: u.OrderLineID, VehicleID: u.VehicleID, VIN: u.VIN,
		SupplierCompanyID: u.SupplierCompanyID, DealerCompanyID: u.DealerCompanyID,
		HandoverFactID: *u.HandoverFactID, Revision: u.Version}
}

func saleUnreserved(u *model.ConsignmentUnit) error {
	if u.CurrentRetailDealID != nil {
		return apperr.New(apperr.ErrConflict, "consignment_retail_reserved", "VIN is already reserved for retail")
	}
	return nil
}

func saleForDeal(u *model.ConsignmentUnit, dealID string) error {
	if u.CurrentRetailDealID == nil || *u.CurrentRetailDealID != dealID {
		return apperr.New(apperr.ErrConflict, "consignment_retail_mismatch", "VIN is not reserved for this retail deal")
	}
	return nil
}

func (s *Consignment) saleEvent(ctx context.Context, st Store, u *model.ConsignmentUnit, actorID, eventType string) error {
	details, err := json.Marshal(map[string]string{"agreementId": u.AgreementID, "orderId": u.OrderID, "vehicleId": u.VehicleID})
	if err != nil {
		return err
	}
	return st.Events().Append(ctx, &model.Event{ID: uuid.NewString(), CompanyID: u.DealerCompanyID,
		EventType: eventType, ResourceType: "consignment_unit", ResourceID: u.ID,
		ActorUserID: actorID, OccurredAt: s.clock(), Details: details})
}

// CheckSale is the current listing/ordinary-fallback check. It returns
// (nil,nil) only when there is no live consignment VIN row at all. A caller
// creating or publishing a listing must pass its bound outer transaction.
func (s *Consignment) CheckSale(ctx context.Context, sellerID, vehicleID string) (*SaleAuthority, error) {
	if uuid.Validate(sellerID) != nil || uuid.Validate(vehicleID) != nil {
		return nil, apperr.FieldError("vehicleId", "seller and VIN identity are required")
	}
	var out *SaleAuthority
	err := s.store.InTx(ctx, func(st Store) error {
		u, err := s.lockSaleUnit(ctx, st, sellerID, vehicleID)
		if err != nil || u == nil {
			return err
		}
		if err := saleUnreserved(u); err != nil {
			return err
		}
		out = saleAuthority(u)
		return nil
	})
	return out, err
}

// ReserveSale associates exactly one retail deal with the locked live unit.
// The caller retains its outer bound transaction through physical hold creation.
func (s *Consignment) ReserveSale(ctx context.Context, sellerID, vehicleID, dealID, actorID string) (*SaleAuthority, error) {
	if uuid.Validate(sellerID) != nil || uuid.Validate(vehicleID) != nil || uuid.Validate(dealID) != nil || uuid.Validate(actorID) != nil {
		return nil, apperr.FieldError("consignment", "seller, vehicle, deal and actor IDs are required")
	}
	var out *SaleAuthority
	err := s.store.InTx(ctx, func(st Store) error {
		u, err := s.lockSaleUnit(ctx, st, sellerID, vehicleID)
		if err != nil || u == nil {
			return err
		}
		if err := saleUnreserved(u); err != nil {
			return err
		}
		expected := u.Version
		u.CurrentRetailDealID = &dealID
		u.UpdatedAt = s.clock()
		if err := st.Consignments().UpdateUnit(ctx, u, expected); err != nil {
			return err
		}
		if err := s.saleEvent(ctx, st, u, actorID, "consignment.retail_reserved"); err != nil {
			return err
		}
		out = saleAuthority(u)
		return nil
	})
	return out, err
}

// SaleForDeal rechecks the existing association before customer delivery.
func (s *Consignment) SaleForDeal(ctx context.Context, sellerID, vehicleID, dealID string) (*SaleAuthority, error) {
	if uuid.Validate(sellerID) != nil || uuid.Validate(vehicleID) != nil || uuid.Validate(dealID) != nil {
		return nil, apperr.FieldError("consignment", "seller, vehicle and deal IDs are required")
	}
	var out *SaleAuthority
	err := s.store.InTx(ctx, func(st Store) error {
		u, err := s.lockSaleUnit(ctx, st, sellerID, vehicleID)
		if err != nil {
			return err
		}
		if u == nil {
			return apperr.ErrNotFound
		}
		if err := saleForDeal(u, dealID); err != nil {
			return err
		}
		out = saleAuthority(u)
		return nil
	})
	return out, err
}

// CancelSale clears only this deal's current association. Its retail record,
// commerce event and supplier settlement obligation remain durable.
func (s *Consignment) CancelSale(ctx context.Context, sellerID, vehicleID, dealID, actorID string) error {
	if uuid.Validate(sellerID) != nil || uuid.Validate(vehicleID) != nil || uuid.Validate(dealID) != nil || uuid.Validate(actorID) != nil {
		return apperr.FieldError("consignment", "seller, vehicle, deal and actor IDs are required")
	}
	return s.store.InTx(ctx, func(st Store) error {
		u, err := s.lockSaleUnit(ctx, st, sellerID, vehicleID)
		if err != nil {
			return err
		}
		if u == nil {
			return apperr.ErrNotFound
		}
		if err := saleForDeal(u, dealID); err != nil {
			return err
		}
		expected := u.Version
		u.CurrentRetailDealID = nil
		u.UpdatedAt = s.clock()
		if err := st.Consignments().UpdateUnit(ctx, u, expected); err != nil {
			return err
		}
		return s.saleEvent(ctx, st, u, actorID, "consignment.retail_cancelled")
	})
}

// CompleteSale records the real customer delivery, never a listing, contract,
// payment or reservation. The inventory delivery and retail deal transition
// must be in the same outer bound transaction as this update.
func (s *Consignment) CompleteSale(ctx context.Context, sellerID, vehicleID, dealID, deliveryRef, actorID string, at time.Time) error {
	if uuid.Validate(sellerID) != nil || uuid.Validate(vehicleID) != nil || uuid.Validate(dealID) != nil || uuid.Validate(actorID) != nil || deliveryRef == "" || at.IsZero() {
		return apperr.FieldError("consignment", "seller, vehicle, deal, delivery reference, actor and time are required")
	}
	return s.store.InTx(ctx, func(st Store) error {
		u, err := s.lockSaleUnit(ctx, st, sellerID, vehicleID)
		if err != nil {
			return err
		}
		if u == nil {
			return apperr.ErrNotFound
		}
		if err := saleForDeal(u, dealID); err != nil {
			return err
		}
		expected := u.Version
		u.Status = model.ConsignmentSold
		u.RetailDeliveryRef = deliveryRef
		u.SoldAt = &at
		u.SoldBy = &actorID
		u.UpdatedAt = s.clock()
		if err := st.Consignments().UpdateUnit(ctx, u, expected); err != nil {
			return err
		}
		return s.saleEvent(ctx, st, u, actorID, "consignment.sold")
	})
}

// physicalOrder locks a converted order's source before a physical decision and
// re-reads the order after that lock. Ordinary orders retain their existing path.
func (s *Fulfilment) physicalOrder(ctx context.Context, st Store, o *model.Order) (*model.Order, string, *model.ReservationCredit, error) {
	if o.Source != "reservation" {
		return o, "", nil, nil
	}
	credit, err := NewReservationPayment(s.Deps).CreditForOrderLocked(ctx, st, o)
	if err != nil {
		return nil, "", nil, err
	}
	current, err := st.Deals().Order(ctx, o.SupplierCompanyID, o.ID)
	if err != nil {
		return nil, "", nil, err
	}
	terms := current.DecodeTerms().Reservation
	source, _, err := st.Reservations().ByOrderForUpdate(ctx, current.ID)
	if err != nil {
		return nil, "", nil, err
	}
	if current.Source != "reservation" || terms == nil || terms.ReservationID == "" ||
		terms.BookingDigest == "" || terms.ContinuationDigest == "" || terms.ContinuationRevision < 1 ||
		terms.ReservationID != source.ID || terms.BookingDigest != source.CurrentDigest ||
		terms.ContinuationDigest != source.ContinuationAcceptedDigest || terms.ContinuationRevision != source.ContinuationAcceptedRevision ||
		(terms.Mode != "purchase" && terms.Mode != "consignment") {
		return nil, "", nil, apperr.New(apperr.ErrConflict, "handover_source_invalid", "accepted reservation handover source is invalid")
	}
	accepted, err := decodeReservationContinuation(source.ContinuationAccepted)
	if err != nil {
		return nil, "", nil, err
	}
	if accepted == nil {
		return nil, "", nil, apperr.New(apperr.ErrConflict, "handover_source_invalid", "accepted continuation is missing")
	}
	acceptedDigest, _ := reservationDigest(*accepted)
	expected := model.ReservationOrderTerms{
		ReservationID: source.ID, BookingDigest: source.CurrentDigest,
		ContinuationDigest: source.ContinuationAcceptedDigest, ContinuationRevision: source.ContinuationAcceptedRevision,
		Mode: accepted.Mode, AllowPartialPurchaseHandover: accepted.AllowPartialPurchaseHandover,
		MinimumPaid: accepted.MinimumPaid, HandoverConditions: accepted.HandoverConditions, HandoverTerms: accepted.HandoverTerms,
	}
	expectedDigest, _ := reservationDigest(expected)
	actualDigest, _ := reservationDigest(terms)
	if acceptedDigest != source.ContinuationAcceptedDigest || expectedDigest != actualDigest {
		return nil, "", nil, apperr.New(apperr.ErrConflict, "handover_source_changed", "accepted handover terms changed")
	}
	return current, terms.Mode, credit, nil
}

// purchaseHandoverAllowed uses accepted invoice evidence, including any
// assigned reservation credit. Before invoice issue, the unique accepted credit
// is the only paid money; pending or rejected claims never contribute.
func (s *Fulfilment) purchaseHandoverAllowed(ctx context.Context, st Store, o *model.Order, credit *model.ReservationCredit) error {
	terms := o.DecodeTerms()
	provenance := terms.Reservation
	if provenance == nil || provenance.Mode != "purchase" {
		return apperr.New(apperr.ErrConflict, "handover_source_invalid", "purchase agreement is missing")
	}
	total := model.TermsTotal(terms)
	face, valid := total.Parse()
	if !valid || face.Sign() <= 0 {
		return apperr.New(apperr.ErrConflict, "handover_money_invalid", "order total is invalid")
	}
	paid := new(big.Int)
	if credit != nil {
		if credit.Currency != total.Currency {
			return apperr.New(apperr.ErrConflict, "handover_money_invalid", "credit currency differs from order")
		}
		amount, err := invoiceMinor(credit.AmountMinor, false)
		if err != nil {
			return err
		}
		paid.Set(amount)
	}
	invoices, err := st.Invoices().ForOrder(ctx, o.ID)
	if err != nil {
		return err
	}
	var issued *model.Invoice
	for i := range invoices {
		if invoices[i].Status != "issued" {
			continue
		}
		if issued != nil {
			return apperr.New(apperr.ErrConflict, "invoice_exists", "multiple issued invoices exist")
		}
		issued = &invoices[i]
	}
	if issued != nil {
		if issued.OrderID != o.ID || issued.SupplierCompanyID != o.SupplierCompanyID || issued.BuyerCompanyID != o.BuyerCompanyID ||
			issued.Currency != total.Currency || issued.TotalMinor != total.AmountMinor {
			return apperr.New(apperr.ErrConflict, "handover_invoice_mismatch", "issued invoice differs from accepted order")
		}
		view, err := NewInvoice(s.Deps).view(ctx, st, issued)
		if err != nil {
			return err
		}
		paid, valid = view.Paid.Parse()
		if !valid {
			return apperr.New(apperr.ErrConflict, "handover_money_invalid", "accepted payment is invalid")
		}
	}
	if paid.Cmp(face) > 0 {
		return apperr.New(apperr.ErrConflict, "handover_money_invalid", "accepted payment exceeds order total")
	}
	if paid.Cmp(face) == 0 {
		return nil
	}
	if !provenance.AllowPartialPurchaseHandover {
		return apperr.New(apperr.ErrConflict, "handover_payment_due", "accepted order payment does not permit handover")
	}
	if provenance.MinimumPaid != nil {
		minimum, ok := provenance.MinimumPaid.Parse()
		if !ok || provenance.MinimumPaid.Currency != total.Currency || minimum.Cmp(face) > 0 {
			return apperr.New(apperr.ErrConflict, "handover_money_invalid", "accepted minimum payment is invalid")
		}
		if paid.Cmp(minimum) < 0 {
			return apperr.New(apperr.ErrConflict, "handover_payment_due", "accepted minimum payment has not been reached")
		}
	}
	return nil
}

func (s *Fulfilment) prepareConsignmentAllocations(ctx context.Context, st Store, o *model.Order, allocations []model.Allocation) error {
	if o.Source != "reservation" || o.DecodeTerms().Reservation == nil || o.DecodeTerms().Reservation.Mode != "consignment" {
		return nil
	}
	agreement, err := st.Consignments().AgreementByOrder(ctx, o.ID)
	if err != nil {
		return err
	}
	for _, allocation := range allocations {
		if _, err := NewConsignment(s.Deps).PrepareUnitLocked(ctx, st, agreement, allocation); err != nil {
			return err
		}
	}
	return nil
}

// handoverConsignment moves one selected, prepared VIN at a time, so the
// inventory fact can be linked to exactly that unit in the same transaction.
func (s *Fulfilment) handoverConsignment(ctx context.Context, st Store, o *model.Order, allocations []model.Allocation, ids []string, warehouseID, actorID string, at time.Time) error {
	provenance := o.DecodeTerms().Reservation
	agreement, err := st.Consignments().AgreementByOrder(ctx, o.ID)
	if err != nil {
		return err
	}
	if provenance == nil || agreement.OrderID != o.ID || agreement.ReservationID != provenance.ReservationID ||
		agreement.SupplierCompanyID != o.SupplierCompanyID || agreement.DealerCompanyID != o.BuyerCompanyID ||
		agreement.BookingDigest != provenance.BookingDigest || agreement.ContinuationDigest != provenance.ContinuationDigest ||
		agreement.ContinuationRevision != provenance.ContinuationRevision {
		return apperr.New(apperr.ErrConflict, "consignment_source_changed", "accepted consignment authority changed")
	}
	for _, id := range ids {
		var allocation *model.Allocation
		for i := range allocations {
			if allocations[i].VehicleID == id && (allocations[i].Status == "allocated" || allocations[i].Status == "shipped") {
				allocation = &allocations[i]
				break
			}
		}
		if allocation == nil {
			return apperr.FieldError("vehicleIds", "VIN has no current order allocation")
		}
		unit, err := st.Consignments().Unit(ctx, agreement.ID, id)
		if err != nil {
			return err
		}
		if unit.Status != model.ConsignmentPrepared || unit.OrderLineID != allocation.LineID || unit.VIN != allocation.VIN ||
			unit.SupplierCompanyID != o.SupplierCompanyID || unit.DealerCompanyID != o.BuyerCompanyID {
			return apperr.New(apperr.ErrConflict, "consignment_not_prepared", "VIN cannot be handed over")
		}
		operation := uuid.NewString()
		factID, err := s.stock.Custody(st.Bind(ctx), CustodyMove{
			OwnerCompanyID: o.SupplierCompanyID, SourceCustodianCompanyID: o.SupplierCompanyID,
			DestinationCompanyID: o.BuyerCompanyID, DestinationWarehouseID: warehouseID,
			OrderID: o.ID, AgreementID: agreement.ID, OperationRef: operation, ActorUserID: actorID,
			VehicleID: id, At: at,
		})
		if err != nil {
			return buyerWarehouseErr(err)
		}
		if _, err := NewConsignment(s.Deps).MarkHandedOverLocked(ctx, st, agreement.ID, id, unit.Version, factID, operation, actorID, at); err != nil {
			return err
		}
	}
	return nil
}
