package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/feedback/internal/model"
)

type FeedbackRepo struct{ db *gorm.DB }

func NewFeedbackRepo(db *gorm.DB) *FeedbackRepo { return &FeedbackRepo{db: db} }

// UpsertBatch inserts or updates by (source_id, original_id). Returns count of newly inserted rows.
func (r *FeedbackRepo) UpsertBatch(ctx context.Context, items []*model.Feedback) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	// Use ON DUPLICATE KEY UPDATE to update mutable fields; new inserts get auto id.
	tx := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source_id"}, {Name: "original_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"content", "images", "videos", "phone_model", "app_version", "channel_id",
			"qq", "file_url", "raw_json", "platform", "platform_id",
			"user_name", "user_mode", "app_name", "updated_at",
		}),
	}).Create(items)
	return tx.RowsAffected, tx.Error
}

func (r *FeedbackRepo) FindByID(ctx context.Context, id uint64) (*model.Feedback, error) {
	var f model.Feedback
	err := r.db.WithContext(ctx).First(&f, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ExistsOriginal reports whether (source_id, original_id) already exists.
func (r *FeedbackRepo) ExistsOriginal(ctx context.Context, sourceID uint64, originalID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Feedback{}).
		Where("source_id = ? AND original_id = ?", sourceID, originalID).
		Count(&n).Error
	return n > 0, err
}

// ListQuery is the filter set for feedback list API.
type ListQuery struct {
	StartTime   *time.Time
	EndTime     *time.Time
	AppID       *int
	Platforms   []string
	PlatformIDs []int
	Categories       []string
	BusinessModules  []string
	Sentiment        string
	AppVersion  string
	UserModes   []int
	UserID      string
	Keyword     string
	OrderBy     string
	OrderDir    string
	Page        int
	PageSize    int
}

// Whitelisted operators not used here; we hardcode safe predicates.

func (r *FeedbackRepo) List(ctx context.Context, q ListQuery) ([]model.Feedback, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.Feedback{})

	if q.StartTime != nil {
		tx = tx.Where("original_created_at >= ?", *q.StartTime)
	}
	if q.EndTime != nil {
		tx = tx.Where("original_created_at <= ?", *q.EndTime)
	}
	if q.AppID != nil {
		tx = tx.Where("app_id = ?", *q.AppID)
	}
	if len(q.Platforms) > 0 {
		tx = tx.Where("platform IN ?", q.Platforms)
	}
	if len(q.PlatformIDs) > 0 {
		tx = tx.Where("platform_id IN ?", q.PlatformIDs)
	}
	if len(q.Categories) > 0 {
		tx = tx.Where("category IN ?", q.Categories)
	}
	if len(q.BusinessModules) > 0 {
		tx = tx.Where("business_module IN ?", q.BusinessModules)
	}
	if q.Sentiment != "" {
		tx = tx.Where("sentiment = ?", q.Sentiment)
	}
	if q.AppVersion != "" {
		tx = tx.Where("app_version = ?", q.AppVersion)
	}
	if len(q.UserModes) > 0 {
		tx = tx.Where("user_mode IN ?", q.UserModes)
	}
	if q.UserID != "" {
		tx = tx.Where("user_id = ?", q.UserID)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		// MATCH AGAINST requires the ngram full-text index; fallback to LIKE for short queries.
		if len([]rune(kw)) >= 2 {
			tx = tx.Where("MATCH(content) AGAINST(? IN BOOLEAN MODE)", "+"+kw+"*")
		} else {
			tx = tx.Where("content LIKE ?", "%"+kw+"%")
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderCol, ok := model.FeedbackSortableColumns[q.OrderBy]
	if !ok {
		orderCol = "original_created_at"
	}
	dir := "DESC"
	if strings.ToLower(q.OrderDir) == "asc" {
		dir = "ASC"
	}

	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size <= 0 || size > 100 {
		size = 20
	}

	var rows []model.Feedback
	err := tx.Order(fmt.Sprintf("%s %s", orderCol, dir)).
		Limit(size).Offset((page - 1) * size).Find(&rows).Error
	return rows, total, err
}

func (r *FeedbackRepo) UpdateCategory(ctx context.Context, id uint64, category, businessModule string, status int) error {
	updates := map[string]any{
		"category":        category,
		"business_module": businessModule,
		"category_status": status,
		"classified_at":   time.Now(),
	}
	return r.db.WithContext(ctx).Model(&model.Feedback{}).
		Where("id = ?", id).Updates(updates).Error
}

// UpdateClassification writes back LLM classification result.
func (r *FeedbackRepo) UpdateClassification(ctx context.Context, id uint64, category, businessModule, sentiment string, confidence float64, status int) error {
	updates := map[string]any{
		"category":         category,
		"business_module":  businessModule,
		"sentiment":        sentiment,
		"category_status":  status,
		"classified_at":    time.Now(),
	}
	return r.db.WithContext(ctx).Model(&model.Feedback{}).
		Where("id = ?", id).Updates(updates).Error
}

// ListPendingClassification returns up to limit feedbacks with category_status=0.
func (r *FeedbackRepo) ListPendingClassification(ctx context.Context, limit int) ([]model.Feedback, error) {
	var rows []model.Feedback
	err := r.db.WithContext(ctx).
		Where("category_status = ?", model.CategoryStatusPending).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// MarkClassificationFailed sets category_status to failed.
func (r *FeedbackRepo) MarkClassificationFailed(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&model.Feedback{}).
		Where("id = ?", id).
		Update("category_status", model.CategoryStatusFailed).Error
}

// ResetFailedClassifications resets category_status from failed back to pending
// for items that failed older than the given duration, enabling automatic retry.
func (r *FeedbackRepo) ResetFailedClassifications(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := r.db.WithContext(ctx).Model(&model.Feedback{}).
		Where("category_status = ? AND classified_at < ?", model.CategoryStatusFailed, cutoff).
		Update("category_status", model.CategoryStatusPending)
	return result.RowsAffected, result.Error
}
