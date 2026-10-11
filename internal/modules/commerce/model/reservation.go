package model

import "time"

type ReservationStatus string

const (
	ReservationRequested ReservationStatus = "requested"
	ReservationOffered   ReservationStatus = "offered"
	ReservationActive    ReservationStatus = "active"
	ReservationExpired   ReservationStatus = "expired"
	ReservationCancelled ReservationStatus = "cancelled"
	ReservationConverted ReservationStatus = "converted"
)

// Reservation is a dealer request for existing VINs or units in one future
// supply lot. Requested and offered terms are immutable JSON versions; Current
// is the accepted version that activated the hold.
type Reservation struct {
	ID, BuyerCompanyID, SupplierCompanyID         string `gorm:"type:uuid"`
	Status                                        ReservationStatus
	RequestedTerms, OfferedTerms, CurrentTerms    []byte `gorm:"type:jsonb"`
	RequestedDigest, OfferedDigest, CurrentDigest string
	RequestedRevision, OfferedRevision, CurrentRevision int64
	OrderID *string `gorm:"type:uuid"`
	ContinuationProposed, ContinuationAccepted []byte `gorm:"type:jsonb"`
	ContinuationProposedDigest, ContinuationAcceptedDigest string
	ContinuationProposedRevision, ContinuationAcceptedRevision int64
	ContinuationProposedBy, ContinuationAcceptedBy *string `gorm:"type:uuid"`
	ExpiresAt                                     *time.Time
	AdvanceDueAt                                  *time.Time
	ExtendedBy                                    *string `gorm:"type:uuid"`
	ExtendedAt                                    *time.Time
	ReleasedAt                                    *time.Time
	ReleaseReason                                 string
	Version                                       int64
	CreatedBy                                     string `gorm:"type:uuid"`
	CreatedAt, UpdatedAt                          time.Time
}

func (Reservation) TableName() string { return "commerce_reservations" }

type ReservationDueCandidate struct {
	ID string
	Deadline time.Time
}

// ReservationLine identifies exactly one physical VIN or a homogeneous future
// quantity. It is retained as the source identity for future supply claims.
type ReservationLine struct {
	ID, ReservationID                                                string  `gorm:"type:uuid"`
	VehicleID, SupplyLotID                                           *string `gorm:"type:uuid"`
	ModelID, ModelSpecificationVersion, ExteriorColor, InteriorColor string
	Quantity                                                         int
}

func (ReservationLine) TableName() string { return "commerce_reservation_lines" }
