package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"justixauto/internal/modules/commerce/model"
)

func TestContractDTOUsesStructuredSnapshotsAndArrays(t *testing.T) {
	signed := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	v := &model.ContractVersion{ID: "contract", OrderID: "order", CompanyID: "seller", State: model.ContractSigned, Kind: model.ContractInitial, ExternalFileIDs: []byte(`["external"]`), SignedFileIDs: []byte(`["scan","external"]`), Content: "A&B <terms>", PartySnapshot: []byte(`{"parties":{"buyer":{"name":"Buyer"},"supplier":{"name":"Seller"}},"vehicles":[{"vin":"VIN"}]}`), TermsSnapshot: []byte(`{"future":{"nested":123456789012345678901234567890}}`), TermsDigest: "digest", SourceOrderRevision: 4, SignedOn: &signed, Version: 7}
	dto, err := toContract(v, 9)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"fileIds":["external","scan"]`, `"partySnapshot":{`, `"termsSnapshot":{"future":{"nested":123456789012345678901234567890}}`, `"vehicleSnapshot":[{"vin":"VIN"}]`, `"contractRevision":"7"`, `"orderRevision":"9"`, `A\u0026B \u003cterms\u003e`} {
		if !strings.Contains(text, want) {
			t.Fatalf("DTO %s missing %s", text, want)
		}
	}
	if strings.Contains(text, "ExternalFileIDs") || strings.Contains(text, "base64") {
		t.Fatalf("storage field leaked: %s", text)
	}
}

func TestContractSignInputAndRevisionFields(t *testing.T) {
	in, err := contractSignInput("2026-10-11", "paper", []string{"scan"})
	if err != nil || in.SignedOn.Format(time.DateOnly) != "2026-10-11" || in.SignedFileIDs[0] != "scan" {
		t.Fatalf("sign=%+v err=%v", in, err)
	}
	if _, err := contractSignInput("not-a-date", "", nil); err == nil {
		t.Fatal("invalid date accepted")
	}
	request := draftEditRequest{DraftRevision: 3, BindingIDs: &[]string{}, RefreshSnapshot: true}
	if request.DraftRevision != 3 || request.BindingIDs == nil || !request.RefreshSnapshot {
		t.Fatalf("request lost explicit values: %+v", request)
	}
}
