package model

import "time"

// SupplyLot is distributor-declared, unidentifiable future supply. It is a
// capacity record, never a substitute for an inventory vehicle or VIN.
type SupplyLot struct {
	ID                           string `gorm:"primaryKey;type:uuid"`
	SupplierCompanyID            string `gorm:"type:uuid"`
	ModelID                      string `gorm:"type:uuid"`
	ModelSpecificationVersion    string
	ExteriorColor, InteriorColor string
	TotalQuantity                int
	ExpectedArrival              *time.Time
	Version                      int64
	CreatedAt, UpdatedAt         time.Time
}

func (SupplyLot) TableName() string { return "commerce_supply_lots" }

// SupplyClaim is one committed future unit. P17 creates it when a reservation
// becomes active. Keeping claims unit-granular means materialization changes
// the same durable identity rather than creating a second future commitment.
// The nullable IDs deliberately have no reservation-table foreign keys: that
// owner schema is introduced by P17a after this migration.
type SupplyClaim struct {
	ID                string  `gorm:"primaryKey;type:uuid"`
	SupplyLotID       string  `gorm:"type:uuid"`
	ReservationID     *string `gorm:"type:uuid"`
	ReservationLineID *string `gorm:"type:uuid"`
	OrderID           *string `gorm:"type:uuid"`
	OrderLineID       *string `gorm:"type:uuid"`
	State             string
	// Kind, reference and unit ordinal identify one materialization permanently.
	// Quantity receipts can consume several claims under the same reference;
	// VIN allocations use ordinal 1.
	MaterializationKind        string
	MaterializationRef         *string `gorm:"type:uuid"`
	MaterializationUnitOrdinal *int32
	MaterializedAt              *time.Time
	CreatedAt                   time.Time
}

func (SupplyClaim) TableName() string { return "commerce_supply_claims" }

// SupplyClaimAllocation records each physical association of a consumed future
// unit. Exactly one row per claim can be current; older rows survive a VIN
// replacement or reservation-to-order holder transfer.
type SupplyClaimAllocation struct {
	ID string `gorm:"primaryKey;type:uuid"`
	ClaimID, ReservationLineID, VehicleID, HolderID string `gorm:"type:uuid"`
	HolderType string
	OrderLineID *string `gorm:"type:uuid"`
	Current bool
	CreatedAt time.Time
}

func (SupplyClaimAllocation) TableName() string { return "commerce_supply_claim_allocations" }

// SupplyCapacity is the projected ledger for a lot. Available plus Held plus
// Materialized always equals TotalQuantity; released claims are absent from
// the two committed quantities.
type SupplyCapacity struct {
	Available, Held, Materialized int
}
