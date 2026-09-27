package service

import (
	"testing"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
)

var testCatalog = []model.Permission{
	{Key: "retail.read", Scope: model.RoleScopeCompany, Assignable: true},
	{Key: "inventory.read", Scope: model.RoleScopeCompany, Assignable: true},
	{Key: "platform.audit.read", Scope: model.RoleScopePlatform, Assignable: true},
	{Key: "platform.users.manage", Scope: model.RoleScopePlatform, Assignable: false},
}

func TestPermissionsValidatedAgainstCatalog(t *testing.T) {
	var v apperr.Validation
	got := permissions(&v, testCatalog, model.RoleScopeCompany, []string{"retail.read", "inventory.read", "retail.read"})
	if v.Err() != nil || len(got) != 2 || got[0] != "inventory.read" {
		t.Fatalf("company role: got %v, err %v", got, v.Err())
	}
	for _, c := range []struct {
		scope, key, why string
	}{
		{model.RoleScopeCompany, "platform.audit.read", "platform permission in a company role"},
		{model.RoleScopePlatform, "retail.read", "company permission in a platform role"},
		{model.RoleScopePlatform, "platform.users.manage", "non-assignable permission"},
		{model.RoleScopeCompany, "unknown.key", "permission missing from the catalog"},
	} {
		v = apperr.Validation{}
		permissions(&v, testCatalog, c.scope, []string{c.key})
		if v.Err() == nil {
			t.Errorf("%s must be rejected", c.why)
		}
	}
}

func TestValidPermission(t *testing.T) {
	var v apperr.Validation
	p := validPermission(&v, PermissionInput{Key: "retail.reports.read", Scope: "company", Name: "Продажи: отчёты", Assignable: true})
	if v.Err() != nil || p.Key != "retail.reports.read" {
		t.Fatalf("valid permission rejected: %v", v.Err())
	}
	for _, in := range []PermissionInput{
		{Key: "Retail.Read", Scope: "company", Name: "x"},
		{Key: "retail", Scope: "company", Name: "x"},
		{Key: "retail.read", Scope: "global", Name: "x"},
		{Key: "retail.read", Scope: "company", Name: ""},
	} {
		v = apperr.Validation{}
		validPermission(&v, in)
		if v.Err() == nil {
			t.Errorf("%+v must be rejected", in)
		}
	}
	if permissionAuditID("retail.read") != permissionAuditID("retail.read") {
		t.Error("audit IDs must be stable per key")
	}
}

func TestDeletedPermissionsGrantNothing(t *testing.T) {
	admin := model.RoleCompanyAdmin
	roles := []model.Role{
		{Name: "Кассир", Permissions: []string{"retail.read", "retail.deleted"}},
		{SystemKey: &admin},
	}
	// The live catalog no longer lists retail.deleted (soft-deleted).
	resolveGrants(roles, testCatalog)
	if got := roles[0].Permissions; len(got) != 1 || got[0] != "retail.read" {
		t.Errorf("custom role grants = %v, want only the live retail.read", got)
	}
	if got := roles[1].Permissions; len(got) != 2 {
		t.Errorf("company admin grants = %v, want the 2 live company permissions", got)
	}
}
