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
