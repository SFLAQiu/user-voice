package model

import "time"

// MetricTimeBucket 反馈指标时间桶预聚合行，构成时间网格。
// 每条 feedback 的 original_created_at 截断到整分钟后，只落入一个 bucket_time。
// 同维度的 60 个 1m 桶 SUM(feedback_count) = 1h 值，1440 = 1d 值，10080 = 1w 值。
type MetricTimeBucket struct {
	ID                     uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	BucketTime             time.Time `gorm:"column:bucket_time;not null;comment:时间桶时刻（1m粒度，截断到整分钟，本地时区）" json:"bucket_time"`
	Granularity            string    `gorm:"column:granularity;size:4;not null;default:1m;comment:粒度级别：1m/1h/1d" json:"granularity"`
	DimensionSchemaVersion int       `gorm:"column:dimension_schema_version;not null;default:1;comment:维度结构版本号，新增维度字段时递增" json:"dimension_schema_version"`
	AppID                  int       `gorm:"column:app_id;not null;default:0;comment:应用ID" json:"app_id"`
	PlatformID             int       `gorm:"column:platform_id;not null;default:0;comment:平台ID（0=未知）" json:"platform_id"`
	Category               string    `gorm:"column:category;size:64;not null;default:'';comment:分类（空=待分类）" json:"category"`
	BusinessModule         string    `gorm:"column:business_module;size:64;not null;default:'';comment:业务模块（空=未知）" json:"business_module"`
	Sentiment              string    `gorm:"column:sentiment;size:16;not null;default:'';comment:情感倾向（空=未分析）" json:"sentiment"`
	AppVersion             string    `gorm:"column:app_version;size:64;not null;default:'';comment:应用版本" json:"app_version"`
	CategoryStatus         int       `gorm:"column:category_status;not null;default:0;comment:分类状态：0待分类 1已分类 2失败 3手动" json:"category_status"`
	FeedbackCount          int       `gorm:"column:feedback_count;not null;default:0;comment:该桶+维度组合的反馈数量" json:"feedback_count"`
	CreatedAt              time.Time `gorm:"comment:记录创建时间" json:"created_at"`
	UpdatedAt              time.Time `gorm:"comment:记录更新时间" json:"updated_at"`
}

func (MetricTimeBucket) TableName() string { return "metric_time_buckets" }

// MetricDimensionConfig 时间网格维度配置，记录哪些 feedback 字段作为维度。
// 新增/删除维度时触发全量重建（dimension_schema_version 递增）。
type MetricDimensionConfig struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	Field        string    `gorm:"column:field;size:64;not null;uniqueIndex:uk_field;comment:feedbacks 表字段名（如 app_version）" json:"field"`
	Label        string    `gorm:"column:label;size:128;not null;comment:维度中文名（如 应用版本）" json:"label"`
	DataType     string    `gorm:"column:data_type;size:16;not null;default:string;comment:字段类型：string|int" json:"data_type"`
	DefaultValue string    `gorm:"column:default_value;size:64;not null;default:'';comment:空值时的默认填充（string→空串，int→0）" json:"default_value"`
	Status       int       `gorm:"column:status;not null;default:1;comment:1启用 0禁用" json:"status"`
	SortOrder    int       `gorm:"column:sort_order;not null;default:0;comment:排序权重" json:"sort_order"`
	CreatedAt    time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (MetricDimensionConfig) TableName() string { return "metric_dimension_configs" }

// MetricBucketConfig 活跃版本配置（单行表），确保查询只读已完成的版本数据。
// 重建/backfill 期间写入 v+1 行，完成后才切换 active_dimension_version。
type MetricBucketConfig struct {
	ID                     uint64    `gorm:"primaryKey;default:1;comment:固定为1，单行表" json:"id"`
	ActiveDimensionVersion int       `gorm:"column:active_dimension_version;not null;default:1;comment:当前查询使用的活跃维度版本号" json:"active_dimension_version"`
	UpdatedAt              time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (MetricBucketConfig) TableName() string { return "metric_bucket_configs" }

// MetricBucketUpdateLog 时间桶更新日志，记录每次对 metric_time_buckets 的操作批次。
// 区分操作来源（全量回填、增量更新、分类变更等），便于追溯数据变更情况。
type MetricBucketUpdateLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	OpType       string    `gorm:"column:op_type;size:32;not null;comment:操作类型：backfill|increment|classify_move|version_switch|cleanup|reconcile" json:"op_type"`
	Version      int       `gorm:"column:version;not null;default:0;comment:涉及的维度版本号" json:"version"`
	AffectedRows int       `gorm:"column:affected_rows;not null;default:0;comment:影响行数（反馈数/桶行数）" json:"affected_rows"`
	DurationMs   int       `gorm:"column:duration_ms;not null;default:0;comment:操作耗时（毫秒）" json:"duration_ms"`
	Status       string    `gorm:"column:status;size:16;not null;default:success;comment:执行状态：success|failed" json:"status"`
	Message      string    `gorm:"column:message;size:512;comment:描述信息或错误消息" json:"message,omitempty"`
	TriggeredBy  string    `gorm:"column:triggered_by;size:64;comment:触发来源：manual|sync|classify|system|dimension_change" json:"triggered_by,omitempty"`
	CreatedAt    time.Time `gorm:"comment:记录创建时间" json:"created_at"`
}

func (MetricBucketUpdateLog) TableName() string { return "metric_bucket_update_logs" }
