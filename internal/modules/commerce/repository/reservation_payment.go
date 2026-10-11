package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/database"
)

type ReservationPaymentRepository struct{ db *gorm.DB }

func (r *ReservationPaymentRepository) conn(ctx context.Context) *gorm.DB {
	return database.Conn(ctx, r.db).WithContext(ctx)
}

func (r *ReservationPaymentRepository) Evidence(ctx context.Context, reservationID string) ([]model.ReservationEvidence, error) {
	rows := []model.ReservationEvidence{}
	err := r.conn(ctx).Where("reservation_id = ?", reservationID).Order("created_at, id").Find(&rows).Error
	return rows, database.Translate(err)
}
func (r *ReservationPaymentRepository) GetEvidence(ctx context.Context, id string) (*model.ReservationEvidence, error) {
	var row model.ReservationEvidence
	if err := r.conn(ctx).Where("id = ?", id).Take(&row).Error; err != nil { return nil, database.Translate(err) }
	return &row, nil
}
func (r *ReservationPaymentRepository) AddEvidence(ctx context.Context, e *model.ReservationEvidence) error {
	return database.Translate(r.conn(ctx).Create(e).Error)
}
func (r *ReservationPaymentRepository) DecideEvidence(ctx context.Context, e *model.ReservationEvidence, expected int64) error {
	err := database.UpdateVersioned(r.conn(ctx), &model.ReservationEvidence{}, e.ID, expected, map[string]any{
		"status": e.Status, "decision_reason": e.DecisionReason, "decided_by": e.DecidedBy, "decided_at": e.DecidedAt,
	})
	if err == nil { e.Version = expected + 1 }
	return err
}
func (r *ReservationPaymentRepository) Refund(ctx context.Context, reservationID string) (*model.ReservationRefund, error) {
	var row model.ReservationRefund
	if err := r.conn(ctx).Where("reservation_id = ?", reservationID).Take(&row).Error; err != nil { return nil, database.Translate(err) }
	return &row, nil
}
func (r *ReservationPaymentRepository) CreateRefund(ctx context.Context, refund *model.ReservationRefund) error {
	return database.Translate(r.conn(ctx).Create(refund).Error)
}
func (r *ReservationPaymentRepository) ConfirmRefund(ctx context.Context, refund *model.ReservationRefund) error {
	result := r.conn(ctx).Model(&model.ReservationRefund{}).Where("reservation_id = ? AND status = 'requested'", refund.ReservationID).Updates(map[string]any{
		"status": refund.Status, "amount_minor": refund.AmountMinor, "currency": refund.Currency,
		"external_reference": refund.ExternalReference, "attachment_ids": refund.AttachmentIDs,
		"refunded_on": refund.RefundedOn, "confirmed_by": refund.ConfirmedBy, "confirmed_at": refund.ConfirmedAt,
	})
	if err := database.Translate(result.Error); err != nil { return err }
	if result.RowsAffected != 1 { return apperr.ErrStale }
	return nil
}
func (r *ReservationPaymentRepository) CreditForReservation(ctx context.Context, reservationID string) (*model.ReservationCredit, error) {
	var row model.ReservationCredit
	if err := r.conn(ctx).Where("reservation_id = ?", reservationID).Take(&row).Error; err != nil { return nil, database.Translate(err) }
	return &row, nil
}
func (r *ReservationPaymentRepository) CreditForOrder(ctx context.Context, orderID string) (*model.ReservationCredit, error) {
	var row model.ReservationCredit
	if err := r.conn(ctx).Where("order_id = ?", orderID).Take(&row).Error; err != nil { return nil, database.Translate(err) }
	return &row, nil
}
func (r *ReservationPaymentRepository) CreateCredit(ctx context.Context, credit *model.ReservationCredit) error {
	return database.Translate(r.conn(ctx).Create(credit).Error)
}
func (r *ReservationPaymentRepository) AttachInvoice(ctx context.Context, reservationID, invoiceID string, at time.Time) error {
	result := r.conn(ctx).Model(&model.ReservationCredit{}).Where("reservation_id = ? AND invoice_id IS NULL", reservationID).Updates(map[string]any{
		"invoice_id": invoiceID, "invoice_attached_at": at,
	})
	if err := database.Translate(result.Error); err != nil { return err }
	if result.RowsAffected != 1 { return apperr.ErrStale }
	return nil
}
