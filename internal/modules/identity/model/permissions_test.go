package model

import (
	"slices"
	"testing"
)

func TestSystemRolePermissionsAreCatalogued(t *testing.T) {
	for role, perms := range systemRolePermissions {
		for _, p := range perms {
			if !slices.ContainsFunc(Catalog, func(c PermissionInfo) bool { return c.Key == p }) {
				t.Fatalf("%s grants uncatalogued permission %s", role, p)
			}
		}
	}
	// Administration powers never include company business permissions.
	if slices.Contains(systemRolePermissions[RolePlatformAdmin], PermCompanyEdit) {
		t.Fatal("platform admin must not implicitly edit companies as a member")
	}
}

func TestCompanyKeysTakesOnlyCompanyScope(t *testing.T) {
	got := CompanyKeys([]Permission{
		{Key: "retail.read", Scope: RoleScopeCompany},
		{Key: "platform.audit.read", Scope: RoleScopePlatform},
		{Key: "inventory.read", Scope: RoleScopeCompany},
	})
	if !slices.Equal(got, []string{"retail.read", "inventory.read"}) {
		t.Fatalf("CompanyKeys = %v", got)
	}
}

func TestCompanyRolesAssignableOnlyInTheirCompany(t *testing.T) {
	admin, a, b := RoleCompanyAdmin, "company-a", "company-b"
	platform := RolePlatformAdmin
	if !(Role{SystemKey: &admin}).AssignableIn(a) {
		t.Error("the company administrator is assignable in every company")
	}
	if (Role{SystemKey: &platform}).AssignableIn(a) {
		t.Error("the platform administrator is never assignable by companies")
	}
	own := Role{Scope: RoleScopeCompany, CompanyID: &a}
	if !own.AssignableIn(a) || own.AssignableIn(b) {
		t.Error("a company role is assignable only in its own company")
	}
	if (Role{Scope: RoleScopeCompany}).AssignableIn(a) {
		t.Error("a role without a company is not assignable by companies")
	}
}
