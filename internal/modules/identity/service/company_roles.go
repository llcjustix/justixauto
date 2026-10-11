package service

import (
	"context"

	"github.com/google/uuid"

	"justixauto/internal/modules/identity/model"
)

const accountantRoleName = "Бухгалтер"

var accountantPermissionKeys = []string{
	"commerce.invoices.manage",
	"commerce.payments.accept",
	"commerce.read",
	"documents.read",
	"documents.sensitive.download",
	"documents.upload",
	"retail.invoices.manage",
	"retail.payments.accept",
	"retail.read",
}

// createAccountantRole gives each new seller an editable private accountant
// preset. Provider companies do not have the seller accounting work surface.
func (s *Company) createAccountantRole(ctx context.Context, st Store, c *model.Company) error {
	if c.Kind != model.KindSeller {
		return nil
	}
	companyID := c.ID
	r := &model.Role{
		ID: uuid.NewString(), Name: accountantRoleName, Scope: model.RoleScopeCompany, CompanyID: &companyID,
		Permissions: accountantPermissionKeys, Version: 1, CreatedAt: c.CreatedAt, UpdatedAt: c.CreatedAt,
	}
	return st.Roles().Create(ctx, r)
}
