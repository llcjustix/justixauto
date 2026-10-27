package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/database"
)

type AuditRepository struct{ db *gorm.DB }

func (r *AuditRepository) Append(ctx context.Context, e *model.AuditEvent) error {
	// The trigger owns chain allocation. RETURNING reads its final values; do not
	// allocate positions or retry inserts in memory (sequence gaps are harmless).
	return translate(database.Conn(ctx, r.db).WithContext(ctx).
		Clauses(clause.Returning{}).Create(e).Error)
}

func (r *AuditRepository) Chain(ctx context.Context) (*model.AuditChain, error) {
	var chain model.AuditChain
	if err := translate(database.Conn(ctx, r.db).WithContext(ctx).
		Select("identity_audit_chain.*, public.identity_audit_boundary_v1(stream_id, legacy_max_seq, legacy_count, created_at) AS reconstructed_boundary").
		Where("singleton = ?", true).Take(&chain).Error); err != nil {
		return nil, err
	}
	return &chain, nil
}

func (r *AuditRepository) Get(ctx context.Context, id string) (*model.AuditEvent, error) {
	var row model.AuditEvidence
	err := translate(database.Conn(ctx, r.db).WithContext(ctx).Table("identity_audit_events AS event").
		Select("event.*, host(event.client_ip) AS client_ip_host").Where("event.id = ?", id).Take(&row).Error)
	row.ClientIP = row.ClientIPHost
	return &row.AuditEvent, err
}

func (r *AuditRepository) GetByIndex(ctx context.Context, index int64) (*model.AuditEvent, error) {
	var event model.AuditEvent
	err := translate(database.Conn(ctx, r.db).WithContext(ctx).Where("chain_index = ?", index).Take(&event).Error)
	return &event, err
}

// CanonicalPayload is also available to bounded evidence consumers. Export uses
// Page's single SELECT so reconstruction and displayed fields share a snapshot.
func (r *AuditRepository) CanonicalPayload(ctx context.Context, id string) ([]byte, error) {
	var row struct{ Payload []byte }
	err := translate(database.Conn(ctx, r.db).WithContext(ctx).
		Table("identity_audit_events AS event").
		Select("public.identity_audit_canonical_v1(head.stream_id, event) AS payload").
		Joins("CROSS JOIN public.identity_audit_chain AS head").
		Where("event.id = ? AND head.singleton = ?", id, true).Take(&row).Error)
	return row.Payload, err
}

func (r *AuditRepository) Page(ctx context.Context, f model.AuditFilter, w model.AuditWindow) ([]model.AuditEvidence, error) {
	q := database.Conn(ctx, r.db).WithContext(ctx).Table("identity_audit_events AS event")
	if w.Reconstruct {
		q = q.Select("event.*, host(event.client_ip) AS client_ip_host, public.identity_audit_canonical_v1(?, event) AS reconstructed_payload", w.StreamID)
	} else {
		q = q.Select("event.*, host(event.client_ip) AS client_ip_host")
	}
	if w.Ascending {
		if w.LegacyOnly {
			q = q.Where("chain_index IS NULL AND seq <= ?", w.LegacyMaxSeq).Order("seq ASC")
			if w.After != nil {
				q = q.Where("seq > ?", w.After.Seq)
			}
		} else {
			q = q.Where("chain_index <= ?", w.FenceIndex).Order("chain_index ASC")
			if w.After != nil {
				q = q.Where("chain_index > ?", *w.After.ChainIndex)
			}
		}
	} else {
		q = q.Where("(chain_index <= ? OR (chain_index IS NULL AND seq <= ?))", w.FenceIndex, w.LegacyMaxSeq).
			Order("chain_index DESC NULLS LAST").Order("seq DESC")
		if w.After != nil {
			if w.After.ChainIndex != nil {
				q = q.Where("(chain_index < ? OR chain_index IS NULL)", *w.After.ChainIndex)
			} else {
				q = q.Where("chain_index IS NULL AND seq < ?", w.After.Seq)
			}
		}
	}
	if f.ResourceType != "" {
		q = q.Where("resource_type = ?", f.ResourceType)
	}
	if f.ResourceID != "" {
		q = q.Where("resource_id = ?", f.ResourceID)
	}
	if f.ActorID != "" {
		q = q.Where("actor_user_id = ?", f.ActorID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.IP != "" {
		q = q.Where("client_ip = ?::inet", f.IP)
	}
	if f.Actor != "" {
		// Literal case-insensitive substring search; wildcard characters are data.
		term := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(f.Actor) + "%"
		q = q.Where("(actor_login ILIKE ? OR actor_display_name ILIKE ?)", term, term)
	}
	rows := []model.AuditEvidence{}
	err := translate(q.Limit(w.Limit).Offset(w.Offset).Find(&rows).Error)
	for i := range rows {
		rows[i].ClientIP = rows[i].ClientIPHost
	}
	return rows, err
}

func (r *AuditRepository) List(ctx context.Context, f model.AuditFilter) ([]model.AuditEvent, error) {
	limit, offset := pageDefaults(f.Limit, f.Offset)
	q := database.Conn(ctx, r.db).WithContext(ctx).
		Order("chain_index DESC NULLS LAST").Order("seq DESC").Limit(limit).Offset(offset)
	if f.ResourceType != "" {
		q = q.Where("resource_type = ?", f.ResourceType)
	}
	if f.ResourceID != "" {
		q = q.Where("resource_id = ?", f.ResourceID)
	}
	if f.ActorID != "" {
		q = q.Where("actor_user_id = ?", f.ActorID)
	}
	events := []model.AuditEvent{}
	if err := translate(q.Find(&events).Error); err != nil {
		return nil, err
	}
	return events, nil
}
