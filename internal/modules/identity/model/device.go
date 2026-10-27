package model

import "time"

// KnownDevice records account-specific cookie recognition, never authentication.
type KnownDevice struct {
	UserID      string `gorm:"primaryKey;type:uuid"`
	DeviceHash  string `gorm:"primaryKey"`
	FirstSeenAt time.Time
}

func (KnownDevice) TableName() string { return "identity_known_devices" }
