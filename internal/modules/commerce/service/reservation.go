package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
	"justixauto/internal/pkg/money"
)

type Reservation struct {
	Deps
	reservationStock ReservationStock
}

func NewReservation(d Deps, stock ReservationStock) *Reservation {
	return &Reservation{Deps: d, reservationStock: stock}
}

type ReservationTerms struct {
	UnitPrice    *money.Money `json:"unitPrice,omitempty"`
	ExpiresAt    *time.Time   `json:"expiresAt,omitempty"`
	Advance      *money.Money `json:"advance,omitempty"`
	AdvanceDueAt *time.Time   `json:"advanceDueAt,omitempty"`
}
type ReservationLineInput struct {
	VehicleID   *string        `json:"vehicleId,omitempty"`
	SupplyLotID *string        `json:"supplyLotId,omitempty"`
	Quantity    jsonx.Quantity `json:"quantity"`
}
type ReservationRequest struct {
	SupplierCompanyID string                 `json:"supplierCompanyId"`
	Lines             []ReservationLineInput `json:"lines"`
	DesiredTerms      ReservationTerms       `json:"desiredTerms"`
}
type ReservationConfirmation struct {
	Terms ReservationTerms          `json:"terms"`
	Lines []ReservationLineQuantity `json:"lines,omitempty"`
}
type ReservationLineQuantity struct {
	LineID   string         `json:"lineId"`
	Quantity jsonx.Quantity `json:"quantity"`
}
type ReservationSnapshotLine struct {
	LineID                    string         `json:"lineId"`
	VehicleID                 *string        `json:"vehicleId,omitempty"`
	SupplyLotID               *string        `json:"supplyLotId,omitempty"`
	ModelID                   string         `json:"modelId"`
	ModelSpecificationVersion string         `json:"modelSpecificationVersion"`
	ExteriorColor             string         `json:"exteriorColor"`
	InteriorColor             string         `json:"interiorColor"`
	Quantity                  jsonx.Quantity `json:"quantity"`
}
type ReservationSnapshot struct {
	Terms ReservationTerms          `json:"terms"`
	Lines []ReservationSnapshotLine `json:"lines"`
}

// ReservationContinuation is an independent mutual agreement. It never
// replaces CurrentTerms, whose digest is bound to reservation payment evidence.
type ReservationContinuation struct {
	Mode                         string       `json:"mode"`
	Terms                        model.Terms  `json:"terms"`
	AllowPartialPurchaseHandover bool         `json:"allowPartialPurchaseHandover"`
	MinimumPaid                  *money.Money `json:"minimumPaid,omitempty"`
	HandoverConditions           string       `json:"handoverConditions,omitempty"`
	HandoverTerms                string       `json:"handoverTerms"`
}
type ReservationConversion struct {
	Reservation *ReservationView
	Order       *model.Order
}
type ReservationView struct {
	Reservation                                model.Reservation
	Lines                                      []model.ReservationLine
	Requested, Offered, Current                *ReservationSnapshot
	ContinuationProposed, ContinuationAccepted *ReservationContinuation
	Buyer, Supplier                            Company
	AllowedActions                             []string
	CancellationState                          string
}

func reservationSnapshot(lines []model.ReservationLine, terms ReservationTerms) ReservationSnapshot {
	out := ReservationSnapshot{Terms: terms, Lines: make([]ReservationSnapshotLine, 0, len(lines))}
	for _, l := range lines {
		out.Lines = append(out.Lines, ReservationSnapshotLine{
			LineID: l.ID, VehicleID: l.VehicleID, SupplyLotID: l.SupplyLotID,
			ModelID: l.ModelID, ModelSpecificationVersion: l.ModelSpecificationVersion, ExteriorColor: l.ExteriorColor,
			InteriorColor: l.InteriorColor, Quantity: jsonx.Quantity(l.Quantity),
		})
	}
	// Database ordering is by ID. Normalize so the digest is independent of client order.
	slices.SortFunc(out.Lines, func(a, b ReservationSnapshotLine) int { return strings.Compare(a.LineID, b.LineID) })
	return out
}

func decodeReservationSnapshot(raw []byte) (*ReservationSnapshot, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var snap ReservationSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func decodeReservationContinuation(raw []byte) (*ReservationContinuation, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var terms ReservationContinuation
	if err := json.Unmarshal(raw, &terms); err != nil {
		return nil, err
	}
	return &terms, nil
}

func (s *Reservation) view(ctx context.Context, r *model.Reservation, lines []model.ReservationLine, companyID string) (*ReservationView, error) {
	requested, err := decodeReservationSnapshot(r.RequestedTerms)
	if err != nil {
		return nil, err
	}
	offered, err := decodeReservationSnapshot(r.OfferedTerms)
	if err != nil {
		return nil, err
	}
	current, err := decodeReservationSnapshot(r.CurrentTerms)
	if err != nil {
		return nil, err
	}
	proposed, err := decodeReservationContinuation(r.ContinuationProposed)
	if err != nil {
		return nil, err
	}
	accepted, err := decodeReservationContinuation(r.ContinuationAccepted)
	if err != nil {
		return nil, err
	}
	buyer, err := s.directory.Company(ctx, r.BuyerCompanyID)
	if err != nil {
		return nil, err
	}
	supplier, err := s.directory.Company(ctx, r.SupplierCompanyID)
	if err != nil {
		return nil, err
	}
	v := &ReservationView{Reservation: *r, Lines: lines, Requested: requested, Offered: offered, Current: current, ContinuationProposed: proposed, ContinuationAccepted: accepted, Buyer: *buyer, Supplier: *supplier, AllowedActions: []string{}}
	if r.Status == model.ReservationRequested && companyID == r.SupplierCompanyID {
		v.AllowedActions = append(v.AllowedActions, "confirm")
	}
	if r.Status == model.ReservationOffered && companyID == r.BuyerCompanyID {
		v.AllowedActions = append(v.AllowedActions, "accept")
	}
	if r.Status == model.ReservationActive && r.OrderID == nil && companyID == r.SupplierCompanyID && r.ExpiresAt != nil && r.ExpiresAt.After(s.clock()) {
		v.AllowedActions = append(v.AllowedActions, "allocate")
	}
	if r.Status == model.ReservationActive && r.OrderID == nil && r.ExpiresAt != nil && r.ExpiresAt.After(s.clock()) {
		v.AllowedActions = append(v.AllowedActions, "proposeContinuation")
		if proposed != nil && r.ContinuationProposedBy != nil && *r.ContinuationProposedBy != companyID && r.ContinuationAcceptedDigest != r.ContinuationProposedDigest {
			v.AllowedActions = append(v.AllowedActions, "acceptContinuation")
		}
		if accepted != nil && r.ContinuationAcceptedDigest == r.ContinuationProposedDigest {
			v.AllowedActions = append(v.AllowedActions, "convert")
		}
	}
	if r.Status == model.ReservationCancelled {
		v.CancellationState = "cancelled"
	}
	if r.Status == model.ReservationExpired {
		v.CancellationState = "expired"
	}
	if r.OrderID != nil {
		v.CancellationState = "converted"
	}
	return v, nil
}

func (s *Reservation) lifecycleViewLocked(ctx context.Context, st Store, r *model.Reservation, lines []model.ReservationLine, companyID string) (*ReservationView, error) {
	v, err := s.view(ctx, r, lines, companyID)
	if err != nil || r.Status != model.ReservationActive || r.OrderID != nil {
		return v, err
	}
	m, err := NewReservationPayment(s.Deps).MoneyLocked(ctx, st, r)
	if err != nil {
		return nil, err
	}
	switch {
	case m.PendingCount != 0:
		v.CancellationState = "awaitingEvidenceDecision"
	case m.Accepted.AmountMinor != "0" && m.RefundStatus != "confirmed":
		v.CancellationState = "awaitingRefund"
	default:
		v.CancellationState = "ready"
	}
	if companyID == r.SupplierCompanyID {
		v.AllowedActions = append(v.AllowedActions, "extend")
	}
	if v.CancellationState == "ready" {
		v.AllowedActions = append(v.AllowedActions, "cancel")
	}
	if (v.CancellationState == "awaitingEvidenceDecision" || v.CancellationState == "awaitingRefund") && m.Refund == nil {
		v.AllowedActions = append(v.AllowedActions, "requestRefund")
	}
	return v, nil
}

func (s *Reservation) dealer(ctx context.Context, p *auth.Principal) error {
	if err := p.Allow(model.PermTrade); err != nil {
		return err
	}
	if err := p.AllowNewCommercialOperation(); err != nil {
		return err
	}
	c, err := s.tradingCompany(ctx, p.CompanyID, "companyId")
	if err != nil {
		return err
	}
	if c.UserType == nil || *c.UserType != "dealer" {
		return apperr.New(apperr.ErrForbidden, "dealer_required", "only a verified dealer can request a reservation")
	}
	return nil
}

func (s *Reservation) distributor(ctx context.Context, p *auth.Principal) error {
	if err := p.Allow(model.PermTrade); err != nil {
		return err
	}
	if err := p.AllowNewCommercialOperation(); err != nil {
		return err
	}
	c, err := s.tradingCompany(ctx, p.CompanyID, "companyId")
	if err != nil {
		return err
	}
	if c.UserType == nil || *c.UserType != "distributor" {
		return apperr.New(apperr.ErrForbidden, "distributor_required", "only a verified distributor can confirm a reservation")
	}
	return nil
}

func (s *Reservation) requirePartner(ctx context.Context, st Store, a, b string) error {
	active, err := st.Partnerships().ActiveBetween(ctx, a, b)
	if err != nil {
		return err
	}
	if !active {
		return apperr.New(apperr.ErrConflict, "partnership_required", "an active partnership is required for new reservations")
	}
	return nil
}

func (s *Reservation) continuationSource(r *model.Reservation) (*ReservationSnapshot, error) {
	if r.Status != model.ReservationActive || r.OrderID != nil || r.ExpiresAt == nil || !r.ExpiresAt.After(s.clock()) {
		return nil, apperr.New(apperr.ErrConflict, "reservation_not_convertible", "reservation is not an active unconverted hold")
	}
	snap, err := decodeReservationSnapshot(r.CurrentTerms)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, apperr.New(apperr.ErrConflict, "reservation_terms_changed", "accepted booking terms are unavailable")
	}
	canonicalDigest, _ := reservationDigest(*snap)
	if canonicalDigest != r.CurrentDigest {
		return nil, apperr.New(apperr.ErrConflict, "reservation_terms_changed", "accepted booking terms are unavailable")
	}
	return snap, nil
}

func (s *Reservation) normalizeContinuation(source *ReservationSnapshot, in ReservationContinuation) (ReservationContinuation, error) {
	var v apperr.Validation
	if in.Mode != "purchase" && in.Mode != "consignment" {
		v.Add("mode", "must be purchase or consignment")
	}
	if in.Mode != "purchase" && in.AllowPartialPurchaseHandover {
		v.Add("allowPartialPurchaseHandover", "applies only to purchase")
	}
	if !in.AllowPartialPurchaseHandover && (in.MinimumPaid != nil || in.HandoverConditions != "") {
		v.Add("allowPartialPurchaseHandover", "agree to partial purchase handover before setting its conditions")
	}
	if strings.TrimSpace(in.HandoverTerms) == "" || len(in.HandoverTerms) > 2000 {
		v.Add("handoverTerms", "explicit handover terms are required, at most 2000 characters")
	}
	if len(in.HandoverConditions) > 2000 {
		v.Add("handoverConditions", "at most 2000 characters")
	}
	if !slices.Contains(model.Routes, in.Terms.Route) {
		v.Add("terms.route", "choose an accepted route")
	}
	if in.Terms.PaymentSchedule == nil || len(in.Terms.PaymentSchedule) > 60 {
		v.Add("terms.paymentSchedule", "give the explicit settlement schedule, at most 60 installments")
	}
	if len(in.Terms.Lines) != len(source.Lines) {
		v.Add("terms.lines", "include every accepted source line exactly once")
	}
	ids, total, currency := normalizeContinuationLines(source, &in, &v)
	if !money.Fits(total) {
		v.Add("terms.lines", "total is too large")
	}
	if in.MinimumPaid != nil {
		minimum, valid := in.MinimumPaid.Parse()
		if !valid || minimum.Sign() < 0 || in.MinimumPaid.Currency != currency || minimum.Cmp(total) > 0 {
			v.Add("minimumPaid", "must be exact money in the order currency within the face total")
		}
	}
	normalizedSchedule := make([]model.Installment, 0, len(in.Terms.PaymentSchedule))
	scheduled := validateSchedule(&v, in.Terms.PaymentSchedule, ids, &normalizedSchedule, &currency)
	if len(in.Terms.PaymentSchedule) != 0 && scheduled.Cmp(total) != 0 {
		v.Add("terms.paymentSchedule", "installments must sum to the exact order total")
	}
	in.Terms.PaymentSchedule = normalizedSchedule
	if in.Terms.Reservation != nil {
		v.Add("terms.reservation", "source provenance is assigned by the server")
	}
	if err := v.Err(); err != nil {
		return ReservationContinuation{}, err
	}
	return in, nil
}

func (s *Reservation) ProposeContinuation(ctx context.Context, p *auth.Principal, id string, expected int64, in ReservationContinuation) (*ReservationView, error) {
	if err := p.Allow(model.PermTrade); err != nil {
		return nil, err
	}
	var out *ReservationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		source, err := s.continuationSource(r)
		if err != nil {
			return err
		}
		terms, err := s.normalizeContinuation(source, in)
		if err != nil {
			return err
		}
		digest, raw := reservationDigest(terms)
		r.ContinuationProposed, r.ContinuationProposedDigest, r.ContinuationProposedRevision = raw, digest, r.Version+1
		r.ContinuationProposedBy = &p.CompanyID
		r.ContinuationAccepted, r.ContinuationAcceptedDigest, r.ContinuationAcceptedRevision, r.ContinuationAcceptedBy = nil, "", 0, nil
		r.UpdatedAt = s.clock()
		if err := st.Reservations().Update(ctx, r, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.continuation_proposed", "reservation", r.ID, "", map[string]any{"digest": digest}); err != nil {
			return err
		}
		out, err = s.lifecycleViewLocked(ctx, st, r, lines, p.CompanyID)
		return err
	})
	return out, err
}

func (s *Reservation) AcceptContinuation(ctx context.Context, p *auth.Principal, id string, expected int64, digest string, proposedRevision int64) (*ReservationView, error) {
	if err := p.Allow(model.PermTrade); err != nil {
		return nil, err
	}
	var out *ReservationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if _, err := s.continuationSource(r); err != nil {
			return err
		}
		if r.ContinuationProposedBy == nil || *r.ContinuationProposedBy == p.CompanyID {
			return apperr.New(apperr.ErrForbidden, "wrong_party", "the other party accepts continuation")
		}
		proposed, err := decodeReservationContinuation(r.ContinuationProposed)
		if err != nil {
			return err
		}
		if proposed == nil {
			return apperr.New(apperr.ErrConflict, "continuation_missing", "proposal is unavailable")
		}
		canonicalDigest, _ := reservationDigest(*proposed)
		if digest == "" || digest != r.ContinuationProposedDigest || proposedRevision != r.ContinuationProposedRevision || canonicalDigest != digest {
			return apperr.New(apperr.ErrConflict, "continuation_changed", "accept the exact current continuation proposal")
		}
		r.ContinuationAccepted = append([]byte(nil), r.ContinuationProposed...)
		r.ContinuationAcceptedDigest, r.ContinuationAcceptedRevision, r.ContinuationAcceptedBy = digest, r.Version+1, &p.CompanyID
		r.UpdatedAt = s.clock()
		if err := st.Reservations().Update(ctx, r, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.continuation_accepted", "reservation", r.ID, "", map[string]any{"digest": digest}); err != nil {
			return err
		}
		out, err = s.lifecycleViewLocked(ctx, st, r, lines, p.CompanyID)
		return err
	})
	return out, err
}

func (s *Reservation) Convert(ctx context.Context, p *auth.Principal, id string, expected int64, digest string, acceptedRevision int64) (*ReservationConversion, error) {
	if err := p.Allow(model.PermTrade); err != nil {
		return nil, err
	}
	var out *ReservationConversion
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if _, err := s.continuationSource(r); err != nil {
			return err
		}
		moneyView, err := NewReservationPayment(s.Deps).MoneyLocked(ctx, st, r)
		if err != nil {
			return err
		}
		if r.AdvanceDueAt != nil && !r.AdvanceDueAt.After(s.clock()) && !moneyView.EvidenceEverSubmitted {
			return apperr.New(apperr.ErrConflict, "reservation_advance_due", "reservation advance deadline has passed without evidence")
		}
		order, now, err := s.createContinuationOrderLocked(ctx, st, p, r, digest, acceptedRevision)
		if err != nil {
			return err
		}
		r.Status, r.OrderID, r.UpdatedAt = model.ReservationConverted, &order.ID, now
		if err := st.Reservations().Update(ctx, r, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "reservation.converted", "reservation", r.ID, "", map[string]any{"orderId": order.ID, "continuationDigest": digest}); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "order.created", "order", order.ID, "", map[string]any{"source": "reservation", "reservationId": r.ID}); err != nil {
			return err
		}
		view, err := s.view(ctx, r, lines, p.CompanyID)
		if err != nil {
			return err
		}
		out = &ReservationConversion{Reservation: view, Order: order}
		return nil
	})
	return out, err
}

func reservationDigest(v any) (string, []byte) {
	raw, _ := json.Marshal(v)
	return model.Digest(raw), raw
}

func completeTerms(t ReservationTerms) bool {
	return t.UnitPrice != nil && t.ExpiresAt != nil && t.UnitPrice.Currency != "" && t.UnitPrice.AmountMinor != ""
}

func validTerms(v *apperr.Validation, t ReservationTerms, required bool, now time.Time) {
	if required && !completeTerms(t) {
		v.Add("terms", "price, currency and expiry are required for confirmation")
		return
	}
	if t.UnitPrice != nil {
		if amount, ok := t.UnitPrice.Parse(); !ok || amount.Sign() <= 0 {
			v.Add("terms.unitPrice", "must be a positive exact money value")
		}
	}
	if t.Advance != nil {
		if amount, ok := t.Advance.Parse(); !ok || amount.Sign() <= 0 {
			v.Add("terms.advance", "must be a positive exact money value")
		}
		if t.UnitPrice != nil && t.Advance.Currency != t.UnitPrice.Currency {
			v.Add("terms.advance", "currency must match unit price")
		}
	}
	if t.AdvanceDueAt != nil && t.Advance == nil {
		v.Add("terms.advanceDueAt", "requires advance")
	}
	if t.Advance != nil && t.AdvanceDueAt == nil {
		v.Add("terms.advanceDueAt", "required with advance")
	}
	if t.ExpiresAt != nil && !t.ExpiresAt.After(now) {
		v.Add("terms.expiresAt", "must be in the future")
	}
	if t.AdvanceDueAt != nil {
		if !t.AdvanceDueAt.After(now) {
			v.Add("terms.advanceDueAt", "must be in the future")
		}
		if t.ExpiresAt != nil && t.AdvanceDueAt.After(*t.ExpiresAt) {
			v.Add("terms.advanceDueAt", "must not be after expiry")
		}
	}
}

func (s *Reservation) validateLines(ctx context.Context, st Store, supplier string, in []ReservationLineInput, id string) ([]model.ReservationLine, error) {
	var v apperr.Validation
	if len(in) == 0 || len(in) > 100 {
		v.Add("lines", "list 1-100 lines")
	}
	out := make([]model.ReservationLine, 0, len(in))
	seen := map[string]bool{}
	for i, x := range in {
		l := model.ReservationLine{ID: uuid.NewString(), ReservationID: id}
		f := "lines." + fmt.Sprint(i)
		if x.Quantity < 1 || x.Quantity > math.MaxInt32 {
			v.Add(f+".quantity", "must be a positive SQL quantity")
			continue
		}
		l.Quantity = int(x.Quantity)
		if (x.VehicleID == nil) == (x.SupplyLotID == nil) {
			v.Add(f, "choose exactly one vehicle or supply lot")
			continue
		}
		if x.VehicleID != nil {
			if uuid.Validate(*x.VehicleID) != nil {
				v.Add(f+".vehicleId", "must be a valid ID")
				continue
			}
			if seen["vin:"+*x.VehicleID] {
				v.Add(f+".vehicleId", "duplicate vehicle")
				continue
			}
			seen["vin:"+*x.VehicleID] = true
			if x.Quantity != 1 {
				v.Add(f+".quantity", "VIN reservation quantity must be 1")
			}
			vehicle, err := s.stock.Vehicle(ctx, supplier, *x.VehicleID)
			if err != nil {
				v.Add(f+".vehicleId", "not available from supplier")
				continue
			}
			l.VehicleID = x.VehicleID
			l.ModelID, l.ModelSpecificationVersion, l.ExteriorColor, l.InteriorColor = vehicle.ModelID, vehicle.ModelSpecificationVersion, vehicle.ExteriorColor, vehicle.InteriorColor
		} else {
			if uuid.Validate(*x.SupplyLotID) != nil {
				v.Add(f+".supplyLotId", "must be a valid ID")
				continue
			}
			if seen["lot:"+*x.SupplyLotID] {
				v.Add(f+".supplyLotId", "duplicate supply lot")
				continue
			}
			seen["lot:"+*x.SupplyLotID] = true
			lot, err := st.Supply().Lot(ctx, *x.SupplyLotID)
			if err != nil || lot.SupplierCompanyID != supplier {
				v.Add(f+".supplyLotId", "not available from supplier")
				continue
			}
			l.SupplyLotID = x.SupplyLotID
			l.ModelID, l.ModelSpecificationVersion, l.ExteriorColor, l.InteriorColor = lot.ModelID, lot.ModelSpecificationVersion, lot.ExteriorColor, lot.InteriorColor
		}
		out = append(out, l)
	}
	return out, v.Err()
}

func (s *Reservation) Request(ctx context.Context, p *auth.Principal, in ReservationRequest) (*ReservationView, error) {
	if err := s.dealer(ctx, p); err != nil {
		return nil, err
	}
	if uuid.Validate(in.SupplierCompanyID) != nil || in.SupplierCompanyID == p.CompanyID {
		return nil, apperr.FieldError("supplierCompanyId", "must be another company")
	}
	supplier, err := s.tradingCompany(ctx, in.SupplierCompanyID, "supplierCompanyId")
	if err != nil {
		return nil, err
	}
	if supplier.UserType == nil || *supplier.UserType != "distributor" {
		return nil, apperr.New(apperr.ErrConflict, "distributor_required", "supplier must be a verified distributor")
	}
	var v apperr.Validation
	validTerms(&v, in.DesiredTerms, false, s.clock())
	if err := v.Err(); err != nil {
		return nil, err
	}
	now := s.clock()
	r := &model.Reservation{ID: uuid.NewString(), BuyerCompanyID: p.CompanyID, SupplierCompanyID: in.SupplierCompanyID, Status: model.ReservationRequested, RequestedRevision: 1, Version: 1, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now}
	var lines []model.ReservationLine
	var out *ReservationView
	err = s.store.InTx(ctx, func(st Store) error {
		if err := s.requirePartner(ctx, st, p.CompanyID, in.SupplierCompanyID); err != nil {
			return err
		}
		var e error
		lines, e = s.validateLines(ctx, st, in.SupplierCompanyID, in.Lines, r.ID)
		if e != nil {
			return e
		}
		r.RequestedDigest, r.RequestedTerms = reservationDigest(reservationSnapshot(lines, in.DesiredTerms))
		if err := st.Reservations().Create(ctx, r, lines); err != nil {
			return err
		}
		out, err = s.view(ctx, r, lines, p.CompanyID)
		if err != nil {
			return err
		}
		return s.event(ctx, st, p, "reservation.requested", "reservation", r.ID, "", nil)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Reservation) Get(ctx context.Context, p *auth.Principal, id string) (*ReservationView, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	if uuid.Validate(id) != nil {
		return nil, apperr.ErrNotFound
	}
	var out *ReservationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		out, err = s.lifecycleViewLocked(ctx, st, r, lines, p.CompanyID)
		return err
	})
	return out, err
}

func (s *Reservation) List(ctx context.Context, p *auth.Principal, limit, offset int) ([]ReservationView, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	rows, err := s.store.Reservations().List(ctx, p.CompanyID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]ReservationView, 0, len(rows))
	for _, candidate := range rows {
		err := s.store.InTx(ctx, func(st Store) error {
			r, lines, err := st.Reservations().ForUpdate(ctx, candidate.ID)
			if err != nil {
				return err
			}
			if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID {
				return apperr.ErrNotFound
			}
			v, err := s.lifecycleViewLocked(ctx, st, r, lines, p.CompanyID)
			if err != nil {
				return err
			}
			out = append(out, *v)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Reservation) Confirm(ctx context.Context, p *auth.Principal, id string, expected int64, in ReservationConfirmation) (*ReservationView, error) {
	if err := s.distributor(ctx, p); err != nil {
		return nil, err
	}
	var v apperr.Validation
	validTerms(&v, in.Terms, true, s.clock())
	if err := v.Err(); err != nil {
		return nil, err
	}
	var out *ReservationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if r.Status != model.ReservationRequested {
			return apperr.New(apperr.ErrConflict, "reservation_confirmed", "reservation is no longer awaiting confirmation")
		}
		requested, err := decodeReservationSnapshot(r.RequestedTerms)
		if err != nil {
			return err
		}
		if requested == nil {
			return apperr.New(apperr.ErrConflict, "reservation_terms_missing", "requested terms are unavailable")
		}
		snapshot := reservationSnapshot(lines, in.Terms)
		if err := applyReservationQuantities(&snapshot, in.Lines); err != nil {
			return err
		}
		digest, raw := reservationDigest(snapshot)
		r.OfferedTerms, r.OfferedDigest = raw, digest
		r.OfferedRevision = expected + 1
		if digest == r.RequestedDigest && completeTerms(requested.Terms) {
			if err := s.activate(ctx, st, r, snapshot, digest); err != nil {
				return err
			}
		} else {
			r.Status = model.ReservationOffered
			r.UpdatedAt = s.clock()
			if err := st.Reservations().Update(ctx, r, expected); err != nil {
				return err
			}
		}
		out, err = s.view(ctx, r, lines, p.CompanyID)
		if err != nil {
			return err
		}
		return s.event(ctx, st, p, "reservation.confirmed", "reservation", r.ID, "", map[string]any{"digest": digest})
	})
	return out, err
}

func (s *Reservation) Accept(ctx context.Context, p *auth.Principal, id string, expected int64, digest string) (*ReservationView, error) {
	if err := s.dealer(ctx, p); err != nil {
		return nil, err
	}
	var out *ReservationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.BuyerCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		if r.Status != model.ReservationOffered || digest == "" || digest != r.OfferedDigest {
			return apperr.New(apperr.ErrConflict, "reservation_terms_changed", "the exact offered terms must be accepted")
		}
		snapshot, err := decodeReservationSnapshot(r.OfferedTerms)
		if err != nil {
			return err
		}
		if snapshot == nil {
			return apperr.New(apperr.ErrConflict, "reservation_terms_missing", "offered terms are unavailable")
		}
		var v apperr.Validation
		validTerms(&v, snapshot.Terms, true, s.clock())
		if err := v.Err(); err != nil {
			return err
		}
		if err := s.activate(ctx, st, r, *snapshot, digest); err != nil {
			return err
		}
		out, err = s.view(ctx, r, lines, p.CompanyID)
		if err != nil {
			return err
		}
		return s.event(ctx, st, p, "reservation.activated", "reservation", r.ID, "", map[string]any{"digest": digest})
	})
	return out, err
}

func (s *Reservation) activate(ctx context.Context, st Store, r *model.Reservation, snapshot ReservationSnapshot, digest string) error {
	if r.Status != model.ReservationRequested && r.Status != model.ReservationOffered {
		return apperr.New(apperr.ErrConflict, "reservation_already_active", "reservation cannot be activated again")
	}
	var v apperr.Validation
	validTerms(&v, snapshot.Terms, true, s.clock())
	if err := v.Err(); err != nil {
		return err
	}
	// Lock future lots in a stable order before acquiring VIN locks. All claims and
	// stock holds share the transaction carried by Store.Bind.
	lots := make([]string, 0)
	for _, l := range snapshot.Lines {
		if l.SupplyLotID != nil {
			lots = append(lots, *l.SupplyLotID)
		}
	}
	slices.Sort(lots)
	for _, id := range lots {
		if _, err := st.Supply().LockLot(ctx, id); err != nil {
			return err
		}
	}
	vinIDs := make([]string, 0)
	for _, l := range snapshot.Lines {
		if l.VehicleID != nil {
			if l.Quantity != 1 {
				return apperr.FieldError("lines.quantity", "VIN quantity must be 1")
			}
			vinIDs = append(vinIDs, *l.VehicleID)
		} else {
			if err := s.claimFutureReservationLine(ctx, st, r, l); err != nil {
				return err
			}
		}
	}
	if len(vinIDs) > 0 {
		slices.Sort(vinIDs)
		if err := s.reservationStock.ReserveReservation(st.Bind(ctx), r.SupplierCompanyID, r.ID, vinIDs); err != nil {
			return err
		}
	}
	r.Status = model.ReservationActive
	r.CurrentTerms, r.CurrentDigest = r.OfferedTerms, digest
	r.CurrentRevision = r.Version + 1
	r.ExpiresAt = snapshot.Terms.ExpiresAt
	r.AdvanceDueAt = snapshot.Terms.AdvanceDueAt
	r.UpdatedAt = s.clock()
	return st.Reservations().Update(ctx, r, r.Version)
}

func normalizeContinuationLines(source *ReservationSnapshot, in *ReservationContinuation, v *apperr.Validation) (map[string]bool, *big.Int, string) {
	sources := make(map[string]ReservationSnapshotLine, len(source.Lines))
	for _, line := range source.Lines {
		sources[line.LineID] = line
	}
	seen := map[string]bool{}
	ids := map[string]bool{}
	total := new(big.Int)
	currency := ""
	for i := range in.Terms.Lines {
		line := &in.Terms.Lines[i]
		field := fmt.Sprintf("terms.lines.%d", i)
		sourceLine, ok := sources[line.OfferLineID]
		if !ok || seen[line.OfferLineID] {
			v.Add(field+".offerLineId", "must identify one unique accepted source line")
		} else {
			seen[line.OfferLineID] = true
			if line.ModelID != sourceLine.ModelID || line.ModelSpecificationVersion != sourceLine.ModelSpecificationVersion || line.ExteriorColor != sourceLine.ExteriorColor || line.InteriorColor != sourceLine.InteriorColor || line.Quantity != sourceLine.Quantity {
				v.Add(field, "model, specification, colors and quantity must match the accepted source")
			}
		}
		if line.LineID == "" {
			line.LineID = uuid.NewString()
		}
		if uuid.Validate(line.LineID) != nil || ids[line.LineID] {
			v.Add(field+".lineId", "must be a unique ID")
		}
		ids[line.LineID] = true
		price, valid := line.UnitPrice.Parse()
		if !valid || price.Sign() <= 0 {
			v.Add(field+".unitPrice", "must be positive exact money")
		} else {
			sameCurrency(v, &currency, field+".unitPrice", line.UnitPrice.Currency)
			total.Add(total, new(big.Int).Mul(price, big.NewInt(int64(line.Quantity))))
		}
	}
	return ids, total, currency
}

func (s *Reservation) createContinuationOrderLocked(ctx context.Context, st Store, p *auth.Principal, r *model.Reservation, digest string, acceptedRevision int64) (*model.Order, time.Time, error) {
	agreement, err := decodeReservationContinuation(r.ContinuationAccepted)
	if err != nil {
		return nil, time.Time{}, err
	}
	if agreement == nil {
		return nil, time.Time{}, apperr.New(apperr.ErrConflict, "continuation_missing", "accepted continuation is unavailable")
	}
	canonicalDigest, _ := reservationDigest(*agreement)
	if digest == "" || digest != r.ContinuationAcceptedDigest || digest != r.ContinuationProposedDigest || acceptedRevision != r.ContinuationAcceptedRevision || canonicalDigest != digest {
		return nil, time.Time{}, apperr.New(apperr.ErrConflict, "continuation_changed", "the exact mutual continuation agreement is required")
	}
	terms := agreement.Terms
	terms.Reservation = &model.ReservationOrderTerms{ReservationID: r.ID, BookingDigest: r.CurrentDigest, ContinuationDigest: digest, ContinuationRevision: acceptedRevision, Mode: agreement.Mode, AllowPartialPurchaseHandover: agreement.AllowPartialPurchaseHandover, MinimumPaid: agreement.MinimumPaid, HandoverConditions: agreement.HandoverConditions, HandoverTerms: agreement.HandoverTerms}
	raw, err := json.Marshal(terms)
	if err != nil {
		return nil, time.Time{}, err
	}
	now := s.clock()
	order := &model.Order{ID: uuid.NewString(), BuyerCompanyID: r.BuyerCompanyID, SupplierCompanyID: r.SupplierCompanyID, Source: "reservation", Terms: raw, Status: model.OrderAccepted, Version: 1, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now}
	if err := st.Deals().CreateOrder(ctx, order); err != nil {
		return nil, time.Time{}, err
	}
	lineMap := make(map[string]string, len(terms.Lines))
	for _, line := range terms.Lines {
		lineMap[line.OfferLineID] = line.LineID
	}
	allocations, err := s.TransferObligations(ctx, st, r, order.ID, lineMap)
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(allocations) != 0 {
		if err := st.Fulfilment().AddAllocations(ctx, allocations); err != nil {
			return nil, time.Time{}, err
		}
	}
	if _, err := NewReservationPayment(s.Deps).LinkCreditLocked(ctx, st, r, order); err != nil {
		return nil, time.Time{}, err
	}
	if agreement.Mode == "consignment" {
		if _, err := NewConsignment(s.Deps).CreateFromConversionLocked(ctx, st, r, order, allocations); err != nil {
			return nil, time.Time{}, err
		}
	}
	return order, now, nil
}

func applyReservationQuantities(snapshot *ReservationSnapshot, lines []ReservationLineQuantity) error {
	if len(lines) > 0 {
		if len(lines) != len(snapshot.Lines) {
			return apperr.FieldError("lines", "must include every source line")
		}
		quantities := make(map[string]jsonx.Quantity, len(lines))
		for _, q := range lines {
			if uuid.Validate(q.LineID) != nil || q.Quantity < 1 || q.Quantity > math.MaxInt32 {
				return apperr.FieldError("lines", "each line needs a valid ID and positive SQL quantity")
			}
			if _, exists := quantities[q.LineID]; exists {
				return apperr.FieldError("lines", "duplicate line ID")
			}
			quantities[q.LineID] = q.Quantity
		}
		for i := range snapshot.Lines {
			q, exists := quantities[snapshot.Lines[i].LineID]
			if !exists || snapshot.Lines[i].VehicleID != nil && q != 1 {
				return apperr.FieldError("lines", "unknown line or invalid VIN quantity")
			}
			snapshot.Lines[i].Quantity = q
		}
	}
	return nil
}

func (s *Reservation) claimFutureReservationLine(ctx context.Context, st Store, r *model.Reservation, l ReservationSnapshotLine) error {
	if l.SupplyLotID == nil || l.Quantity < 1 || l.Quantity > math.MaxInt32 {
		return apperr.FieldError("lines.quantity", "invalid future quantity")
	}
	lot, err := st.Supply().Lot(ctx, *l.SupplyLotID)
	if err != nil {
		return err
	}
	if lot.SupplierCompanyID != r.SupplierCompanyID || lot.ModelID != l.ModelID || lot.ModelSpecificationVersion != l.ModelSpecificationVersion || lot.ExteriorColor != l.ExteriorColor || lot.InteriorColor != l.InteriorColor {
		return apperr.New(apperr.ErrConflict, "supply_source_changed", "future supply source has changed")
	}
	capacity, err := st.Supply().Capacity(ctx, lot.ID)
	if err != nil {
		return err
	}
	if capacity.Available < int(l.Quantity) {
		return apperr.New(apperr.ErrConflict, "supply_unavailable", "future supply is exhausted")
	}
	const batchSize = 500
	for start := 0; start < int(l.Quantity); start += batchSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := min(start+batchSize, int(l.Quantity))
		claims := make([]model.SupplyClaim, end-start)
		for i := range claims {
			claims[i] = model.SupplyClaim{ID: uuid.NewString(), SupplyLotID: lot.ID, ReservationID: &r.ID, ReservationLineID: &l.LineID, State: "held", CreatedAt: s.clock()}
		}
		if err := st.Supply().CreateClaims(ctx, claims); err != nil {
			return err
		}
	}
	return nil
}
