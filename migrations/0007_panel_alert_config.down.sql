SET NAMES utf8mb4;

ALTER TABLE panels DROP COLUMN alert_config_json;
ALTER TABLE alert_rules DROP COLUMN last_notify_at;
ALTER TABLE alert_rules DROP INDEX idx_panel_id;