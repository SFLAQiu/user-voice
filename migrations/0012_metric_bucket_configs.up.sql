-- 活跃版本配置表（单行表，存储当前查询使用的 dimension_schema_version）
-- 解决 MAX(dimension_schema_version) 子查询在重建期间读到未完成数据的问题
CREATE TABLE IF NOT EXISTS metric_bucket_configs (
  id                    INT UNSIGNED PRIMARY KEY DEFAULT 1 COMMENT '固定为1，单行表',
  active_dimension_version INT NOT NULL DEFAULT 1         COMMENT '当前查询使用的活跃维度版本号，重建/backfill完成后才切换',
  updated_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  CONSTRAINT chk_single_row CHECK (id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='时间桶版本配置（单行表）';

-- 初始版本为 1
INSERT INTO metric_bucket_configs (id, active_dimension_version) VALUES (1, 1);