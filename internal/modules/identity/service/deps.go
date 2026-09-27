// Package service holds identity's business rules. It imports model and
// internal/pkg only; it must never import echo, gorm, repository or handler.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/validate"
)

// Deps is shared by all identity services.
type Deps struct {
	store Store
	now   func() time.Time
}

// NewDeps builds the shared dependencies every identity service embeds.
func NewDeps(store Store, now func() time.Time) Deps { return Deps{store: store, now: now} }

func (d Deps) clock() time.Time { return d.now().UTC() }

// audit appends an audit event inside the caller's transaction.
func (d Deps) audit(ctx context.Context, st Store, actor *auth.Principal, action, resourceType, resourceID string, companyID *string, reason string, details map[string]any) error {
	raw, err := json.Marshal(details)
	if err != nil {
		return err
	}
	e := &model.AuditEvent{
		ID: uuid.NewString(), OccurredAt: d.clock(), Action: action,
		ResourceType: resourceType, ResourceID: resourceID, CompanyID: companyID,
		Reason: reason, Details: raw,
	}
	if actor != nil {
		e.ActorUserID = &actor.UserID
	}
	return st.Audit().Append(ctx, e)
}

// isMember reports whether the user has an active membership in the company.
func (d Deps) isMember(ctx context.Context, st Store, userID, companyID string) (bool, error) {
	_, err := st.Memberships().Active(ctx, userID, companyID)
	if errors.Is(err, apperr.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Shared input checks (see internal/pkg/validate).
var (
	validID   = validate.IDs
	text      = validate.Text
	email     = validate.Email
	reason    = validate.Reason
	uniqueIDs = validate.UniqueIDs
)

// revision formats an optimistic-locking version for an ETag/If-Match value.
func revision(v int64) string { return strconv.FormatInt(v, 10) }

// withPermissions resolves each role's grants against the live catalog in
// PostgreSQL: the company administrator holds every company permission, the
// platform administrator its fixed set, and custom roles their stored keys
// minus soft-deleted permissions (a deleted permission grants nothing).
func (d Deps) withPermissions(ctx context.Context, st Store, roles []model.Role) error {
	if len(roles) == 0 {
		return nil
	}
	catalog, err := st.Permissions().List(ctx)
	if err != nil {
		return err
	}
	resolveGrants(roles, catalog)
	return nil
}

// resolveGrants sets each role's grants from the live catalog (see
// withPermissions).
func resolveGrants(roles []model.Role, catalog []model.Permission) {
	live := make(map[string]bool, len(catalog))
	for _, p := range catalog {
		live[p.Key] = true
	}
	company := model.CompanyKeys(catalog)
	for i := range roles {
		r := &roles[i]
		switch {
		case r.SystemKey == nil:
			r.Permissions = slices.DeleteFunc(slices.Clone(r.Permissions), func(k string) bool { return !live[k] })
		case *r.SystemKey == model.RoleCompanyAdmin:
			r.Permissions = company
		default:
			r.Permissions = model.EffectivePermissions(*r)
		}
	}
}

// rolesIn returns the roles a user acts with in companyID: platform roles
// (held by the user) plus the company roles of the user's active membership
// in that company (user decision 2026-09-27: roles are per company). With no
// company, or no active membership, only platform roles count.
func (d Deps) rolesIn(ctx context.Context, st Store, userID, companyID string) ([]model.Role, error) {
	roles, err := st.Roles().UserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	if companyID == "" {
		return roles, nil
	}
	m, err := st.Memberships().Active(ctx, userID, companyID)
	if errors.Is(err, apperr.ErrNotFound) {
		return roles, nil
	}
	if err != nil {
		return nil, err
	}
	company, err := st.Roles().MembershipRoles(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	return append(roles, company...), nil
}
