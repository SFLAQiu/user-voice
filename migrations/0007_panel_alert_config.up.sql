-- Panel alert config: panels.alert_config_json + alert_rules.last_notify_at + alert_rules.idx_panel_id
SET NAMES utf8mb4;

-- 面板告警配置 JSON：threshold, condition_op, level, silence_minutes, eval_interval_sec, channel_ids
ALTER TABLE panels ADD COLUMN alert_config_json JSON NULL
  COMMENT '面板告警配置 JSON：threshold, condition_op, level, silence_minutes, eval_interval_sec, channel_ids'
  AFTER refresh_seconds;

-- 补齐 alert_rules.last_notify_at 列（Go model 已有但初始化 SQL 缺少）
ALTER TABLE alert_rules ADD COLUMN last_notify_at DATETIME NULL
  COMMENT '最近通知时间，用于静默期判断'
  AFTER consecutive_fires;

-- 为 panel_id 添加索引，便于告警引擎和告警管理页面按面板过滤
ALTER TABLE alert_rules ADD INDEX idx_panel_id (panel_id);