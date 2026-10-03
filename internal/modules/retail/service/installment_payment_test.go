package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/money"
)

func (r *paymentDeals) InstallmentPayment(_ context.Context, company, id string) (*model.InstallmentPayment, error) {
	for _, p := range r.groups {
		if p.CompanyID == company && p.ID == id {
			return &p, nil
		}
	}
	return nil, apperr.ErrNotFound
}

func (r *paymentDeals) InstallmentPaymentEvidence(_ context.Context, id string) ([]model.Evidence, error) {
	out := []model.Evidence{}
	for _, e := range r.evidence {
		if e.PaymentGroupID != nil && *e.PaymentGroupID == id {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InvoiceID < out[j].InvoiceID })
	return out, nil
}

func (r *paymentDeals) UpdateInstallmentPayment(_ context.Context, parent *model.InstallmentPayment, expected int64) error {
	if !r.inTx || !r.dealLocked || !r.locked {
		panic("parent update before financial locks")
	}
	if r.failParent {
		return errors.New("parent update failed")
	}
	for i, old := range r.groups {
		if old.ID == parent.ID {
			if old.Version != expected {
				return apperr.ErrStale
			}
			parent.Version = expected + 1
			r.groups[i] = *parent
			return nil
		}
	}
	return apperr.ErrNotFound
}

func groupInput(st *paymentStore) InstallmentPaymentInput {
	return InstallmentPaymentInput{EvidenceInput: paymentInput("150", "receipt"), Allocations: []InstallmentPaymentAllocationInput{
		{InvoiceID: st.repo.invoices[2].ID, Amount: money.Money{AmountMinor: "100", Currency: "USD"}},
		{InvoiceID: st.repo.invoices[1].ID, Amount: money.Money{AmountMinor: "50", Currency: "USD"}},
	}}
}

func TestInstallmentPaymentGroupManualAdvance(t *testing.T) {
	for _, state := range []string{"reserved", "delivered"} {
		t.Run(state, func(t *testing.T) {
			s, st, p, id := paymentFixture()
			st.repo.deal.Status = state
			before, _ := json.Marshal(struct {
				Deal     *model.Deal
				Plan     *model.InstallmentPlan
				Invoices []model.Invoice
			}{st.repo.deal, st.repo.plan, st.repo.invoices})
			view, err := s.SubmitInstallmentPayment(context.Background(), p, st.repo.deal.ID, groupInput(st))
			if err != nil {
				t.Fatal(err)
			}
			if view.Payment.AmountMinor != "150" || view.Payment.Status != "submitted" || len(view.Allocations) != 2 || view.Allocations[0].Number != 2 || view.Allocations[1].Amount.AmountMinor != "50" {
				t.Fatalf("manual allocations changed: %+v", view)
			}
			if !reflect.DeepEqual(st.repo.locks, []string{"deal", id, st.repo.invoices[2].ID}) {
				t.Fatalf("lock order: %v", st.repo.locks)
			}
			after, _ := json.Marshal(struct {
				Deal     *model.Deal
				Plan     *model.InstallmentPlan
				Invoices []model.Invoice
			}{st.repo.deal, st.repo.plan, st.repo.invoices})
			if string(before) != string(after) {
				t.Fatal("saved terms or delivery state changed")
			}
			v, err := s.Invoice(context.Background(), p, id)
			if err != nil {
				t.Fatal(err)
			}
			paymentBalance(t, v, "0", "50", "100", "50")
			if len(v.EvidenceActions) != 0 {
				t.Fatal("group child exposes decision")
			}
			if _, err := s.DecideEvidence(context.Background(), p, v.Evidence[0].ID, 1, true, true, ""); !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("child accepted: %v", err)
			}
			if len(st.events.events) != 1 || st.events.events[0].ActorUserID != p.UserID {
				t.Fatal("audit missing")
			}
			var audit map[string]any
			if err := json.Unmarshal(st.events.events[0].Details, &audit); err != nil {
				t.Fatal(err)
			}
			if audit["paymentGroupId"] != view.Payment.ID || len(audit["evidenceIds"].([]any)) != 2 {
				t.Fatalf("incomplete audit: %v", audit)
			}
		})
	}
}

func TestInstallmentPaymentGroupValidation(t *testing.T) {
	for _, name := range []string{"empty", "too-many", "duplicate", "sum", "zero", "fraction", "currency", "claim-currency", "nonmonthly", "foreign", "unlinked", "cancelled", "draft", "company", "branch", "permission", "over-capacity"} {
		t.Run(name, func(t *testing.T) {
			s, st, p, _ := paymentFixture()
			in := groupInput(st)
			switch name {
			case "empty":
				in.Allocations = nil
			case "too-many":
				in.Allocations = make([]InstallmentPaymentAllocationInput, 1001)
			case "duplicate":
				in.Allocations[1].InvoiceID = in.Allocations[0].InvoiceID
			case "sum":
				in.ClaimedAmount.AmountMinor = "149"
			case "zero":
				in.Allocations[0].Amount.AmountMinor = "0"
			case "fraction":
				in.Allocations[0].Amount.AmountMinor = "1.5"
			case "currency":
				in.Allocations[0].Amount.Currency = "EUR"
			case "claim-currency":
				in.ClaimedAmount.Currency = "EUR"
				for i := range in.Allocations {
					in.Allocations[i].Amount.Currency = "EUR"
				}
			case "nonmonthly":
				in.Allocations[0].InvoiceID = st.repo.invoices[0].ID
			case "foreign":
				st.repo.invoices[2].CompanyID = "foreign"
			case "unlinked":
				st.repo.invoices[2].InstallmentPlanID = nil
			case "cancelled":
				st.repo.deal.Status = "cancelled"
			case "draft":
				st.repo.deal.Status, st.repo.plan = "reserved", nil
			case "company":
				p.CompanyID = "foreign"
			case "branch":
				p.BranchScope.Mode, p.BranchScope.BranchIDs = "SELECTED", []string{"foreign"}
			case "permission":
				delete(p.Permissions, model.PermDeals)
			case "over-capacity":
				in.Allocations[0].Amount.AmountMinor, in.ClaimedAmount.AmountMinor = "101", "151"
			}
			if _, err := s.SubmitInstallmentPayment(context.Background(), p, st.repo.deal.ID, in); err == nil {
				t.Fatal("invalid claim accepted")
			}
			if len(st.repo.groups)+len(st.repo.evidence)+len(st.events.events) != 0 {
				t.Fatal("invalid claim persisted")
			}
		})
	}
}

func TestInstallmentPaymentGroupReferences(t *testing.T) {
	for _, kind := range []string{"legacy", "group"} {
		for _, status := range []string{"submitted", "accepted", "rejected"} {
			t.Run(kind+"/"+status, func(t *testing.T) {
				s, st, p, id := paymentFixture()
				if kind == "legacy" {
					st.repo.evidence = []model.Evidence{{ID: "legacy", InvoiceID: id, AmountMinor: "10", Currency: "USD", Status: status, ExternalReference: "receipt"}}
				} else {
					st.repo.groups = []model.InstallmentPayment{{ID: "old", DealID: st.repo.deal.ID, CompanyID: p.CompanyID, Status: status, ExternalReference: "receipt"}}
				}
				// Disjoint row: the natural key is still protected across the plan.
				in := groupInput(st)
				in.Allocations, in.ClaimedAmount.AmountMinor = in.Allocations[:1], "100"
				in.ExternalReference = " receipt "
				_, err := s.SubmitInstallmentPayment(context.Background(), p, st.repo.deal.ID, in)
				if status == "rejected" {
					if err != nil {
						t.Fatal(err)
					}
				} else if !errors.Is(err, apperr.ErrConflict) {
					t.Fatalf("reference reused: %v", err)
				}
			})
		}
	}
	t.Run("retry-disjoint", func(t *testing.T) {
		s, st, p, id := paymentFixture()
		if _, err := s.SubmitEvidence(context.Background(), p, id, paymentInput("10", "same")); err != nil {
			t.Fatal(err)
		}
		if _, err := s.SubmitEvidence(context.Background(), p, st.repo.invoices[2].ID, paymentInput("10", "same")); !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("retry allocated elsewhere: %v", err)
		}
		if len(st.repo.groups) != 1 || len(st.repo.evidence) != 1 {
			t.Fatal("retry persisted")
		}
	})
}

func TestInstallmentPaymentGroupRollback(t *testing.T) {
	for _, fail := range []string{"file", "second-allocation", "audit"} {
		t.Run(fail, func(t *testing.T) {
			s, st, p, _ := paymentFixture()
			s.files = &paymentFiles{repo: st.repo, fail: fail == "file"}
			in := groupInput(st)
			in.AttachmentIDs = []string{"00000000-0000-0000-0000-000000000010"}
			if fail == "second-allocation" {
				st.repo.failAddAt = 2
			}
			if fail == "audit" {
				st.events.failType = "deal.payment_submitted"
			}
			if _, err := s.SubmitInstallmentPayment(context.Background(), p, st.repo.deal.ID, in); err == nil {
				t.Fatal("expected failure")
			}
			if len(st.repo.groups)+len(st.repo.evidence)+len(st.events.events)+len(st.repo.shares) != 0 {
				t.Fatal("partial transaction persisted")
			}
		})
	}
}

func TestInstallmentPaymentGroupPostLockCapacity(t *testing.T) {
	s, st, p, id := paymentFixture()
	st.repo.onLock = func() {
		st.repo.evidence = append(st.repo.evidence, model.Evidence{InvoiceID: id, AmountMinor: "60", Status: "submitted", ExternalReference: "competitor"})
	}
	if _, err := s.SubmitInstallmentPayment(context.Background(), p, st.repo.deal.ID, groupInput(st)); err == nil {
		t.Fatal("stale preview accepted")
	}
	if len(st.repo.groups) != 0 {
		t.Fatal("overclaim persisted")
	}
}

type paymentStock struct {
	Stock
	releases int
}

func (s *paymentStock) Release(context.Context, string, string) error { s.releases++; return nil }

func TestInstallmentPaymentGroupCancellationOrdering(t *testing.T) {
	for _, status := range []string{"submitted", "accepted"} {
		t.Run("payment-wins-"+status, func(t *testing.T) {
			s, st, p, id := paymentFixture()
			st.repo.deal.Status = "reserved"
			stock := &paymentStock{}
			s.stock = stock
			st.repo.onDealLock = func() {
				st.repo.evidence = append(st.repo.evidence, model.Evidence{InvoiceID: id, AmountMinor: "20", Status: status})
			}
			if _, err := s.Cancel(context.Background(), p, st.repo.deal.ID, st.repo.deal.Version, "cancel sale"); !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("cancel ignored payment: %v", err)
			}
			if stock.releases != 0 || st.repo.deal.Status != "reserved" {
				t.Fatal("cancel side effects")
			}
			want := []string{"deal", st.repo.invoices[0].ID, id, st.repo.invoices[2].ID}
			if !reflect.DeepEqual(st.repo.locks, want) {
				t.Fatalf("cancel lock order: %v", st.repo.locks)
			}
		})
	}
	t.Run("cancel-wins", func(t *testing.T) {
		s, st, p, id := paymentFixture()
		st.repo.deal.Status = "reserved"
		stock := &paymentStock{}
		s.stock = stock
		if _, err := s.Cancel(context.Background(), p, st.repo.deal.ID, st.repo.deal.Version, "cancel sale"); err != nil {
			t.Fatal(err)
		}
		if stock.releases != 1 || st.repo.deal.Status != "cancelled" {
			t.Fatal("cancel did not persist")
		}
		if _, err := s.SubmitEvidence(context.Background(), p, id, paymentInput("10", "late")); err == nil {
			t.Fatal("payment after cancel")
		}
	})
	t.Run("cancel-observed-after-lock", func(t *testing.T) {
		s, st, p, _ := paymentFixture()
		st.repo.onDealLock = func() { st.repo.deal.Status = "cancelled" }
		if _, err := s.SubmitInstallmentPayment(context.Background(), p, st.repo.deal.ID, groupInput(st)); err == nil {
			t.Fatal("trusted pre-lock sale state")
		}
	})
	t.Run("legacy-reject-then-cancel", func(t *testing.T) {
		s, st, p, id := paymentFixture()
		st.repo.deal.Status = "reserved"
		s.stock = &paymentStock{}
		v, err := legacyPayment(s, st, p, id, "20", "legacy")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.DecideEvidence(context.Background(), p, v.Evidence[0].ID, 1, false, false, "not received"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Cancel(context.Background(), p, st.repo.deal.ID, st.repo.deal.Version, "cancel sale"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInstallmentPaymentGroupDecisionValidation(t *testing.T) {
	for _, name := range []string{"permission", "company", "branch", "confirmation", "reason", "stale", "replay", "reload-version", "reload-state", "reload-children", "cancelled", "mixed-status", "sum", "currency", "reference", "files", "date", "submitter", "decision", "link", "duplicate-row", "missing-child", "over-reserved"} {
		t.Run(name, func(t *testing.T) {
			s, st, p, _ := paymentFixture()
			ctx := context.Background()
			group, err := s.SubmitInstallmentPayment(ctx, p, st.repo.deal.ID, groupInput(st))
			if err != nil {
				t.Fatal(err)
			}
			expected, accept, confirmation, why := int64(1), true, true, ""
			want := apperr.ErrConflict
			switch name {
			case "permission":
				delete(p.Permissions, model.PermPaymentsAccept)
				want = apperr.ErrForbidden
			case "company":
				p.CompanyID = "foreign"
				want = apperr.ErrNotFound
			case "branch":
				p.BranchScope.Mode, p.BranchScope.BranchIDs = "SELECTED", []string{"foreign"}
				want = apperr.ErrNotFound
			case "confirmation":
				confirmation = false
				want = nil // field validation, asserted by error below
			case "reason":
				accept = false
				want = nil
			case "stale":
				expected = 2
				want = apperr.ErrStale
			case "replay":
				if _, err := s.DecideInstallmentPayment(ctx, p, group.Payment.ID, 1, true, true, ""); err != nil {
					t.Fatal(err)
				}
				expected = 2
			case "reload-version":
				st.repo.onLock = func() { st.repo.groups[0].Version++ }
				want = apperr.ErrStale
			case "reload-state":
				st.repo.onLock = func() { st.repo.groups[0].Status = "accepted" }
			case "reload-children":
				st.repo.onLock = func() { st.repo.evidence[0].Status = "accepted" }
			case "cancelled":
				st.repo.onDealLock = func() { st.repo.deal.Status = "cancelled" }
			case "mixed-status":
				st.repo.evidence[0].Status = "accepted"
			case "sum":
				st.repo.evidence[0].AmountMinor = "90"
			case "currency":
				st.repo.evidence[0].Currency = "EUR"
			case "reference":
				st.repo.evidence[0].ExternalReference = "other"
			case "files":
				st.repo.evidence[0].AttachmentIDs = []byte(`["foreign"]`)
			case "date":
				st.repo.evidence[0].PaidOn = st.repo.evidence[0].PaidOn.AddDate(0, 0, 1)
			case "submitter":
				st.repo.evidence[0].SubmittedBy = "foreign"
			case "decision":
				st.repo.evidence[0].DecidedBy = &p.UserID
			case "link":
				st.repo.invoices[2].InstallmentPlanID = nil
			case "duplicate-row":
				st.repo.evidence[0].InvoiceID = st.repo.evidence[1].InvoiceID
			case "missing-child":
				st.repo.evidence = st.repo.evidence[:1]
			case "over-reserved":
				st.repo.evidence = append(st.repo.evidence, model.Evidence{InvoiceID: st.repo.evidence[0].InvoiceID, AmountMinor: "1", Status: "submitted"})
			}
			before, _ := json.Marshal([]any{st.repo.groups, st.repo.evidence, st.events.events})
			_, err = s.DecideInstallmentPayment(ctx, p, group.Payment.ID, expected, accept, confirmation, why)
			if err == nil || (want != nil && !errors.Is(err, want)) {
				t.Fatalf("decision error = %v, want %v", err, want)
			}
			after, _ := json.Marshal([]any{st.repo.groups, st.repo.evidence, st.events.events})
			if string(before) != string(after) {
				t.Fatal("denied decision persisted changes")
			}
		})
	}
}

func TestInstallmentPaymentGroupDecisionRollback(t *testing.T) {
	for _, decision := range []string{"accepted", "rejected"} {
		for _, fail := range []string{"second-child", "parent", "audit"} {
			t.Run(decision+"/"+fail, func(t *testing.T) {
				s, st, p, _ := paymentFixture()
				ctx := context.Background()
				group, err := s.SubmitInstallmentPayment(ctx, p, st.repo.deal.ID, groupInput(st))
				if err != nil {
					t.Fatal(err)
				}
				switch fail {
				case "second-child":
					st.repo.failUpdateAt = 2
				case "parent":
					st.repo.failParent = true
				case "audit":
					st.events.failType = "deal.payment_" + decision
				}
				before, _ := json.Marshal([]any{st.repo.groups, st.repo.evidence, st.events.events})
				if _, err := s.DecideInstallmentPayment(ctx, p, group.Payment.ID, 1, decision == "accepted", true, "not received"); err == nil {
					t.Fatal("expected transaction failure")
				}
				after, _ := json.Marshal([]any{st.repo.groups, st.repo.evidence, st.events.events})
				if string(before) != string(after) {
					t.Fatal("partial decision persisted")
				}
			})
		}
	}
}

func TestInstallmentPaymentGroupSettlement(t *testing.T) {
	s, st, p, id := paymentFixture()
	ctx := context.Background()
	st.repo.deal.Status = "reserved"
	before, _ := json.Marshal([]any{st.repo.deal, st.repo.plan, st.repo.invoices})
	checklist, err := s.checklist(ctx, st, st.repo.deal)
	if err != nil {
		t.Fatal(err)
	}
	in := groupInput(st)
	in.ClaimedAmount.AmountMinor, in.Allocations[1].Amount.AmountMinor = "200", "100"
	group, err := s.SubmitInstallmentPayment(ctx, p, st.repo.deal.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(group.AllowedActions, []string{"accept", "reject"}) {
		t.Fatal("submit response lacks finance actions")
	}
	view, err := s.Get(ctx, p, st.repo.deal.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan := view.InstallmentPlan
	if plan.SettlementState != "outstanding" || plan.Outstanding.AmountMinor != "200" || plan.Pending.AmountMinor != "200" || plan.Available.AmountMinor != "0" || len(plan.Payments) != 1 || len(plan.AllowedActions) != 0 {
		t.Fatalf("pending payoff projection: %+v", plan)
	}
	// A finance-only reviewer sees all months from a single invoice.
	delete(p.Permissions, model.PermDeals)
	invoice, err := s.Invoice(ctx, p, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(invoice.PaymentGroups) != 1 || len(invoice.PaymentGroups[0].Allocations) != 2 || !reflect.DeepEqual(invoice.PaymentGroups[0].AllowedActions, []string{"accept", "reject"}) || len(invoice.EvidenceActions) != 0 {
		t.Fatalf("incomplete finance review: %+v", invoice)
	}
	st.repo.locks = nil
	accepted, err := s.DecideInstallmentPayment(ctx, p, group.Payment.ID, 1, true, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Payment.Status != "accepted" || accepted.Payment.Version != 2 || len(accepted.AllowedActions) != 0 || len(accepted.Allocations) != 2 {
		t.Fatalf("decision response: %+v", accepted)
	}
	if !reflect.DeepEqual(st.repo.locks, []string{"deal", id, st.repo.invoices[2].ID}) {
		t.Fatalf("decision lock order: %v", st.repo.locks)
	}
	for _, e := range st.repo.evidence {
		if e.Status != "accepted" || e.Version != 2 || e.DecidedBy == nil || *e.DecidedBy != p.UserID || e.DecidedAt == nil {
			t.Fatalf("child decision: %+v", e)
		}
	}
	view, err = s.Get(ctx, p, st.repo.deal.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan = view.InstallmentPlan
	if plan.SettlementState != "settled" || plan.State != "planned" || plan.Outstanding.AmountMinor != "0" || plan.Paid.AmountMinor != "200" || plan.Pending.AmountMinor != "0" || plan.Payments[0].Payment.Status != "accepted" {
		t.Fatalf("confirmed settlement: %+v", plan)
	}
	if view.Invoices[0].Outstanding.AmountMinor != "800000" || !reflect.DeepEqual(checklist, view.Checklist) || st.repo.deal.Status != "reserved" {
		t.Fatal("monthly settlement changed nonmonthly debt/checklist/handover")
	}
	after, _ := json.Marshal([]any{st.repo.deal, st.repo.plan, st.repo.invoices})
	if string(before) != string(after) {
		t.Fatal("saved schedule or sale changed")
	}
	var audit map[string]any
	if err := json.Unmarshal(st.events.events[1].Details, &audit); err != nil {
		t.Fatal(err)
	}
	if audit["paymentGroupId"] != group.Payment.ID || audit["decision"] != "accepted" || len(audit["evidenceIds"].([]any)) != 2 || len(audit["invoiceIds"].([]any)) != 2 || st.events.events[1].ActorUserID != p.UserID {
		t.Fatalf("incomplete decision audit: %v", audit)
	}
}

func TestInstallmentPaymentGroupRejectAndProjectionActions(t *testing.T) {
	s, st, p, id := paymentFixture()
	ctx := context.Background()
	st.repo.deal.Status = "reserved"
	group, err := s.SubmitInstallmentPayment(ctx, p, st.repo.deal.ID, groupInput(st))
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.Get(ctx, p, st.repo.deal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.InstallmentPlan.Available.AmountMinor != "50" || !reflect.DeepEqual(view.InstallmentPlan.AllowedActions, []string{"submit-installment-payment"}) || len(view.InstallmentPlan.Payments) != 1 {
		t.Fatalf("partial pending projection: %+v", view.InstallmentPlan)
	}
	delete(p.Permissions, model.PermPaymentsAccept)
	invoice, err := s.Invoice(ctx, p, id)
	if err != nil || len(invoice.PaymentGroups[0].AllowedActions) != 0 {
		t.Fatalf("manager decision actions: %+v %v", invoice, err)
	}
	delete(p.Permissions, model.PermDeals)
	view, err = s.Get(ctx, p, st.repo.deal.ID)
	if err != nil || len(view.InstallmentPlan.AllowedActions) != 0 {
		t.Fatalf("read-only plan actions: %+v %v", view, err)
	}
	p.Permissions[model.PermPaymentsAccept] = true
	rejected, err := s.DecideInstallmentPayment(ctx, p, group.Payment.ID, 1, false, false, "  not received  ")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Payment.Status != "rejected" || rejected.Payment.DecisionReason != "not received" || rejected.Payment.Version != 2 || len(rejected.AllowedActions) != 0 {
		t.Fatalf("rejection response: %+v", rejected)
	}
	for _, allocation := range rejected.Allocations {
		invoice, err = s.Invoice(ctx, p, allocation.InvoiceID)
		if err != nil {
			t.Fatal(err)
		}
		paymentBalance(t, invoice, "0", "0", "100", "100")
	}
	p.Permissions[model.PermDeals] = true
	// Rejection permits a retry using the same receipt reference, on all rows.
	if _, err := s.SubmitInstallmentPayment(ctx, p, st.repo.deal.ID, groupInput(st)); err != nil {
		t.Fatal(err)
	}
	view, err = s.Get(ctx, p, st.repo.deal.ID)
	if err != nil || len(view.InstallmentPlan.Payments) != 2 {
		t.Fatalf("rejected receipt history lost: %+v %v", view, err)
	}
	latest := st.repo.groups[1]
	if _, err := s.DecideInstallmentPayment(ctx, p, latest.ID, latest.Version, false, false, "not received"); err != nil {
		t.Fatal(err)
	}
	s.stock = &paymentStock{}
	_, err = s.Cancel(ctx, p, st.repo.deal.ID, st.repo.deal.Version, "cancel sale")
	if err != nil {
		t.Fatal(err)
	}
	view, err = s.Get(ctx, p, st.repo.deal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.InstallmentPlan.State != "cancelled" || len(view.InstallmentPlan.AllowedActions) != 0 || len(view.InstallmentPlan.Payments) != 2 {
		t.Fatalf("cancelled receipt projection: %+v", view.InstallmentPlan)
	}
}

func TestInstallmentPaymentGroupEmptyProjection(t *testing.T) {
	s, st, p, _ := paymentFixture()
	view, err := s.Get(context.Background(), p, st.repo.deal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.InstallmentPlan.Payments == nil || len(view.InstallmentPlan.Payments) != 0 || view.InstallmentPlan.SettlementState != "outstanding" || view.InstallmentPlan.Available.AmountMinor != "200" {
		t.Fatalf("empty payment collection: %+v", view.InstallmentPlan)
	}
	for _, invoice := range view.Invoices {
		if invoice.PaymentGroups == nil || len(invoice.PaymentGroups) != 0 {
			t.Fatal("invoice empty collection is not []")
		}
	}
}
