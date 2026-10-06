package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type EnumConfigRepo struct{ db *gorm.DB }

func NewEnumConfigRepo(db *gorm.DB) *EnumConfigRepo { return &EnumConfigRepo{db: db} }

func (r *EnumConfigRepo) List(ctx context.Context) ([]model.EnumConfig, error) {
	var out []model.EnumConfig
	err := r.db.WithContext(ctx).Order("field ASC").Find(&out).Error
	return out, err
}

func (r *EnumConfigRepo) FindByID(ctx context.Context, id uint64) (*model.EnumConfig, error) {
	var ec model.EnumConfig
	err := r.db.WithContext(ctx).First(&ec, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ec, nil
}

func (r *EnumConfigRepo) FindByField(ctx context.Context, field string) (*model.EnumConfig, error) {
	var ec model.EnumConfig
	err := r.db.WithContext(ctx).Where("field = ?", field).First(&ec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ec, nil
}

func (r *EnumConfigRepo) Create(ctx context.Context, ec *model.EnumConfig) error {
	return r.db.WithContext(ctx).Create(ec).Error
}

func (r *EnumConfigRepo) Update(ctx context.Context, ec *model.EnumConfig) error {
	return r.db.WithContext(ctx).Save(ec).Error
}

func (r *EnumConfigRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.EnumConfig{}, id).Error
}