ALTER TABLE alert_records ADD COLUMN metric_timestamp DATETIME NULL COMMENT '触发告警的指标数据点的时间戳（数据桶时刻），用于图表标注对齐' AFTER triggered_at;
