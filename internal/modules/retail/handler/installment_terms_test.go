package handler

import (
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/money"
)

func TestInstallmentTermsDTOMapsDraftAndAction(t *testing.T) {
	v := &model.InstallmentDraft{PolicyID: "own-interest-free-equal", PolicyVersion: 1, Price: money.Money{AmountMinor: "1000", Currency: "USD"}, DownPayment: money.Money{AmountMinor: "100", Currency: "USD"}, TermMonths: 2, FirstDueDate: "2026-01-31", ScheduledTotal: money.Money{AmountMinor: "900", Currency: "USD"}, RegularPayment: money.Money{AmountMinor: "450", Currency: "USD"}, Rows: []model.InstallmentDraftRow{{Number: 1, DueDate: "2026-01-31", Amount: money.Money{AmountMinor: "450", Currency: "USD"}, Balance: money.Money{AmountMinor: "450", Currency: "USD"}}}}
	d := toInstallmentDraft(v)
	if d == nil || d.Rows[0].Balance.AmountMinor != "450" || toInstallmentDraft(nil) != nil {
		t.Fatalf("mapping failed: %+v", d)
	}
}
