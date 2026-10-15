package model

import "time"

// ReservationEvidence is an immutable assertion about an external advance.
// Only its decision fields and version change after submission.
type ReservationEvidence struct {
	ID, ReservationID     string `gorm:"type:uuid"`
	TermsDigest           string
	TermsRevision         int64
	AmountMinor, Currency string
	PaidOn                time.Time `gorm:"type:date"`
	ExternalReference     string
	AttachmentIDs         []byte `gorm:"type:jsonb"`
	Status                string // submitted | accepted | rejected
	DecisionReason        string
	SubmittedBy           string  `gorm:"type:uuid"`
	DecidedBy             *string `gorm:"type:uuid"`
	Version               int64
	CreatedAt             time.Time
	DecidedAt             *time.Time
}

func (ReservationEvidence) TableName() string { return "commerce_reservation_evidence" }

// ReservationRefund records cancellation intent and, separately, confirmation
// of an actual full external refund. Accepted evidence remains untouched.
type ReservationRefund struct {
	ReservationID                            string `gorm:"primaryKey;type:uuid"`
	Status                                   string // requested | confirmed
	RequestedBy                              string `gorm:"type:uuid"`
	RequestedAt                              time.Time
	RequestReason                            string
	AmountMinor, Currency, ExternalReference string
	AttachmentIDs                            []byte     `gorm:"type:jsonb"`
	RefundedOn                               *time.Time `gorm:"type:date"`
	ConfirmedBy                              *string    `gorm:"type:uuid"`
	ConfirmedAt                              *time.Time
}

func (ReservationRefund) TableName() string { return "commerce_reservation_refunds" }

// ReservationCredit links accepted reservation money to one source order.
// InvoiceID is attached later, once an issued invoice exists.
type ReservationCredit struct {
	ReservationID                     string `gorm:"primaryKey;type:uuid"`
	OrderID                           string `gorm:"type:uuid"`
	BuyerCompanyID, SupplierCompanyID string `gorm:"type:uuid"`
	AmountMinor, Currency             string
	EvidenceIDs                       []byte  `gorm:"type:jsonb"`
	InvoiceID                         *string `gorm:"type:uuid"`
	CreatedAt                         time.Time
	InvoiceAttachedAt                 *time.Time
}

func (ReservationCredit) TableName() string { return "commerce_reservation_credits" }
