package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

// MetricDimensionConfigRepo 操作维度配置表。
type MetricDimensionConfigRepo struct {
	db *gorm.DB
}

func NewMetricDimensionConfigRepo(db *gorm.DB) *MetricDimensionConfigRepo {
	return &MetricDimensionConfigRepo{db: db}
}

// List 返回所有启用的维度配置，按 sort_order 排序。
func (r *MetricDimensionConfigRepo) List(ctx context.Context) ([]model.MetricDimensionConfig, error) {
	var items []model.MetricDimensionConfig
	err := r.db.WithContext(ctx).
		Where("status = 1").
		Order("sort_order ASC").
		Find(&items).Error
	return items, err
}

// ListAll 返回所有维度配置（含禁用），按 sort_order 排序。
func (r *MetricDimensionConfigRepo) ListAll(ctx context.Context) ([]model.MetricDimensionConfig, error) {
	var items []model.MetricDimensionConfig
	err := r.db.WithContext(ctx).
		Order("sort_order ASC").
		Find(&items).Error
	return items, err
}

// GetByField 按字段名查找维度配置。
func (r *MetricDimensionConfigRepo) GetByField(ctx context.Context, field string) (*model.MetricDimensionConfig, error) {
	var item model.MetricDimensionConfig
	err := r.db.WithContext(ctx).
		Where("field = ?", field).
		First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Create 新增维度配置。
func (r *MetricDimensionConfigRepo) Create(ctx context.Context, item *model.MetricDimensionConfig) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// Delete 删除维度配置（按 field）。
func (r *MetricDimensionConfigRepo) Delete(ctx context.Context, field string) error {
	return r.db.WithContext(ctx).
		Where("field = ?", field).
		Delete(&model.MetricDimensionConfig{}).Error
}

// Update 更新维度配置。
func (r *MetricDimensionConfigRepo) Update(ctx context.Context, item *model.MetricDimensionConfig) error {
	return r.db.WithContext(ctx).Save(item).Error
}