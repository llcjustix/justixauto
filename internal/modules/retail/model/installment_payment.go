package model

import "time"

// InstallmentPayment is one externally identified receipt allocated across one
// or more monthly invoices of a saved installment plan.
type InstallmentPayment struct {
	ID                string `gorm:"primaryKey;type:uuid"`
	CompanyID         string `gorm:"type:uuid"`
	DealID            string `gorm:"type:uuid"`
	InstallmentPlanID string `gorm:"type:uuid"`
	AmountMinor       string
	Currency          string
	PaidOn            time.Time `gorm:"type:date"`
	ExternalReference string
	AttachmentIDs     []byte `gorm:"type:jsonb"`
	Status            string
	DecisionReason    string
	SubmittedBy       string  `gorm:"type:uuid"`
	DecidedBy         *string `gorm:"type:uuid"`
	Version           int64
	CreatedAt         time.Time
	DecidedAt         *time.Time
}

func (InstallmentPayment) TableName() string { return "retail_installment_payments" }
