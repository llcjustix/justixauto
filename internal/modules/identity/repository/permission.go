package repository

import (
	"context"

	"gorm.io/gorm"

	"justixauto/internal/modules/identity/model"
)

// PermissionRepository reads the permission catalog kept in PostgreSQL.
type PermissionRepository struct{ db *gorm.DB }

// List returns the whole catalog ordered by name.
func (r *PermissionRepository) List(ctx context.Context) ([]model.Permission, error) {
	perms := []model.Permission{}
	err := r.db.WithContext(ctx).Order("name, key").Find(&perms).Error
	return perms, translate(err)
}
