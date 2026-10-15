package model

import "time"

// ConsignmentAgreement identifies the mutually accepted continuation. The
// order's terms remain the sole settlement and handover terms.
type ConsignmentAgreement struct {
	ID, OrderID, ReservationID, SupplierCompanyID, DealerCompanyID string `gorm:"type:uuid"`
	BookingDigest, ContinuationDigest                              string
	ContinuationRevision                                           int64
	Version                                                        int64
	CreatedAt                                                      time.Time
}

func (ConsignmentAgreement) TableName() string { return "commerce_consignment_agreements" }

type ConsignmentUnitStatus string

const (
	ConsignmentPrepared ConsignmentUnitStatus = "prepared"
	ConsignmentHeld     ConsignmentUnitStatus = "held"
	ConsignmentSold     ConsignmentUnitStatus = "sold"
	ConsignmentReturned ConsignmentUnitStatus = "returned"
)

// A unit is the serialization point for sale and return decisions on one VIN.
// Current associations can clear on cancellation/resolution; events and the
// owning retail/return records keep their immutable history.
type ConsignmentUnit struct {
	ID, AgreementID, OrderID, OrderLineID, VehicleID string `gorm:"type:uuid"`
	VIN                                              string
	SupplierCompanyID, DealerCompanyID               string `gorm:"type:uuid"`
	Status                                           ConsignmentUnitStatus
	HandoverFactID, HandoverOperationRef             *string `gorm:"type:uuid"`
	HandedOverAt                                     *time.Time
	HandedOverBy                                     *string `gorm:"type:uuid"`
	CurrentRetailDealID, PendingReturnID             *string `gorm:"type:uuid"`
	RetailDeliveryRef                                string
	SoldAt                                           *time.Time
	SoldBy                                           *string `gorm:"type:uuid"`
	ReturnedAt                                       *time.Time
	ReturnFactID                                     *string `gorm:"type:uuid"`
	Version                                          int64
	CreatedAt, UpdatedAt                             time.Time
}

func (ConsignmentUnit) TableName() string { return "commerce_consignment_units" }
