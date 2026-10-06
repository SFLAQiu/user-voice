-- 添加 app_version 维度到维度配置和预聚合表

-- 1. 新增维度配置
INSERT INTO metric_dimension_configs (field, label, data_type, default_value, sort_order) VALUES
  ('app_version', '版本号', 'string', '', 8);

-- 2. 添加列到预聚合表
ALTER TABLE metric_time_buckets ADD COLUMN `app_version` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '应用版本';

-- 3. 重建唯一索引（包含 app_version）
ALTER TABLE metric_time_buckets DROP INDEX uk_bucket_dims;
ALTER TABLE metric_time_buckets ADD UNIQUE KEY uk_bucket_dims (
  bucket_time, granularity, dimension_schema_version,
  app_id, platform_id, category, business_module, sentiment, category_status, app_version
);
