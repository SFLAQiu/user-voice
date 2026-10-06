package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type LLMProviderRepo struct{ db *gorm.DB }

func NewLLMProviderRepo(db *gorm.DB) *LLMProviderRepo { return &LLMProviderRepo{db: db} }

func (r *LLMProviderRepo) Create(ctx context.Context, p *model.LLMProvider) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *LLMProviderRepo) Update(ctx context.Context, p *model.LLMProvider) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *LLMProviderRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.LLMProvider{}, id).Error
}

func (r *LLMProviderRepo) FindByID(ctx context.Context, id uint64) (*model.LLMProvider, error) {
	var p model.LLMProvider
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *LLMProviderRepo) FindByName(ctx context.Context, name string) (*model.LLMProvider, error) {
	var p model.LLMProvider
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *LLMProviderRepo) List(ctx context.Context) ([]model.LLMProvider, error) {
	var out []model.LLMProvider
	err := r.db.WithContext(ctx).Order("priority ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *LLMProviderRepo) ListEnabled(ctx context.Context) ([]model.LLMProvider, error) {
	var out []model.LLMProvider
	err := r.db.WithContext(ctx).
		Where("status = ?", model.LLMProviderEnabled).
		Order("priority ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *LLMProviderRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.LLMProvider{}).Count(&n).Error
	return n, err
}