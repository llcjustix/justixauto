package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"sort"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
	"justixauto/internal/pkg/validate"
)

type InstallmentPlanRowInput struct {
	DueDate string      `json:"dueDate"`
	Amount  money.Money `json:"amount"`
}

type InstallmentPlanInput struct {
	ContractTotal money.Money               `json:"contractTotal"`
	Rows          []InstallmentPlanRowInput `json:"rows"`
}

type installmentPlanRow struct {
	due    time.Time
	amount *big.Int
}

// SaveInstallmentPlan stores only terms explicitly copied from a signed
// contract; it never calculates a repayment schedule.
func (s *Deal) SaveInstallmentPlan(ctx context.Context, p *auth.Principal, id string, expected int64, in InstallmentPlanInput) (*model.Deal, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	var v apperr.Validation
	total, ok := in.ContractTotal.Parse()
	if !ok || total.Sign() <= 0 {
		v.Add("contractTotal", "must be a positive amount in minor units with a currency code")
	}
	if len(in.Rows) < 1 || len(in.Rows) > 1000 {
		v.Add("rows", "must contain between 1 and 1000 rows")
	}
	rows := make([]installmentPlanRow, len(in.Rows))
	for n, input := range in.Rows {
		amount, valid := input.Amount.Parse()
		if !valid || amount.Sign() <= 0 {
			v.Add("rows", "each row amount must be positive money")
			continue
		}
		due, err := time.Parse(time.DateOnly, input.DueDate)
		if err != nil {
			v.Add("rows", "each dueDate must be a valid YYYY-MM-DD date")
			continue
		}
		rows[n] = installmentPlanRow{due: due, amount: amount}
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	var d *model.Deal
	err := s.store.InTx(ctx, func(st Store) error {
		var err error
		if d, err = s.deal(ctx, st, p, id, -1); err != nil {
			return err
		}
		if d.Version != expected {
			return apperr.ErrStale
		}
		if d.PaymentScheme != "own-installment" || (d.Status != "reserved" && d.Status != "delivered") {
			return apperr.New(apperr.ErrConflict, "installment_plan_ineligible", "the sale must be an active own-installment sale")
		}
		if d.InstallmentDraft != nil {
			return apperr.New(apperr.ErrConflict, "installment_draft_exists", "generated terms finalize automatically; manual rows cannot replace them")
		}
		if d.ContractSignedOn == nil || d.ContractReference == "" {
			return apperr.New(apperr.ErrConflict, "signed_contract_required", "record the signed contract first")
		}
		if in.ContractTotal.Currency != d.Currency {
			return apperr.FieldError("contractTotal", "must be in "+d.Currency)
		}
		if existing, err := st.Deals().InstallmentPlan(ctx, d.ID); err == nil && existing != nil {
			return apperr.New(apperr.ErrConflict, "installment_plan_exists", "the sale already has an installment plan")
		} else if err != nil && !errors.Is(err, apperr.ErrNotFound) {
			return err
		}
		down, err := initialInstallmentInvoice(ctx, st, d.ID)
		if err != nil {
			return err
		}
		if down == nil {
			return apperr.New(apperr.ErrConflict, "initial_invoice_required", "issue the first-installment invoice first")
		}
		downAmount := amount(down.AmountMinor)
		if down.Currency != d.Currency || downAmount.Sign() <= 0 {
			return apperr.New(apperr.ErrConflict, "initial_invoice_invalid", "the first-installment invoice must be positive and in the sale currency")
		}
		scheduled := new(big.Int)
		for n, row := range rows {
			if row.amount == nil {
				return apperr.FieldError("rows", "each row is required")
			}
			if in.Rows[n].Amount.Currency != d.Currency {
				return apperr.FieldError("rows", "each row amount must be in "+d.Currency)
			}
			scheduled.Add(scheduled, row.amount)
		}
		expectedRows := new(big.Int).Sub(total, downAmount)
		if expectedRows.Sign() < 0 || scheduled.Cmp(expectedRows) != 0 {
			return apperr.FieldError("rows", "row total must equal contractTotal less the first-installment invoice")
		}
		if err := s.materializeInstallmentPlan(ctx, st, p, d, down, total.String(), rows); err != nil {
			return err
		}
		d.UpdatedAt = s.clock()
		if err := st.Deals().Update(ctx, d, expected); err != nil {
			return err
		}
		return nil
	})
	return d, err
}

func initialInstallmentInvoice(ctx context.Context, st Store, dealID string) (*model.Invoice, error) {
	invoices, err := st.Deals().Invoices(ctx, dealID)
	if err != nil {
		return nil, err
	}
	var down *model.Invoice
	for i := range invoices {
		if invoices[i].Purpose == "first-installment" && invoices[i].Status == "issued" {
			if down != nil {
				return nil, apperr.New(apperr.ErrConflict, "initial_invoice_ambiguous", "multiple initial invoices exist")
			}
			down = &invoices[i]
		}
	}
	return down, nil
}

// Validate against the supported policy without replacing the saved snapshot.
func validateInstallmentDraft(d *model.Deal) error {
	draft := d.InstallmentDraft
	if draft.PolicyID != installmentDraftPolicy || draft.PolicyVersion != 1 {
		return apperr.New(apperr.ErrConflict, "installment_policy_unsupported", "the saved installment policy is unsupported")
	}
	generated, err := GenerateInstallmentDraft(d.Price(), InstallmentTermsInput{DownPayment: draft.DownPayment, TermMonths: draft.TermMonths, FirstDueDate: draft.FirstDueDate})
	if err != nil || !reflect.DeepEqual(draft, generated) {
		return apperr.New(apperr.ErrConflict, "installment_draft_invalid", "saved installment terms do not match the sale and policy")
	}
	return nil
}

// finalizeInstallmentDraft joins the originating command's transaction. That
// command owns the single versioned deal update and its own audit event.
func (s *Deal) finalizeInstallmentDraft(ctx context.Context, st Store, p *auth.Principal, d *model.Deal) error {
	if d.InstallmentDraft == nil {
		return nil
	}
	if plan, err := st.Deals().InstallmentPlan(ctx, d.ID); err == nil && plan != nil {
		return nil // Existing obligations, including legacy plans, are immutable.
	} else if err != nil && !errors.Is(err, apperr.ErrNotFound) {
		return err
	}
	if d.PaymentScheme != "own-installment" || (d.Status != "reserved" && d.Status != "delivered") {
		return apperr.New(apperr.ErrConflict, "installment_plan_ineligible", "the sale must be an active own-installment sale")
	}
	if err := validateInstallmentDraft(d); err != nil {
		return err
	}
	down, err := initialInstallmentInvoice(ctx, st, d.ID)
	if err != nil {
		return err
	}
	if down != nil && (down.Currency != d.InstallmentDraft.DownPayment.Currency || down.AmountMinor != d.InstallmentDraft.DownPayment.AmountMinor) {
		return apperr.New(apperr.ErrConflict, "first_installment_mismatch", "issued first installment must match installment terms")
	}
	if down == nil || d.ContractSignedOn == nil || d.ContractReference == "" {
		return nil
	}
	rows := make([]installmentPlanRow, len(d.InstallmentDraft.Rows))
	for n, row := range d.InstallmentDraft.Rows {
		// Exact snapshot validation above guarantees positive amounts and dates.
		due, _ := time.Parse(time.DateOnly, row.DueDate)
		rows[n] = installmentPlanRow{due: due, amount: amount(row.Amount.AmountMinor)}
	}
	return s.materializeInstallmentPlan(ctx, st, p, d, down, d.PriceMinor, rows)
}

// materializeInstallmentPlan only inserts validated obligations and their audit
// event. Both generated and legacy manual paths supply their validated rows.
func (s *Deal) materializeInstallmentPlan(ctx context.Context, st Store, p *auth.Principal, d *model.Deal, down *model.Invoice, total string, rows []installmentPlanRow) error {
	customer, err := st.CRM().Customer(ctx, d.CompanyID, d.CustomerID)
	if err != nil {
		return err
	}
	plan := &model.InstallmentPlan{ID: uuid.NewString(), DealID: d.ID, CompanyID: d.CompanyID, ContractReference: d.ContractReference,
		ContractSignedOn: *d.ContractSignedOn, ContractFileIDs: append([]byte(nil), d.ContractFileIDs...), ContractTotalMinor: total, Currency: d.Currency,
		DownPaymentInvoiceID: down.ID, DownPaymentMinor: down.AmountMinor, CreatedBy: p.UserID, CreatedAt: s.clock()}
	if err := st.Deals().CreateInstallmentPlan(ctx, plan); err != nil {
		return err
	}
	for n, row := range rows {
		number, planID, due := n+1, plan.ID, row.due
		i := &model.Invoice{ID: uuid.NewString(), DealID: d.ID, CompanyID: d.CompanyID, Purpose: "monthly-installment", AmountMinor: row.amount.String(), Currency: d.Currency,
			RecipientSnapshot: customer.DisplayName, DueDate: &due, InstallmentPlanID: &planID, InstallmentNumber: &number, Status: "issued", Version: 1, CreatedAt: s.clock()}
		if err := st.Deals().CreateInvoice(ctx, i); err != nil {
			return err
		}
	}
	return s.event(ctx, st, p, "deal.installment_plan_created", "deal", d.ID, "", map[string]any{"installmentPlanId": plan.ID})
}

type InstallmentPlanRowView struct {
	Number  int
	Invoice InvoiceView
}

type InstallmentPlanView struct {
	Plan                                       model.InstallmentPlan
	State                                      string
	Rows                                       []InstallmentPlanRowView
	ScheduledTotal, Paid, Pending, Outstanding money.Money
	Available                                  money.Money
	SettlementState                            string
	AllowedActions                             []string
	Payments                                   []InstallmentPaymentView
}

// Reuse the invoice views so each row and its full invoice share one evidence
// snapshot within this response, even when another row changes concurrently.
func (s *Deal) installmentPlanView(ctx context.Context, st Store, d *model.Deal, invoices []InvoiceView) (*InstallmentPlanView, error) {
	p, err := st.Deals().InstallmentPlan(ctx, d.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := &InstallmentPlanView{Plan: *p, State: map[bool]string{true: "active"}[d.Status == "delivered"], Rows: []InstallmentPlanRowView{}, SettlementState: "outstanding", AllowedActions: []string{}, Payments: []InstallmentPaymentView{}}
	if d.Status == "reserved" {
		v.State = "planned"
	}
	if d.Status == "cancelled" {
		v.State = "cancelled"
	}
	scheduled, paid, pending, outstanding := new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	available := new(big.Int)
	seenPayments := map[string]bool{}
	for i := range invoices {
		iv := &invoices[i]
		if iv.Invoice.Purpose != "monthly-installment" || iv.Invoice.InstallmentPlanID == nil || *iv.Invoice.InstallmentPlanID != p.ID || iv.Invoice.InstallmentNumber == nil {
			continue
		}
		scheduled.Add(scheduled, amount(iv.Invoice.AmountMinor))
		paid.Add(paid, amount(iv.Paid.AmountMinor))
		pending.Add(pending, amount(iv.Pending.AmountMinor))
		outstanding.Add(outstanding, amount(iv.Outstanding.AmountMinor))
		available.Add(available, amount(iv.Available.AmountMinor))
		for _, action := range iv.AllowedActions {
			if action == "submit-payment" {
				v.AllowedActions = []string{"submit-installment-payment"}
			}
		}
		for _, payment := range iv.PaymentGroups {
			if !seenPayments[payment.Payment.ID] {
				seenPayments[payment.Payment.ID] = true
				v.Payments = append(v.Payments, payment)
			}
		}
		v.Rows = append(v.Rows, InstallmentPlanRowView{Number: *iv.Invoice.InstallmentNumber, Invoice: *iv})
	}
	sort.Slice(v.Rows, func(i, j int) bool { return v.Rows[i].Number < v.Rows[j].Number })
	v.ScheduledTotal, v.Paid, v.Pending, v.Outstanding = money.Of(scheduled, p.Currency), money.Of(paid, p.Currency), money.Of(pending, p.Currency), money.Of(outstanding, p.Currency)
	v.Available = money.Of(available, p.Currency)
	if len(v.Rows) > 0 && outstanding.Sign() == 0 {
		v.SettlementState = "settled"
	}
	return v, nil
}

func copiedIDs(raw []byte) []string { var out []string; _ = json.Unmarshal(raw, &out); return out }

func validateInstallmentPlanID(id string) error { return validate.IDs(id) }

func (s *Deal) monthlyPaymentEligibility(ctx context.Context, st Store, d *model.Deal, i *model.Invoice) error {
	if i.Purpose != "monthly-installment" {
		return nil
	}
	ineligible := apperr.New(apperr.ErrConflict, "monthly_payment_ineligible", "monthly payment evidence requires a reserved or delivered linked installment sale")
	if d.PaymentScheme != "own-installment" || (d.Status != "reserved" && d.Status != "delivered") || i.Status != "issued" ||
		i.CompanyID != d.CompanyID || i.DealID != d.ID ||
		i.InstallmentPlanID == nil || i.InstallmentNumber == nil || *i.InstallmentNumber < 1 || i.Currency != d.Currency {
		return ineligible
	}
	plan, err := st.Deals().InstallmentPlan(ctx, d.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		return ineligible
	}
	if err != nil {
		return err
	}
	if plan == nil || plan.ID != *i.InstallmentPlanID || plan.CompanyID != d.CompanyID || plan.DealID != d.ID || plan.Currency != i.Currency {
		return ineligible
	}
	return nil
}

// actorInvoiceView projects actions in the service; DTOs only copy them.
func (s *Deal) actorInvoiceView(ctx context.Context, st Store, p *auth.Principal, i *model.Invoice) (*InvoiceView, error) {
	if p == nil || p.CompanyID == "" || p.CompanyID != i.CompanyID {
		return nil, apperr.ErrNotFound
	}
	d, err := s.deal(ctx, st, p, i.DealID, -1)
	if err != nil {
		return nil, err
	}
	v, err := s.invoiceView(ctx, st, i)
	if err != nil {
		return nil, err
	}
	seenPayments := map[string]bool{}
	for _, e := range v.Evidence {
		if e.PaymentGroupID == nil || seenPayments[*e.PaymentGroupID] {
			continue
		}
		seenPayments[*e.PaymentGroupID] = true
		group, err := st.Deals().InstallmentPayment(ctx, p.CompanyID, *e.PaymentGroupID)
		if err != nil {
			return nil, err
		}
		payment, err := s.installmentPaymentView(ctx, st, p, d, group)
		if err != nil {
			return nil, err
		}
		v.PaymentGroups = append(v.PaymentGroups, *payment)
	}
	if err := s.monthlyPaymentEligibility(ctx, st, d, i); err != nil {
		if errors.Is(err, apperr.ErrConflict) {
			return v, nil
		}
		return nil, err
	}
	if p.Has(model.PermDeals) && i.Status == "issued" && amount(v.Available.AmountMinor).Sign() > 0 {
		v.AllowedActions = []string{"submit-payment"}
	}
	if p.Has(model.PermPaymentsAccept) {
		for _, e := range v.Evidence {
			if e.Status == "submitted" && e.PaymentGroupID == nil {
				v.EvidenceActions[e.ID] = []string{"accept", "reject"}
			}
		}
	}
	return v, nil
}
