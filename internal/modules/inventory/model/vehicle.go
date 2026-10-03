package model

import "time"

// VehicleUnit is one physical vehicle. The VIN never changes.
type VehicleUnit struct {
	ID                 string `gorm:"primaryKey;type:uuid"`
	VIN                string `gorm:"column:vin"`
	ModelID            string `gorm:"type:uuid"`
	SpecVersion        int
	ExteriorColor      *string
	InteriorColor      *string
	OwnerCompanyID     *string `gorm:"type:uuid"`
	CustodianCompanyID *string `gorm:"type:uuid"`
	Version            int64
	CreatedAt          time.Time
}

func (VehicleUnit) TableName() string { return "inventory_vehicle_units" }

// Placement says in which warehouse a vehicle is; none means outside storage.
type Placement struct {
	VehicleID      string  `gorm:"primaryKey;type:uuid"`
	WarehouseID    string  `gorm:"type:uuid"`
	ReceiptBatchID *string `gorm:"type:uuid"`
	PlacedAt       time.Time
}

func (Placement) TableName() string { return "inventory_placements" }

// VehicleFilter narrows a company's vehicle unit listing.
type VehicleFilter struct {
	CompanyID   string
	ModelID     string
	Placement   string // "warehouse", "outside" or "any"
	WarehouseID string
	Search      string
	// Eligible restricts the listing to stock the active company owns and can
	// currently reserve. Generic registry listings retain their wider visibility.
	Eligible bool
	Limit    int
	Offset   int
}

// VehicleRow is a vehicle with its current placement (if any).
type VehicleRow struct {
	VehicleUnit
	WarehouseID    *string
	ReceiptBatchID *string
	PlacedAt       *time.Time
	Reserved       bool // held by an order or a retail sale
}
