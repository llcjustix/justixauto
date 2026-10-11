package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"justixauto/internal/modules/identity/model"
)

type accountantRoleStore struct {
	Store
	roles RoleRepository
}

func (s accountantRoleStore) Roles() RoleRepository { return s.roles }

type accountantRoleRepository struct {
	RoleRepository
	created *model.Role
	err     error
}

func (r *accountantRoleRepository) Create(_ context.Context, role *model.Role) error {
	if r.err != nil {
		return r.err
	}
	copy := *role
	copy.Permissions = slices.Clone(role.Permissions)
	r.created = &copy
	return nil
}

func TestCreateAccountantRoleCreatesPrivateSellerPreset(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	repo := &accountantRoleRepository{}
	c := &model.Company{ID: "seller-1", Kind: model.KindSeller, CreatedAt: now}
	if err := (&Company{}).createAccountantRole(context.Background(), accountantRoleStore{roles: repo}, c); err != nil {
		t.Fatalf("create preset: %v", err)
	}
	if repo.created == nil || repo.created.CompanyID == nil || *repo.created.CompanyID != c.ID {
		t.Fatalf("preset must be private to seller: %+v", repo.created)
	}
	if repo.created.Name != accountantRoleName || repo.created.Scope != model.RoleScopeCompany || repo.created.SystemKey != nil {
		t.Fatalf("preset must be an ordinary editable company role: %+v", repo.created)
	}
	if !repo.created.AssignableIn(c.ID) || repo.created.AssignableIn("other-company") {
		t.Fatalf("preset must only be assignable in its seller company: %+v", repo.created)
	}
	want := []string{"commerce.invoices.manage", "commerce.payments.accept", "commerce.read", "documents.read", "documents.sensitive.download", "documents.upload", "retail.invoices.manage", "retail.payments.accept", "retail.read"}
	if !slices.Equal(repo.created.Permissions, want) {
		t.Fatalf("unexpected preset permissions: got %v want %v", repo.created.Permissions, want)
	}
}

func TestCreateAccountantRoleSkipsProvidersAndPropagatesWriteFailure(t *testing.T) {
	repo := &accountantRoleRepository{}
	provider := &model.Company{ID: "bank-1", Kind: model.KindBank}
	if err := (&Company{}).createAccountantRole(context.Background(), accountantRoleStore{roles: repo}, provider); err != nil {
		t.Fatalf("provider must not get seller accountant preset: %v", err)
	}
	if repo.created != nil {
		t.Fatalf("provider unexpectedly got a preset: %+v", repo.created)
	}
	want := errors.New("role write failed")
	repo.err = want
	if err := (&Company{}).createAccountantRole(context.Background(), accountantRoleStore{roles: repo}, &model.Company{ID: "seller-2", Kind: model.KindSeller}); !errors.Is(err, want) {
		t.Fatalf("role write error = %v, want %v", err, want)
	}
}
