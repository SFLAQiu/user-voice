SET NAMES utf8mb4;

-- 增加 pending_duration_sec 列：待确认时长（秒），0 表示立即告警
ALTER TABLE alert_rules ADD COLUMN pending_duration_sec INT NOT NULL DEFAULT 0
  COMMENT '待确认时长（秒）：指标突破阈值后需持续高于阈值的时间，0=立即告警'
  AFTER eval_interval_sec;