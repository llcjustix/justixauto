package model

import "time"

type ConsignmentReturnStatus string

const (
	ConsignmentReturnPending   ConsignmentReturnStatus = "pending"
	ConsignmentReturnConfirmed ConsignmentReturnStatus = "confirmed"
	ConsignmentReturnRejected  ConsignmentReturnStatus = "rejected"
)

// ConsignmentReturn records one supplier request and the dealer's decision.
// The accepted order remains the financial obligation throughout this history.
type ConsignmentReturn struct {
	ID, UnitID, AgreementID, OrderID, VehicleID                string `gorm:"type:uuid"`
	SupplierCompanyID, DealerCompanyID, DestinationWarehouseID string `gorm:"type:uuid"`
	Reason                                                     string
	Status                                                     ConsignmentReturnStatus
	RequestedBy                                                string `gorm:"type:uuid"`
	RequestedAt                                                time.Time
	DecidedBy                                                  *string `gorm:"type:uuid"`
	DecidedAt                                                  *time.Time
	DecisionReason                                             string
	MovedAt                                                    *time.Time
	MovementFactID, MovementOperationRef                       *string `gorm:"type:uuid"`
	Version                                                    int64
	UpdatedAt                                                  time.Time
}

func (ConsignmentReturn) TableName() string { return "commerce_consignment_returns" }
