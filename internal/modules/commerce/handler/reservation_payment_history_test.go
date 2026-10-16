package handler

import (
	"encoding/json"
	"testing"
	"time"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

func TestReservationAccountingHistoryMapsEvidenceActorsAndRefundProvenance(t *testing.T) {
	requestedAt := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	confirmedAt := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)
	refundedOn := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	decidedBy, confirmedBy := "accountant", "refund-accountant"
	view := &service.ReservationMoneyView{
		ReservationID: "reservation", Revision: 9007199254740993,
		AgreedAdvance: money.Money{AmountMinor: "10000000000000000001", Currency: "USD"},
		Accepted:      money.Money{AmountMinor: "10000000000000000001", Currency: "USD"},
		Pending:       money.Money{AmountMinor: "0", Currency: "USD"}, Refunded: money.Money{AmountMinor: "10000000000000000001", Currency: "USD"},
		RefundObligation: money.Money{AmountMinor: "10000000000000000001", Currency: "USD"}, RefundStatus: "confirmed",
		Evidence: []model.ReservationEvidence{{ID: "evidence", AmountMinor: "10000000000000000001", Currency: "USD", PaidOn: refundedOn, Status: "accepted", SubmittedBy: "buyer", DecidedBy: &decidedBy, CreatedAt: requestedAt, DecidedAt: &confirmedAt}},
		Refund:   &model.ReservationRefund{Status: "confirmed", RequestedBy: "buyer", RequestedAt: requestedAt, RequestReason: "cancelled", AmountMinor: "10000000000000000001", Currency: "USD", ExternalReference: "REF-1", AttachmentIDs: []byte(`["private-1"]`), RefundedOn: &refundedOn, ConfirmedBy: &confirmedBy, ConfirmedAt: &confirmedAt},
	}
	d := toReservationAccounting(&auth.Principal{}, view)
	if d.Evidence[0].SubmittedBy != "buyer" || d.Evidence[0].DecidedBy == nil || *d.Evidence[0].DecidedBy != decidedBy {
		t.Fatalf("evidence actors: %#v", d.Evidence[0])
	}
	if d.Refund == nil || d.Refund.Amount == nil || d.Refund.Amount.AmountMinor != "10000000000000000001" || d.Refund.RefundedOn == nil || *d.Refund.RefundedOn != "2026-10-02" || d.Refund.AttachmentIDs == nil || len(*d.Refund.AttachmentIDs) != 1 || (*d.Refund.AttachmentIDs)[0] != "private-1" || d.Refund.ConfirmedBy == nil || *d.Refund.ConfirmedBy != confirmedBy {
		t.Fatalf("confirmed refund: %#v", d.Refund)
	}
	encoded, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) == 0 {
		t.Fatal("empty accounting JSON")
	}
}

func TestReservationAccountingHistoryOmitsUnconfirmedRefundFields(t *testing.T) {
	requestedAt := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	if d := toReservationAccounting(&auth.Principal{}, &service.ReservationMoneyView{}); d.Refund != nil {
		t.Fatalf("missing refund: %#v", d.Refund)
	}
	d := toReservationAccounting(&auth.Principal{}, &service.ReservationMoneyView{RefundStatus: "requested", Refund: &model.ReservationRefund{Status: "requested", RequestedBy: "buyer", RequestedAt: requestedAt, RequestReason: "cancelled"}})
	if d.Refund == nil || d.Refund.Amount != nil || d.Refund.ExternalReference != nil || d.Refund.AttachmentIDs != nil || d.Refund.RefundedOn != nil || d.Refund.ConfirmedBy != nil || d.Refund.ConfirmedAt != nil {
		t.Fatalf("requested refund fabricated confirmation: %#v", d.Refund)
	}
}
