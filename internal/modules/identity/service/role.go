package service

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

// NewRole builds the role service.
func NewRole(d Deps) *Role { return &Role{d} }

// RoleInput is a role: a name and a set of permissions (user decisions
// 2026-09-26). Admin prepares platform roles from platform permissions; each
// company creates its own private roles from company permissions.
type RoleInput struct {
	Name           string   `json:"name"`
	PermissionKeys []string `json:"permissionKeys"`
}

// Role manages roles and their permission grants. Permissions come from the
// catalog kept in PostgreSQL (identity.permissions).
type Role struct{ Deps }

// all returns every role with the grants of built-in roles filled in.
func (s *Role) all(ctx context.Context, st Store) ([]model.Role, error) {
	roles, err := st.Roles().List(ctx)
	if err != nil {
		return nil, err
	}
	return roles, s.withPermissions(ctx, st, roles)
}

// List returns the roles managed in Admin: built-in and platform roles, not
// any company's own roles.
func (s *Role) List(ctx context.Context) ([]model.Role, error) {
	roles, err := s.all(ctx, s.store)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(roles, func(r model.Role) bool { return r.CompanyID != nil }), nil
}

// ForCompany returns the roles a company's admins may give their employees:
// the built-in company administrator and the company's own roles.
func (s *Role) ForCompany(ctx context.Context, st Store, companyID string) ([]model.Role, error) {
	roles, err := s.all(ctx, st)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(roles, func(r model.Role) bool { return !r.AssignableIn(companyID) }), nil
}

// Catalog returns the permissions of one scope from PostgreSQL.
func (s *Role) Catalog(ctx context.Context, scope string) ([]model.Permission, error) {
	catalog, err := s.store.Permissions().List(ctx)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(catalog, func(p model.Permission) bool { return p.Scope != scope }), nil
}

// permissions validates keys against the catalog: each must exist, be
// assignable and belong to scope. Duplicates are dropped; the result is sorted.
func permissions(v *apperr.Validation, catalog []model.Permission, scope string, keys []string) []string {
	out := []string{}
	for _, k := range keys {
		i := slices.IndexFunc(catalog, func(p model.Permission) bool { return p.Key == k })
		switch {
		case i < 0 || !catalog[i].Assignable:
			v.Add("permissionKeys", "unknown or non-assignable permission "+k)
		case catalog[i].Scope != scope:
			v.Add("permissionKeys", "permission "+k+" cannot be used in a "+scope+" role")
		case !slices.Contains(out, k):
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

func duplicateRole(err error) error {
	if err != nil && isConflict(err) {
		return apperr.New(apperr.ErrConflict, "role_duplicate", "a role with this name already exists")
	}
	return err
}

// create stores a new role of scope (and companyID for a company role).
func (s *Role) create(ctx context.Context, actor *auth.Principal, scope string, companyID *string, in RoleInput) (*model.Role, error) {
	var v apperr.Validation
	now := s.clock()
	r := &model.Role{
		ID: uuid.NewString(), Name: text(&v, "name", in.Name, 1, 100), Scope: scope, CompanyID: companyID,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	err := s.store.InTx(ctx, func(st Store) error {
		catalog, err := st.Permissions().List(ctx)
		if err != nil {
			return err
		}
		r.Permissions = permissions(&v, catalog, scope, in.PermissionKeys)
		if err := v.Err(); err != nil {
			return err
		}
		if err := st.Roles().Create(ctx, r); err != nil {
			return duplicateRole(err)
		}
		return s.audit(ctx, st, actor, "role.created", "role", r.ID, companyID, "", map[string]any{"name": r.Name, "permissions": r.Permissions})
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// update replaces a role's name and permissions after owns accepts it.
// Built-in roles are fixed.
func (s *Role) update(ctx context.Context, actor *auth.Principal, id string, expected int64, in RoleInput, owns func(*model.Role) bool) (*model.Role, error) {
	if err := validID(id); err != nil {
		return nil, err
	}
	var result *model.Role
	err := s.store.InTx(ctx, func(st Store) error {
		r, err := st.Roles().Get(ctx, id)
		if err != nil {
			return err
		}
		if !owns(r) {
			return apperr.ErrNotFound
		}
		if r.System() {
			return apperr.New(apperr.ErrConflict, "system_role", "system roles cannot be changed")
		}
		if r.Version != expected {
			return apperr.ErrStale
		}
		catalog, err := st.Permissions().List(ctx)
		if err != nil {
			return err
		}
		var v apperr.Validation
		before := r.Permissions
		r.Name = text(&v, "name", in.Name, 1, 100)
		r.Permissions = permissions(&v, catalog, r.Scope, in.PermissionKeys)
		if err := v.Err(); err != nil {
			return err
		}
		r.UpdatedAt = s.clock()
		if err := st.Roles().Update(ctx, r, expected); err != nil {
			return duplicateRole(err)
		}
		result = r
		return s.audit(ctx, st, actor, "role.updated", "role", r.ID, r.CompanyID, "", map[string]any{"permissionsBefore": before, "permissionsAfter": r.Permissions})
	})
	return result, err
}

// Create prepares a platform role (platform permissions) in Admin.
func (s *Role) Create(ctx context.Context, actor *auth.Principal, in RoleInput) (*model.Role, error) {
	return s.create(ctx, actor, model.RoleScopePlatform, nil, in)
}

// Update edits a platform role in Admin; company roles are not visible here.
func (s *Role) Update(ctx context.Context, actor *auth.Principal, id string, expected int64, in RoleInput) (*model.Role, error) {
	return s.update(ctx, actor, id, expected, in, func(r *model.Role) bool { return r.CompanyID == nil })
}
