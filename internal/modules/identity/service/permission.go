package service

import (
	"context"
	"regexp"

	"github.com/google/uuid"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

// PermissionInput adds a permission to the catalog in PostgreSQL (user
// decision 2026-09-27: managed in Admin). The key and scope are fixed once
// created; code decides what a permission unlocks.
type PermissionInput struct {
	Key        string `json:"key"`
	Scope      string `json:"scope"`
	Name       string `json:"name"`
	Assignable bool   `json:"assignable"`
}

// UpdatePermissionInput changes a permission's name and whether roles may hold it.
type UpdatePermissionInput struct {
	Name       string `json:"name"`
	Assignable bool   `json:"assignable"`
}

// permissionKey: <module>.<resource>.<action>, lowercase, at least two parts.
var permissionKey = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// permissionAuditID gives a permission a stable UUID for the audit log,
// whose resource IDs are UUIDs; the key itself goes into the details.
func permissionAuditID(key string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("justixauto:permission:"+key)).String()
}

func validPermission(v *apperr.Validation, in PermissionInput) *model.Permission {
	if len(in.Key) > 100 || !permissionKey.MatchString(in.Key) {
		v.Add("key", "use lowercase words separated by dots, e.g. retail.reports.read")
	}
	if in.Scope != model.RoleScopePlatform && in.Scope != model.RoleScopeCompany {
		v.Add("scope", "must be platform or company")
	}
	return &model.Permission{Key: in.Key, Scope: in.Scope, Name: text(v, "name", in.Name, 1, 200), Assignable: in.Assignable}
}

// AllPermissions returns the whole catalog.
func (s *Role) AllPermissions(ctx context.Context) ([]model.Permission, error) {
	return s.store.Permissions().List(ctx)
}

// CreatePermission adds a permission to the catalog.
func (s *Role) CreatePermission(ctx context.Context, actor *auth.Principal, in PermissionInput) (*model.Permission, error) {
	var v apperr.Validation
	p := validPermission(&v, in)
	if err := v.Err(); err != nil {
		return nil, err
	}
	err := s.store.InTx(ctx, func(st Store) error {
		if err := st.Permissions().Create(ctx, p); err != nil {
			if isConflict(err) {
				return apperr.New(apperr.ErrConflict, "permission_exists", "a permission with this key already exists")
			}
			return err
		}
		return s.audit(ctx, st, actor, "permission.created", "permission", permissionAuditID(p.Key), nil, "", map[string]any{"key": p.Key, "scope": p.Scope, "name": p.Name, "assignable": p.Assignable})
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UpdatePermission renames a permission or changes whether roles may hold it.
func (s *Role) UpdatePermission(ctx context.Context, actor *auth.Principal, key string, in UpdatePermissionInput) (*model.Permission, error) {
	var v apperr.Validation
	name := text(&v, "name", in.Name, 1, 200)
	if err := v.Err(); err != nil {
		return nil, err
	}
	var result *model.Permission
	err := s.store.InTx(ctx, func(st Store) error {
		catalog, err := st.Permissions().List(ctx)
		if err != nil {
			return err
		}
		var p *model.Permission
		for i := range catalog {
			if catalog[i].Key == key {
				p = &catalog[i]
			}
		}
		if p == nil {
			return apperr.ErrNotFound
		}
		before := map[string]any{"name": p.Name, "assignable": p.Assignable}
		p.Name, p.Assignable = name, in.Assignable
		if err := st.Permissions().Update(ctx, p); err != nil {
			return err
		}
		result = p
		return s.audit(ctx, st, actor, "permission.updated", "permission", permissionAuditID(p.Key), nil, "", map[string]any{"key": p.Key, "before": before, "name": p.Name, "assignable": p.Assignable})
	})
	return result, err
}
