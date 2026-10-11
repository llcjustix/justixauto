package model

import "time"

// DemoScoring is an explicitly demonstrative completion record. It is not a
// credit decision, provider response, insurance application, or coverage.
type DemoScoring struct {
	ID           string `gorm:"primaryKey;type:uuid"`
	CompanyID    string `gorm:"type:uuid"`
	DealID       string `gorm:"type:uuid"`
	DealRevision int64
	TermsDigest  string
	Mode         string
	Result       string
	CompletedBy  string `gorm:"type:uuid"`
	CompletedAt  time.Time
	CreatedAt    time.Time
}

func (DemoScoring) TableName() string { return "retail_demo_scoring_runs" }
