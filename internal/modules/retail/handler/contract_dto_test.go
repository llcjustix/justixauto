package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
)

func TestContractDTOUsesExplicitStructuredFields(t *testing.T) {
	signed := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	v := &model.ContractVersion{ID: "contract", DealID: "deal", State: "signed", Kind: "contract", SignedOn: &signed, FileIDs: []byte(`["file-a"]`), Content: "plain text", PartySnapshot: []byte(`{"seller":{"name":"Seller"},"customer":{"name":"Buyer"},"vehicle":{"vin":"VIN"}}`), TermsSnapshot: []byte(`{"price":{"amountMinor":"100","currency":"USD"},"paymentScheme":"cash"}`), Revision: 7}
	dto, err := toContract(v, 9)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"bindingIds":["file-a"]`, `"partySnapshot":{`, `"termsSnapshot":{`, `"contractRevision":"7"`, `"dealRevision":"9"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("DTO %s missing %s", text, want)
		}
	}
	if strings.Contains(text, "file_ids") || strings.Contains(text, "PartySnapshot") {
		t.Fatalf("model fields leaked: %s", text)
	}
}

func TestContractDTORejectsMalformedStoredFields(t *testing.T) {
	base := &model.ContractVersion{PartySnapshot: []byte(`{}`), TermsSnapshot: []byte(`{}`), FileIDs: []byte(`[]`)}
	for _, mutate := range []func(*model.ContractVersion){
		func(v *model.ContractVersion) { v.PartySnapshot = []byte(`null`) },
		func(v *model.ContractVersion) { v.TermsSnapshot = []byte(`not-json`) },
		func(v *model.ContractVersion) { v.FileIDs = []byte(`null`) },
	} {
		v := *base
		mutate(&v)
		if _, err := toContract(&v, 1); err == nil {
			t.Fatal("malformed stored contract was serialized")
		}
	}
}
