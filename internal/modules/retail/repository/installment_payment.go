package repository

import (
	"context"

	"gorm.io/gorm/clause"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/database"
)

// LockDeal takes the company-scoped deal lock before any monthly invoice locks.
// Callers must bind their transaction in ctx.
func (r *DealRepository) LockDeal(ctx context.Context, companyID, id string) (*model.Deal, error) {
	var d model.Deal
	err := database.Conn(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND company_id = ?", id, companyID).Take(&d).Error
	if err != nil {
		return nil, database.Translate(err)
	}
	return &d, nil
}

// LockInstallmentInvoices locks selected monthly invoices in UUID order after
// LockDeal has established the common deal lock.
func (r *DealRepository) LockInstallmentInvoices(ctx context.Context, companyID, dealID string, invoiceIDs []string) ([]model.Invoice, error) {
	invoices := []model.Invoice{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("company_id = ? AND deal_id = ? AND purpose = ? AND id IN ?", companyID, dealID, "monthly-installment", invoiceIDs).
		Order("id ASC").Find(&invoices).Error
	return invoices, database.Translate(err)
}

func (r *DealRepository) CreateInstallmentPayment(ctx context.Context, p *model.InstallmentPayment) error {
	return database.Translate(database.Conn(ctx, r.db).WithContext(ctx).Create(p).Error)
}

func (r *DealRepository) InstallmentPayment(ctx context.Context, companyID, id string) (*model.InstallmentPayment, error) {
	var p model.InstallmentPayment
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("id = ? AND company_id = ?", id, companyID).Take(&p).Error
	if err != nil {
		return nil, database.Translate(err)
	}
	return &p, nil
}

func (r *DealRepository) InstallmentPaymentByReference(ctx context.Context, companyID, dealID, externalReference string) (*model.InstallmentPayment, error) {
	var p model.InstallmentPayment
	err := database.Conn(ctx, r.db).WithContext(ctx).
		Where("company_id = ? AND deal_id = ? AND external_reference = ? AND status <> ?", companyID, dealID, externalReference, "rejected").
		Take(&p).Error
	if err != nil {
		return nil, database.Translate(err)
	}
	return &p, nil
}

// InstallmentPaymentEvidence returns every allocation in stable invoice order,
// so review and decision callers always see the complete parent group.
func (r *DealRepository) InstallmentPaymentEvidence(ctx context.Context, paymentID string) ([]model.Evidence, error) {
	evidence := []model.Evidence{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("payment_group_id = ?", paymentID).
		Order("invoice_id ASC").Order("id ASC").Find(&evidence).Error
	return evidence, database.Translate(err)
}

func (r *DealRepository) UpdateInstallmentPayment(ctx context.Context, p *model.InstallmentPayment, expected int64) error {
	err := database.UpdateVersioned(database.Conn(ctx, r.db).WithContext(ctx), &model.InstallmentPayment{}, p.ID, expected, map[string]any{
		"status": p.Status, "decision_reason": p.DecisionReason, "decided_by": p.DecidedBy, "decided_at": p.DecidedAt,
	})
	if err == nil {
		p.Version = expected + 1
	}
	return err
}
