package service

import (
	"context"
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/money"
)

func termsInput(down, first string, months int) InstallmentTermsInput {
	return InstallmentTermsInput{DownPayment: money.Money{AmountMinor: down, Currency: "USD"}, TermMonths: months, FirstDueDate: first}
}

func TestInstallmentTermsGeneratorExactDivisionAndRemainder(t *testing.T) {
	for _, tc := range []struct {
		price, down string
		months      int
		want        []string
	}{
		{"1000", "100", 3, []string{"300", "300", "300"}},
		{"1001", "100", 3, []string{"300", "300", "301"}},
		{"1000", "999", 1, []string{"1"}},
		{"9007199254740993001", "1", 3, []string{"3002399751580331000", "3002399751580331000", "3002399751580331000"}},
		{"99999999999999999999999999999999999999", "1", 2, []string{"49999999999999999999999999999999999999", "49999999999999999999999999999999999999"}},
	} {
		d, err := GenerateInstallmentDraft(money.Money{AmountMinor: tc.price, Currency: "USD"}, termsInput(tc.down, "2026-01-31", tc.months))
		if err != nil || len(d.Rows) != len(tc.want) {
			t.Fatalf("generate %#v: draft=%+v err=%v", tc, d, err)
		}
		for n, want := range tc.want {
			if d.Rows[n].Amount.AmountMinor != want {
				t.Fatalf("row %d: got %s want %s", n, d.Rows[n].Amount.AmountMinor, want)
			}
		}
		if d.Rows[len(d.Rows)-1].Balance.AmountMinor != "0" {
			t.Fatalf("nonzero final balance: %+v", d.Rows)
		}
	}
}

func TestInstallmentTermsGeneratorAnchoredDates(t *testing.T) {
	for _, tc := range []struct {
		first  string
		months int
		want   []string
	}{
		{"2025-01-31", 3, []string{"2025-01-31", "2025-02-28", "2025-03-31"}},
		{"2024-01-31", 3, []string{"2024-01-31", "2024-02-29", "2024-03-31"}},
		{"2026-11-30", 3, []string{"2026-11-30", "2026-12-30", "2027-01-30"}},
	} {
		d, err := GenerateInstallmentDraft(money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("100", tc.first, tc.months))
		if err != nil {
			t.Fatal(err)
		}
		for n, want := range tc.want {
			if d.Rows[n].DueDate != want {
				t.Fatalf("%s row %d: %s", tc.first, n, d.Rows[n].DueDate)
			}
		}
	}
}

func TestInstallmentTermsGeneratorRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		price money.Money
		in    InstallmentTermsInput
	}{
		{"missing-down", money.Money{AmountMinor: "1000", Currency: "USD"}, InstallmentTermsInput{TermMonths: 3, FirstDueDate: "2026-01-01"}},
		{"zero-down", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("0", "2026-01-01", 3)},
		{"full-down", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("1000", "2026-01-01", 3)},
		{"negative-down", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("-1", "2026-01-01", 3)},
		{"currency", money.Money{AmountMinor: "1000", Currency: "USD"}, InstallmentTermsInput{DownPayment: money.Money{AmountMinor: "100", Currency: "EUR"}, TermMonths: 3, FirstDueDate: "2026-01-01"}},
		{"zero-months", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("100", "2026-01-01", 0)},
		{"too-many-months", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("100", "2026-01-01", 1001)},
		{"principal-too-small", money.Money{AmountMinor: "5", Currency: "USD"}, termsInput("4", "2026-01-01", 2)},
		{"malformed-date", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("100", "not-a-date", 3)},
		{"year-zero", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("100", "0000-01-01", 3)},
		{"past-date-valid", money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("100", "2020-01-01", 3)},
	} {
		d, err := GenerateInstallmentDraft(tc.price, tc.in)
		if tc.name == "past-date-valid" {
			if err != nil || d == nil {
				t.Fatalf("past date rejected: %v", err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("%s accepted: %+v", tc.name, d)
		}
	}
	if _, err := GenerateInstallmentDraft(money.Money{AmountMinor: "1000", Currency: "USD"}, termsInput("100", "9999-12-31", 2)); err == nil {
		t.Fatal("overflow year accepted")
	}
}

func TestInstallmentTermsLegacySavePersistsDraftWithoutInvoices(t *testing.T) {
	s, st, p := installmentFixture(false)
	st.deals.deal.ContractSignedOn, st.deals.deal.ContractReference = nil, ""
	d, err := s.SaveInstallmentTerms(context.Background(), p, st.deals.deal.ID, 7, termsInput("800000", "2026-01-31", 2))
	if err != nil || d.InstallmentDraft == nil || d.Version != 8 || len(st.deals.invoices) != 1 || len(st.events.events) != 1 {
		t.Fatalf("save result=%+v err=%v invoices=%d events=%+v", d, err, len(st.deals.invoices), st.events.events)
	}
	if st.events.events[0].EventType != "deal.installment_terms_saved" || d.InstallmentDraft.Rows[1].Balance.AmountMinor != "0" {
		t.Fatalf("bad persisted draft: %+v", d.InstallmentDraft)
	}
}

func TestInstallmentTermsLegacySaveGuards(t *testing.T) {
	s, st, p := installmentFixture(false)
	if _, err := s.SaveInstallmentTerms(context.Background(), p, st.deals.deal.ID, 6, termsInput("800000", "2026-01-01", 2)); err != apperr.ErrStale {
		t.Fatalf("version: %v", err)
	}
	st.deals.deal.Status = "cancelled"
	if _, err := s.SaveInstallmentTerms(context.Background(), p, st.deals.deal.ID, 7, termsInput("800000", "2026-01-01", 2)); err == nil {
		t.Fatal("cancelled accepted")
	}
	st.deals.deal.Status, st.deals.deal.PaymentScheme = "delivered", "cash"
	if _, err := s.SaveInstallmentTerms(context.Background(), p, st.deals.deal.ID, 7, termsInput("800000", "2026-01-01", 2)); err == nil {
		t.Fatal("cash accepted")
	}
	st.deals.deal.PaymentScheme = "own-installment"
	st.deals.plan = &model.InstallmentPlan{ID: "plan"}
	if _, err := s.SaveInstallmentTerms(context.Background(), p, st.deals.deal.ID, 7, termsInput("800000", "2026-01-01", 2)); err == nil {
		t.Fatal("final plan accepted")
	}
}

func TestInstallmentTermsLegacySaveRejectsIssuedDownMismatch(t *testing.T) {
	s, st, p := installmentFixture(false)
	if _, err := s.SaveInstallmentTerms(context.Background(), p, st.deals.deal.ID, 7, termsInput("700000", "2026-01-01", 2)); err == nil {
		t.Fatal("issued down mismatch accepted")
	}
	if st.deals.deal.InstallmentDraft != nil || st.deals.deal.Version != 7 {
		t.Fatal("mismatch persisted draft")
	}
}
