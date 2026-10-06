package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/feedback/internal/model"
	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/repository"
)

// MetricBucketService 管理时间网格预聚合管道。
// 活跃版本通过 metric_bucket_configs 表持久化，确保重建期间查询只读已完成数据。
type MetricBucketService struct {
	bucketRepo *repository.MetricTimeBucketRepo
	logRepo    *repository.MetricBucketUpdateLogRepo
	db         *gorm.DB
	version    int // 内存缓存，从 metric_bucket_configs 加载
}

func NewMetricBucketService(bucketRepo *repository.MetricTimeBucketRepo, logRepo *repository.MetricBucketUpdateLogRepo, db *gorm.DB) *MetricBucketService {
	return &MetricBucketService{bucketRepo: bucketRepo, logRepo: logRepo, db: db}
}

// InitVersion 从 metric_bucket_configs 表加载活跃版本，并确保配置行存在。
func (s *MetricBucketService) InitVersion(ctx context.Context) {
	var cfg model.MetricBucketConfig
	err := s.db.WithContext(ctx).Where("id = 1").First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		cfg = model.MetricBucketConfig{ID: 1, ActiveDimensionVersion: 1}
		if err := s.db.WithContext(ctx).Create(&cfg).Error; err != nil {
			logger.L.Warn("metric-bucket: create config row", zap.Error(err))
			s.version = 1
			return
		}
	} else if err != nil {
		logger.L.Warn("metric-bucket: load config", zap.Error(err))
		s.version = 1
		return
	}
	s.version = cfg.ActiveDimensionVersion
	logger.L.Info("metric-bucket: active version loaded", zap.Int("version", s.version))
}

// Version 返回当前活跃版本号（内存缓存）。
func (s *MetricBucketService) Version() int {
	return s.version
}

// SetVersion 持久化更新活跃版本号到 metric_bucket_configs 表。
// 重建/backfill 完成后调用，确保查询切换到新版本。
func (s *MetricBucketService) SetVersion(ctx context.Context, v int) {
	s.version = v
	if err := s.db.WithContext(ctx).
		Model(&model.MetricBucketConfig{}).
		Where("id = 1").
		Update("active_dimension_version", v).Error; err != nil {
		logger.L.Warn("metric-bucket: persist version", zap.Error(err))
	}
	logger.L.Info("metric-bucket: version switched", zap.Int("version", v))
	s.logRepo.Create(context.Background(), &model.MetricBucketUpdateLog{
		OpType:       "version_switch",
		Version:      v,
		AffectedRows: 0,
		DurationMs:   0,
		Status:       "success",
		Message:      fmt.Sprintf("活跃版本切换到 v%d", v),
		TriggeredBy:  "system",
	})
}

// supportedGranularities 时间网格支持的粒度列表。
// 同步入库/分类变更时为每个粒度各写一行桶数据。
var supportedGranularities = []string{"1m", "1h", "1d"}

// truncators 将 feedback 时间截断到对应粒度的整点时刻。
// 1d 使用 time.Date 截断到本地时区午夜，而非 UTC Truncate。
var truncators = map[string]func(time.Time) time.Time{
	"1m": func(t time.Time) time.Time { return t.Truncate(time.Minute) },
	"1h": func(t time.Time) time.Time { return t.Truncate(time.Hour) },
	"1d": func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()) },
}

// IncrementFromFeedbacks 同步入库时批量增量写入桶行。
// 每条 feedback 为每个支持粒度（1m/1h）各生成一行桶数据。
func (s *MetricBucketService) IncrementFromFeedbacks(ctx context.Context, feedbacks []*model.Feedback) error {
	if len(feedbacks) == 0 {
		return nil
	}
	start := time.Now()
	items := make([]model.MetricTimeBucket, 0, len(feedbacks)*len(supportedGranularities))
	for _, fb := range feedbacks {
		dims := repository.BucketDimsFromFeedback(*fb)
		for _, g := range supportedGranularities {
			bucketTime := truncators[g](fb.OriginalCreatedAt)
			items = append(items, model.MetricTimeBucket{
				BucketTime:             bucketTime,
				Granularity:            g,
				DimensionSchemaVersion: s.version,
				AppID:                  dims.AppID,
				PlatformID:             dims.PlatformID,
				AppVersion:             dims.AppVersion,
				Category:               dims.Category,
				BusinessModule:         dims.BusinessModule,
				Sentiment:              dims.Sentiment,
				CategoryStatus:         dims.CategoryStatus,
			})
		}
	}
	if err := s.bucketRepo.IncrementBatch(ctx, items); err != nil {
		s.logRepo.Create(context.Background(), &model.MetricBucketUpdateLog{
			OpType:       "increment",
			Version:      s.version,
			AffectedRows: len(feedbacks),
			DurationMs:   int(time.Since(start).Milliseconds()),
			Status:       "failed",
			Message:      err.Error(),
			TriggeredBy:  "sync",
		})
		return fmt.Errorf("increment feedbacks: %w", err)
	}
	s.logRepo.Create(context.Background(), &model.MetricBucketUpdateLog{
		OpType:       "increment",
		Version:      s.version,
		AffectedRows: len(feedbacks),
		DurationMs:   int(time.Since(start).Milliseconds()),
		Status:       "success",
		Message:      fmt.Sprintf("增量写入 %d 条反馈", len(feedbacks)),
		TriggeredBy:  "sync",
	})
	return nil
}

// MoveFeedbackBucket 维度变更时将一条 feedback 的 count 从旧维度组合移到新维度组合。
// 为每个支持粒度（1m/1h）同时执行旧行 -1、新行 +1。
func (s *MetricBucketService) MoveFeedbackBucket(ctx context.Context, fb model.Feedback, oldDims repository.BucketDims) error {
	return s.MoveFeedbackBucketWithSource(ctx, fb, oldDims, "classify", "classify_move", "分类变更")
}

// MoveFeedbackBucketWithSource 支持指定日志来源与描述，用于区分自动分类/手动分类等场景。
func (s *MetricBucketService) MoveFeedbackBucketWithSource(ctx context.Context, fb model.Feedback, oldDims repository.BucketDims, triggeredBy, opType, actionDesc string) error {
	start := time.Now()
	newDims := repository.BucketDimsFromFeedback(fb)
	for _, g := range supportedGranularities {
		bucketTime := truncators[g](fb.OriginalCreatedAt)
		if err := s.bucketRepo.MoveFeedbackBucket(ctx, bucketTime, g, s.version, oldDims, newDims); err != nil {
			s.logRepo.Create(context.Background(), &model.MetricBucketUpdateLog{
				OpType:       opType,
				Version:      s.version,
				AffectedRows: 1,
				DurationMs:   int(time.Since(start).Milliseconds()),
				Status:       "failed",
				Message:      fmt.Errorf("move feedback bucket (%s): %w", g, err).Error(),
				TriggeredBy:  triggeredBy,
			})
			return fmt.Errorf("move feedback bucket (%s): %w", g, err)
		}
	}
	s.logRepo.Create(context.Background(), &model.MetricBucketUpdateLog{
		OpType:       opType,
		Version:      s.version,
		AffectedRows: 1,
		DurationMs:   int(time.Since(start).Milliseconds()),
		Status:       "success",
		Message:      fmt.Sprintf("%s移动反馈 #%d", actionDesc, fb.ID),
		TriggeredBy:  triggeredBy,
	})
	return nil
}

// ReconcileRecent 对账最近同步的反馈，补充回填期间遗漏的桶行。
// backfillStartTime: 回填开始时间，仅处理此时间之后入库的反馈。
func (s *MetricBucketService) ReconcileRecent(ctx context.Context, backfillStartTime time.Time) error {
	start := time.Now()
	var feedbacks []*model.Feedback
	if err := s.db.WithContext(ctx).
		Where("created_at >= ?", backfillStartTime).
		Find(&feedbacks).Error; err != nil {
		s.logRepo.Create(context.Background(), &model.MetricBucketUpdateLog{
			OpType:       "reconcile",
			Version:      s.version,
			AffectedRows: 0,
			DurationMs:   int(time.Since(start).Milliseconds()),
			Status:       "failed",
			Message:      fmt.Errorf("查询回填窗口内反馈: %w", err).Error(),
			TriggeredBy:  "system",
		})
		return fmt.Errorf("查询回填窗口内反馈: %w", err)
	}
	if len(feedbacks) == 0 {
		return nil
	}
	logger.L.Info("metric-bucket: reconciling recent feedbacks", zap.Int("count", len(feedbacks)), zap.Time("since", backfillStartTime))
	err := s.IncrementFromFeedbacks(ctx, feedbacks)
	s.logRepo.Create(context.Background(), &model.MetricBucketUpdateLog{
		OpType:       "reconcile",
		Version:      s.version,
		AffectedRows: len(feedbacks),
		DurationMs:   int(time.Since(start).Milliseconds()),
		Status:       "success",
		Message:      fmt.Sprintf("对账补漏 %d 条反馈", len(feedbacks)),
		TriggeredBy:  "system",
	})
	if err != nil {
		return err
	}
	return nil
}
