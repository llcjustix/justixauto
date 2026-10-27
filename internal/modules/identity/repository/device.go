package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/database"
)

type KnownDeviceRepository struct{ db *gorm.DB }

// Recognize atomically inserts the first sighting. A conflicting committed row
// means this account has seen this digest; another account's row is unrelated.
func (r *KnownDeviceRepository) Recognize(ctx context.Context, userID, deviceHash string, firstSeenAt time.Time) (bool, error) {
	device := model.KnownDevice{UserID: userID, DeviceHash: deviceHash, FirstSeenAt: firstSeenAt}
	res := database.Conn(ctx, r.db).Clauses(clause.OnConflict{DoNothing: true}).Create(&device)
	if res.Error != nil {
		return false, translate(res.Error)
	}
	return res.RowsAffected == 0, nil
}
