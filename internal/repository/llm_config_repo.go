package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type LLMConfigRepo struct{ db *gorm.DB }

func NewLLMConfigRepo(db *gorm.DB) *LLMConfigRepo { return &LLMConfigRepo{db: db} }

func (r *LLMConfigRepo) List(ctx context.Context) ([]model.LLMConfig, error) {
	var out []model.LLMConfig
	err := r.db.WithContext(ctx).Order("type ASC, sort_order ASC").Find(&out).Error
	return out, err
}

func (r *LLMConfigRepo) ListByType(ctx context.Context, t string) ([]model.LLMConfig, error) {
	var out []model.LLMConfig
	err := r.db.WithContext(ctx).
		Where("type = ? AND enabled = ?", t, 1).
		Order("sort_order ASC").Find(&out).Error
	return out, err
}

func (r *LLMConfigRepo) FindByID(ctx context.Context, id uint64) (*model.LLMConfig, error) {
	var c model.LLMConfig
	err := r.db.WithContext(ctx).First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *LLMConfigRepo) FindDefaultByType(ctx context.Context, t string) (*model.LLMConfig, error) {
	var c model.LLMConfig
	err := r.db.WithContext(ctx).
		Where("type = ? AND is_default = ? AND enabled = ?", t, 1, 1).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *LLMConfigRepo) FindByTypeAndName(ctx context.Context, t, name string) (*model.LLMConfig, error) {
	var c model.LLMConfig
	err := r.db.WithContext(ctx).Where("type = ? AND name = ?", t, name).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *LLMConfigRepo) Create(ctx context.Context, c *model.LLMConfig) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *LLMConfigRepo) Update(ctx context.Context, c *model.LLMConfig) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *LLMConfigRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.LLMConfig{}, id).Error
}