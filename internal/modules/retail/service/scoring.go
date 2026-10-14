package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

const (
	demoScoringMode   = "demo"
	demoScoringResult = "completed"
)

// Scoring records an optional technical demonstration. It deliberately makes
// no financial assessment.
type Scoring struct{ Deps }

func NewScoring(d Deps) *Scoring { return &Scoring{d} }

type demoScoringFingerprint struct {
	CustomerID       string                  `json:"customerId"`
	Vehicle          *model.VehicleSnapshot  `json:"vehicle"`
	Price            string                  `json:"priceMinor"`
	Currency         string                  `json:"currency"`
	PaymentScheme    string                  `json:"paymentScheme"`
	InstallmentTerms *model.InstallmentDraft `json:"installmentTerms"`
}

func demoTermsDigest(d *model.Deal) (string, error) {
	raw, err := json.Marshal(demoScoringFingerprint{CustomerID: d.CustomerID, Vehicle: d.VehicleSnapshot, Price: d.PriceMinor, Currency: d.Currency, PaymentScheme: d.PaymentScheme, InstallmentTerms: d.InstallmentDraft})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Scoring) Run(ctx context.Context, p *auth.Principal, id string, expected int64) (*model.DemoScoring, error) {
	if err := p.Allow(model.PermDeals); err != nil {
		return nil, err
	}
	var out *model.DemoScoring
	err := s.store.InTx(ctx, func(st Store) error {
		d, err := (&Deal{Deps: s.Deps}).deal(ctx, st, p, id, expected)
		if err != nil {
			return err
		}
		if d.PaymentScheme != "own-installment" || d.InstallmentDraft == nil {
			return apperr.New(apperr.ErrConflict, "demo_scoring_ineligible", "demo scoring is available only for a new own-installment sale with terms")
		}
		digest, err := demoTermsDigest(d)
		if err != nil {
			return err
		}
		now := s.clock()
		out = &model.DemoScoring{ID: uuid.NewString(), CompanyID: p.CompanyID, DealID: d.ID, DealRevision: d.Version, TermsDigest: digest, Mode: demoScoringMode, Result: demoScoringResult, CompletedBy: p.UserID, CompletedAt: now, CreatedAt: now}
		if err := st.Scoring().Create(ctx, out); err != nil {
			return err
		}
		d.UpdatedAt = now
		if err := st.Deals().Update(ctx, d, d.Version); err != nil {
			return err
		}
		return s.event(ctx, st, p, "deal.demo_scoring_completed", "deal", d.ID, "", map[string]any{"mode": demoScoringMode, "result": demoScoringResult})
	})
	return out, err
}

// Current returns the latest recorded run and whether it still covers the
// current signing subject and terms. A missing run remains absent.
func (s *Scoring) Current(ctx context.Context, p *auth.Principal, d *model.Deal) (*model.DemoScoring, bool, error) {
	v, err := s.store.Scoring().Current(ctx, p.CompanyID, d.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	current, err := demoScoringCurrentFor(d, v)
	return v, current, err
}

func currentDemoScoring(ctx context.Context, st Store, d *model.Deal) (bool, error) {
	if d.PaymentScheme != "own-installment" {
		return true, nil
	}
	v, err := st.Scoring().Current(ctx, d.CompanyID, d.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return demoScoringCurrentFor(d, v)
}

func demoScoringCurrentFor(d *model.Deal, v *model.DemoScoring) (bool, error) {
	if v == nil {
		return false, nil
	}
	digest, err := demoTermsDigest(d)
	if err != nil {
		return false, err
	}
	return v.Mode == demoScoringMode && v.Result == demoScoringResult && v.TermsDigest == digest, nil
}
