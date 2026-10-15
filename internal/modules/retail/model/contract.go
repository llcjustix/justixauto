package model

import (
	"time"

	"justixauto/internal/pkg/money"
)

type ContractTemplate struct {
	ID                   string `gorm:"primaryKey;type:uuid"`
	CompanyID            string `gorm:"type:uuid"`
	Name                 string
	Body                 string
	Version              int64
	CreatedBy            string `gorm:"type:uuid"`
	CreatedAt, UpdatedAt time.Time
}

func (ContractTemplate) TableName() string { return "retail_contract_templates" }

// ContractVersion is immutable once signed. Draft rows may be revised through
// their template/body before signing; signed and amendment rows are append-only.
type ContractVersion struct {
	ID                   string  `gorm:"primaryKey;type:uuid"`
	DealID               string  `gorm:"type:uuid"`
	CompanyID            string  `gorm:"type:uuid"`
	TemplateID           *string `gorm:"type:uuid"`
	TemplateVersion      *int64
	PriorSignedVersionID *string `gorm:"type:uuid"`
	Kind                 string
	State                string
	Reference            string
	SignedOn             *time.Time `gorm:"type:date"`
	FileIDs              []byte     `gorm:"type:jsonb"`
	Content              string
	PartySnapshot        []byte `gorm:"type:jsonb"`
	TermsSnapshot        []byte `gorm:"type:jsonb"`
	CreatedBy            string `gorm:"type:uuid"`
	CreatedAt            time.Time
	Revision             int64
	DealRevision         int64 `gorm:"-" json:"dealRevision,omitempty"`
}

func (ContractVersion) TableName() string { return "retail_contract_versions" }

// ContractPartySnapshot contains the private facts captured when a draft is
// prepared. It is stored as JSON so later profile or company edits cannot
// change a reviewed or signed document.
type ContractPartySnapshot struct {
	Seller   ContractSellerSnapshot   `json:"seller"`
	Customer ContractCustomerSnapshot `json:"customer"`
	Vehicle  *VehicleSnapshot         `json:"vehicle"`
}

type ContractSellerSnapshot struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	LegalName          string `json:"legalName"`
	RegistrationNumber string `json:"registrationNumber"`
	TaxID              string `json:"taxId"`
	BankRequisites     string `json:"bankRequisites"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
}

type ContractCustomerSnapshot struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Phone             string  `json:"phone"`
	PassportSeries    string  `json:"passportSeries"`
	PassportNumber    string  `json:"passportNumber"`
	PassportIssuer    string  `json:"passportIssuer"`
	PersonalID        string  `json:"personalId"`
	RegisteredAddress string  `json:"registeredAddress"`
	PassportIssuedOn  *string `json:"passportIssuedOn"`
	PassportExpiresOn *string `json:"passportExpiresOn"`
	BirthDate         *string `json:"birthDate"`
}

// ContractTermsSnapshot records the exact sale economics observed while the
// draft was prepared. It is evidence, never an instruction to recalculate a
// payment schedule.
type ContractTermsSnapshot struct {
	Price         money.Money                  `json:"price"`
	PaymentScheme string                       `json:"paymentScheme"`
	Installment   *ContractInstallmentSnapshot `json:"installment,omitempty"`
	Legacy        bool                         `json:"legacy,omitempty"`
}

type ContractInstallmentSnapshot struct {
	Source               string                          `json:"source"`
	PolicyID             string                          `json:"policyId"`
	PolicyVersion        int                             `json:"policyVersion"`
	Price                money.Money                     `json:"price"`
	DownPayment          money.Money                     `json:"downPayment"`
	ScheduledTotal       money.Money                     `json:"scheduledTotal"`
	RegularPayment       money.Money                     `json:"regularPayment"`
	TermMonths           int                             `json:"termMonths"`
	FirstDueDate         string                          `json:"firstDueDate"`
	PlanID               string                          `json:"planId"`
	DownPaymentInvoiceID string                          `json:"downPaymentInvoiceId"`
	Obligations          []ContractInstallmentObligation `json:"obligations"`
}

type ContractInstallmentObligation struct {
	InvoiceID string      `json:"invoiceId"`
	Number    *int        `json:"number"`
	Purpose   string      `json:"purpose"`
	Amount    money.Money `json:"amount"`
	DueDate   *string     `json:"dueDate"`
}
