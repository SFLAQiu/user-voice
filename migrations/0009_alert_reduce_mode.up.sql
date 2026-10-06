-- 告警规则新增缩减模式字段：max（峰值）| avg（均值）| last（最新）| sum（总量）
-- 保留 XDimension 分桶后逐桶数据缩减为触发值，max 模式确保图表上触发值可与阈值线对齐
ALTER TABLE alert_rules ADD COLUMN reduce_mode VARCHAR(16) NOT NULL DEFAULT 'max' COMMENT '告警缩减模式：max|avg|last|sum' AFTER time_window_sec;
-- 存量规则统一改为 max
UPDATE alert_rules SET reduce_mode = 'max' WHERE reduce_mode = '' OR reduce_mode IS NULL OR reduce_mode = 'sum';