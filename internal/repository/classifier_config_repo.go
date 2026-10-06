package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type ClassifierConfigRepo struct{ db *gorm.DB }

func NewClassifierConfigRepo(db *gorm.DB) *ClassifierConfigRepo {
	return &ClassifierConfigRepo{db: db}
}

func (r *ClassifierConfigRepo) List(ctx context.Context) ([]model.ClassifierConfig, error) {
	var out []model.ClassifierConfig
	err := r.db.WithContext(ctx).Order("type ASC, sort_order ASC").Find(&out).Error
	return out, err
}

func (r *ClassifierConfigRepo) ListByType(ctx context.Context, t string) ([]model.ClassifierConfig, error) {
	var out []model.ClassifierConfig
	err := r.db.WithContext(ctx).
		Where("type = ? AND enabled = ?", t, 1).
		Order("sort_order ASC").Find(&out).Error
	return out, err
}

func (r *ClassifierConfigRepo) FindByID(ctx context.Context, id uint64) (*model.ClassifierConfig, error) {
	var c model.ClassifierConfig
	err := r.db.WithContext(ctx).First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ClassifierConfigRepo) FindByTypeAndName(ctx context.Context, t, name string) (*model.ClassifierConfig, error) {
	var c model.ClassifierConfig
	err := r.db.WithContext(ctx).Where("type = ? AND name = ?", t, name).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ClassifierConfigRepo) FindDefaultPrompt(ctx context.Context) (*model.ClassifierConfig, error) {
	var c model.ClassifierConfig
	err := r.db.WithContext(ctx).
		Where("type = ? AND is_default = ? AND enabled = ?", model.ClassifierConfigTypePrompt, 1, 1).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ClassifierConfigRepo) Create(ctx context.Context, c *model.ClassifierConfig) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *ClassifierConfigRepo) Update(ctx context.Context, c *model.ClassifierConfig) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *ClassifierConfigRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.ClassifierConfig{}, id).Error
}