package repository

import (
	"context"

	"gorm.io/gorm"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
)

// PermissionRepository reads the permission catalog kept in PostgreSQL.
type PermissionRepository struct{ db *gorm.DB }

// List returns the live catalog (soft-deleted permissions excluded) by name.
func (r *PermissionRepository) List(ctx context.Context) ([]model.Permission, error) {
	perms := []model.Permission{}
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Order("name, key").Find(&perms).Error
	return perms, translate(err)
}

// GetAny returns a permission by key, deleted or not.
func (r *PermissionRepository) GetAny(ctx context.Context, key string) (*model.Permission, error) {
	var p model.Permission
	if err := r.db.WithContext(ctx).Where("key = ?", key).Take(&p).Error; err != nil {
		return nil, translate(err)
	}
	return &p, nil
}

// Create adds a permission; a duplicate key is a conflict.
func (r *PermissionRepository) Create(ctx context.Context, p *model.Permission) error {
	return translate(r.db.WithContext(ctx).Create(p).Error)
}

// Update saves a permission's name, assignability and deletion mark; the key
// and scope never change (roles and code refer to them).
func (r *PermissionRepository) Update(ctx context.Context, p *model.Permission) error {
	res := r.db.WithContext(ctx).Model(&model.Permission{}).Where("key = ?", p.Key).
		Updates(map[string]any{"name": p.Name, "assignable": p.Assignable, "deleted_at": p.DeletedAt})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
