package service

import (
	"context"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/money"
	"justixauto/internal/pkg/validate"
)

// validateTerms normalizes terms and returns the exact total. All amounts
// share one currency (no conversion); a payment schedule must add up to the
// total exactly. Line and installment IDs are generated when missing.
func (d Deps) validateTerms(ctx context.Context, v *apperr.Validation, in model.Terms) model.Terms {
	return d.validateTermsSelections(ctx, v, in, false, nil)
}

// New orders and quotations require a chosen pair. Offers may leave either
// color open. Addenda retain unchanged saved facts, including legacy unknowns.
func (d Deps) validateTermsSelections(ctx context.Context, v *apperr.Validation, in model.Terms, required bool, previous []model.Line) model.Terms {
	out := model.Terms{
		Route: in.Route, Lines: []model.Line{}, PaymentSchedule: []model.Installment{},
		DeliveryTerms: validate.Text(v, "terms.deliveryTerms", in.DeliveryTerms, 0, 2000),
		WarrantyTerms: validate.Text(v, "terms.warrantyTerms", in.WarrantyTerms, 0, 2000),
		ServiceTerms:  validate.Text(v, "terms.serviceTerms", in.ServiceTerms, 0, 2000),
	}
	if !slices.Contains(model.Routes, in.Route) {
		v.Add("terms.route", "must be factory, foreign-direct, in-transit or local")
	}
	if len(in.Lines) == 0 || len(in.Lines) > 100 {
		v.Add("terms.lines", "list 1-100 lines")
	}
	currency := ""
	ids := map[string]bool{}
	total := d.validateLines(ctx, v, in.Lines, ids, &out.Lines, &currency, required, previous)
	if !money.Fits(total) {
		v.Add("terms.lines", "total is too large")
	}
	if len(in.PaymentSchedule) > 60 {
		v.Add("terms.paymentSchedule", "at most 60 installments")
	}
	scheduled := validateSchedule(v, in.PaymentSchedule, ids, &out.PaymentSchedule, &currency)
	if len(in.PaymentSchedule) > 0 && scheduled.Cmp(total) != 0 {
		v.Add("terms.paymentSchedule", "installments must add up to the total "+total.String())
	}
	return out
}

// sameCurrency records the first currency seen and flags any later amount
// that uses a different one.
func sameCurrency(v *apperr.Validation, currency *string, field, c string) {
	if *currency == "" {
		*currency = c
	} else if c != *currency {
		v.Add(field, "all amounts must use one currency ("+*currency+")")
	}
}

// validateLines normalizes order lines, checks each against the catalog and
// returns the exact total (unit price * quantity, summed). Generated or
// duplicate line IDs are tracked in ids so the payment schedule can also
// reject collisions against them.
func (d Deps) validateLines(ctx context.Context, v *apperr.Validation, lines []model.Line, ids map[string]bool, out *[]model.Line, currency *string, required bool, previous []model.Line) *big.Int {
	total := new(big.Int)
	saved := make(map[string]model.Line, len(previous))
	for _, l := range previous {
		saved[l.LineID] = l
	}
	catalog := map[string]*Model{}
	for i, l := range lines {
		field := "terms.lines." + strconv.Itoa(i)
		if l.LineID == "" {
			l.LineID = uuid.NewString()
		}
		if uuid.Validate(l.LineID) != nil || ids[l.LineID] {
			v.Add(field+".lineId", "must be a unique ID")
		}
		ids[l.LineID] = true
		if l.Quantity < 1 || l.Quantity > 10_000 {
			v.Add(field+".quantity", "must be 1-10000")
		}
		price, ok := l.UnitPrice.Parse()
		if !ok || price.Sign() == 0 {
			v.Add(field+".unitPrice", "must be a positive amount in minor units with a currency code")
		} else {
			sameCurrency(v, currency, field+".unitPrice", l.UnitPrice.Currency)
			total.Add(total, new(big.Int).Mul(price, big.NewInt(int64(l.Quantity))))
		}
		unchanged := false
		if old, ok := saved[l.LineID]; ok && old.ModelID == l.ModelID {
			// Older clients omit these fields on unrelated commercial edits.
			// Omission is not permission to repin from the current catalog.
			if l.ModelSpecificationVersion == "" {
				l.ModelSpecificationVersion = old.ModelSpecificationVersion
			}
			if l.ExteriorColor == "" {
				l.ExteriorColor = old.ExteriorColor
			}
			if l.InteriorColor == "" {
				l.InteriorColor = old.InteriorColor
			}
			l.OfferLineID = old.OfferLineID
			unchanged = l.ModelSpecificationVersion == old.ModelSpecificationVersion && l.ExteriorColor == old.ExteriorColor && l.InteriorColor == old.InteriorColor
		}
		if uuid.Validate(l.ModelID) != nil {
			v.Add(field+".modelId", "must be a valid ID")
		} else if !unchanged {
			l = d.selectColors(ctx, v, field, l, required, catalog)
		}
		*out = append(*out, l)
	}
	return total
}

// selectColors resolves only new/explicitly changed facts. A missing version
// is pinned once per model in this request; an explicit version never falls
// back to the current palette.
func (d Deps) selectColors(ctx context.Context, v *apperr.Validation, field string, l model.Line, required bool, models map[string]*Model) model.Line {
	m, ok := models[l.ModelID]
	if !ok {
		var err error
		m, err = d.catalog.Model(ctx, l.ModelID)
		if err != nil || m == nil {
			v.Add(field+".modelId", "unknown vehicle model")
			return l
		}
		models[l.ModelID] = m
	}
	l.ModelSpecificationVersion = strings.TrimSpace(l.ModelSpecificationVersion)
	if l.ModelSpecificationVersion == "" {
		l.ModelSpecificationVersion = m.CurrentSpecificationVersion
	}
	for _, spec := range m.Specifications {
		if spec.Version != "" && spec.Version == l.ModelSpecificationVersion {
			l.ExteriorColor = selectColor(v, field+".exteriorColor", l.ExteriorColor, spec.ExteriorColors, required)
			l.InteriorColor = selectColor(v, field+".interiorColor", l.InteriorColor, spec.InteriorColors, required)
			return l
		}
	}
	v.Add(field+".modelSpecificationVersion", "unknown specification version")
	return l
}

func selectColor(v *apperr.Validation, field, value string, palette []string, required bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		if !required {
			return ""
		}
		if len(palette) == 1 {
			return palette[0]
		}
		v.Add(field, "choose one color from the specification palette")
		return ""
	}
	for _, allowed := range palette {
		if strings.EqualFold(value, allowed) {
			return allowed
		}
	}
	v.Add(field, "must be a color in the specification palette")
	return value
}

// validateSchedule normalizes the payment schedule and returns the total
// amount scheduled. ids also tracks line IDs so schedule IDs cannot collide
// with them.
func validateSchedule(v *apperr.Validation, schedule []model.Installment, ids map[string]bool, out *[]model.Installment, currency *string) *big.Int {
	scheduled := new(big.Int)
	for i, p := range schedule {
		field := "terms.paymentSchedule." + strconv.Itoa(i)
		if p.ID == "" {
			p.ID = uuid.NewString()
		}
		if uuid.Validate(p.ID) != nil || ids[p.ID] {
			v.Add(field+".id", "must be a unique ID")
		}
		ids[p.ID] = true
		amount, ok := p.Amount.Parse()
		if !ok || amount.Sign() == 0 {
			v.Add(field+".amount", "must be a positive amount in minor units with a currency code")
		} else {
			sameCurrency(v, currency, field+".amount", p.Amount.Currency)
			scheduled.Add(scheduled, amount)
		}
		if _, err := time.Parse(time.DateOnly, p.DueDate); err != nil {
			v.Add(field+".dueDate", "must be a date (YYYY-MM-DD)")
		}
		*out = append(*out, p)
	}
	return scheduled
}

func validateAudience(v *apperr.Validation, own string, in model.Audience) model.Audience {
	out := model.Audience{Mode: in.Mode, PartnerCompanyIDs: validate.UniqueIDs(v, "audience.partnerCompanyIds", in.PartnerCompanyIDs)}
	switch in.Mode {
	case "all-active":
		if len(out.PartnerCompanyIDs) > 0 {
			v.Add("audience.partnerCompanyIds", "must be empty for all-active")
		}
	case "selected":
		if len(out.PartnerCompanyIDs) == 0 || len(out.PartnerCompanyIDs) > 500 {
			v.Add("audience.partnerCompanyIds", "select 1-500 partners")
		}
		if slices.Contains(out.PartnerCompanyIDs, own) {
			v.Add("audience.partnerCompanyIds", "cannot include your own company")
		}
	default:
		v.Add("audience.mode", "must be all-active or selected")
	}
	return out
}
