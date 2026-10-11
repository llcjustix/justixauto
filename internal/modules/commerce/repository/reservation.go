package repository

import (
	"context"
	"time"
	"gorm.io/gorm/clause"
	"gorm.io/gorm"
	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/database"
)

type ReservationRepository struct{ db *gorm.DB }

func (r *ReservationRepository) Create(ctx context.Context, x *model.Reservation, lines []model.ReservationLine) error {
	return database.Conn(ctx, r.db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(x).Error; err != nil {
			return database.Translate(err)
		}
		return database.Translate(tx.Create(&lines).Error)
	})
}
func (r *ReservationRepository) Lines(ctx context.Context, id string) ([]model.ReservationLine, error) {
	var lines []model.ReservationLine
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("reservation_id = ?", id).Order("id").Find(&lines).Error
	return lines, database.Translate(err)
}
func (r *ReservationRepository) Get(ctx context.Context, companyID, id string) (*model.Reservation, []model.ReservationLine, error) {
	var x model.Reservation
	db := database.Conn(ctx, r.db).WithContext(ctx)
	if err := db.Where("id = ? AND (buyer_company_id = ? OR supplier_company_id = ?)", id, companyID, companyID).Take(&x).Error; err != nil {
		return nil, nil, database.Translate(err)
	}
	lines, err := r.Lines(ctx, id)
	return &x, lines, err
}
func (r *ReservationRepository) List(ctx context.Context, companyID string, limit, offset int) ([]model.Reservation, error) {
	limit, offset = database.Page(limit, offset)
	rows := []model.Reservation{}
	err := database.Conn(ctx, r.db).WithContext(ctx).Where("buyer_company_id = ? OR supplier_company_id = ?", companyID, companyID).Order("updated_at DESC, id").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, database.Translate(err)
}
func (r *ReservationRepository) ForUpdate(ctx context.Context, id string) (*model.Reservation, []model.ReservationLine, error) {
	db := database.Conn(ctx, r.db).WithContext(ctx)
	var x model.Reservation
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&x).Error; err != nil {
		return nil, nil, database.Translate(err)
	}
	var lines []model.ReservationLine
	if err := db.Where("reservation_id = ?", id).Order("id").Find(&lines).Error; err != nil {
		return nil, nil, database.Translate(err)
	}
	return &x, lines, nil
}
func (r *ReservationRepository) ByOrderForUpdate(ctx context.Context, orderID string) (*model.Reservation, []model.ReservationLine, error) {
	db := database.Conn(ctx,r.db).WithContext(ctx)
	var x model.Reservation
	if err:=db.Clauses(clause.Locking{Strength:"UPDATE"}).Where("order_id = ?",orderID).Take(&x).Error;err!=nil{return nil,nil,database.Translate(err)}
	lines,err:=r.Lines(ctx,x.ID)
	return &x,lines,err
}
func (r *ReservationRepository) Update(ctx context.Context, x *model.Reservation, expected int64) error {
	err := database.UpdateVersioned(database.Conn(ctx, r.db).WithContext(ctx), &model.Reservation{}, x.ID, expected, map[string]any{"status": x.Status, "offered_terms": x.OfferedTerms, "offered_digest": x.OfferedDigest, "offered_revision": x.OfferedRevision, "current_terms": x.CurrentTerms, "current_digest": x.CurrentDigest, "current_revision": x.CurrentRevision, "order_id":x.OrderID,"continuation_proposed":x.ContinuationProposed,"continuation_accepted":x.ContinuationAccepted,"continuation_proposed_digest":x.ContinuationProposedDigest,"continuation_accepted_digest":x.ContinuationAcceptedDigest,"continuation_proposed_revision":x.ContinuationProposedRevision,"continuation_accepted_revision":x.ContinuationAcceptedRevision,"continuation_proposed_by":x.ContinuationProposedBy,"continuation_accepted_by":x.ContinuationAcceptedBy,"expires_at": x.ExpiresAt, "advance_due_at":x.AdvanceDueAt,"extended_by":x.ExtendedBy,"extended_at":x.ExtendedAt,"released_at":x.ReleasedAt,"release_reason":x.ReleaseReason,"updated_at": x.UpdatedAt})
	if err == nil {
		x.Version = expected + 1
	}
	return err
}

// DueCandidates is a bounded keyset scan. Selection is only a hint: callers
// lock and recheck every row before changing it.
func (r *ReservationRepository) DueCandidates(ctx context.Context, now time.Time, afterAt *time.Time, afterID string, limit int) ([]model.ReservationDueCandidate, error) {
	const deadline = "LEAST(expires_at, COALESCE(advance_due_at, expires_at))"
	db := database.Conn(ctx, r.db).WithContext(ctx).Model(&model.Reservation{}).
		Select("id, "+deadline+" AS deadline").
		Where("status = ? AND order_id IS NULL AND expires_at IS NOT NULL AND "+deadline+" <= ?", model.ReservationActive, now)
	if afterAt != nil { db = db.Where("("+deadline+", id) > (?, ?)", *afterAt, afterID) }
	rows := []model.ReservationDueCandidate{}
	err := db.Order(deadline+", id").Limit(limit).Scan(&rows).Error
	return rows, database.Translate(err)
}

