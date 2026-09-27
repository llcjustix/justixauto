package repository

import (
	"context"

	"gorm.io/gorm"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
)

// PermissionRepository reads the permission catalog kept in PostgreSQL.
type PermissionRepository struct{ db *gorm.DB }

// List returns the whole catalog ordered by name.
func (r *PermissionRepository) List(ctx context.Context) ([]model.Permission, error) {
	perms := []model.Permission{}
	err := r.db.WithContext(ctx).Order("name, key").Find(&perms).Error
	return perms, translate(err)
}

// Create adds a permission; a duplicate key is a conflict.
func (r *PermissionRepository) Create(ctx context.Context, p *model.Permission) error {
	return translate(r.db.WithContext(ctx).Create(p).Error)
}

// Update changes a permission's name and assignability; the key and scope
// never change (roles and code refer to them).
func (r *PermissionRepository) Update(ctx context.Context, p *model.Permission) error {
	res := r.db.WithContext(ctx).Model(&model.Permission{}).Where("key = ?", p.Key).
		Updates(map[string]any{"name": p.Name, "assignable": p.Assignable})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
