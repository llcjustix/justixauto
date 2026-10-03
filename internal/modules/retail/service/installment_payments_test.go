package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/google/uuid"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

// These fakes assert serialization order, not real PostgreSQL concurrency.
type paymentDeals struct {
	*installmentDeals
	evidence     []model.Evidence
	inTx, locked bool
	onLock       func()
	reads        int
	dealLocked   bool
	locks        []string
	onDealLock   func()
	groups       []model.InstallmentPayment
	shares       []string
	addCalls     int
	failAddAt    int
	updateCalls  int
	failUpdateAt int
	failParent   bool
}

func (r *paymentDeals) LockDeal(ctx context.Context, company, id string) (*model.Deal, error) {
	if !r.inTx || r.locked {
		panic("deal lock outside transaction or after invoice lock")
	}
	r.dealLocked = true
	r.locks = append(r.locks, "deal")
	if r.onDealLock != nil {
		r.onDealLock()
		r.onDealLock = nil
	}
	return r.Deal(ctx, company, id)
}
func (r *paymentDeals) LockInstallmentInvoices(ctx context.Context, company, dealID string, ids []string) ([]model.Invoice, error) {
	if !r.dealLocked || !sort.StringsAreSorted(ids) {
		panic("monthly lock requires deal first and stable order")
	}
	rows := []model.Invoice{}
	for _, id := range ids {
		i, err := r.Invoice(ctx, company, id)
		if err != nil || i.DealID != dealID || i.Purpose != "monthly-installment" {
			continue
		}
		i, err = r.LockInvoice(ctx, company, id)
		if err != nil {
			return nil, err
		}
		rows = append(rows, *i)
	}
	return rows, nil
}
func (r *paymentDeals) CreateInstallmentPayment(ctx context.Context, p *model.InstallmentPayment) error {
	if !r.dealLocked || !r.locked {
		panic("group insert before locks")
	}
	if _, err := r.InstallmentPaymentByReference(ctx, p.CompanyID, p.DealID, p.ExternalReference); err == nil {
		return apperr.ErrConflict
	}
	r.groups = append(r.groups, *p)
	return nil
}
func (r *paymentDeals) InstallmentPaymentByReference(_ context.Context, company, dealID, ref string) (*model.InstallmentPayment, error) {
	for _, p := range r.groups {
		if p.CompanyID == company && p.DealID == dealID && p.ExternalReference == ref && p.Status != "rejected" {
			return &p, nil
		}
	}
	return nil, apperr.ErrNotFound
}

func (r *paymentDeals) Deal(_ context.Context, company, id string) (*model.Deal, error) {
	if r.deal.ID != id || r.deal.CompanyID != company {
		return nil, apperr.ErrNotFound
	}
	return r.deal, nil
}
func (r *paymentDeals) Invoice(_ context.Context, company, id string) (*model.Invoice, error) {
	for _, i := range r.invoices {
		if i.ID == id && i.CompanyID == company {
			return &i, nil
		}
	}
	return nil, apperr.ErrNotFound
}
func (r *paymentDeals) LockInvoice(ctx context.Context, company, id string) (*model.Invoice, error) {
	if !r.inTx {
		panic("lock outside transaction")
	}
	i, err := r.Invoice(ctx, company, id)
	if err != nil {
		return nil, err
	}
	r.locked = true
	r.locks = append(r.locks, id)
	if r.onLock != nil {
		r.onLock()
		r.onLock = nil
	}
	return i, nil
}
func (r *paymentDeals) Evidence(_ context.Context, invoice string) ([]model.Evidence, error) {
	if r.inTx && !r.locked {
		panic("balance read before invoice lock")
	}
	r.reads++
	out := []model.Evidence{}
	for _, e := range r.evidence {
		if e.InvoiceID == invoice {
			out = append(out, e)
		}
	}
	return out, nil
}
func (r *paymentDeals) GetEvidence(_ context.Context, id string) (*model.Evidence, error) {
	for _, e := range r.evidence {
		if e.ID == id {
			return &e, nil
		}
	}
	return nil, apperr.ErrNotFound
}
func (r *paymentDeals) AddEvidence(_ context.Context, e *model.Evidence) error {
	if !r.locked {
		panic("insert before lock")
	}
	r.addCalls++
	if r.addCalls == r.failAddAt {
		return errors.New("allocation insert failed")
	}
	for _, old := range r.evidence {
		if old.InvoiceID == e.InvoiceID && old.ExternalReference == e.ExternalReference && old.Status != "rejected" {
			return apperr.ErrConflict
		}
	}
	r.evidence = append(r.evidence, *e)
	return nil
}
func (r *paymentDeals) UpdateEvidence(_ context.Context, e *model.Evidence, expected int64) error {
	if !r.locked {
		panic("decision before lock")
	}
	r.updateCalls++
	if r.updateCalls == r.failUpdateAt {
		return errors.New("allocation update failed")
	}
	for n, old := range r.evidence {
		if old.ID == e.ID {
			if old.Version != expected {
				return apperr.ErrStale
			}
			e.Version = expected + 1
			r.evidence[n] = *e
			return nil
		}
	}
	return apperr.ErrNotFound
}

type paymentEvents struct{ *installmentEvents }

func (r paymentEvents) For(context.Context, string, string) ([]model.Event, error) {
	return r.events, nil
}

type paymentStore struct {
	*installmentStore
	repo *paymentDeals
}

type paymentTxKey struct{}

func (s *paymentStore) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, paymentTxKey{}, s.repo)
}

func (s *paymentStore) Deals() DealRepository   { return s.repo }
func (s *paymentStore) Events() EventRepository { return paymentEvents{s.events} }
func (s *paymentStore) InTx(_ context.Context, fn func(Store) error) error {
	if s.repo.inTx {
		return fn(s)
	}
	s.repo.inTx, s.repo.locked, s.repo.dealLocked = true, false, false
	defer func() { s.repo.inTx, s.repo.locked, s.repo.dealLocked = false, false, false }()
	type state struct {
		Evidence []model.Evidence
		Groups   []model.InstallmentPayment
		Events   []model.Event
		Shares   []string
		Deal     model.Deal
	}
	raw, _ := json.Marshal(state{s.repo.evidence, s.repo.groups, s.events.events, s.repo.shares, *s.repo.deal})
	var old state
	if err := json.Unmarshal(raw, &old); err != nil {
		return err
	}
	err := fn(s)
	if err != nil {
		s.repo.evidence, s.repo.groups, s.events.events, s.repo.shares = old.Evidence, old.Groups, old.Events, old.Shares
		*s.repo.deal = old.Deal
	}
	return err
}

type paymentInsurance struct{}

func (paymentInsurance) Approved(context.Context, string, string) (bool, error) { return true, nil }

func paymentFixture() (*Deal, *paymentStore, *auth.Principal, string) {
	s, old, p := installmentFixture(false)
	planID, rowID, otherID := "00000000-0000-0000-0000-000000000007", "00000000-0000-0000-0000-000000000008", "00000000-0000-0000-0000-000000000009"
	old.deals.plan = &model.InstallmentPlan{ID: planID, DealID: old.deals.deal.ID, CompanyID: p.CompanyID, Currency: "USD", DownPaymentInvoiceID: old.deals.invoices[0].ID, DownPaymentMinor: "800000"}
	for n, id := range []string{rowID, otherID} {
		number := n + 1
		old.deals.invoices = append(old.deals.invoices, model.Invoice{ID: id, CompanyID: p.CompanyID, DealID: old.deals.deal.ID, Purpose: "monthly-installment", AmountMinor: "100", Currency: "USD", Status: "issued", InstallmentPlanID: &planID, InstallmentNumber: &number})
	}
	st := &paymentStore{installmentStore: old, repo: &paymentDeals{installmentDeals: old.deals}}
	s.store, s.insurance = st, paymentInsurance{}
	p.Permissions[model.PermPaymentsAccept] = true
	return s, st, p, rowID
}
func paymentInput(amount, ref string) EvidenceInput {
	return EvidenceInput{ClaimedAmount: money.Money{AmountMinor: amount, Currency: "USD"}, PaidOn: "2026-09-30", ExternalReference: ref}
}

// Seed historical ungrouped evidence explicitly; new submissions are groups.
func legacyPayment(s *Deal, st *paymentStore, p *auth.Principal, id, value, ref string) (*InvoiceView, error) {
	e := model.Evidence{ID: uuid.NewString(), InvoiceID: id, AmountMinor: value, Currency: "USD", ExternalReference: ref,
		Status: "submitted", Version: 1, SubmittedBy: p.UserID}
	st.repo.evidence = append(st.repo.evidence, e)
	return s.Invoice(context.Background(), p, id)
}
func paymentBalance(t *testing.T, v *InvoiceView, paid, pending, outstanding, available string) {
	t.Helper()
	got := []string{v.Paid.AmountMinor, v.Pending.AmountMinor, v.Outstanding.AmountMinor, v.Available.AmountMinor}
	if !reflect.DeepEqual(got, []string{paid, pending, outstanding, available}) {
		t.Fatalf("balance %v", got)
	}
}

func TestInstallmentPaymentPartialLifecycle(t *testing.T) {
	s, st, p, id := paymentFixture()
	ctx := context.Background()
	v, err := legacyPayment(s, st, p, id, "60", "first")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "0", "60", "100", "40")
	if _, err := s.SubmitEvidence(ctx, p, id, paymentInput("41", "over")); err == nil {
		t.Fatal("overclaim allowed")
	}
	if _, err := s.SubmitEvidence(ctx, p, id, paymentInput("1", "first")); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	v, err = s.DecideEvidence(ctx, p, v.Evidence[0].ID, 1, true, true, "")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "60", "0", "40", "40")
	if v.Evidence[0].DecidedBy == nil || *v.Evidence[0].DecidedBy != p.UserID || v.Evidence[0].DecidedAt == nil {
		t.Fatal("decision author/time missing")
	}
	v, err = legacyPayment(s, st, p, id, "40", "second")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "60", "40", "40", "0")
	if len(v.AllowedActions) != 0 {
		t.Fatal("fully reserved row is actionable")
	}
	if _, err := s.SubmitEvidence(ctx, p, id, paymentInput("1", "third")); err == nil {
		t.Fatal("pending coverage ignored")
	}
	v, err = s.DecideEvidence(ctx, p, v.Evidence[1].ID, 1, false, false, "not received")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "60", "0", "40", "40")
	if v.Evidence[1].DecisionReason != "not received" {
		t.Fatal("reason lost")
	}
	if _, err := s.SubmitEvidence(ctx, p, id, paymentInput("40", "second")); err != nil {
		t.Fatalf("rejected reference not reusable: %v", err)
	}
	other := st.repo.invoices[2].ID
	if _, err := s.SubmitEvidence(ctx, p, other, paymentInput("10", "first")); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("plan-wide legacy reference guard: %v", err)
	}
	down, err := s.Invoice(ctx, p, st.repo.invoices[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, down, "0", "0", "800000", "800000")
	full, err := s.Get(ctx, p, st.repo.deal.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan := full.InstallmentPlan
	if plan.Paid.AmountMinor != "60" || plan.Pending.AmountMinor != "40" || plan.Outstanding.AmountMinor != "140" || plan.ScheduledTotal.AmountMinor != "200" {
		t.Fatalf("monthly totals: %+v", plan)
	}
	for _, row := range plan.Rows {
		for _, invoice := range full.Invoices {
			if invoice.Invoice.ID == row.Invoice.Invoice.ID && !reflect.DeepEqual(invoice, row.Invoice) {
				t.Fatal("row differs from full invoice")
			}
		}
	}
}

func TestInstallmentPaymentReloadsDecisionAfterLock(t *testing.T) {
	for _, stale := range []bool{true, false} {
		t.Run(map[bool]string{true: "stale", false: "already-decided"}[stale], func(t *testing.T) {
			s, st, p, id := paymentFixture()
			ctx := context.Background()
			v, err := legacyPayment(s, st, p, id, "30", "one")
			if err != nil {
				t.Fatal(err)
			}
			st.repo.onLock = func() {
				st.repo.evidence[0].Status = "accepted"
				if stale {
					st.repo.evidence[0].Version++
				}
			}
			_, err = s.DecideEvidence(ctx, p, v.Evidence[0].ID, 1, true, true, "")
			want := apperr.ErrConflict
			if stale {
				want = apperr.ErrStale
			}
			if !errors.Is(err, want) {
				t.Fatalf("trusted pre-lock state: %v", err)
			}
		})
	}
}

func TestInstallmentPaymentEligibilityAndPermissions(t *testing.T) {
	for _, name := range []string{"permission", "company", "branch", "cancelled", "scheme", "link", "missing-plan", "currency", "void"} {
		t.Run(name, func(t *testing.T) {
			s, st, p, id := paymentFixture()
			ctx := context.Background()
			v, err := legacyPayment(s, st, p, id, "10", "existing")
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "permission":
				p.Permissions = map[string]bool{model.PermRead: true}
			case "company":
				p.CompanyID = "foreign"
			case "branch":
				p.BranchScope = auth.BranchScope{Mode: "SELECTED", BranchIDs: []string{"foreign"}}
			case "cancelled":
				st.repo.deal.Status = "cancelled"
			case "scheme":
				st.repo.deal.PaymentScheme = "cash"
			case "link":
				bad := "foreign"
				st.repo.invoices[1].InstallmentPlanID = &bad
			case "missing-plan":
				st.repo.plan = nil
			case "currency":
				st.repo.invoices[1].Currency = "EUR"
			case "void":
				st.repo.invoices[1].Status = "void"
			}
			if _, err := s.SubmitEvidence(ctx, p, id, paymentInput("10", "next")); err == nil {
				t.Fatal("submit accepted")
			}
			for _, accept := range []bool{true, false} {
				if _, err := s.DecideEvidence(ctx, p, v.Evidence[0].ID, 1, accept, true, "not received"); err == nil {
					t.Fatal("decision accepted")
				}
			}
			read, err := s.Invoice(ctx, p, id)
			if name == "company" || name == "branch" {
				if err == nil {
					t.Fatal("foreign invoice visible")
				}
				return
			}
			if err != nil || len(read.AllowedActions) != 0 || len(read.EvidenceActions) != 0 {
				t.Fatalf("ineligible actions: %+v %v", read, err)
			}
		})
	}
}

func TestInstallmentPaymentActorActions(t *testing.T) {
	s, st, p, id := paymentFixture()
	ctx := context.Background()
	v, err := legacyPayment(s, st, p, id, "20", "one")
	if err != nil {
		t.Fatal(err)
	}
	evidenceID := v.Evidence[0].ID
	for _, perm := range []string{model.PermRead, model.PermDeals, model.PermPaymentsAccept} {
		p.Permissions = map[string]bool{perm: true}
		v, err = s.Invoice(ctx, p, id)
		if err != nil {
			t.Fatal(err)
		}
		if (len(v.AllowedActions) == 1) != (perm == model.PermDeals) || (len(v.EvidenceActions[evidenceID]) == 2) != (perm == model.PermPaymentsAccept) {
			t.Fatalf("wrong actions for %s: %+v", perm, v)
		}
	}
	v, err = s.DecideEvidence(ctx, p, evidenceID, 1, true, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(v.EvidenceActions) != 0 {
		t.Fatal("decided evidence actionable")
	}
	p.Permissions[model.PermDeals] = true
	v, err = legacyPayment(s, st, p, id, "80", "rest")
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.DecideEvidence(ctx, p, v.Evidence[1].ID, 1, true, true, "")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "100", "0", "0", "0")
	if len(v.AllowedActions) != 0 {
		t.Fatal("paid row actionable")
	}
}

func TestInstallmentPaymentExactAmountValidationAndNonmonthly(t *testing.T) {
	s, st, p, id := paymentFixture()
	ctx := context.Background()
	st.repo.invoices[1].AmountMinor = "90071992547409931234"
	v, err := s.SubmitEvidence(ctx, p, id, paymentInput("90071992547409931233", "large"))
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "0", "90071992547409931233", "90071992547409931234", "1")
	for _, invalid := range []EvidenceInput{paymentInput("0", "zero"), paymentInput("-1", "negative"), paymentInput("1.1", "fraction"), {ClaimedAmount: money.Money{AmountMinor: "1", Currency: "EUR"}, PaidOn: "2026-09-30", ExternalReference: "currency"}} {
		if _, err := s.SubmitEvidence(ctx, p, id, invalid); err == nil {
			t.Fatal("invalid money accepted")
		}
	}
	if _, err := s.DecideEvidence(ctx, p, v.Evidence[0].ID, 1, true, false, ""); err == nil {
		t.Fatal("confirmation omitted")
	}
	if _, err := s.DecideEvidence(ctx, p, v.Evidence[0].ID, 1, false, false, ""); err == nil {
		t.Fatal("reason omitted")
	}
	st.repo.deal.Status, st.repo.deal.PaymentScheme, st.repo.plan = "reserved", "cash", nil
	id = st.repo.invoices[0].ID
	v, err = s.SubmitEvidence(ctx, p, id, paymentInput("20", "ordinary"))
	if err != nil {
		t.Fatalf("nonmonthly submit: %v", err)
	}
	v, err = s.DecideEvidence(ctx, p, v.Evidence[0].ID, 1, true, true, "")
	if err != nil {
		t.Fatal(err)
	}
	paymentBalance(t, v, "20", "0", "799980", "799980")
}

type paymentFiles struct {
	repo       *paymentDeals
	fail       bool
	evidenceID string
}

func (f *paymentFiles) Share(ctx context.Context, owner, file, company, resource, id string) error {
	if ctx.Value(paymentTxKey{}) != f.repo {
		panic("file share outside ambient transaction")
	}
	if !f.repo.locked || owner != f.repo.deal.CompanyID || company != owner || resource != "retail.payment-evidence" || file == "" {
		panic("invalid attachment sharing scope/order")
	}
	f.evidenceID = id
	if f.fail {
		return apperr.ErrNotFound
	}
	f.repo.shares = append(f.repo.shares, id)
	return nil
}
func TestInstallmentPaymentAttachmentAndAuditAtomicity(t *testing.T) {
	s, st, p, id := paymentFixture()
	f := &paymentFiles{repo: st.repo, fail: true}
	s.files = f
	in := paymentInput("10", "file-payment")
	in.AttachmentIDs = []string{"00000000-0000-0000-0000-000000000010"}
	if _, err := s.SubmitEvidence(context.Background(), p, id, in); err == nil || len(st.repo.evidence) != 0 || len(st.events.events) != 0 {
		t.Fatal("failed file claim persisted")
	}
	f.fail = false
	v, err := s.SubmitEvidence(context.Background(), p, id, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.events.events) != 1 || st.events.events[0].EventType != "deal.payment_submitted" || v.Evidence[0].ID != f.evidenceID || string(v.Evidence[0].AttachmentIDs) != `["00000000-0000-0000-0000-000000000010"]` {
		t.Fatal("attachment or audit association missing")
	}
}
