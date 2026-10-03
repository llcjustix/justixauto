package service

import (
	"context"
	"errors"
	"math/big"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

const installmentDraftPolicy = "own-interest-free-equal"

type InstallmentTermsInput struct {
	DownPayment  money.Money `json:"downPayment"`
	TermMonths   int         `json:"termMonths"`
	FirstDueDate string      `json:"firstDueDate"`
}

// GenerateInstallmentDraft produces the interest-free snapshot with exact
// minor-unit arithmetic. The last row receives the division remainder.
func GenerateInstallmentDraft(price money.Money, in InstallmentTermsInput) (*model.InstallmentDraft, error) {
	var v apperr.Validation
	p, priceOK := price.Parse()
	if !priceOK || p.Sign() <= 0 {
		v.Add("price", "must be positive money")
	}
	down, downOK := in.DownPayment.Parse()
	if !downOK || in.DownPayment.Currency != price.Currency {
		v.Add("downPayment", "must be positive money in the sale currency")
	}
	if priceOK && downOK && (down.Sign() <= 0 || down.Cmp(p) >= 0) {
		v.Add("downPayment", "must be greater than zero and less than price")
	}
	if in.TermMonths < 1 || in.TermMonths > 1000 {
		v.Add("termMonths", "must be an integer from 1 to 1000")
	}
	first, err := time.Parse(time.DateOnly, in.FirstDueDate)
	if err != nil || first.Year() < 1 || first.Year() > 9999 {
		v.Add("firstDueDate", "must be a valid YYYY-MM-DD date in years 1 to 9999")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	principal := new(big.Int).Sub(p, down)
	months := big.NewInt(int64(in.TermMonths))
	if principal.Cmp(months) < 0 {
		return nil, apperr.FieldError("termMonths", "principal minor units must be at least termMonths")
	}
	lastYear := first.Year() + (int(first.Month())-1+in.TermMonths-1)/12
	if lastYear > 9999 {
		return nil, apperr.FieldError("firstDueDate", "generated dates must not exceed year 9999")
	}
	regular, remainder := new(big.Int), new(big.Int)
	regular.QuoRem(principal, months, remainder)
	balance := new(big.Int).Set(principal)
	draft := &model.InstallmentDraft{PolicyID: installmentDraftPolicy, PolicyVersion: 1, Price: price, DownPayment: in.DownPayment, TermMonths: in.TermMonths, FirstDueDate: first.Format(time.DateOnly), ScheduledTotal: money.Of(principal, price.Currency), RegularPayment: money.Of(regular, price.Currency), Rows: make([]model.InstallmentDraftRow, 0, in.TermMonths)}
	for i := 0; i < in.TermMonths; i++ {
		amount := new(big.Int).Set(regular)
		if i == in.TermMonths-1 {
			amount.Add(amount, remainder)
		}
		balance.Sub(balance, amount)
		due := installmentDueDate(first, i)
		draft.Rows = append(draft.Rows, model.InstallmentDraftRow{Number: i + 1, DueDate: due.Format(time.DateOnly), Amount: money.Of(amount, price.Currency), Balance: money.Of(balance, price.Currency)})
	}
	return draft, nil
}

func installmentDueDate(first time.Time, offset int) time.Time {
	month := int(first.Month()) + offset
	year := first.Year() + (month-1)/12
	month = (month-1)%12 + 1
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := first.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// SaveInstallmentTerms records generated terms for a legacy sale and finalizes
// them in the same transaction when the signed contract and invoice exist.
func (s *Deal) SaveInstallmentTerms(ctx context.Context, p *auth.Principal, id string, expected int64, in InstallmentTermsInput) (*model.Deal, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	var result *model.Deal
	err := s.store.InTx(ctx, func(st Store) error {
		d, err := s.deal(ctx, st, p, id, -1)
		if err != nil {
			return err
		}
		if d.Version != expected {
			return apperr.ErrStale
		}
		if d.Status != "reserved" && d.Status != "delivered" {
			return apperr.New(apperr.ErrConflict, "deal_closed", "the sale is "+d.Status)
		}
		if d.PaymentScheme != "own-installment" {
			return apperr.FieldError("installmentTerms", "only allowed for own-installment sales")
		}
		plan, err := st.Deals().InstallmentPlan(ctx, d.ID)
		if err != nil && !errors.Is(err, apperr.ErrNotFound) {
			return err
		}
		if plan != nil {
			return apperr.New(apperr.ErrConflict, "installment_plan_exists", "the sale already has an installment plan")
		}
		draft, err := GenerateInstallmentDraft(d.Price(), in)
		if err != nil {
			return err
		}
		down, err := initialInstallmentInvoice(ctx, st, d.ID)
		if err != nil {
			return err
		}
		if down != nil && (down.Currency != draft.DownPayment.Currency || down.AmountMinor != draft.DownPayment.AmountMinor) {
			return apperr.New(apperr.ErrConflict, "first_installment_mismatch", "issued first installment must match installment terms")
		}
		d.InstallmentDraft, d.UpdatedAt = draft, s.clock()
		if err := s.finalizeInstallmentDraft(ctx, st, p, d); err != nil {
			return err
		}
		if err := st.Deals().Update(ctx, d, expected); err != nil {
			return err
		}
		if err := s.event(ctx, st, p, "deal.installment_terms_saved", "deal", d.ID, "", nil); err != nil {
			return err
		}
		result = d
		return nil
	})
	return result, err
}
