package model

import "time"

// InstallmentPlan is the immutable copy of the signed agreement behind an
// own-installment sale. Individual monthly obligations remain invoices.
type InstallmentPlan struct {
	ID                   string `gorm:"primaryKey;type:uuid"`
	DealID               string `gorm:"type:uuid"`
	CompanyID            string `gorm:"type:uuid"`
	ContractReference    string
	ContractSignedOn     time.Time `gorm:"type:date"`
	ContractFileIDs      []byte    `gorm:"type:jsonb"`
	ContractTotalMinor   string
	Currency             string
	DownPaymentInvoiceID string `gorm:"type:uuid"`
	DownPaymentMinor     string
	CreatedBy            string `gorm:"type:uuid"`
	CreatedAt            time.Time
}

func (InstallmentPlan) TableName() string { return "retail_installment_plans" }
