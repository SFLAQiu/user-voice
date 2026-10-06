package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

// MetricTimeBucketRepo 操作预聚合时间桶表。
type MetricTimeBucketRepo struct {
	db *gorm.DB
}

func NewMetricTimeBucketRepo(db *gorm.DB) *MetricTimeBucketRepo {
	return &MetricTimeBucketRepo{db: db}
}

// IncrementBatch 批量增量写入：INSERT ... ON DUPLICATE KEY UPDATE feedback_count = feedback_count + VALUES(feedback_count)。
// 用于同步入库时写入新反馈的桶行。
func (r *MetricTimeBucketRepo) IncrementBatch(ctx context.Context, items []model.MetricTimeBucket) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			result := tx.Exec(
				`INSERT INTO metric_time_buckets
					(bucket_time, granularity, dimension_schema_version, app_id, platform_id, app_version, category, business_module, sentiment, category_status, feedback_count, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, NOW(), NOW())
				ON DUPLICATE KEY UPDATE feedback_count = feedback_count + 1, updated_at = NOW()`,
				item.BucketTime, item.Granularity, item.DimensionSchemaVersion,
				item.AppID, item.PlatformID, item.AppVersion,
				item.Category, item.BusinessModule, item.Sentiment, item.CategoryStatus,
			)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

// DecrementRow 减少指定桶行的 feedback_count（维度变更时旧行 -1）。
func (r *MetricTimeBucketRepo) DecrementRow(ctx context.Context, bucketTime time.Time, granularity string, version int, dims BucketDims) error {
	return r.db.WithContext(ctx).Exec(
		`UPDATE metric_time_buckets SET feedback_count = feedback_count - 1
		WHERE bucket_time = ? AND granularity = ? AND dimension_schema_version = ?
		  AND app_id = ? AND platform_id = ? AND app_version = ?
		  AND category = ? AND business_module = ? AND sentiment = ? AND category_status = ?`,
		bucketTime, granularity, version,
		dims.AppID, dims.PlatformID, dims.AppVersion,
		dims.Category, dims.BusinessModule, dims.Sentiment, dims.CategoryStatus,
	).Error
}

// IncrementRow 增加指定桶行的 feedback_count（维度变更时新行 +1）。
func (r *MetricTimeBucketRepo) IncrementRow(ctx context.Context, item model.MetricTimeBucket) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO metric_time_buckets
			(bucket_time, granularity, dimension_schema_version, app_id, platform_id, app_version, category, business_module, sentiment, category_status, feedback_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, NOW(), NOW())
		ON DUPLICATE KEY UPDATE feedback_count = feedback_count + 1, updated_at = NOW()`,
		item.BucketTime, item.Granularity, item.DimensionSchemaVersion,
		item.AppID, item.PlatformID, item.AppVersion,
		item.Category, item.BusinessModule, item.Sentiment, item.CategoryStatus,
	).Error
}

// MoveFeedbackBucket 将一条 feedback 的 count 从旧维度组合移到新维度组合。
// 在同一个事务中执行：旧行 -1，新行 +1。
func (r *MetricTimeBucketRepo) MoveFeedbackBucket(ctx context.Context, bucketTime time.Time, granularity string, version int, oldDims, newDims BucketDims) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 旧行 -1
		if err := tx.Exec(
			`UPDATE metric_time_buckets SET feedback_count = feedback_count - 1
			WHERE bucket_time = ? AND granularity = ? AND dimension_schema_version = ?
			  AND app_id = ? AND platform_id = ? AND app_version = ?
			  AND category = ? AND business_module = ? AND sentiment = ? AND category_status = ?`,
			bucketTime, granularity, version,
			oldDims.AppID, oldDims.PlatformID, oldDims.AppVersion,
			oldDims.Category, oldDims.BusinessModule, oldDims.Sentiment, oldDims.CategoryStatus,
		).Error; err != nil {
			return err
		}
		// 新行 +1
		if err := tx.Exec(
			`INSERT INTO metric_time_buckets
				(bucket_time, granularity, dimension_schema_version, app_id, platform_id, app_version, category, business_module, sentiment, category_status, feedback_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, NOW(), NOW())
			ON DUPLICATE KEY UPDATE feedback_count = feedback_count + 1, updated_at = NOW()`,
			bucketTime, granularity, version,
			newDims.AppID, newDims.PlatformID, newDims.AppVersion,
			newDims.Category, newDims.BusinessModule, newDims.Sentiment, newDims.CategoryStatus,
		).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetCurrentVersion 获取当前活跃的 dimension_schema_version（最大值）。
func (r *MetricTimeBucketRepo) GetCurrentVersion(ctx context.Context) (int, error) {
	var version int
	err := r.db.WithContext(ctx).Model(&model.MetricTimeBucket{}).
		Select("COALESCE(MAX(dimension_schema_version), 1)").
		Scan(&version).Error
	return version, err
}

// CleanupZeroRows 删除 feedback_count=0 的空行（对账时清理）。
func (r *MetricTimeBucketRepo) CleanupZeroRows(ctx context.Context, version int) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM metric_time_buckets WHERE dimension_schema_version = ? AND feedback_count = 0`,
		version,
	).Error
}

// DeleteByVersion 删除指定版本的所有行（重建完成后清理旧版本）。
func (r *MetricTimeBucketRepo) DeleteByVersion(ctx context.Context, version int) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM metric_time_buckets WHERE dimension_schema_version = ?`,
		version,
	).Error
}

// BucketDims 封装维度字段，便于传递。
type BucketDims struct {
	AppID          int
	PlatformID     int
	AppVersion     string
	Category       string
	BusinessModule string
	Sentiment      string
	CategoryStatus int
}

// FromFeedback 从 Feedback 对象提取维度字段。
// platform_id NULL → 0；分类相关字段按 category_status 决定是否为空。
func BucketDimsFromFeedback(fb model.Feedback) BucketDims {
	platformID := 0
	if fb.PlatformID != nil {
		platformID = *fb.PlatformID
	}
	category := ""
	businessModule := ""
	sentiment := ""
	// 只有已分类(1)或手动(3)时，维度字段才有实际值
	if fb.CategoryStatus == model.CategoryStatusClassified || fb.CategoryStatus == model.CategoryStatusManual {
		category = fb.Category
		businessModule = fb.BusinessModule
		sentiment = fb.Sentiment
	}
	return BucketDims{
		AppID:          fb.AppID,
		PlatformID:     platformID,
		AppVersion:     fb.AppVersion,
		Category:       category,
		BusinessModule: businessModule,
		Sentiment:      sentiment,
		CategoryStatus: fb.CategoryStatus,
	}
}