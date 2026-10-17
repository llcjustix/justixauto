package model

import "time"

// AuditEvent is an append-only record of a sensitive change.
type AuditEvent struct {
	ID               string `gorm:"primaryKey;type:uuid"`
	Seq              int64  `gorm:"->"` // database allocation, not committed chain order
	OccurredAt       time.Time
	ActorUserID      *string `gorm:"type:uuid"`
	Action           string
	ResourceType     string
	ResourceID       *string `gorm:"type:uuid"`
	CompanyID        *string `gorm:"type:uuid"`
	Reason           string
	Details          []byte `gorm:"type:jsonb"`
	ActorLogin       *string
	ActorDisplayName *string
	Outcome          *string
	ErrorCode        *string
	ClientIP         *string `gorm:"type:inet"`
	RequestID        *string `gorm:"type:uuid"`
	SessionHash      *string
	DeviceHash       *string
	Metadata         []byte `gorm:"type:jsonb"`
	BeforeValues     []byte `gorm:"type:jsonb"`
	AfterValues      []byte `gorm:"type:jsonb"`
	ChainIndex       *int64 `gorm:"->"`
	ChainVersion     *int16 `gorm:"->"`
	PrevHash         []byte `gorm:"->;type:bytea"`
	EventHash        []byte `gorm:"->;type:bytea"`
	CanonicalPayload []byte `gorm:"->;type:bytea"`
	StreamID         string `gorm:"-"` // read context only; no directory enrichment
}

func (AuditEvent) TableName() string { return "identity_audit_events" }

// AuditOrder distinguishes chained stream positions from legacy sequence
// order. Possessing a chain position alone does not verify an event's integrity.
type AuditOrder struct {
	ChainIndex *int64
	Seq        int64
}

func (e AuditEvent) Order() AuditOrder { return AuditOrder{ChainIndex: e.ChainIndex, Seq: e.Seq} }

// AuditChain is the immutable boundary plus the current committed stream head.
// Read it once to fence a paged read; Seq is never a safe commit fence.
type AuditChain struct {
	Singleton             bool
	StreamID              string `gorm:"type:uuid"`
	ChainVersion          int16
	LastIndex             int64
	LastHash              []byte
	GenesisHash           []byte
	LegacyMaxSeq          int64
	LegacyCount           int64
	CreatedAt             time.Time
	BoundaryPayload       []byte
	ReconstructedBoundary []byte `gorm:"->"`
}

func (AuditChain) TableName() string { return "identity_audit_chain" }

type AuditFilter struct {
	ResourceType string
	ResourceID   string
	ActorID      string
	Actor        string
	IP           string
	Action       string
	Cursor       string
	Limit        int
	Offset       int
}

// AuditWindow is a committed chain fence, never an allocation-sequence fence.
// Ascending pages are export segments; descending pages include chain then legacy.
type AuditWindow struct {
	StreamID     string
	FenceIndex   int64
	LegacyMaxSeq int64
	After        *AuditOrder
	Ascending    bool
	LegacyOnly   bool
	Reconstruct  bool
	Limit        int
	Offset       int
}

type AuditEvidence struct {
	AuditEvent           `gorm:"embedded"`
	ReconstructedPayload []byte  `gorm:"->"`
	ClientIPHost         *string `gorm:"->"`
}
