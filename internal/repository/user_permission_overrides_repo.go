package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type UserPermissionOverridesRepo struct{ db *gorm.DB }

func NewUserPermissionOverridesRepo(db *gorm.DB) *UserPermissionOverridesRepo {
	return &UserPermissionOverridesRepo{db: db}
}

func (r *UserPermissionOverridesRepo) FindByUserID(ctx context.Context, userID uint64) ([]model.UserPermissionOverride, error) {
	var out []model.UserPermissionOverride
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&out).Error
	return out, err
}

func (r *UserPermissionOverridesRepo) Replace(ctx context.Context, userID uint64, overrides []model.UserPermissionOverride) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserPermissionOverride{}).Error; err != nil {
			return err
		}
		if len(overrides) == 0 {
			return nil
		}
		return tx.Create(&overrides).Error
	})
}
