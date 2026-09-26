package model

import "time"

// Role scopes (user decisions 2026-09-26), derived from a role's permissions:
// platform roles are for JustixAuto staff; company roles are prepared in Admin
// and assigned by company admins to their employees.
const (
	RoleScopePlatform = "platform"
	RoleScopeCompany  = "company"
)

// Role is a named group of permissions prepared by the platform admin.
type Role struct {
	ID          string  `gorm:"primaryKey;type:uuid"`
	SystemKey   *string // set for built-in roles; their permissions come from code
	Name        string
	Scope       string   // RoleScopePlatform or RoleScopeCompany
	CompanyID   *string  // set for a company's own (private) role
	Permissions []string `gorm:"-"`
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Role) TableName() string { return "identity.roles" }

func (r Role) System() bool { return r.SystemKey != nil }

// AssignableIn reports whether the admins of companyID may give r to their
// employees: the built-in company administrator or the company's own roles.
func (r Role) AssignableIn(companyID string) bool {
	if r.SystemKey != nil {
		return *r.SystemKey == RoleCompanyAdmin
	}
	return r.CompanyID != nil && *r.CompanyID == companyID
}

// Permission is one entry of the permission catalog kept in PostgreSQL (user
// decision 2026-09-26): what can be put into roles, its scope and its name.
type Permission struct {
	Key        string `gorm:"primaryKey"`
	Scope      string // RoleScopePlatform or RoleScopeCompany
	Name       string
	Assignable bool
}

func (Permission) TableName() string { return "identity.permissions" }
