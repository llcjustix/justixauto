package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/money"
)

func TestContractPreviewPreservesPlainTextSnapshotValues(t *testing.T) {
	d := &model.Deal{PriceMinor: "100", Currency: "USD", PaymentScheme: "cash", VehicleSnapshot: &model.VehicleSnapshot{VIN: "VIN<&"}}
	got := renderContract("{{seller.name}} {{customer.name}} {{vehicle.vin}} {{price.currency}}", &CompanyRequisites{Name: "A&B"}, &model.Customer{DisplayName: "<Buyer>"}, d)
	if got != "A&B <Buyer> VIN<& USD" || !strings.Contains(got, "<Buyer>") {
		t.Fatalf("preview = %q", got)
	}
}

func TestContractSignConsumesDraftOnce(t *testing.T) {
	s, st, p := contractFixture()
	v, err := s.Sign(context.Background(), p, contractDealID, contractDraftID, 7, contractSignInput(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != contractDraftID || v.State != "signed" || v.Revision != 2 || len(st.contracts.rows) != 1 {
		t.Fatalf("draft was not consumed: %#v %#v", v, st.contracts.rows)
	}
	if st.contracts.rows[contractDraftID].State != "signed" || !st.files.sawBoundContext || len(st.files.shares) != 1 || !strings.HasSuffix(st.files.shares[0], ":"+contractDraftID) {
		t.Fatalf("sign binding mismatch: %#v %#v", st.files, st.contracts.rows[contractDraftID])
	}
	for _, revision := range []int64{7, 8} {
		if _, err := s.Sign(context.Background(), p, contractDealID, contractDraftID, revision, contractSignInput(), nil); err == nil {
			t.Fatalf("repeat sign with deal revision %d succeeded", revision)
		}
	}
}

func TestContractSignRollbackAndGuards(t *testing.T) {
	for _, name := range []string{"cas", "files", "deal", "event", "company", "branch", "permission", "amendment-parent"} {
		t.Run(name, func(t *testing.T) {
			s, st, p := contractFixture()
			var prior *string
			switch name {
			case "cas":
				st.contracts.failSign = true
			case "files":
				st.files.fail = true
			case "deal":
				st.deals.failUpdate = true
			case "event":
				st.events.fail = true
			case "company":
				p.CompanyID = "20000000-0000-0000-0000-000000000001"
			case "branch":
				p.BranchScope.Mode, p.BranchScope.BranchIDs = "SELECTED", []string{"20000000-0000-0000-0000-000000000002"}
			case "permission":
				delete(p.Permissions, model.PermDeals)
			case "amendment-parent":
				id := "10000000-0000-0000-0000-000000000009"
				st.contracts.rows[id] = model.ContractVersion{ID: id, CompanyID: contractCompany, DealID: "10000000-0000-0000-0000-000000000099", State: "signed", Revision: 1}
				prior = &id
			}
			_, err := s.Sign(context.Background(), p, contractDealID, contractDraftID, 7, contractSignInput(), prior)
			if err == nil {
				t.Fatal("invalid sign succeeded")
			}
			if got := st.contracts.rows[contractDraftID]; got.State != "draft" || got.Revision != 1 {
				t.Fatalf("draft persisted after %s: %#v", name, got)
			}
			if st.deals.deal.Version != 7 || len(st.events.events) != 0 || len(st.files.shares) != 0 {
				t.Fatalf("transaction leaked after %s", name)
			}
		})
	}
}

func TestContractAmendmentAndReadGuards(t *testing.T) {
	s, st, p := contractFixture()
	priorID := "10000000-0000-0000-0000-000000000008"
	st.contracts.rows[priorID] = model.ContractVersion{ID: priorID, CompanyID: contractCompany, DealID: contractDealID, State: "signed", Revision: 1}
	if _, err := s.Sign(context.Background(), p, contractDealID, contractDraftID, 7, contractSignInput(), nil); err == nil {
		t.Fatal("base signature after signed version succeeded")
	}
	if _, err := s.Sign(context.Background(), p, contractDealID, contractDraftID, 7, contractSignInput(), &priorID); err != nil {
		t.Fatalf("explicit amendment failed: %v", err)
	}
	if got := st.contracts.rows[contractDraftID]; got.Kind != "amendment" || got.PriorSignedVersionID == nil || *got.PriorSignedVersionID != priorID {
		t.Fatalf("amendment link = %#v", got)
	}
	delete(p.Permissions, model.PermRead)
	if _, err := s.Contracts(context.Background(), p, contractDealID); !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("read without permission = %v", err)
	}
}

func TestContractDraftExternalEditAndRevisionGuards(t *testing.T) {
	s, st, p := contractFixture()
	v, err := s.Draft(context.Background(), p, contractDealID, 7, DraftContractInput{BindingIDs: []string{contractFile}})
	if err != nil {
		t.Fatal(err)
	}
	if v.Content != "" || v.Revision != 1 || v.DealRevision != 8 || len(st.files.shares) != 1 {
		t.Fatalf("external draft = %#v %#v", v, st.files)
	}
	if _, err := s.Draft(context.Background(), p, contractDealID, 8, DraftContractInput{}); err == nil {
		t.Fatal("empty draft succeeded")
	}
	body, files := "edited {{customer.name}}", []string{contractFile}
	edited, err := s.EditDraft(context.Background(), p, contractDealID, v.ID, 8, EditDraftContractInput{Body: &body, BindingIDs: &files, DraftRevision: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Content != "edited Buyer" || edited.Revision != 2 || edited.DealRevision != 9 {
		t.Fatalf("edit = %#v", edited)
	}
	if _, err := s.EditDraft(context.Background(), p, contractDealID, v.ID, 9, EditDraftContractInput{DraftRevision: "01"}); err == nil {
		t.Fatal("noncanonical draft revision succeeded")
	}
	if _, err := s.EditDraft(context.Background(), p, contractDealID, v.ID, 8, EditDraftContractInput{DraftRevision: "2"}); !errors.Is(err, apperr.ErrStale) {
		t.Fatalf("stale deal = %v", err)
	}
}

func TestContractTemplateBodyOverrideAndFileUnion(t *testing.T) {
	s, st, p := contractFixture()
	templateID := "10000000-0000-0000-0000-000000000010"
	st.contracts.templates[templateID] = model.ContractTemplate{ID: templateID, CompanyID: contractCompany, Body: "template", Version: 3}
	v, err := s.Draft(context.Background(), p, contractDealID, 7, DraftContractInput{TemplateID: templateID, Body: "override {{seller.name}}", BindingIDs: []string{contractFile}})
	if err != nil {
		t.Fatal(err)
	}
	if v.Content != "override Seller" || v.TemplateVersion == nil || *v.TemplateVersion != 3 {
		t.Fatalf("template provenance/body = %#v", v)
	}
	scan := "10000000-0000-0000-0000-000000000011"
	signed, err := s.Sign(context.Background(), p, contractDealID, v.ID, 8, SignContractInput{SignedOn: "2026-10-11", Reference: "SC-2", BindingIDs: []string{scan, contractFile}, DraftRevision: "1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(signed.FileIDs, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != contractFile || got[1] != scan {
		t.Fatalf("file union = %#v", got)
	}
	if _, err := s.EditDraft(context.Background(), p, contractDealID, v.ID, 9, EditDraftContractInput{DraftRevision: "2"}); err == nil {
		t.Fatal("signed draft was editable")
	}
}

func TestContractSnapshotCapturesPrivateTermsAndStaysStable(t *testing.T) {
	s, st, p := contractFixture()
	issued, birth := time.Date(2020, 2, 3, 0, 0, 0, 0, time.UTC), time.Date(1990, 1, 2, 0, 0, 0, 0, time.UTC)
	st.crm.customer = &model.Customer{ID: st.deals.deal.CustomerID, CompanyID: contractCompany, DisplayName: "Buyer", Phone: "1", PassportSeries: "AA", PassportNumber: "2", PassportIssuer: "Issuer", PassportIssuedOn: &issued, BirthDate: &birth, PersonalID: "pid", RegisteredAddress: "addr"}
	st.deals.deal.PriceMinor, st.deals.deal.Currency, st.deals.deal.PaymentScheme = "1000", "USD", "own-installment"
	st.deals.deal.VehicleSnapshot = &model.VehicleSnapshot{VIN: "VIN<&", ModelID: "model"}
	st.deals.deal.InstallmentDraft = &model.InstallmentDraft{PolicyID: "policy", PolicyVersion: 1, Price: money.Money{AmountMinor: "1000", Currency: "USD"}, DownPayment: money.Money{AmountMinor: "100", Currency: "USD"}, TermMonths: 2, FirstDueDate: "2026-11-01", ScheduledTotal: money.Money{AmountMinor: "900", Currency: "USD"}, RegularPayment: money.Money{AmountMinor: "450", Currency: "USD"}, Rows: []model.InstallmentDraftRow{{Number: 1, DueDate: "2026-11-01", Amount: money.Money{AmountMinor: "450", Currency: "USD"}}}}
	content, party, terms, err := s.snapshot(context.Background(), st, p, st.deals.deal, "{{seller.name}} {{customer.passportNumber}} {{vehicle.vin}}")
	if err != nil {
		t.Fatal(err)
	}
	if content != "Seller 2 VIN<&" || !strings.Contains(string(party), "passportNumber") || !strings.Contains(string(terms), "policy") || !strings.Contains(string(terms), "scheduledTotal") {
		t.Fatalf("snapshot = %q %s %s", content, party, terms)
	}
	partyBefore, termsBefore := append([]byte(nil), party...), append([]byte(nil), terms...)
	st.crm.customer.DisplayName, st.deals.deal.InstallmentDraft.Rows[0].Amount.AmountMinor = "later", "999"
	if string(party) != string(partyBefore) || string(terms) != string(termsBefore) {
		t.Fatal("snapshot bytes changed after live edits")
	}
}

func TestContractSignRejectsStaleTerms(t *testing.T) {
	s, st, p := contractFixture()
	st.deals.deal.PaymentScheme, st.deals.deal.PriceMinor, st.deals.deal.Currency = "own-installment", "100", "USD"
	st.deals.deal.InstallmentDraft = &model.InstallmentDraft{PolicyID: "policy", PolicyVersion: 1, Price: money.Money{AmountMinor: "100", Currency: "USD"}, DownPayment: money.Money{AmountMinor: "10", Currency: "USD"}, TermMonths: 1, FirstDueDate: "2026-11-01", ScheduledTotal: money.Money{AmountMinor: "90", Currency: "USD"}, RegularPayment: money.Money{AmountMinor: "90", Currency: "USD"}}
	_, _, terms, err := s.snapshot(context.Background(), st, p, st.deals.deal, "x")
	if err != nil {
		t.Fatal(err)
	}
	draft := st.contracts.rows[contractDraftID]
	draft.TermsSnapshot = terms
	st.contracts.rows[contractDraftID] = draft
	st.deals.deal.InstallmentDraft.DownPayment.AmountMinor = "11"
	if _, err := s.Sign(context.Background(), p, contractDealID, contractDraftID, 7, contractSignInput(), nil); err == nil {
		t.Fatal("stale economics signed")
	}
}

func TestContractSignFinalizesExistingInstallmentDraft(t *testing.T) {
	s, st, p := contractFixture()
	d := st.deals.deal
	d.Status, d.PaymentScheme, d.PriceMinor, d.Currency = "reserved", "own-installment", "100", "USD"
	installment, err := GenerateInstallmentDraft(d.Price(), InstallmentTermsInput{DownPayment: money.Money{AmountMinor: "10", Currency: "USD"}, TermMonths: 1, FirstDueDate: "2026-11-01"})
	if err != nil {
		t.Fatal(err)
	}
	d.InstallmentDraft = installment
	st.deals.invoices = []model.Invoice{{ID: "first", DealID: d.ID, CompanyID: d.CompanyID, Purpose: "first-installment", AmountMinor: "10", Currency: "USD", Status: "issued"}}
	if _, err := NewScoring(s.Deps).Run(context.Background(), p, d.ID, 7); err != nil {
		t.Fatalf("demo scoring: %v", err)
	}
	draft, err := s.Draft(context.Background(), p, d.ID, 8, DraftContractInput{Body: "reviewed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sign(context.Background(), p, d.ID, draft.ID, 9, SignContractInput{SignedOn: "2026-10-11", Reference: "SC-1", BindingIDs: []string{contractFile}, DraftRevision: "1"}, nil); err != nil {
		t.Fatal(err)
	}
	if st.deals.plan == nil || len(st.deals.invoices) != 2 || st.deals.plan.ContractReference != "SC-1" {
		t.Fatalf("sign did not finalize saved draft: %#v %#v", st.deals.plan, st.deals.invoices)
	}
	row := st.deals.invoices[1]
	if row.AmountMinor != installment.Rows[0].Amount.AmountMinor || row.DueDate == nil || row.DueDate.Format(time.DateOnly) != installment.Rows[0].DueDate || row.InstallmentNumber == nil || *row.InstallmentNumber != installment.Rows[0].Number {
		t.Fatalf("signed plan row diverged from saved draft: %#v %#v", row, installment.Rows[0])
	}
}

func TestContractSnapshotUsesExistingPlanWithoutRecalculation(t *testing.T) {
	s, st, _ := contractFixture()
	st.deals.deal.PaymentScheme, st.deals.deal.PriceMinor, st.deals.deal.Currency = "own-installment", "1000", "USD"
	planID, downID, monthlyID := "10000000-0000-0000-0000-000000000012", "10000000-0000-0000-0000-000000000013", "10000000-0000-0000-0000-000000000014"
	st.deals.plan = &model.InstallmentPlan{ID: planID, DealID: contractDealID, DownPaymentInvoiceID: downID, DownPaymentMinor: "100", ContractTotalMinor: "1000", Currency: "USD"}
	number, due := 1, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	st.deals.invoices = []model.Invoice{{ID: downID, Purpose: "first-installment", AmountMinor: "100", Currency: "USD"}, {ID: monthlyID, Purpose: "monthly-installment", AmountMinor: "900", Currency: "USD", InstallmentPlanID: &planID, InstallmentNumber: &number, DueDate: &due}}
	terms, err := s.termsSnapshot(context.Background(), st, st.deals.deal)
	if err != nil {
		t.Fatal(err)
	}
	if terms.Installment == nil || terms.Installment.Source != "plan" || terms.Installment.DownPayment.AmountMinor != "100" || len(terms.Installment.Obligations) != 2 || terms.Installment.Obligations[1].Amount.AmountMinor != "900" {
		t.Fatalf("plan terms = %#v", terms.Installment)
	}
}
