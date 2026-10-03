package handler

import (
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/modules/retail/service"
	"justixauto/internal/pkg/money"
)

func TestInstallmentPlanDTOUsesImmutableSnapshot(t *testing.T) {
	due := time.Date(2026, 12, 5, 0, 0, 0, 0, time.UTC)
	number := 1
	v := &service.InstallmentPlanView{Plan: model.InstallmentPlan{ID: "plan", ContractReference: "signed", ContractSignedOn: due, ContractFileIDs: []byte(`["file"]`), ContractTotalMinor: "900", Currency: "USD", DownPaymentInvoiceID: "down", DownPaymentMinor: "100"}, State: "active", ScheduledTotal: money.Money{AmountMinor: "800", Currency: "USD"}, Paid: money.Money{AmountMinor: "0", Currency: "USD"}, Pending: money.Money{AmountMinor: "0", Currency: "USD"}, Outstanding: money.Money{AmountMinor: "800", Currency: "USD"}, Rows: []service.InstallmentPlanRowView{{Number: 1, Invoice: service.InvoiceView{Invoice: model.Invoice{ID: "row", AmountMinor: "800", Currency: "USD", DueDate: &due, InstallmentNumber: &number}, Outstanding: money.Money{AmountMinor: "800", Currency: "USD"}, Pending: money.Money{AmountMinor: "0", Currency: "USD"}}}}}
	d := toInstallmentPlan(v)
	if d.ContractReference != "signed" || d.DownPayment.AmountMinor != "100" || len(d.Rows) != 1 || d.Rows[0].Number != 1 || d.Rows[0].Available.AmountMinor != "800" {
		t.Fatalf("bad plan DTO: %+v", d)
	}
}
