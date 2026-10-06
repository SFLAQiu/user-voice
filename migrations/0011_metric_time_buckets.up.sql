-- 时间网格预聚合表 + 维度配置表
-- 用于仪表盘图表统计和告警，替代实时聚合查询，确保跨粒度值一致性

-- 维度配置表（记录哪些 feedback 字段作为时间网格维度）
CREATE TABLE IF NOT EXISTS metric_dimension_configs (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  field        VARCHAR(64) NOT NULL COMMENT 'feedbacks 表字段名（如 app_version）',
  label        VARCHAR(128) NOT NULL COMMENT '维度中文名（如 应用版本）',
  data_type    VARCHAR(16) NOT NULL DEFAULT 'string' COMMENT '字段类型：string|int',
  default_value VARCHAR(64) NOT NULL DEFAULT '' COMMENT '空值时的默认填充（string→空串，int→0）',
  status       TINYINT NOT NULL DEFAULT 1 COMMENT '1启用 0禁用',
  sort_order   INT NOT NULL DEFAULT 0 COMMENT '排序权重',
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_field (field)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='时间网格维度配置表';

-- 初始维度数据
INSERT INTO metric_dimension_configs (field, label, data_type, default_value, sort_order) VALUES
  ('app_id',          '应用',     'int',    '0', 2),
  ('platform_id',     '平台',     'int',    '0', 3),
  ('category',        '分类',     'string', '', 4),
  ('business_module', '业务模块', 'string', '', 5),
  ('sentiment',       '情感倾向', 'string', '', 6),
  ('category_status', '分类状态', 'int',    '0', 7);

-- 时间桶预聚合表（时间网格）
CREATE TABLE IF NOT EXISTS metric_time_buckets (
  id                       BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  bucket_time              DATETIME NOT NULL           COMMENT '时间桶时刻（1m粒度，截断到整分钟，本地时区）',
  granularity              VARCHAR(4) NOT NULL DEFAULT '1m' COMMENT '粒度级别：1m',
  dimension_schema_version INT NOT NULL DEFAULT 1     COMMENT '维度结构版本号，新增维度字段时递增',
  app_id                   INT NOT NULL DEFAULT 0      COMMENT '应用ID',
  platform_id              INT NOT NULL DEFAULT 0      COMMENT '平台ID（0=未知）',
  category                 VARCHAR(64) NOT NULL DEFAULT '' COMMENT '分类（空=待分类）',
  business_module          VARCHAR(64) NOT NULL DEFAULT '' COMMENT '业务模块（空=未知）',
  sentiment                VARCHAR(16) NOT NULL DEFAULT '' COMMENT '情感倾向（空=未分析）',
  category_status          TINYINT NOT NULL DEFAULT 0  COMMENT '分类状态：0待分类 1已分类 2失败 3手动',
  feedback_count           INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '该桶+维度组合的反馈数量',
  created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  updated_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
  UNIQUE KEY uk_bucket_dims (bucket_time, granularity, dimension_schema_version, app_id, platform_id, category, business_module, sentiment, category_status),
  INDEX idx_version_gran_time (dimension_schema_version, granularity, bucket_time),
  INDEX idx_app_time (app_id, bucket_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='反馈指标时间桶预聚合表（时间网格）';

-- 移除 source_id 维度列（旧版本可能已存在；MySQL 8 不支持 DROP COLUMN IF EXISTS）
SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'metric_time_buckets' AND COLUMN_NAME = 'source_id');
SET @sql = IF(@col_exists > 0, 'ALTER TABLE metric_time_buckets DROP COLUMN source_id', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;