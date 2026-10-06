-- 时间桶更新日志表
-- 记录每次对 metric_time_buckets 的操作批次，区分更新来源

CREATE TABLE IF NOT EXISTS metric_bucket_update_logs (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  op_type       VARCHAR(32) NOT NULL COMMENT '操作类型：backfill|increment|classify_move|version_switch|cleanup|reconcile',
  version       INT NOT NULL DEFAULT 0 COMMENT '涉及的维度版本号',
  affected_rows INT NOT NULL DEFAULT 0 COMMENT '影响行数（反馈数/桶行数）',
  duration_ms   INT NOT NULL DEFAULT 0 COMMENT '操作耗时（毫秒）',
  status        VARCHAR(16) NOT NULL DEFAULT 'success' COMMENT '执行状态：success|failed',
  message       VARCHAR(512) COMMENT '描述信息或错误消息',
  triggered_by  VARCHAR(64) COMMENT '触发来源：manual|sync|classify|system|dimension_change',
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  INDEX idx_op_type (op_type),
  INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='时间桶更新日志表';