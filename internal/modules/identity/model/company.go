// Package model holds identity's GORM models, enums and the permission
// catalog. It imports internal/pkg only; it must never import service,
// repository or handler.
package model

import "time"

// CompanyKind: a seller using the Realization app, or a provider (bank, MFO,
// insurance company) connected through Admin → Integrations.
type CompanyKind string

const (
	KindSeller    CompanyKind = "seller"
	KindBank      CompanyKind = "bank"
	KindMFO       CompanyKind = "mfo"
	KindInsurance CompanyKind = "insurance"
)

func (k CompanyKind) Valid() bool {
	switch k {
	case KindSeller, KindBank, KindMFO, KindInsurance:
		return true
	}
	return false
}

func (k CompanyKind) Provider() bool { return k == KindBank || k == KindMFO || k == KindInsurance }

// CompanyAccess is platform access, not legal/compliance verification.
type CompanyAccess string

const (
	AccessDraft     CompanyAccess = "draft"
	AccessActive    CompanyAccess = "active"
	AccessSuspended CompanyAccess = "suspended"
)

// SellerUserType is the platform-assigned commercial type of a seller.
// It deliberately belongs to the company, not to its members' job roles.
type SellerUserType string

const (
	SellerUserTypeDealer      SellerUserType = "dealer"
	SellerUserTypeDistributor SellerUserType = "distributor"
)

func (t SellerUserType) Valid() bool {
	return t == SellerUserTypeDealer || t == SellerUserTypeDistributor
}

// CompanyVerification is separate from operational platform access.
type CompanyVerification string

const (
	VerificationUnverified CompanyVerification = "unverified"
	VerificationSubmitted  CompanyVerification = "submitted"
	VerificationVerified   CompanyVerification = "verified"
	VerificationReturned   CompanyVerification = "returned"
)

// LegalForm is optional company profile information; it has no compliance
// meaning on its own.
type LegalForm string

const (
	LegalFormLLC            LegalForm = "llc"
	LegalFormSoleProprietor LegalForm = "sole_proprietor"
	LegalFormOther          LegalForm = "other"
)

func (f LegalForm) Valid() bool {
	return f == "" || f == LegalFormLLC || f == LegalFormSoleProprietor || f == LegalFormOther
}

// Company is a registered organization. Requisites can change; the ID never does.
type Company struct {
	ID                  string `gorm:"primaryKey;type:uuid"`
	Kind                CompanyKind
	Name                string
	LegalName           string
	Country             string // label as entered
	CountryKey          string // catalogue key when chosen from the list
	Region              string
	RegionKey           string
	RegistrationNumber  string
	LegalForm           LegalForm
	TaxID               string
	BankRequisites      string
	Email               string
	Address             string
	Phone               string
	Status              CompanyAccess
	StatusReason        string
	UserType            *SellerUserType
	Verification        CompanyVerification
	VerificationReason  string
	VerificationActorID *string
	VerificationAt      *time.Time
	Version             int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	// DeletedAt marks a soft-deleted company: hidden everywhere, memberships
	// no longer grant access, the row stays for history.
	DeletedAt *time.Time
}

func (Company) TableName() string { return "identity_companies" }

// CompanyFilter narrows List; zero values mean "any".
type CompanyFilter struct {
	Query  string // part of the name
	Kind   CompanyKind
	Access CompanyAccess
	Limit  int
	Offset int
}
