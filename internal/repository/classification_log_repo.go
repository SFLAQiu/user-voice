package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type ClassificationLogRepo struct{ db *gorm.DB }

func NewClassificationLogRepo(db *gorm.DB) *ClassificationLogRepo {
	return &ClassificationLogRepo{db: db}
}

func (r *ClassificationLogRepo) Create(ctx context.Context, log *model.ClassificationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *ClassificationLogRepo) ListByFeedbackID(ctx context.Context, feedbackID uint64) ([]model.ClassificationLog, error) {
	var out []model.ClassificationLog
	err := r.db.WithContext(ctx).
		Where("feedback_id = ?", feedbackID).
		Order("id DESC").Find(&out).Error
	return out, err
}

func (r *ClassificationLogRepo) CountByFeedbackID(ctx context.Context, feedbackID uint64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ClassificationLog{}).
		Where("feedback_id = ?", feedbackID).Count(&count).Error
	return int(count), err
}