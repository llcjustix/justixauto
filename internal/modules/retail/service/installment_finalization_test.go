package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

func finalizationFixture(t *testing.T) (*Deal, *installmentStore, *auth.Principal) {
	t.Helper()
	s, st, p := installmentFixture(false)
	st.deals.deal.Status = "reserved"
	draft, err := GenerateInstallmentDraft(st.deals.deal.Price(), termsInput("800000", "2026-01-31", 3))
	if err != nil {
		t.Fatal(err)
	}
	st.deals.deal.InstallmentDraft = draft
	return s, st, p
}

func finalizationCommand(ctx context.Context, s *Deal, st *installmentStore, p *auth.Principal, origin string) error {
	d := st.deals.deal
	switch origin {
	case "contract":
		_, err := s.RecordContract(ctx, p, d.ID, d.Version, "2026-09-30", "SIGNED-2", nil)
		return err
	case "invoice":
		_, err := s.IssueInvoice(ctx, p, d.ID, d.Version, InvoiceInput{Purpose: "first-installment", Amount: money.Money{AmountMinor: "800000", Currency: "USD"}, RecipientSnapshot: "Buyer"})
		return err
	default:
		_, err := s.SaveInstallmentTerms(ctx, p, d.ID, d.Version, termsInput("800000", "2026-01-31", 3))
		return err
	}
}

func finalizationState(t *testing.T, st *installmentStore) string {
	t.Helper()
	b, err := json.Marshal([]any{st.deals.deal, st.deals.plan, st.deals.invoices, st.deals.evidence, st.events.events})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertFinalized(t *testing.T, st *installmentStore, revision int64) {
	t.Helper()
	d, plan := st.deals.deal, st.deals.plan
	if plan == nil || d.Version != revision || len(st.deals.invoices) != 4 {
		t.Fatalf("plan=%+v revision=%d invoices=%d", plan, d.Version, len(st.deals.invoices))
	}
	if plan.ContractTotalMinor != d.PriceMinor || plan.DownPaymentMinor != "800000" || plan.DownPaymentInvoiceID != st.deals.invoices[0].ID || plan.ContractReference != d.ContractReference || !plan.ContractSignedOn.Equal(*d.ContractSignedOn) {
		t.Fatalf("wrong contract snapshot: %+v", plan)
	}
	for n, row := range d.InstallmentDraft.Rows {
		i := st.deals.invoices[n+1]
		if i.AmountMinor != row.Amount.AmountMinor || i.Currency != row.Amount.Currency || i.DueDate.Format("2006-01-02") != row.DueDate || i.InstallmentPlanID == nil || *i.InstallmentPlanID != plan.ID || i.InstallmentNumber == nil || *i.InstallmentNumber != row.Number {
			t.Fatalf("row %d differs from saved snapshot: %+v / %+v", n, i, row)
		}
	}
	created := 0
	for _, e := range st.events.events {
		if e.EventType == "deal.installment_plan_created" {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("plan events=%d", created)
	}
}

func TestInstallmentFinalizationLifecycleOrders(t *testing.T) {
	for _, order := range [][]string{{"contract", "invoice"}, {"invoice", "contract"}, {"terms"}} {
		t.Run(order[0], func(t *testing.T) {
			s, st, p := finalizationFixture(t)
			if order[0] == "terms" {
				st.deals.deal.InstallmentDraft = nil
				st.deals.deal.Status = "delivered"
				st.deals.evidence = map[string][]model.Evidence{st.deals.invoices[0].ID: {{ID: "receipt", AmountMinor: "100000", Currency: "USD", Status: "accepted"}}}
			} else {
				st.deals.deal.ContractSignedOn, st.deals.deal.ContractReference = nil, ""
				st.deals.invoices = nil
			}
			for n, command := range order {
				if err := finalizationCommand(context.Background(), s, st, p, command); err != nil {
					t.Fatal(err)
				}
				if n == 0 && len(order) == 2 && (st.deals.plan != nil || len(st.deals.invoices) > 1) {
					t.Fatal("plan created before last prerequisite")
				}
			}
			assertFinalized(t, st, 7+int64(len(order)))
			if st.deals.updateCalls != len(order) || len(st.events.events) != len(order)+1 {
				t.Fatalf("updates=%d events=%d", st.deals.updateCalls, len(st.events.events))
			}
			row := &st.deals.invoices[1]
			if st.deals.deal.Status == "reserved" {
				v, err := s.actorInvoiceView(context.Background(), st, p, row)
				if err != nil || len(v.AllowedActions) != 1 || v.AllowedActions[0] != "submit-payment" || s.monthlyPaymentEligibility(context.Background(), st, st.deals.deal, row) != nil {
					t.Fatalf("advance monthly payment unavailable before delivery: %+v %v", v, err)
				}
				st.deals.deal.Status = "delivered"
			}
			v, err := s.actorInvoiceView(context.Background(), st, p, row)
			if err != nil || len(v.AllowedActions) != 1 || v.AllowedActions[0] != "submit-payment" {
				t.Fatalf("delivered plan not serviceable: %+v %v", v, err)
			}
		})
	}
}

func TestInstallmentFinalizationIncompletePrerequisites(t *testing.T) {
	for _, missing := range []string{"both", "contract", "reference", "invoice"} {
		t.Run(missing, func(t *testing.T) {
			s, st, p := finalizationFixture(t)
			st.deals.deal.Status = "delivered"
			if missing == "both" || missing == "contract" {
				st.deals.deal.ContractSignedOn = nil
			}
			if missing == "reference" {
				st.deals.deal.ContractReference = ""
			}
			if missing == "both" || missing == "invoice" {
				st.deals.invoices = nil
			}
			if err := finalizationCommand(context.Background(), s, st, p, "terms"); err != nil {
				t.Fatal(err)
			}
			if st.deals.plan != nil || len(st.deals.invoices) > 1 || st.deals.deal.InstallmentDraft == nil || st.deals.deal.Version != 8 {
				t.Fatal("missing facts were fabricated or draft lost")
			}
		})
	}
}

func TestInstallmentFinalizationRollback(t *testing.T) {
	for _, origin := range []string{"contract", "invoice", "terms"} {
		for _, failure := range []string{"plan", "second-row", "plan-event", "origin-event", "stale-update", "concurrent-plan"} {
			t.Run(origin+"/"+failure, func(t *testing.T) {
				s, st, p := finalizationFixture(t)
				if origin == "invoice" {
					st.deals.invoices = nil
				}
				if origin == "terms" {
					st.deals.deal.InstallmentDraft = nil
				}
				switch failure {
				case "plan":
					st.deals.failPlan = errors.New("plan insert failed")
				case "second-row":
					st.deals.failInvoiceAt = 2
					if origin == "invoice" {
						st.deals.failInvoiceAt = 3
					}
				case "plan-event":
					st.events.failType = "deal.installment_plan_created"
				case "origin-event":
					st.events.failType = map[string]string{"contract": "deal.contract_recorded", "invoice": "deal.invoice_issued", "terms": "deal.installment_terms_saved"}[origin]
				case "stale-update":
					// Simulate another writer winning the optimistic update after
					// our prerequisite read and inserts; not a database race test.
					st.deals.failUpdate = apperr.ErrStale
				case "concurrent-plan":
					// Simulate unique deal-plan conflict after the absence read.
					st.deals.failPlan = apperr.ErrConflict
				}
				before := finalizationState(t, st)
				err := finalizationCommand(context.Background(), s, st, p, origin)
				if err == nil || finalizationState(t, st) != before {
					t.Fatalf("origin or partial obligations escaped rollback: %v", err)
				}
				if failure == "stale-update" && !errors.Is(err, apperr.ErrStale) {
					t.Fatalf("wrong version error: %v", err)
				}
			})
		}
	}
}

func TestInstallmentFinalizationRejectsMismatches(t *testing.T) {
	for _, origin := range []string{"contract", "invoice", "terms"} {
		for _, mismatch := range []string{"amount", "currency", "ambiguous"} {
			t.Run(origin+"/"+mismatch, func(t *testing.T) {
				s, st, p := finalizationFixture(t)
				if origin == "invoice" {
					// Must reject even before a contract exists.
					st.deals.deal.ContractSignedOn = nil
					st.deals.invoices = nil
					if mismatch == "amount" {
						st.deals.deal.InstallmentDraft.DownPayment.AmountMinor = "700000"
					} else if mismatch == "currency" {
						st.deals.deal.InstallmentDraft.DownPayment.Currency = "EUR"
					} else {
						st.deals.invoices = []model.Invoice{{Purpose: "first-installment", Status: "issued"}}
					}
				} else if mismatch == "amount" {
					st.deals.invoices[0].AmountMinor = "700000"
				} else if mismatch == "currency" {
					st.deals.invoices[0].Currency = "EUR"
				} else {
					st.deals.invoices = append(st.deals.invoices, st.deals.invoices[0])
				}
				before := finalizationState(t, st)
				if err := finalizationCommand(context.Background(), s, st, p, origin); !errors.Is(err, apperr.ErrConflict) || finalizationState(t, st) != before {
					t.Fatalf("mismatch not atomically rejected: %v", err)
				}
			})
		}
	}
	for _, corrupt := range []string{"policy", "version", "price", "row", "balance", "number", "date", "total", "regular", "invalid-input"} {
		t.Run(corrupt, func(t *testing.T) {
			s, st, p := finalizationFixture(t)
			draft := st.deals.deal.InstallmentDraft
			switch corrupt {
			case "policy":
				draft.PolicyID = "unknown"
			case "version":
				draft.PolicyVersion++
			case "price":
				st.deals.deal.PriceMinor = "3400001"
			case "row":
				draft.Rows[0].Amount.AmountMinor = "1"
			case "balance":
				draft.Rows[0].Balance.AmountMinor = "0"
			case "number":
				draft.Rows[0].Number++
			case "date":
				draft.Rows[1].DueDate = "2026-03-01"
			case "total":
				draft.ScheduledTotal.AmountMinor = "1"
			case "regular":
				draft.RegularPayment.Currency = "EUR"
			case "invalid-input":
				draft.TermMonths = 0
			}
			before := finalizationState(t, st)
			if err := finalizationCommand(context.Background(), s, st, p, "contract"); !errors.Is(err, apperr.ErrConflict) || finalizationState(t, st) != before {
				t.Fatalf("corrupt snapshot accepted: %v", err)
			}
		})
	}
}

func TestInstallmentFinalizationGuards(t *testing.T) {
	for _, origin := range []string{"contract", "invoice", "terms"} {
		for _, guard := range []string{"permission", "company", "branch", "cancelled"} {
			t.Run(origin+"/"+guard, func(t *testing.T) {
				s, st, p := finalizationFixture(t)
				want := apperr.ErrConflict
				switch guard {
				case "permission":
					p.Permissions = nil
					want = apperr.ErrForbidden
				case "company":
					p.CompanyID = "other-company"
					want = apperr.ErrNotFound
				case "branch":
					p.BranchScope.Mode = "SELECTED"
					want = apperr.ErrNotFound
				case "cancelled":
					st.deals.deal.Status = "cancelled"
				}
				before := finalizationState(t, st)
				if err := finalizationCommand(context.Background(), s, st, p, origin); !errors.Is(err, want) || finalizationState(t, st) != before {
					t.Fatalf("guard failed: %v want %v", err, want)
				}
			})
		}
	}
}

func TestInstallmentFinalizationImmutablePlansAndManualGuard(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		t.Run(map[bool]string{true: "legacy", false: "generated"}[legacy], func(t *testing.T) {
			s, st, p := finalizationFixture(t)
			manual := InstallmentPlanInput{ContractTotal: money.Money{AmountMinor: "9999999", Currency: "USD"}, Rows: []InstallmentPlanRowInput{{DueDate: "2026-10-01", Amount: money.Money{AmountMinor: "9000000", Currency: "USD"}}, {DueDate: "2020-01-01", Amount: money.Money{AmountMinor: "199999", Currency: "USD"}}}}
			if _, err := s.SaveInstallmentPlan(context.Background(), p, st.deals.deal.ID, 7, manual); !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("manual rows replaced draft: %v", err)
			}
			if legacy {
				st.deals.deal.InstallmentDraft = nil
				if _, err := s.SaveInstallmentPlan(context.Background(), p, st.deals.deal.ID, 7, manual); err != nil {
					t.Fatal(err)
				}
			} else if err := finalizationCommand(context.Background(), s, st, p, "contract"); err != nil {
				t.Fatal(err)
			}
			st.deals.evidence = map[string][]model.Evidence{st.deals.invoices[1].ID: {{ID: "accepted-receipt", Status: "accepted", AmountMinor: "123", Currency: "USD", AttachmentIDs: []byte("[\"receipt\"]")}}}
			immutable := func() string {
				b, _ := json.Marshal([]any{st.deals.plan, st.deals.invoices, st.deals.evidence})
				return string(b)
			}
			before := immutable()
			if !legacy {
				// A finalized plan is authoritative even if a later policy is
				// unsupported. Never recalculate or rewrite existing obligations.
				st.deals.deal.InstallmentDraft.PolicyVersion = 999
			}
			if _, err := s.RecordContract(context.Background(), p, st.deals.deal.ID, 8, "2026-09-29", "AMENDED-REF", nil); err != nil || immutable() != before || st.deals.deal.Version != 9 {
				t.Fatalf("contract re-record rewrote final facts: %v", err)
			}
			if _, err := s.SaveInstallmentTerms(context.Background(), p, st.deals.deal.ID, 9, termsInput("800000", "2026-01-01", 1)); !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("terms changed final plan: %v", err)
			}
			if _, err := s.SaveInstallmentPlan(context.Background(), p, st.deals.deal.ID, 9, manual); !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("duplicate manual plan: %v", err)
			}
			if _, err := s.RecordContract(context.Background(), p, st.deals.deal.ID, 8, "2026-09-29", "STALE", nil); !errors.Is(err, apperr.ErrStale) {
				t.Fatalf("stale command: %v", err)
			}
			if immutable() != before {
				t.Fatal("immutable obligations or receipt evidence changed")
			}
		})
	}
}
