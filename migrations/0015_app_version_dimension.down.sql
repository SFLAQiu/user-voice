-- 回滚 app_version 维度

-- 1. 恢复旧唯一索引（不含 app_version）
ALTER TABLE metric_time_buckets DROP INDEX uk_bucket_dims;
ALTER TABLE metric_time_buckets ADD UNIQUE KEY uk_bucket_dims (
  bucket_time, granularity, dimension_schema_version,
  app_id, platform_id, category, business_module, sentiment, category_status
);

-- 2. 删除列
ALTER TABLE metric_time_buckets DROP COLUMN `app_version`;

-- 3. 删除维度配置
DELETE FROM metric_dimension_configs WHERE field = 'app_version';
