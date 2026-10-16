package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

// Older reservation units also read the new lifecycle projection. Their
// fixtures have no payment evidence, so an empty locked ledger is sufficient.
func (s *reservationTestStore) ReservationPayments() ReservationPaymentRepository {
	return emptyLifecycleMoney{}
}

type emptyLifecycleMoney struct{ ReservationPaymentRepository }

func (emptyLifecycleMoney) Evidence(context.Context, string) ([]model.ReservationEvidence, error) {
	return nil, nil
}

func (emptyLifecycleMoney) Refund(context.Context, string) (*model.ReservationRefund, error) {
	return nil, apperr.ErrNotFound
}

func (emptyLifecycleMoney) CreditForReservation(context.Context, string) (*model.ReservationCredit, error) {
	return nil, apperr.ErrNotFound
}

type lifecycleStore struct {
	*reservationTestStore
	evidence               []model.ReservationEvidence
	refund                 *model.ReservationRefund
	credit                 *model.ReservationCredit
	locked                 bool
	beforeLock             func()
	failMoney, failRelease bool
}
type lifecycleSnapshot struct {
	Reservations map[string]model.Reservation
	Claims       map[string][]model.SupplyClaim
	Holds        map[string]string
	Associations []model.SupplyClaimAllocation
	Events       int
	Evidence     []model.ReservationEvidence
	Refund       *model.ReservationRefund
}

func (s *lifecycleStore) InTx(_ context.Context, fn func(Store) error) error {
	raw, _ := json.Marshal(lifecycleSnapshot{Reservations: s.reservations, Claims: s.claims, Holds: s.holds, Associations: s.associations, Events: s.events, Evidence: s.evidence, Refund: s.refund})
	var before lifecycleSnapshot
	_ = json.Unmarshal(raw, &before)
	s.locked = false
	if err := fn(s); err != nil {
		s.reservations, s.claims, s.holds, s.associations, s.events, s.evidence, s.refund = before.Reservations, before.Claims, before.Holds, before.Associations, before.Events, before.Evidence, before.Refund
		s.locked = false
		return err
	}
	s.locked = false
	return nil
}

type lifecycleBoundKey struct{}

func (s *lifecycleStore) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, lifecycleBoundKey{}, true)
}

func (s *lifecycleStore) Reservations() ReservationRepository {
	return lifecycleReservationRepo{reservationTestRepo{s: s.reservationTestStore}, s}
}

func (s *lifecycleStore) ReservationPayments() ReservationPaymentRepository {
	return lifecycleMoneyRepo{ReservationPaymentRepository: emptyLifecycleMoney{}, s: s}
}

type lifecycleReservationRepo struct {
	reservationTestRepo
	s *lifecycleStore
}

func (r lifecycleReservationRepo) ForUpdate(ctx context.Context, id string) (*model.Reservation, []model.ReservationLine, error) {
	if r.s.beforeLock != nil {
		f := r.s.beforeLock
		r.s.beforeLock = nil
		f()
	}
	x, lines, err := r.reservationTestRepo.ForUpdate(ctx, id)
	if err == nil {
		r.s.locked = true
	}
	return x, lines, err
}

func (r lifecycleReservationRepo) DueCandidates(ctx context.Context, now time.Time, after *time.Time, afterID string, limit int) ([]model.ReservationDueCandidate, error) {
	var out []model.ReservationDueCandidate
	for id, x := range r.s.reservations {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if x.Status != model.ReservationActive || x.OrderID != nil || x.ExpiresAt == nil {
			continue
		}
		deadline := *x.ExpiresAt
		if x.AdvanceDueAt != nil && x.AdvanceDueAt.Before(deadline) {
			deadline = *x.AdvanceDueAt
		}
		if deadline.After(now) || after != nil && (deadline.Before(*after) || deadline.Equal(*after) && id <= afterID) {
			continue
		}
		out = append(out, model.ReservationDueCandidate{ID: id, Deadline: deadline})
	}
	slices.SortFunc(out, func(a, b model.ReservationDueCandidate) int {
		if d := a.Deadline.Compare(b.Deadline); d != 0 {
			return d
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type lifecycleMoneyRepo struct {
	ReservationPaymentRepository
	s *lifecycleStore
}

func (r lifecycleMoneyRepo) check() error {
	if !r.s.locked {
		return errors.New("money read without row lock")
	}
	if r.s.failMoney {
		return errors.New("money read failed")
	}
	return nil
}

func (r lifecycleMoneyRepo) Evidence(_ context.Context, id string) ([]model.ReservationEvidence, error) {
	if err := r.check(); err != nil {
		return nil, err
	}
	out := []model.ReservationEvidence{}
	for _, e := range r.s.evidence {
		if e.ReservationID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r lifecycleMoneyRepo) Refund(_ context.Context, id string) (*model.ReservationRefund, error) {
	if err := r.check(); err != nil {
		return nil, err
	}
	if r.s.refund == nil || r.s.refund.ReservationID != id {
		return nil, apperr.ErrNotFound
	}
	return r.s.refund, nil
}

func (r lifecycleMoneyRepo) CreditForReservation(_ context.Context, id string) (*model.ReservationCredit, error) {
	if err := r.check(); err != nil {
		return nil, err
	}
	if r.s.credit == nil || r.s.credit.ReservationID != id {
		return nil, apperr.ErrNotFound
	}
	return r.s.credit, nil
}

type lifecycleStock struct {
	reservationTestStock
	s *lifecycleStore
}

func (x lifecycleStock) ReleaseReservation(ctx context.Context, id string, ids []string, _ string) error {
	if ctx.Value(lifecycleBoundKey{}) != true {
		return errors.New("inventory context not bound")
	}
	if x.s.failRelease {
		return errors.New("release failed")
	}
	for _, vin := range ids {
		if x.s.holds[vin] != id {
			return apperr.ErrConflict
		}
		delete(x.s.holds, vin)
	}
	return nil
}

type reservationLifecycleSetup struct {
	service       *Reservation
	store         *lifecycleStore
	buyer         *auth.Principal
	supplier      *auth.Principal
	reservationID string
	lotID         string
	vehicleID     string
}

func lifecycleFixture(t *testing.T, paid bool) reservationLifecycleSetup {
	t.Helper()
	setup := reservationFixture()
	s, base, buyer, supplier, lot, vin := setup.service, setup.store, setup.dealer, setup.supplier, setup.lotID, setup.vehicleID
	terms := reservationTerms()
	if paid {
		advance := money.Money{AmountMinor: "100", Currency: "USD"}
		due := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
		terms.Advance = &advance
		terms.AdvanceDueAt = &due
	}
	v, err := s.Request(context.Background(), buyer, ReservationRequest{SupplierCompanyID: supplier.CompanyID, Lines: []ReservationLineInput{{SupplyLotID: &lot, Quantity: 1}, {VehicleID: &vin, Quantity: 1}}, DesiredTerms: terms})
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.Confirm(context.Background(), supplier, v.Reservation.ID, v.Reservation.Version, ReservationConfirmation{Terms: terms})
	if err != nil {
		t.Fatal(err)
	}
	if v.Reservation.Status != model.ReservationActive {
		t.Fatalf("activation: %s", v.Reservation.Status)
	}
	st := &lifecycleStore{reservationTestStore: base}
	s.store = st
	s.reservationStock = lifecycleStock{reservationTestStock: reservationTestStock{store: base}, s: st}
	return reservationLifecycleSetup{
		service:       s,
		store:         st,
		buyer:         buyer,
		supplier:      supplier,
		reservationID: v.Reservation.ID,
		lotID:         lot,
		vehicleID:     vin,
	}
}

func lifecycleEvidence(st *lifecycleStore, id, status string) {
	r := st.reservations[id]
	st.evidence = append(st.evidence, model.ReservationEvidence{ID: uuid.NewString(), ReservationID: id, TermsDigest: r.CurrentDigest, TermsRevision: r.CurrentRevision, AmountMinor: "100", Currency: "USD", Status: status})
}

func TestReservationExpiry(t *testing.T) {
	ctx := context.Background()
	t.Run("free due releases physical and held only once", func(t *testing.T) {
		setup := lifecycleFixture(t, false)
		s, st, id, lot, vin := setup.service, setup.store, setup.reservationID, setup.lotID, setup.vehicleID
		s.now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
		out, err := s.ExpireDue(ctx, 100, "")
		if err != nil || out.Expired != 1 || st.holds[vin] != "" || st.claims[lot][0].State != "released" || st.reservations[id].Status != model.ReservationExpired {
			t.Fatalf("expiry=%+v err=%v state=%+v", out, err, st.reservations[id])
		}
		if st.reservations[id].ReleaseReason == "" || st.reservations[id].ReleasedAt == nil {
			t.Fatal("release audit missing")
		}
		out, err = s.ExpireDue(ctx, 100, "")
		if err != nil || out.Expired != 0 || st.events != 3 {
			t.Fatalf("repeat=%+v err=%v events=%d", out, err, st.events)
		}
	})
	t.Run("advance deadline and rejected history", func(t *testing.T) {
		setup := lifecycleFixture(t, true)
		s, st, id, vin := setup.service, setup.store, setup.reservationID, setup.vehicleID
		s.now = func() time.Time { return time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC) }
		lifecycleEvidence(st, id, "rejected")
		out, err := s.ExpireDue(ctx, 100, "")
		if err != nil || out.Protected != 1 || st.holds[vin] == "" {
			t.Fatalf("rejected protected=%+v %v", out, err)
		}
		st.evidence = nil
		out, err = s.ExpireDue(ctx, 100, "")
		if err != nil || out.Expired != 1 {
			t.Fatalf("never submitted=%+v %v", out, err)
		}
	})
	t.Run("selection stale after extension or submission", func(t *testing.T) {
		setup := lifecycleFixture(t, true)
		s, st, id, vin := setup.service, setup.store, setup.reservationID, setup.vehicleID
		s.now = func() time.Time { return time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC) }
		st.beforeLock = func() {
			r := st.reservations[id]
			next := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
			r.AdvanceDueAt = &next
			st.reservations[id] = r
		}
		out, err := s.ExpireDue(ctx, 100, "")
		if err != nil || out.Protected != 1 || st.holds[vin] == "" {
			t.Fatalf("extended=%+v %v", out, err)
		}
		st.beforeLock = func() { lifecycleEvidence(st, id, "submitted") }
		s.now = func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }
		out, err = s.ExpireDue(ctx, 100, "")
		if err != nil || out.Protected != 1 || st.holds[vin] == "" {
			t.Fatalf("submitted=%+v %v", out, err)
		}
	})
}

func TestReservationExtension(t *testing.T) {
	setup := lifecycleFixture(t, true)
	s, st, buyer, supplier, id := setup.service, setup.store, setup.buyer, setup.supplier, setup.reservationID
	before := st.reservations[id]
	next := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	if _, err := s.Extend(context.Background(), supplier, id, before.Version, ReservationExtension{ExpiresAt: next, Reason: "supplier approved"}); err != nil {
		t.Fatal(err)
	}
	after := st.reservations[id]
	if after.CurrentDigest != before.CurrentDigest || after.CurrentRevision != before.CurrentRevision || !bytes.Equal(after.CurrentTerms, before.CurrentTerms) || !after.AdvanceDueAt.Equal(*before.AdvanceDueAt) || !after.ExpiresAt.Equal(next) {
		t.Fatalf("accepted source changed: %+v", after)
	}
	if after.ExtendedBy == nil || after.ExtendedAt == nil {
		t.Fatal("extension audit missing")
	}
	if _, err := s.Extend(context.Background(), buyer, id, after.Version, ReservationExtension{ExpiresAt: next.Add(time.Hour), Reason: "buyer attempt"}); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("buyer extension: %v", err)
	}
	if _, err := s.Extend(context.Background(), supplier, id, before.Version, ReservationExtension{ExpiresAt: next.Add(time.Hour), Reason: "stale"}); !errors.Is(err, apperr.ErrStale) {
		t.Fatalf("stale revision: %v", err)
	}
}

func TestReservationCancellation(t *testing.T) {
	t.Run("confirmed full refund required", func(t *testing.T) {
		setup := lifecycleFixture(t, true)
		s, st, buyer, id, lot, vin := setup.service, setup.store, setup.buyer, setup.reservationID, setup.lotID, setup.vehicleID
		lifecycleEvidence(st, id, "accepted")
		if _, err := s.Cancel(context.Background(), buyer, id, st.reservations[id].Version, "buyer requested"); err == nil {
			t.Fatal("accepted advance cancelled")
		}
		st.refund = &model.ReservationRefund{ReservationID: id, Status: "requested"}
		if _, err := s.Cancel(context.Background(), buyer, id, st.reservations[id].Version, "buyer requested"); err == nil {
			t.Fatal("refund request treated as payment")
		}
		st.refund.Status = "confirmed"
		st.refund.AmountMinor = "100"
		st.refund.Currency = "USD"
		v, err := s.Cancel(context.Background(), buyer, id, st.reservations[id].Version, "buyer requested")
		if err != nil || v.Reservation.Status != model.ReservationCancelled || st.holds[vin] != "" || st.claims[lot][0].State != "released" {
			t.Fatalf("cancel=%+v err=%v", v, err)
		}
	})
	t.Run("pending and conversion protect holds", func(t *testing.T) {
		setup := lifecycleFixture(t, true)
		s, st, buyer, id, vin := setup.service, setup.store, setup.buyer, setup.reservationID, setup.vehicleID
		lifecycleEvidence(st, id, "submitted")
		if _, err := s.Cancel(context.Background(), buyer, id, st.reservations[id].Version, "cancel"); err == nil || st.holds[vin] == "" {
			t.Fatalf("pending changed hold: %v", err)
		}
		st.evidence = nil
		r := st.reservations[id]
		orderID := uuid.NewString()
		r.OrderID = &orderID
		st.reservations[id] = r
		if _, err := s.Cancel(context.Background(), buyer, id, r.Version, "cancel"); err == nil || st.holds[vin] == "" {
			t.Fatalf("converted changed hold: %v", err)
		}
	})
}

func TestReservationLifecycle(t *testing.T) {
	t.Run("cancellation during attempt returns progress without expiry", func(t *testing.T) {
		setup := lifecycleFixture(t, false)
		s, st, id, lot, vin := setup.service, setup.store, setup.reservationID, setup.lotID, setup.vehicleID
		s.now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
		ctx, cancel := context.WithCancel(context.Background())
		st.beforeLock = cancel
		out, err := s.ExpireDue(ctx, 1, "")
		if !errors.Is(err, context.Canceled) || out.Selected != 1 || out.Expired != 0 || out.Failed != 1 || out.NextCursor == "" {
			t.Fatalf("cancelled attempt=%+v %v", out, err)
		}
		if st.reservations[id].Status != model.ReservationActive || st.claims[lot][0].State != "held" || st.holds[vin] == "" {
			t.Fatal("cancelled attempt changed hold")
		}
	})
	t.Run("failed first candidate preserves cursor for next due row", func(t *testing.T) {
		setup := lifecycleFixture(t, false)
		s, st, id := setup.service, setup.store, setup.reservationID
		s.now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
		second := st.reservations[id]
		second.ID = uuid.NewString()
		deadline := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
		second.ExpiresAt = &deadline
		st.reservations[second.ID] = second
		st.failMoney = true
		out, err := s.ExpireDue(context.Background(), 1, "")
		if err == nil || out.Failed != 1 || out.NextCursor == "" {
			t.Fatalf("first failure=%+v %v", out, err)
		}
		st.failMoney = false
		s.now = func() time.Time { return time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC) }
		out, err = s.ExpireDue(context.Background(), 1, out.NextCursor)
		if err != nil || out.Expired != 1 || st.reservations[id].Status != model.ReservationActive || st.reservations[second.ID].Status != model.ReservationExpired {
			t.Fatalf("later row=%+v %v", out, err)
		}
	})
	t.Run("materialized association retires without reopening future capacity", func(t *testing.T) {
		setup := lifecycleFixture(t, false)
		s, st, buyer, id, lot, vin := setup.service, setup.store, setup.buyer, setup.reservationID, setup.lotID, setup.vehicleID
		claim := st.claims[lot][0]
		claim.State = "materialized"
		associationID := uuid.NewString()
		claim.MaterializationKind = "vin-allocation"
		claim.MaterializationRef = &associationID
		ordinal := int32(1)
		claim.MaterializationUnitOrdinal = &ordinal
		now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
		claim.MaterializedAt = &now
		st.claims[lot][0] = claim
		st.associations = []model.SupplyClaimAllocation{{ID: associationID, ClaimID: claim.ID, ReservationLineID: *claim.ReservationLineID, VehicleID: vin, HolderType: "commerce-reservation", HolderID: id, Current: true}}
		if _, err := s.Cancel(context.Background(), buyer, id, st.reservations[id].Version, "cancel"); err != nil {
			t.Fatal(err)
		}
		if st.claims[lot][0].State != "materialized" || st.claims[lot][0].MaterializationRef == nil || st.associations[0].Current || st.holds[vin] != "" {
			t.Fatalf("materialized release=%+v assoc=%+v", st.claims[lot][0], st.associations[0])
		}
	})
	t.Run("failed release rolls back claims and state", func(t *testing.T) {
		setup := lifecycleFixture(t, false)
		s, st, buyer, id, lot, vin := setup.service, setup.store, setup.buyer, setup.reservationID, setup.lotID, setup.vehicleID
		st.failRelease = true
		if _, err := s.Cancel(context.Background(), buyer, id, st.reservations[id].Version, "cancel"); err == nil {
			t.Fatal("release failure accepted")
		}
		if st.reservations[id].Status != model.ReservationActive || st.claims[lot][0].State != "held" || st.holds[vin] == "" {
			t.Fatal("release failure did not roll back")
		}
	})
	t.Run("claim and event failures roll back", func(t *testing.T) {
		for _, fail := range []string{"claim", "event"} {
			setup := lifecycleFixture(t, false)
			s, st, buyer, id, lot, vin := setup.service, setup.store, setup.buyer, setup.reservationID, setup.lotID, setup.vehicleID
			if fail == "claim" {
				st.failClaim = true
			} else {
				st.failEvent = true
			}
			if _, err := s.Cancel(context.Background(), buyer, id, st.reservations[id].Version, "cancel"); err == nil {
				t.Fatalf("%s failure accepted", fail)
			}
			if st.reservations[id].Status != model.ReservationActive || st.claims[lot][0].State != "held" || st.holds[vin] == "" {
				t.Fatalf("%s failure did not roll back", fail)
			}
		}
	})
	t.Run("protected first page advances then wraps", func(t *testing.T) {
		setup := lifecycleFixture(t, true)
		s, st, id := setup.service, setup.store, setup.reservationID
		s.now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
		lifecycleEvidence(st, id, "rejected")
		second := st.reservations[id]
		second.ID = uuid.NewString()
		free := reservationTerms()
		free.ExpiresAt = func() *time.Time { d := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC); return &d }()
		raw, _ := json.Marshal(ReservationSnapshot{Terms: free})
		second.CurrentTerms = raw
		second.AdvanceDueAt = nil
		second.ExpiresAt = free.ExpiresAt
		st.reservations[second.ID] = second
		out, err := s.ExpireDue(context.Background(), 1, "")
		if err != nil || out.Protected != 1 || out.NextCursor == "" {
			t.Fatalf("first=%+v %v", out, err)
		}
		out, err = s.ExpireDue(context.Background(), 1, out.NextCursor)
		if err != nil || out.Expired != 1 || out.NextCursor == "" {
			t.Fatalf("second=%+v %v", out, err)
		}
		out, err = s.ExpireDue(context.Background(), 1, out.NextCursor)
		if err != nil || out.NextCursor != "" {
			t.Fatalf("wrap=%+v %v", out, err)
		}
	})
}
