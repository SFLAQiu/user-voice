package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type PermissionGroupRepo struct{ db *gorm.DB }

func NewPermissionGroupRepo(db *gorm.DB) *PermissionGroupRepo {
	return &PermissionGroupRepo{db: db}
}

func (r *PermissionGroupRepo) List(ctx context.Context) ([]model.PermissionGroup, error) {
	var out []model.PermissionGroup
	err := r.db.WithContext(ctx).Order("id asc").Find(&out).Error
	return out, err
}

func (r *PermissionGroupRepo) FindByID(ctx context.Context, id uint64) (*model.PermissionGroup, error) {
	var g model.PermissionGroup
	err := r.db.WithContext(ctx).First(&g, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *PermissionGroupRepo) FindByIDs(ctx context.Context, ids []uint64) ([]model.PermissionGroup, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []model.PermissionGroup
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&out).Error
	return out, err
}

func (r *PermissionGroupRepo) FindByName(ctx context.Context, name string) (*model.PermissionGroup, error) {
	var g model.PermissionGroup
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&g).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *PermissionGroupRepo) Create(ctx context.Context, g *model.PermissionGroup) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *PermissionGroupRepo) Update(ctx context.Context, g *model.PermissionGroup) error {
	return r.db.WithContext(ctx).Model(g).Where("id = ?", g.ID).Updates(map[string]any{
		"name":        g.Name,
		"description": g.Description,
		"permissions": g.Permissions,
	}).Error
}

func (r *PermissionGroupRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", id).Delete(&model.UserPermissionGroup{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.PermissionGroup{}, id).Error
	})
}
