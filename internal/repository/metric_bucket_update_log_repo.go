package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type MetricBucketUpdateLogRepo struct {
	db *gorm.DB
}

func NewMetricBucketUpdateLogRepo(db *gorm.DB) *MetricBucketUpdateLogRepo {
	return &MetricBucketUpdateLogRepo{db: db}
}

func (r *MetricBucketUpdateLogRepo) Create(ctx context.Context, log *model.MetricBucketUpdateLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *MetricBucketUpdateLogRepo) ListRecent(ctx context.Context, limit int) ([]model.MetricBucketUpdateLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []model.MetricBucketUpdateLog
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
