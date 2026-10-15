package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

type scoringRepo struct {
	ScoringRepository
	runs []model.DemoScoring
	fail error
}

func (r *scoringRepo) Create(_ context.Context, v *model.DemoScoring) error {
	if r.fail != nil {
		return r.fail
	}
	r.runs = append(r.runs, *v)
	return nil
}

func (r *scoringRepo) Current(_ context.Context, company, deal string) (*model.DemoScoring, error) {
	for n := len(r.runs) - 1; n >= 0; n-- {
		v := r.runs[n]
		if v.CompanyID == company && v.DealID == deal {
			return &v, nil
		}
	}
	return nil, apperr.ErrNotFound
}

func scoringFixture(t *testing.T) (*Contract, *Scoring, *contractStore, *auth.Principal) {
	t.Helper()
	contracts, st, p := contractFixture()
	d := st.deals.deal
	d.Status, d.PaymentScheme, d.PriceMinor, d.Currency = "reserved", "own-installment", "100", "USD"
	d.VehicleSnapshot = &model.VehicleSnapshot{VehicleID: "10000000-0000-0000-0000-000000000007", VIN: "VIN"}
	terms, err := GenerateInstallmentDraft(d.Price(), InstallmentTermsInput{DownPayment: money.Money{AmountMinor: "10", Currency: "USD"}, TermMonths: 1, FirstDueDate: "2026-11-01"})
	if err != nil {
		t.Fatal(err)
	}
	d.InstallmentDraft = terms
	return contracts, NewScoring(contracts.Deps), st, p
}

func TestDemoScoringDoesNotBlockFirstOwnInstallmentSignature(t *testing.T) {
	contracts, _, st, p := scoringFixture(t)
	draft, err := contracts.Draft(context.Background(), p, st.deals.deal.ID, 7, DraftContractInput{Body: "reviewed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contracts.Sign(context.Background(), p, st.deals.deal.ID, draft.ID, 8, contractSignInput(), nil); err != nil {
		t.Fatalf("unscored first signature = %v", err)
	}
}

func TestDemoScoringCurrentDigestAndGuards(t *testing.T) {
	_, scoring, st, p := scoringFixture(t)
	if _, err := scoring.Run(context.Background(), p, st.deals.deal.ID, 6); !errors.Is(err, apperr.ErrStale) {
		t.Fatalf("stale run = %v", err)
	}
	outsider := *p
	outsider.CompanyID = "20000000-0000-0000-0000-000000000001"
	if _, err := scoring.Run(context.Background(), &outsider, st.deals.deal.ID, 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("outsider run = %v", err)
	}
	if _, err := scoring.Run(context.Background(), p, st.deals.deal.ID, 7); err != nil {
		t.Fatal(err)
	}
	ready, err := currentDemoScoring(context.Background(), st, st.deals.deal)
	if err != nil || !ready {
		t.Fatalf("current = %v, %v", ready, err)
	}
	st.deals.deal.InstallmentDraft.DownPayment.AmountMinor = "11"
	ready, err = currentDemoScoring(context.Background(), st, st.deals.deal)
	if err != nil || ready {
		t.Fatalf("changed terms current = %v, %v", ready, err)
	}
}

func TestDemoScoringFreshnessProjection(t *testing.T) {
	_, scoring, st, p := scoringFixture(t)
	if run, current, err := scoring.Current(context.Background(), p, st.deals.deal); err != nil || run != nil || current {
		t.Fatalf("missing run = %#v, current=%v, err=%v", run, current, err)
	}
	if _, err := scoring.Run(context.Background(), p, st.deals.deal.ID, 7); err != nil {
		t.Fatal(err)
	}
	run, current, err := scoring.Current(context.Background(), p, st.deals.deal)
	if err != nil || run == nil || !current {
		t.Fatalf("current run = %#v, current=%v, err=%v", run, current, err)
	}

	st.deals.deal.Version++
	st.deals.invoices = append(st.deals.invoices, model.Invoice{ID: "invoice", DealID: st.deals.deal.ID})
	st.contracts.rows["document"] = model.ContractVersion{ID: "document", DealID: st.deals.deal.ID, State: "draft"}
	_, current, err = scoring.Current(context.Background(), p, st.deals.deal)
	if err != nil || !current {
		t.Fatalf("incidental activity current=%v, err=%v", current, err)
	}

	st.deals.deal.InstallmentDraft.DownPayment.AmountMinor = "11"
	run, current, err = scoring.Current(context.Background(), p, st.deals.deal)
	if err != nil || run == nil || current {
		t.Fatalf("stale covered terms = %#v, current=%v, err=%v", run, current, err)
	}
}

func TestDemoScoringExcludesCashAndAmendments(t *testing.T) {
	contracts, _, st, p := scoringFixture(t)
	st.deals.deal.PaymentScheme = "cash"
	ready, err := currentDemoScoring(context.Background(), st, st.deals.deal)
	if err != nil || !ready {
		t.Fatalf("cash gate = %v, %v", ready, err)
	}

	contracts, _, st, p = scoringFixture(t)
	prior := "10000000-0000-0000-0000-000000000008"
	st.contracts.rows[prior] = model.ContractVersion{ID: prior, CompanyID: p.CompanyID, DealID: st.deals.deal.ID, State: "signed", Revision: 1}
	draft, err := contracts.Draft(context.Background(), p, st.deals.deal.ID, 7, DraftContractInput{Body: "amend"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contracts.Sign(context.Background(), p, st.deals.deal.ID, draft.ID, 8, contractSignInput(), &prior); err != nil {
		t.Fatalf("amendment sign = %v", err)
	}
}

func TestDemoScoringRollbackDoesNotPersistRun(t *testing.T) {
	_, scoring, st, p := scoringFixture(t)
	st.events.fail = true
	if _, err := scoring.Run(context.Background(), p, st.deals.deal.ID, 7); err == nil {
		t.Fatal("event failure accepted")
	}
	if len(st.scoring.runs) != 0 || st.deals.deal.Version != 7 {
		t.Fatalf("run or revision escaped rollback: %#v / %d", st.scoring.runs, st.deals.deal.Version)
	}
	st.events.fail = false
	st.scoring.fail = errors.New("scoring insert failed")
	if _, err := scoring.Run(context.Background(), p, st.deals.deal.ID, 7); err == nil || len(st.scoring.runs) != 0 {
		t.Fatalf("insert failure = %v, runs=%#v", err, st.scoring.runs)
	}
}
