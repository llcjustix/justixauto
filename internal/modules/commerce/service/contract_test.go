package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/auth"
)

func ptr[T any](v T) *T { return &v }
func TestContractDraftSignsSameRowAndKeepsExactSnapshots(t *testing.T) {
	s, st, files := contractFixture()
	p := contractPrincipal("buyer")
	body := "{{buyer.name}} <{{supplier.taxId}}> A&B"
	d, e := s.DraftCreate(context.Background(), p, "order", 1, DraftInput{Content: &body, ExternalFileIDs: ptr([]string{"external"})})
	if e != nil {
		t.Fatal(e)
	}
	if d.State != model.ContractDraft || d.Version != 1 || st.deals.order.Version != 2 {
		t.Fatalf("draft %#v order %d", d, st.deals.order.Version)
	}
	if !bytes.Contains(d.TermsSnapshot, []byte("12345678901234567890")) {
		t.Fatalf("future JSON lost: %s", d.TermsSnapshot)
	}
	if !bytes.Contains(d.PartySnapshot, []byte("B\\u0026\\u003c")) {
		t.Fatalf("party snapshot %s", d.PartySnapshot)
	}
	original := append([]byte(nil), d.TermsSnapshot...)
	st.deals.order.Terms[0] = 'X'
	if _, e = s.Sign(context.Background(), p, d.ID, 2, 1, SignInput{SignedOn: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), SignedFileIDs: []string{"scan"}}); e == nil {
		t.Fatal("changed accepted terms signed")
	}
	st.deals.order.Terms = append([]byte(nil), original...)
	signed, e := s.Sign(context.Background(), p, d.ID, 2, 1, SignInput{SignedOn: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), SignedFileIDs: []string{"scan"}})
	if e != nil {
		t.Fatal(e)
	}
	if signed.ID != d.ID || signed.State != model.ContractSigned || signed.Version != 2 || len(files.grants) != 3 {
		t.Fatalf("signed %#v grants %v", signed, files.grants)
	}
	if _, e = s.Sign(context.Background(), p, d.ID, 3, 2, SignInput{}); e == nil {
		t.Fatal("signed record changed")
	}
}

func TestContractRollbackAndAmendment(t *testing.T) {
	s, st, files := contractFixture()
	p := contractPrincipal("buyer")
	content := "paper"
	d, e := s.DraftCreate(context.Background(), p, "order", 1, DraftInput{Content: &content})
	if e != nil {
		t.Fatal(e)
	}
	files.fail = true
	if _, e = s.Sign(context.Background(), p, d.ID, 2, 1, SignInput{SignedFileIDs: []string{"scan"}}); e == nil {
		t.Fatal("share failure accepted")
	}
	if got := st.contracts.rows[d.ID]; got.State != model.ContractDraft || st.deals.order.Version != 2 {
		t.Fatalf("rollback failed %#v order %d", got, st.deals.order.Version)
	}
	files.fail = false
	signed, e := s.Sign(context.Background(), p, d.ID, 2, 1, SignInput{SignedFileIDs: []string{"scan"}})
	if e != nil {
		t.Fatal(e)
	}
	amendDraft, e := s.DraftCreate(context.Background(), p, "order", 3, DraftInput{Content: &content})
	if e != nil {
		t.Fatal(e)
	}
	oldBytes := append([]byte(nil), signed.TermsSnapshot...)
	amend, e := s.Amend(context.Background(), p, amendDraft.ID, signed.ID, 4, 1, SignInput{SignedFileIDs: []string{"amend-scan"}})
	if e != nil {
		t.Fatal(e)
	}
	if amend.ID != amendDraft.ID || amend.Kind != model.ContractAmendment || amend.PriorSignedVersionID == nil || *amend.PriorSignedVersionID != signed.ID || st.contracts.rows[signed.ID].State != model.ContractSigned || !bytes.Equal(st.contracts.rows[signed.ID].TermsSnapshot, oldBytes) {
		t.Fatalf("amend %#v", amend)
	}
}

func TestContractPermissionAndPrivateDraft(t *testing.T) {
	s, _, _ := contractFixture()
	none := contractPrincipal("buyer")
	none.Permissions = nil
	body := "x"
	if _, e := s.DraftCreate(context.Background(), none, "order", 1, DraftInput{Content: &body}); e == nil {
		t.Fatal("missing trade permission")
	}
	p := contractPrincipal("buyer")
	d, e := s.DraftCreate(context.Background(), p, "order", 1, DraftInput{Content: &body})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Detail(context.Background(), contractPrincipal("supplier"), d.ID); e == nil {
		t.Fatal("other party read draft")
	}
}

func TestContractPreviewAndDraftEditFileGrants(t *testing.T) {
	s, st, files := contractFixture()
	p := contractPrincipal("buyer")
	template, err := s.TemplateCreate(context.Background(), p, TemplateInput{Name: "paper", Body: "{{buyer.name}} & <{{supplier.taxId}}>"})
	if err != nil {
		t.Fatal(err)
	}
	template, err = s.TemplateEdit(context.Background(), p, template.ID, 1, TemplateInput{Name: "paper v2", Body: "{{buyer.name}} & <{{supplier.taxId}}>"})
	if err != nil || template.Version != 2 {
		t.Fatalf("template edit %#v: %v", template, err)
	}
	if got, e := s.TemplateDetail(context.Background(), p, template.ID); e != nil || got.Body != template.Body {
		t.Fatalf("template detail %#v: %v", got, e)
	}
	preview, err := s.Preview(context.Background(), p, "order", DraftInput{TemplateID: &template.ID})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Content != "B&< & <ST>" || preview.TemplateVersion == nil || *preview.TemplateVersion != 2 || preview.SourceOrderRevision != 1 || len(st.contracts.rows) != 0 || len(files.grants) != 0 || st.deals.order.Version != 1 {
		t.Fatalf("preview %#v rows %d grants %v revision %d", preview, len(st.contracts.rows), files.grants, st.deals.order.Version)
	}
	content := "original"
	draft, err := s.DraftCreate(context.Background(), p, "order", 1, DraftInput{Content: &content, ExternalFileIDs: ptr([]string{"one"})})
	if err != nil {
		t.Fatal(err)
	}
	if len(files.grants) != 1 || files.grants[0] != "buyer/one/buyer/contract/"+draft.ID {
		t.Fatalf("initial private grant %v", files.grants)
	}
	updated, err := s.DraftEdit(context.Background(), p, draft.ID, 2, 1, DraftInput{ExternalFileIDs: ptr([]string{"one", "two"})})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != content || updated.Version != 2 || len(files.grants) != 2 || files.grants[1] != "buyer/two/buyer/contract/"+draft.ID {
		t.Fatalf("edit %#v grants %v", updated, files.grants)
	}
	empty := ""
	if _, err = s.DraftEdit(context.Background(), p, draft.ID, 3, 2, DraftInput{Content: &empty, ExternalFileIDs: ptr([]string{})}); err == nil {
		t.Fatal("explicit empty invalid draft accepted")
	}
	if got := st.contracts.rows[draft.ID]; got.Version != 2 || got.Content != content {
		t.Fatalf("failed edit persisted %#v", got)
	}
}

func TestContractHistoryAndTransactionalFailures(t *testing.T) {
	s, st, files := contractFixture()
	p := contractPrincipal("buyer")
	content := "paper"
	draft, err := s.DraftCreate(context.Background(), p, "order", 1, DraftInput{Content: &content, ExternalFileIDs: ptr([]string{"external"})})
	if err != nil {
		t.Fatal(err)
	}
	files.failAfter = 2 // author-only grant plus first counterparty grant succeeds, next share fails.
	if _, err = s.Sign(context.Background(), p, draft.ID, 2, 1, SignInput{SignedFileIDs: []string{"scan"}}); err == nil {
		t.Fatal("partial file share accepted")
	}
	if len(files.grants) != 1 || st.contracts.rows[draft.ID].State != model.ContractDraft || st.deals.order.Version != 2 {
		t.Fatalf("share rollback grants %v contract %#v order %d", files.grants, st.contracts.rows[draft.ID], st.deals.order.Version)
	}
	files.failAfter = 0
	st.contracts.fail = true
	if _, err = s.Sign(context.Background(), p, draft.ID, 2, 1, SignInput{SignedFileIDs: []string{"scan"}}); err == nil {
		t.Fatal("contract CAS failure accepted")
	}
	st.contracts.fail = false
	st.deals.fail = true
	if _, err = s.Sign(context.Background(), p, draft.ID, 2, 1, SignInput{SignedFileIDs: []string{"scan"}}); err == nil {
		t.Fatal("order CAS failure accepted")
	}
	st.deals.fail = false
	st.failEvent = true
	if _, err = s.Sign(context.Background(), p, draft.ID, 2, 1, SignInput{SignedFileIDs: []string{"scan"}}); err == nil {
		t.Fatal("event failure accepted")
	}
	if len(files.grants) != 1 || st.contracts.rows[draft.ID].State != model.ContractDraft || st.deals.order.Version != 2 || st.events != 1 {
		t.Fatalf("later rollback grants %v contract %#v order %d events %d", files.grants, st.contracts.rows[draft.ID], st.deals.order.Version, st.events)
	}
	st.failEvent = false
	signed, err := s.Sign(context.Background(), p, draft.ID, 2, 1, SignInput{SignedFileIDs: []string{"scan"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []*auth.Principal{contractPrincipal("buyer"), contractPrincipal("supplier")} {
		if _, err = s.Detail(context.Background(), actor, signed.ID); err != nil {
			t.Fatalf("signed detail for %s: %v", actor.CompanyID, err)
		}
		if history, e := s.List(context.Background(), actor, "order", 20, 0); e != nil || len(history) != 1 {
			t.Fatalf("history for %s: %v %#v", actor.CompanyID, e, history)
		}
	}
	outsider := contractPrincipal("outsider")
	if _, err = s.Detail(context.Background(), outsider, signed.ID); err == nil {
		t.Fatal("outsider read signed detail")
	}
}
