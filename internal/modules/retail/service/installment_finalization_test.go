package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

func finalizationFixture(t *testing.T) (*Deal, *installmentStore, *auth.Principal) {
	t.Helper()
	s, st, p := installmentFixture(false)
	p.Permissions[model.PermInvoicesManage] = true
	st.deals.deal.Status = "reserved"
	draft, err := GenerateInstallmentDraft(st.deals.deal.Price(), termsInput("800000", "2026-01-31", 3))
	if err != nil {
		t.Fatal(err)
	}
	st.deals.deal.InstallmentDraft = draft
	return s, st, p
}

func finalizationState(t *testing.T, st *installmentStore) string {
	t.Helper()
	b, err := json.Marshal([]any{st.deals.deal, st.deals.plan, st.deals.invoices, st.events.events})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertIssuedSchedule(t *testing.T, st *installmentStore, revision int64) {
	t.Helper()
	if st.deals.plan == nil || st.deals.deal.Version != revision || len(st.deals.invoices) != 4 {
		t.Fatalf("plan=%+v revision=%d invoices=%d", st.deals.plan, st.deals.deal.Version, len(st.deals.invoices))
	}
	for n, row := range st.deals.deal.InstallmentDraft.Rows {
		i := st.deals.invoices[n+1]
		if i.AmountMinor != row.Amount.AmountMinor || i.Currency != row.Amount.Currency || i.DueDate == nil || i.DueDate.Format("2006-01-02") != row.DueDate || i.InstallmentNumber == nil || *i.InstallmentNumber != row.Number {
			t.Fatalf("row %d differs: %+v / %+v", n, i, row)
		}
	}
}

func TestInstallmentInvoicesIssueExplicitlyAndAtomically(t *testing.T) {
	s, st, p := finalizationFixture(t)
	d, err := s.IssueInstallmentInvoices(context.Background(), p, st.deals.deal.ID, 7)
	if err != nil || d != st.deals.deal {
		t.Fatalf("issue=%+v err=%v", d, err)
	}
	assertIssuedSchedule(t, st, 8)
	if len(st.events.events) != 1 || st.events.events[0].EventType != "deal.installment_plan_created" {
		t.Fatalf("events=%+v", st.events.events)
	}
	if _, err := s.IssueInstallmentInvoices(context.Background(), p, st.deals.deal.ID, 8); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("duplicate=%v", err)
	}
}

func TestInstallmentInvoicesDoNotIssueFromFirstInvoice(t *testing.T) {
	s, st, p := finalizationFixture(t)
	st.deals.invoices = nil
	if _, err := s.IssueInvoice(context.Background(), p, st.deals.deal.ID, 7, InvoiceInput{Purpose: "first-installment", Amount: money.Money{AmountMinor: "800000", Currency: "USD"}, RecipientSnapshot: "Buyer"}); err != nil {
		t.Fatal(err)
	}
	if st.deals.plan != nil || len(st.deals.invoices) != 1 {
		t.Fatalf("first invoice issued monthly obligations: %+v %+v", st.deals.plan, st.deals.invoices)
	}
	if _, err := s.IssueInstallmentInvoices(context.Background(), p, st.deals.deal.ID, 8); err != nil {
		t.Fatal(err)
	}
	assertIssuedSchedule(t, st, 9)
}

func TestInstallmentInvoicesRejectIncompleteOrDeniedWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*installmentStore, *auth.Principal)
		want   error
	}{
		{"permission", func(_ *installmentStore, p *auth.Principal) { p.Permissions = nil }, apperr.ErrForbidden},
		{"company", func(_ *installmentStore, p *auth.Principal) { p.CompanyID = "other" }, apperr.ErrNotFound},
		{"branch", func(_ *installmentStore, p *auth.Principal) { p.BranchScope.Mode = "SELECTED" }, apperr.ErrNotFound},
		{"draft", func(st *installmentStore, _ *auth.Principal) { st.deals.deal.InstallmentDraft = nil }, apperr.ErrConflict},
		{"contract", func(st *installmentStore, _ *auth.Principal) { st.deals.deal.ContractSignedOn = nil }, apperr.ErrConflict},
		{"first", func(st *installmentStore, _ *auth.Principal) { st.deals.invoices = nil }, apperr.ErrConflict},
		{"duplicate first", func(st *installmentStore, _ *auth.Principal) {
			st.deals.invoices = append(st.deals.invoices, st.deals.invoices[0])
		}, apperr.ErrConflict},
		{"first mismatch", func(st *installmentStore, _ *auth.Principal) { st.deals.invoices[0].AmountMinor = "1" }, apperr.ErrConflict},
		{"cancelled", func(st *installmentStore, _ *auth.Principal) { st.deals.deal.Status = "cancelled" }, apperr.ErrConflict},
		{"other scheme", func(st *installmentStore, _ *auth.Principal) { st.deals.deal.PaymentScheme = "cash" }, apperr.ErrConflict},
		{"corrupt draft", func(st *installmentStore, _ *auth.Principal) { st.deals.deal.InstallmentDraft.PolicyVersion = 99 }, apperr.ErrConflict},
		{"stale", func(st *installmentStore, _ *auth.Principal) { st.deals.deal.Version = 8 }, apperr.ErrStale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, st, p := finalizationFixture(t)
			tc.change(st, p)
			before := finalizationState(t, st)
			if _, err := s.IssueInstallmentInvoices(context.Background(), p, st.deals.deal.ID, 7); !errors.Is(err, tc.want) || finalizationState(t, st) != before {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
		})
	}
}

func TestInstallmentInvoicesRollBack(t *testing.T) {
	for _, fail := range []func(*installmentStore){
		func(st *installmentStore) { st.deals.failPlan = errors.New("plan") },
		func(st *installmentStore) { st.deals.failInvoiceAt = 2 },
		func(st *installmentStore) { st.events.failType = "deal.installment_plan_created" },
		func(st *installmentStore) { st.deals.failUpdate = apperr.ErrStale },
	} {
		s, st, p := finalizationFixture(t)
		fail(st)
		before := finalizationState(t, st)
		if _, err := s.IssueInstallmentInvoices(context.Background(), p, st.deals.deal.ID, 7); err == nil || finalizationState(t, st) != before {
			t.Fatalf("err=%v state changed", err)
		}
	}
}
