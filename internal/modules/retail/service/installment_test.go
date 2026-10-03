package service

import (
	"context"
	"testing"

	"justixauto/internal/pkg/money"
)

func TestInstallmentPlanSavesExplicitDeliveredSchedule(t *testing.T) {
	s, st, p := installmentFixture(false)
	d, err := s.SaveInstallmentPlan(context.Background(), p, st.deals.deal.ID, 7, InstallmentPlanInput{ContractTotal: money.Money{AmountMinor: "3400000", Currency: "USD"}, Rows: []InstallmentPlanRowInput{{DueDate: "2020-01-05", Amount: money.Money{AmountMinor: "1200000", Currency: "USD"}}, {DueDate: "2026-12-05", Amount: money.Money{AmountMinor: "1400000", Currency: "USD"}}}})
	if err != nil || d.Version != 8 || st.deals.plan == nil || len(st.deals.invoices) != 3 {
		t.Fatalf("save: deal=%+v err=%v plan=%+v invoices=%d", d, err, st.deals.plan, len(st.deals.invoices))
	}
	if st.deals.plan.ContractTotalMinor != "3400000" || st.deals.invoices[1].InstallmentNumber == nil || *st.deals.invoices[2].InstallmentNumber != 2 {
		t.Fatalf("snapshot or stable numbers missing: %+v %+v", st.deals.plan, st.deals.invoices)
	}
}

func TestInstallmentPlanRollsBackFailedInvoice(t *testing.T) {
	s, st, p := installmentFixture(true)
	_, err := s.SaveInstallmentPlan(context.Background(), p, st.deals.deal.ID, 7, InstallmentPlanInput{ContractTotal: money.Money{AmountMinor: "900000", Currency: "USD"}, Rows: []InstallmentPlanRowInput{{DueDate: "2026-12-05", Amount: money.Money{AmountMinor: "100000", Currency: "USD"}}}})
	if err == nil || st.deals.plan != nil || len(st.deals.invoices) != 1 || st.deals.deal.Version != 7 {
		t.Fatalf("partial plan escaped transaction: err=%v plan=%+v invoices=%d version=%d", err, st.deals.plan, len(st.deals.invoices), st.deals.deal.Version)
	}
}

func TestInstallmentPlanRejectsInvalidSchedule(t *testing.T) {
	s, st, p := installmentFixture(false)
	_, err := s.SaveInstallmentPlan(context.Background(), p, st.deals.deal.ID, 7, InstallmentPlanInput{ContractTotal: money.Money{AmountMinor: "0", Currency: "USD"}, Rows: nil})
	if err == nil {
		t.Fatal("expected invalid schedule rejection")
	}
}
