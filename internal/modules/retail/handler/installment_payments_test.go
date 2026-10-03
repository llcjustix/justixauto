package handler

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/modules/retail/service"
	"justixauto/internal/pkg/money"
)

func TestInstallmentPaymentDTOActionsAndExactBalances(t *testing.T) {
	number := 2
	due := time.Date(2026, 12, 5, 0, 0, 0, 0, time.UTC)
	m := func(n string) money.Money { return money.Money{AmountMinor: n, Currency: "USD"} }
	v := &service.InvoiceView{Invoice: model.Invoice{ID: "invoice", Purpose: "monthly-installment", AmountMinor: "90071992547409931234", Currency: "USD", InstallmentNumber: &number, DueDate: &due},
		Paid: m("1"), Pending: m("2"), Outstanding: m("90071992547409931233"), Available: m("90071992547409931231"), AllowedActions: []string{"submit-payment"}, Evidence: []model.Evidence{{ID: "submitted", Status: "submitted"}, {ID: "accepted", Status: "accepted"}}, EvidenceActions: map[string][]string{"submitted": {"accept", "reject"}}}
	dto := toInvoice(v)
	if dto.Available != v.Available || dto.InstallmentNumber == nil || *dto.InstallmentNumber != 2 || !reflect.DeepEqual(dto.AllowedActions, v.AllowedActions) || !reflect.DeepEqual(dto.Evidence[0].AllowedActions, []string{"accept", "reject"}) || len(dto.Evidence[1].AllowedActions) != 0 {
		t.Fatalf("invoice projection: %+v", dto)
	}
	plan := toInstallmentPlan(&service.InstallmentPlanView{Plan: model.InstallmentPlan{Currency: "USD"}, Rows: []service.InstallmentPlanRowView{{Number: 2, Invoice: *v}}})
	row := plan.Rows[0]
	if row.InvoiceID != dto.ID || row.Available != dto.Available || row.Paid != dto.Paid || row.Pending != dto.Pending || row.Outstanding != dto.Outstanding || !reflect.DeepEqual(row.AllowedActions, dto.AllowedActions) {
		t.Fatalf("row differs: %+v", row)
	}
	v.AllowedActions, v.EvidenceActions, v.Invoice.InstallmentNumber = nil, nil, nil
	raw, err := json.Marshal(toInvoice(v))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"allowedActions":null`) || !strings.Contains(string(raw), `"allowedActions":[]`) || !strings.Contains(string(raw), `"installmentNumber":null`) {
		t.Fatalf("nullable/empty interface: %s", raw)
	}
}

func TestInstallmentPaymentDTOCompleteProjection(t *testing.T) {
	groupID := "group"
	paidOn := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	amount := money.Money{AmountMinor: "90071992547409931234", Currency: "USD"}
	group := service.InstallmentPaymentView{Payment: model.InstallmentPayment{ID: groupID, DealID: "deal", InstallmentPlanID: "plan", AmountMinor: amount.AmountMinor, Currency: amount.Currency, PaidOn: paidOn, ExternalReference: "receipt", AttachmentIDs: []byte(`["file"]`), Status: "submitted", Version: 7}, AllowedActions: []string{"accept", "reject"}, Allocations: []service.InstallmentPaymentAllocationView{{InvoiceID: "invoice", EvidenceID: "evidence", Number: 3, Amount: amount}}}
	invoice := &service.InvoiceView{Invoice: model.Invoice{ID: "invoice", Version: 5}, Evidence: []model.Evidence{{ID: "evidence", PaymentGroupID: &groupID, Version: 2}}, EvidenceActions: map[string][]string{}, PaymentGroups: []service.InstallmentPaymentView{group}, AllowedActions: []string{}}
	invoiceDTO := toInvoice(invoice)
	if invoiceDTO.Evidence[0].PaymentGroupID == nil || *invoiceDTO.Evidence[0].PaymentGroupID != groupID || len(invoiceDTO.PaymentGroups) != 1 || invoiceDTO.PaymentGroups[0].ClaimedAmount != amount || invoiceDTO.PaymentGroups[0].Revision != "7" || len(invoiceDTO.PaymentGroups[0].Allocations) != 1 {
		t.Fatalf("invoice payment group projection: %+v", invoiceDTO)
	}
	plan := toInstallmentPlan(&service.InstallmentPlanView{Plan: model.InstallmentPlan{Currency: "USD"}, Available: amount, SettlementState: "outstanding", AllowedActions: []string{"submit-installment-payment"}, Payments: []service.InstallmentPaymentView{group}})
	if plan.Available != amount || plan.SettlementState != "outstanding" || !reflect.DeepEqual(plan.AllowedActions, []string{"submit-installment-payment"}) || len(plan.Payments) != 1 || plan.Payments[0].Allocations[0].Amount != amount {
		t.Fatalf("plan payment projection: %+v", plan)
	}
	raw, err := json.Marshal(invoiceDTO)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"paymentGroupId":null`) || !strings.Contains(string(raw), `"paymentGroups"`) || !strings.Contains(string(raw), `"amountMinor":"90071992547409931234"`) {
		t.Fatalf("payment group JSON: %s", raw)
	}
}
