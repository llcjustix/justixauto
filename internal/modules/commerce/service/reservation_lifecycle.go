package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/validate"
)

type ReservationExtension struct {
	ExpiresAt time.Time `json:"expiresAt"`
	AdvanceDueAt *time.Time `json:"advanceDueAt,omitempty"`
	Reason string `json:"reason"`
}

type ReservationExpiryPass struct {
	NextCursor string `json:"nextCursor,omitempty"`
	Selected int `json:"selected"`
	Expired int `json:"expired"`
	Protected int `json:"protected"`
	Failed int `json:"failed"`
}

type reservationExpiryCursor struct {
	Deadline time.Time `json:"deadline"`
	ID string `json:"id"`
}

func decodeExpiryCursor(raw string) (*reservationExpiryCursor, error) {
	if raw == "" { return nil, nil }
	bytes, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil { return nil, apperr.FieldError("cursor", "invalid expiry cursor") }
	var c reservationExpiryCursor
	if json.Unmarshal(bytes, &c) != nil || c.Deadline.IsZero() || uuid.Validate(c.ID) != nil {
		return nil, apperr.FieldError("cursor", "invalid expiry cursor")
	}
	return &c, nil
}

func encodeExpiryCursor(c model.ReservationDueCandidate) string {
	raw, _ := json.Marshal(reservationExpiryCursor{Deadline:c.Deadline.UTC(), ID:c.ID})
	return base64.RawURLEncoding.EncodeToString(raw)
}

// ExpireDue scans at most limit due candidates. Each row owns a transaction and
// is rechecked under its reservation lock; cursor progress is never authority
// to release. An error leaves earlier committed expiries intact and reports the
// cursor after the last attempted row, including a failed one.
func (s *Reservation) ExpireDue(ctx context.Context, limit int, cursor string) (ReservationExpiryPass, error) {
	var out ReservationExpiryPass
	if limit < 1 || limit > 100 { return out, apperr.FieldError("limit", "must be 1-100") }
	after, err := decodeExpiryCursor(cursor)
	if err != nil { return out, err }
	if err := ctx.Err(); err != nil { return out, err }
	now := s.clock()
	var afterAt *time.Time
	var afterID string
	if after != nil { afterAt, afterID = &after.Deadline, after.ID }
	rows, err := s.store.Reservations().DueCandidates(ctx, now, afterAt, afterID, limit)
	if err != nil { return out, err }
	var firstErr error
	for _, candidate := range rows {
		if err := ctx.Err(); err != nil { return out, errors.Join(firstErr, err) }
		expired, attemptErr := s.expireCandidate(ctx, candidate.ID, now)
		out.Selected++
		out.NextCursor = encodeExpiryCursor(candidate)
		if attemptErr != nil {
			out.Failed++
			if firstErr == nil { firstErr = attemptErr }
			if ctx.Err() != nil { return out, errors.Join(firstErr, ctx.Err()) }
		} else if expired { out.Expired++ } else { out.Protected++ }
	}
	if len(rows) < limit { out.NextCursor = "" }
	return out, firstErr
}

func (s *Reservation) expireCandidate(ctx context.Context, id string, now time.Time) (bool, error) {
	committed := false
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil { return err }
		if err := ctx.Err(); err != nil { return err }
		if r.Status != model.ReservationActive || r.OrderID != nil || r.ExpiresAt == nil { return nil }
		money, err := NewReservationPayment(s.Deps).MoneyLocked(ctx, st, r)
		if err != nil { return err }
		if money.EvidenceEverSubmitted || money.PendingCount != 0 || money.Accepted.AmountMinor != "0" || money.Credit != nil || money.Refund != nil { return nil }
		due := !r.ExpiresAt.After(now)
		if money.AgreedAdvance.AmountMinor != "0" {
			// A mandatory advance has its own agreed deadline. A historical
			// rejected submission still protects the source above.
			due = r.AdvanceDueAt != nil && !r.AdvanceDueAt.After(now) || due
		}
		if !due { return nil }
		reason := "agreed expiry"
		if money.AgreedAdvance.AmountMinor != "0" && r.AdvanceDueAt != nil && !r.AdvanceDueAt.After(now) { reason = "advance deadline without submitted evidence" }
		if err := s.releaseReservationLocked(ctx, st, r, lines, model.ReservationExpired, reason, now); err != nil { return err }
		details, _ := json.Marshal(map[string]any{"automatic":true,"reason":reason})
		if err := st.Events().Append(ctx, &model.Event{ID:uuid.NewString(), CompanyID:r.SupplierCompanyID, EventType:"reservation.expired", ResourceType:"reservation", ResourceID:r.ID, ActorUserID:uuid.Nil.String(), OccurredAt:now, Reason:reason, Details:details}); err != nil { return err }
		if err := ctx.Err(); err != nil { return err }
		committed = true
		return nil
	})
	return committed && err == nil, err
}

func (s *Reservation) Extend(ctx context.Context, p *auth.Principal, id string, expected int64, in ReservationExtension) (*ReservationView, error) {
	if err := p.Allow(model.PermTrade); err != nil { return nil, err }
	var fields apperr.Validation
	in.Reason = validate.Reason(&fields, in.Reason)
	if err := fields.Err(); err != nil { return nil, err }
	var out *ReservationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil { return err }
		if r.SupplierCompanyID != p.CompanyID { return apperr.ErrNotFound }
		if r.Version != expected { return apperr.ErrStale }
		if r.Status != model.ReservationActive || r.OrderID != nil || r.ExpiresAt == nil { return apperr.New(apperr.ErrConflict,"reservation_not_extendable","reservation is no longer active") }
		money, err := NewReservationPayment(s.Deps).MoneyLocked(ctx, st, r)
		if err != nil { return err }
		if money.Credit != nil || money.Refund != nil { return apperr.New(apperr.ErrConflict,"reservation_money_settling","reservation is settling") }
		now := s.clock()
		if !in.ExpiresAt.After(now) || !in.ExpiresAt.After(*r.ExpiresAt) { return apperr.FieldError("expiresAt","must extend the current deadline into the future") }
		if in.AdvanceDueAt != nil {
			if r.AdvanceDueAt == nil || !in.AdvanceDueAt.After(now) || !in.AdvanceDueAt.After(*r.AdvanceDueAt) || in.AdvanceDueAt.After(in.ExpiresAt) {
				return apperr.FieldError("advanceDueAt","must extend the agreed advance deadline within expiry")
			}
			r.AdvanceDueAt = in.AdvanceDueAt
		} else if r.AdvanceDueAt != nil && r.AdvanceDueAt.After(in.ExpiresAt) { return apperr.FieldError("expiresAt","must not precede the advance deadline") }
		r.ExpiresAt = &in.ExpiresAt
		r.ExtendedBy, r.ExtendedAt, r.UpdatedAt = &p.UserID, &now, now
		if err := st.Reservations().Update(ctx, r, expected); err != nil { return err }
		if err := s.event(ctx, st, p, "reservation.extended", "reservation", r.ID, in.Reason, map[string]any{"expiresAt":r.ExpiresAt,"advanceDueAt":r.AdvanceDueAt}); err != nil { return err }
		out, err = s.lifecycleViewLocked(ctx, st, r, lines, p.CompanyID)
		return err
	})
	return out, err
}

func (s *Reservation) Cancel(ctx context.Context, p *auth.Principal, id string, expected int64, reason string) (*ReservationView, error) {
	if err := p.Allow(model.PermTrade); err != nil { return nil, err }
	var fields apperr.Validation
	reason = validate.Reason(&fields, reason)
	if err := fields.Err(); err != nil { return nil, err }
	var out *ReservationView
	err := s.store.InTx(ctx, func(st Store) error {
		r, lines, err := st.Reservations().ForUpdate(ctx, id)
		if err != nil { return err }
		if r.BuyerCompanyID != p.CompanyID && r.SupplierCompanyID != p.CompanyID { return apperr.ErrNotFound }
		if r.Version != expected { return apperr.ErrStale }
		if r.Status != model.ReservationActive || r.OrderID != nil { return apperr.New(apperr.ErrConflict,"reservation_not_cancellable","reservation is no longer an active hold") }
		money, err := NewReservationPayment(s.Deps).MoneyLocked(ctx, st, r)
		if err != nil { return err }
		eligible, err := NewReservationPayment(s.Deps).ReleaseEligibleLocked(ctx, st, r)
		if err != nil { return err }
		if !eligible {
			code := "refund_required"
			message := "full accountant-confirmed refund is required before cancellation"
			if money.PendingCount != 0 { code, message = "pending_evidence", "decide submitted payment evidence before cancellation" }
			return apperr.New(apperr.ErrConflict,code,message)
		}
		now := s.clock()
		if err := s.releaseReservationLocked(ctx, st, r, lines, model.ReservationCancelled, reason, now); err != nil { return err }
		if err := s.event(ctx, st, p, "reservation.cancelled", "reservation", r.ID, reason, nil); err != nil { return err }
		out, err = s.lifecycleViewLocked(ctx, st, r, lines, p.CompanyID)
		return err
	})
	return out, err
}

// releaseReservationLocked keeps materialized claims consumed permanently.
// Physical reservation holds, including materialized VIN associations, end in
// the same transaction as held-claim release and the versioned status update.
func (s *Reservation) releaseReservationLocked(ctx context.Context, st Store, r *model.Reservation, lines []model.ReservationLine, status model.ReservationStatus, reason string, now time.Time) error {
	if r.Status != model.ReservationActive || r.OrderID != nil { return apperr.New(apperr.ErrConflict,"reservation_not_releasable","reservation no longer owns the hold") }
	lotIDs, vehicleIDs := []string{}, []string{}
	for _, line := range lines {
		if line.SupplyLotID != nil { lotIDs = append(lotIDs, *line.SupplyLotID) }
		if line.VehicleID != nil { vehicleIDs = append(vehicleIDs, *line.VehicleID) }
	}
	slices.Sort(lotIDs); lotIDs = slices.Compact(lotIDs)
	for _, lotID := range lotIDs { if _, err := st.Supply().LockLot(ctx, lotID); err != nil { return err } }
	for _, lotID := range lotIDs {
		claims, err := st.Supply().ClaimsForUpdate(ctx, lotID)
		if err != nil { return err }
		for _, claim := range claims {
			if claim.ReservationID == nil || *claim.ReservationID != r.ID || claim.OrderID != nil || claim.State != "held" { continue }
			claim.State = "released"
			if err := st.Supply().UpdateClaim(ctx, &claim); err != nil { return err }
		}
	}
	associations, err := st.Supply().ClaimAllocations(ctx, r.ID)
	if err != nil { return err }
	for _, association := range associations {
		if !association.Current || association.HolderType != "commerce-reservation" || association.HolderID != r.ID { continue }
		vehicleIDs = append(vehicleIDs, association.VehicleID)
		if err := st.Supply().RetireClaimAllocation(ctx, association.ClaimID, association.ID); err != nil { return err }
	}
	slices.Sort(vehicleIDs); vehicleIDs = slices.Compact(vehicleIDs)
	if len(vehicleIDs) != 0 {
		if err := s.reservationStock.ReleaseReservation(st.Bind(ctx), r.ID, vehicleIDs, reason); err != nil { return err }
	}
	r.Status, r.ReleasedAt, r.ReleaseReason, r.UpdatedAt = status, &now, reason, now
	return st.Reservations().Update(ctx, r, r.Version)
}
