package model

import "time"

type Customer struct {
	ID                  string `gorm:"primaryKey;type:uuid"`
	CompanyID           string `gorm:"type:uuid"`
	DisplayName         string
	Phone               string
	PassportSeries      string
	PassportNumber      string
	PassportIssuer      string
	PassportIssuedOn    *time.Time `gorm:"type:date"`
	PassportExpiresOn   *time.Time `gorm:"type:date"`
	BirthDate           *time.Time `gorm:"type:date"`
	PersonalID          string
	RegisteredAddress   string
	PassportCopyFileIDs []byte `gorm:"type:jsonb"`
	Version             int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (Customer) TableName() string { return "retail_customers" }
