package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/feedback/internal/model"
	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/repository"
)

// RebuildStatus 维度重建进度状态。
type RebuildStatus struct {
	State     string     `json:"state"` // idle | running | completed | failed
	Version   int        `json:"version"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	Message   string     `json:"message,omitempty"`
}

// DimensionSchemaService 维度管理服务，负责维度配置的增删和全量重建。
type DimensionSchemaService struct {
	dimRepo    *repository.MetricDimensionConfigRepo
	bucketRepo *repository.MetricTimeBucketRepo
	bucketSvc  *MetricBucketService
	logRepo    *repository.MetricBucketUpdateLogRepo
	db         *gorm.DB
	mu         sync.Mutex
	status     RebuildStatus
}

func NewDimensionSchemaService(
	dimRepo *repository.MetricDimensionConfigRepo,
	bucketRepo *repository.MetricTimeBucketRepo,
	bucketSvc *MetricBucketService,
	logRepo *repository.MetricBucketUpdateLogRepo,
	db *gorm.DB,
) *DimensionSchemaService {
	return &DimensionSchemaService{
		dimRepo:    dimRepo,
		bucketRepo: bucketRepo,
		bucketSvc:  bucketSvc,
		logRepo:    logRepo,
		db:         db,
		status:     RebuildStatus{State: "idle"},
	}
}

// List 返回所有维度配置（含禁用）。
func (s *DimensionSchemaService) List(ctx context.Context) ([]model.MetricDimensionConfig, error) {
	return s.dimRepo.ListAll(ctx)
}

// AddDimension 新增维度配置并触发重建。
func (s *DimensionSchemaService) AddDimension(ctx context.Context, field, label, dataType, defaultValue string) error {
	if s.status.State == "running" {
		return fmt.Errorf("维度重建正在进行中，请稍后再试")
	}

	existing, err := s.dimRepo.GetByField(ctx, field)
	if err != nil {
		return fmt.Errorf("查询维度配置: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("维度字段 %s 已存在", field)
	}

	item := &model.MetricDimensionConfig{
		Field:        field,
		Label:        label,
		DataType:     dataType,
		DefaultValue: defaultValue,
		Status:       1,
		SortOrder:    len(fmt.Sprintf("%d", 0)),
	}
	if err := s.dimRepo.Create(ctx, item); err != nil {
		return fmt.Errorf("创建维度配置: %w", err)
	}

	s.startRebuild(ctx)
	return nil
}

// RemoveDimension 删除维度配置并触发重建。
func (s *DimensionSchemaService) RemoveDimension(ctx context.Context, field string) error {
	if s.status.State == "running" {
		return fmt.Errorf("维度重建正在进行中，请稍后再试")
	}

	if err := s.dimRepo.Delete(ctx, field); err != nil {
		return fmt.Errorf("删除维度配置: %w", err)
	}

	s.startRebuild(ctx)
	return nil
}

// GetStatus 返回当前重建进度；idle/completed 时从 bucketSvc 获取真实活跃版本。
func (s *DimensionSchemaService) GetStatus() RebuildStatus {
	s.mu.Lock()
	st := s.status
	s.mu.Unlock()
	// idle 或 completed 状态下 version 可能是初始 0，用实际活跃版本替代
	if st.State == "idle" || st.State == "completed" && st.Version == 0 {
		st.Version = s.bucketSvc.Version()
	}
	return st
}

// TriggerBackfill 手动触发全量回填历史数据。
// 使用版本+1策略：写入 v+1 完整数据 → 切换活跃版本 → 删除 v_old → 对账补漏。
// 确保回填期间告警查询仍读旧版本数据，丝滑过渡。
func (s *DimensionSchemaService) TriggerBackfill(ctx context.Context) error {
	s.mu.Lock()
	if s.status.State == "running" {
		s.mu.Unlock()
		return fmt.Errorf("维度重建正在进行中，请稍后再试")
	}
	currentVersion := s.bucketSvc.Version()
	newVersion := currentVersion + 1
	s.status = RebuildStatus{
		State:     "running",
		Version:   newVersion,
		StartedAt: nowPtr(),
		Message:   fmt.Sprintf("开始全量回填，新版本 v%d，当前版本 v%d", newVersion, currentVersion),
	}
	s.mu.Unlock()

	go func() {
		bgCtx := context.Background()
		startTime := time.Now()
		err := s.backfillAndSwitch(bgCtx, newVersion, startTime)
		durationMs := int(time.Since(startTime).Milliseconds())
		s.mu.Lock()
		if err != nil {
			s.status.State = "failed"
			s.status.Message = fmt.Sprintf("回填失败: %v", err)
			logger.L.Error("metric-dimension: backfill failed", zap.Error(err))
			s.logRepo.Create(bgCtx, &model.MetricBucketUpdateLog{
				OpType: "backfill", Version: newVersion, AffectedRows: 0, DurationMs: durationMs,
				Status: "failed", Message: err.Error(), TriggeredBy: "manual",
			})
		} else {
			s.status.State = "completed"
			s.status.Message = fmt.Sprintf("回填完成，活跃版本 v%d", newVersion)
			logger.L.Info("metric-dimension: backfill completed", zap.Int("version", newVersion))
			s.logRepo.Create(bgCtx, &model.MetricBucketUpdateLog{
				OpType: "backfill", Version: newVersion, AffectedRows: 0, DurationMs: durationMs,
				Status: "success", Message: fmt.Sprintf("全量回填完成，活跃版本 v%d", newVersion), TriggeredBy: "manual",
			})
		}
		s.mu.Unlock()
	}()
	return nil
}

// startRebuild 异步启动维度重建流程（版本+1策略）。
func (s *DimensionSchemaService) startRebuild(ctx context.Context) {
	s.mu.Lock()
	newVersion := s.bucketSvc.Version() + 1
	s.status = RebuildStatus{
		State:     "running",
		Version:   newVersion,
		StartedAt: nowPtr(),
		Message:   fmt.Sprintf("开始重建维度版本 v%d", newVersion),
	}
	s.mu.Unlock()

	go func() {
		bgCtx := context.Background()
		startTime := time.Now()
		err := s.rebuild(bgCtx, newVersion, startTime)
		durationMs := int(time.Since(startTime).Milliseconds())
		s.mu.Lock()
		if err != nil {
			s.status.State = "failed"
			s.status.Message = fmt.Sprintf("重建失败: %v", err)
			logger.L.Error("metric-dimension: rebuild failed", zap.Error(err))
			s.logRepo.Create(bgCtx, &model.MetricBucketUpdateLog{
				OpType: "backfill", Version: newVersion, AffectedRows: 0, DurationMs: durationMs,
				Status: "failed", Message: err.Error(), TriggeredBy: "dimension_change",
			})
		} else {
			s.status.State = "completed"
			s.status.Message = fmt.Sprintf("重建完成，活跃版本 v%d", newVersion)
			logger.L.Info("metric-dimension: rebuild completed", zap.Int("version", newVersion))
			s.logRepo.Create(bgCtx, &model.MetricBucketUpdateLog{
				OpType: "backfill", Version: newVersion, AffectedRows: 0, DurationMs: durationMs,
				Status: "success", Message: fmt.Sprintf("维度重建完成，活跃版本 v%d", newVersion), TriggeredBy: "dimension_change",
			})
		}
		s.mu.Unlock()
	}()
}

// rebuild 执行维度重建：ALTER TABLE → 全量回填 v+1 → 切换版本 → 清理 v_old → 对账补漏。
// 整个过程中查询仍读旧版本数据（metric_bucket_configs.active_dimension_version 未切换）。
func (s *DimensionSchemaService) rebuild(ctx context.Context, newVersion int, startTime time.Time) error {
	dims, err := s.dimRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("读取维度配置: %w", err)
	}

	// Step 1: ALTER TABLE — 为新增维度添加列
	columns, err := s.getTableColumns(ctx)
	if err != nil {
		return fmt.Errorf("获取表结构: %w", err)
	}
	for _, dim := range dims {
		colName := dim.Field
		if _, exists := columns[colName]; !exists {
			colDef := s.columnDefinition(dim)
			if err := s.db.WithContext(ctx).Exec(colDef).Error; err != nil {
				return fmt.Errorf("添加列 %s: %w", colName, err)
			}
			logger.L.Info("metric-dimension: added column", zap.String("column", colName))
		}
	}

	// Step 2: 重建 UNIQUE KEY
	if err := s.rebuildUniqueKey(ctx, dims); err != nil {
		return fmt.Errorf("重建 UNIQUE KEY: %w", err)
	}

	// Step 3: 全量回填新版本数据
	if err := s.backfillNewVersion(ctx, dims, newVersion); err != nil {
		return fmt.Errorf("回填数据: %w", err)
	}

	// Step 4: 切换活跃版本（告警和图表查询开始读 v+1）
	s.bucketSvc.SetVersion(ctx, newVersion)

	// Step 5: 对账补漏 — 处理回填期间新增的反馈
	if err := s.bucketSvc.ReconcileRecent(ctx, startTime); err != nil {
		logger.L.Warn("metric-dimension: reconcile recent feedbacks", zap.Error(err))
	}

	// Step 6: 删除旧版本行
	oldVersion := newVersion - 1
	if err := s.bucketRepo.DeleteByVersion(ctx, oldVersion); err != nil {
		logger.L.Warn("metric-dimension: cleanup old version", zap.Error(err))
	} else {
		s.logRepo.Create(ctx, &model.MetricBucketUpdateLog{
			OpType: "cleanup", Version: oldVersion, AffectedRows: 0, DurationMs: 0,
			Status: "success", Message: fmt.Sprintf("清理旧版本 v%d 行", oldVersion), TriggeredBy: "system",
		})
	}

	// Step 7: 清理 count=0 的空行
	if err := s.bucketRepo.CleanupZeroRows(ctx, newVersion); err != nil {
		logger.L.Warn("metric-dimension: cleanup zero rows", zap.Error(err))
	} else {
		s.logRepo.Create(ctx, &model.MetricBucketUpdateLog{
			OpType: "cleanup", Version: newVersion, AffectedRows: 0, DurationMs: 0,
			Status: "success", Message: fmt.Sprintf("清理 v%d 零行", newVersion), TriggeredBy: "system",
		})
	}

	return nil
}

// backfillAndSwitch 执行纯回填（不修改表结构）：回填 v+1 → 切换版本 → 清理 v_old → 对账补漏。
func (s *DimensionSchemaService) backfillAndSwitch(ctx context.Context, newVersion int, startTime time.Time) error {
	dims, err := s.dimRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("读取维度配置: %w", err)
	}

	// Step 1: 全量回填新版本数据
	if err := s.backfillNewVersion(ctx, dims, newVersion); err != nil {
		return fmt.Errorf("回填数据: %w", err)
	}

	// Step 2: 切换活跃版本
	s.bucketSvc.SetVersion(ctx, newVersion)

	// Step 3: 对账补漏 — 处理回填期间新增的反馈
	if err := s.bucketSvc.ReconcileRecent(ctx, startTime); err != nil {
		logger.L.Warn("metric-dimension: reconcile recent feedbacks", zap.Error(err))
	}

	// Step 4: 删除旧版本行
	oldVersion := newVersion - 1
	if err := s.bucketRepo.DeleteByVersion(ctx, oldVersion); err != nil {
		logger.L.Warn("metric-dimension: cleanup old version", zap.Error(err))
	} else {
		s.logRepo.Create(ctx, &model.MetricBucketUpdateLog{
			OpType: "cleanup", Version: oldVersion, AffectedRows: 0, DurationMs: 0,
			Status: "success", Message: fmt.Sprintf("清理旧版本 v%d 行", oldVersion), TriggeredBy: "system",
		})
	}

	// Step 5: 清理 count=0 的空行
	if err := s.bucketRepo.CleanupZeroRows(ctx, newVersion); err != nil {
		logger.L.Warn("metric-dimension: cleanup zero rows", zap.Error(err))
	} else {
		s.logRepo.Create(ctx, &model.MetricBucketUpdateLog{
			OpType: "cleanup", Version: newVersion, AffectedRows: 0, DurationMs: 0,
			Status: "success", Message: fmt.Sprintf("清理 v%d 零行", newVersion), TriggeredBy: "system",
		})
	}

	return nil
}

// getTableColumns 获取 metric_time_buckets 表当前的列名集合。
func (s *DimensionSchemaService) getTableColumns(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.WithContext(ctx).Raw(
		"SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'metric_time_buckets'",
	).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, nil
}

// columnDefinition 生成 ALTER TABLE ADD COLUMN 的 SQL。
func (s *DimensionSchemaService) columnDefinition(dim model.MetricDimensionConfig) string {
	switch dim.DataType {
	case "int":
		return fmt.Sprintf("ALTER TABLE metric_time_buckets ADD COLUMN `%s` INT NOT NULL DEFAULT %s COMMENT '%s'",
			dim.Field, dim.DefaultValue, dim.Label)
	default:
		return fmt.Sprintf("ALTER TABLE metric_time_buckets ADD COLUMN `%s` VARCHAR(64) NOT NULL DEFAULT '%s' COMMENT '%s'",
			dim.Field, dim.DefaultValue, dim.Label)
	}
}

// rebuildUniqueKey 重建唯一索引，包含所有当前维度字段。
func (s *DimensionSchemaService) rebuildUniqueKey(ctx context.Context, dims []model.MetricDimensionConfig) error {
	if err := s.db.WithContext(ctx).Exec("ALTER TABLE metric_time_buckets DROP INDEX uk_bucket_dims").Error; err != nil {
		logger.L.Warn("metric-dimension: drop old unique key", zap.Error(err))
	}

	dimColumns := make([]string, 0, len(dims))
	for _, dim := range dims {
		dimColumns = append(dimColumns, dim.Field)
	}
	allColumns := []string{"bucket_time", "granularity", "dimension_schema_version"}
	allColumns = append(allColumns, dimColumns...)
	colList := ""
	for i, col := range allColumns {
		if i > 0 {
			colList += ", "
		}
		colList += "`" + col + "`"
	}

	sql := fmt.Sprintf("ALTER TABLE metric_time_buckets ADD UNIQUE KEY uk_bucket_dims (%s)", colList)
	if err := s.db.WithContext(ctx).Exec(sql).Error; err != nil {
		return fmt.Errorf("添加唯一索引: %w", err)
	}
	return nil
}

// backfillGranularities 回填时需要生成的粒度列表，与 metric_bucket_service.supportedGranularities 一致。
var backfillGranularities = []struct {
	value      string
	dateFormat string // MySQL DATE_FORMAT 用于生成 bucket_time
}{
	{"1m", "%Y-%m-%d %H:%i:00"},
	{"1h", "%Y-%m-%d %H:00:00"},
	{"1d", "%Y-%m-%d 00:00:00"},
}

// backfillNewVersion 从 feedbacks 表全量回填新版本数据。
// 分三类 category_status 处理，确保维度值准确。
// 为每个支持粒度（1m/1h）各生成一套回填数据。
func (s *DimensionSchemaService) backfillNewVersion(ctx context.Context, dims []model.MetricDimensionConfig, version int) error {
	dimFields := make([]string, 0, len(dims))
	dimSelectParts := make([]string, 0, len(dims))
	dimGroupParts := make([]string, 0, len(dims))
	for _, dim := range dims {
		col := dim.Field
		dimFields = append(dimFields, col)
		if dim.DataType == "int" {
			dimSelectParts = append(dimSelectParts, fmt.Sprintf("IFNULL(%s, %s)", col, dim.DefaultValue))
			dimGroupParts = append(dimGroupParts, fmt.Sprintf("IFNULL(%s, %s)", col, dim.DefaultValue))
		} else {
			dimSelectParts = append(dimSelectParts, fmt.Sprintf("IFNULL(NULLIF(%s, ''), '')", col))
			dimGroupParts = append(dimGroupParts, fmt.Sprintf("IFNULL(NULLIF(%s, ''), '')", col))
		}
	}

	emptyDimSelect := ""
	emptyDimGroup := ""
	for i, dim := range dims {
		if i > 0 {
			emptyDimSelect += ", "
			emptyDimGroup += ", "
		}
		if dim.Field == "category" || dim.Field == "business_module" || dim.Field == "sentiment" {
			emptyDimSelect += "'" + dim.DefaultValue + "'"
			emptyDimGroup += "'" + dim.DefaultValue + "'"
		} else {
			emptyDimSelect += dimSelectParts[i]
			emptyDimGroup += dimGroupParts[i]
		}
	}

	dimFieldsStr := joinStrings(dimFields, ", ")

	// 为每个粒度执行回填
	for _, g := range backfillGranularities {
		bucketTimeExpr := fmt.Sprintf("DATE_FORMAT(original_created_at, '%s')", g.dateFormat)

		// category_status=0（待分类）
		selectCols := bucketTimeExpr + ", '" + g.value + "', " + fmt.Sprintf("%d", version) + ", " + emptyDimSelect + ", COUNT(*), NOW(), NOW()"
		groupCols := bucketTimeExpr + ", " + emptyDimGroup

		sql := fmt.Sprintf(`INSERT INTO metric_time_buckets
			(bucket_time, granularity, dimension_schema_version, %s, feedback_count, created_at, updated_at)
		SELECT %s FROM feedbacks WHERE category_status = 0 GROUP BY %s
		ON DUPLICATE KEY UPDATE feedback_count = feedback_count + VALUES(feedback_count), updated_at = NOW()`,
			dimFieldsStr, selectCols, groupCols,
		)
		if err := s.db.WithContext(ctx).Exec(sql).Error; err != nil {
			return fmt.Errorf("回填待分类数据 (%s): %w", g.value, err)
		}

		// category_status IN (1, 3)（已分类/手动）
		selectCols2 := bucketTimeExpr + ", '" + g.value + "', " + fmt.Sprintf("%d", version) + ", " + joinStrings(dimSelectParts, ", ") + ", COUNT(*), NOW(), NOW()"
		groupCols2 := bucketTimeExpr + ", " + joinStrings(dimGroupParts, ", ")

		sql2 := fmt.Sprintf(`INSERT INTO metric_time_buckets
			(bucket_time, granularity, dimension_schema_version, %s, feedback_count, created_at, updated_at)
		SELECT %s FROM feedbacks WHERE category_status IN (1, 3) GROUP BY %s
		ON DUPLICATE KEY UPDATE feedback_count = feedback_count + VALUES(feedback_count), updated_at = NOW()`,
			dimFieldsStr, selectCols2, groupCols2,
		)
		if err := s.db.WithContext(ctx).Exec(sql2).Error; err != nil {
			return fmt.Errorf("回填已分类数据 (%s): %w", g.value, err)
		}

		// category_status=2（分类失败）
		sql3 := fmt.Sprintf(`INSERT INTO metric_time_buckets
			(bucket_time, granularity, dimension_schema_version, %s, feedback_count, created_at, updated_at)
		SELECT %s FROM feedbacks WHERE category_status = 2 GROUP BY %s
		ON DUPLICATE KEY UPDATE feedback_count = feedback_count + VALUES(feedback_count), updated_at = NOW()`,
			dimFieldsStr, selectCols, groupCols,
		)
		if err := s.db.WithContext(ctx).Exec(sql3).Error; err != nil {
			return fmt.Errorf("回填分类失败数据 (%s): %w", g.value, err)
		}
	}

	return nil
}

func joinStrings(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}
