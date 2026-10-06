-- Initial schema for feedback platform (MySQL 8.0+)
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS=0;

CREATE TABLE IF NOT EXISTS users (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username        VARCHAR(64) NOT NULL UNIQUE,
  password_hash   VARCHAR(255) NOT NULL,
  role            VARCHAR(16) NOT NULL DEFAULT 'viewer'  COMMENT 'admin | viewer',
  status          TINYINT NOT NULL DEFAULT 1             COMMENT '1 enabled, 0 disabled',
  last_login_at   DATETIME NULL,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS login_attempts (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  identifier   VARCHAR(128) NOT NULL COMMENT 'username or ip',
  attempted_at DATETIME NOT NULL,
  success      TINYINT NOT NULL,
  INDEX idx_identifier_time (identifier, attempted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS data_sources (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name            VARCHAR(128) NOT NULL,
  type            VARCHAR(32)  NOT NULL                   COMMENT 'http_api etc.',
  config_json     JSON NOT NULL                           COMMENT 'plugin-specific; sensitive fields encrypted',
  sync_cron       VARCHAR(64) NOT NULL DEFAULT '*/5 * * * *',
  status          TINYINT NOT NULL DEFAULT 1              COMMENT '1 enabled, 0 disabled',
  last_sync_at    DATETIME NULL,
  last_sync_status VARCHAR(32) NULL                       COMMENT 'success|failed|running',
  last_sync_error TEXT NULL,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_name (name),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS feedbacks (
  id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  source_id             BIGINT UNSIGNED NOT NULL,
  original_id           VARCHAR(64)  NOT NULL                COMMENT 'source-side id',
  app_id                INT NOT NULL DEFAULT 0,
  app_name              VARCHAR(64) NULL,
  platform              VARCHAR(32) NULL                     COMMENT 'iOS/Android/HarmonyOS',
  platform_id           INT NULL,
  user_id               VARCHAR(64) NULL,
  user_name             VARCHAR(128) NULL,
  user_mode             INT NULL,
  content               TEXT NOT NULL,
  images                JSON NULL,
  phone_model           VARCHAR(64) NULL,
  app_version           VARCHAR(32) NULL,
  channel_id            VARCHAR(32) NULL,
  qq                    VARCHAR(32) NULL,
  file_url              TEXT NULL,
  raw_json              JSON NULL,
  category              VARCHAR(64) NULL,
  category_confidence   DECIMAL(4,3) NULL,
  category_status       TINYINT NOT NULL DEFAULT 0           COMMENT '0 pending 1 classified 2 failed 3 manual',
  sentiment             VARCHAR(16) NULL                     COMMENT 'positive/neutral/negative',
  classified_at         DATETIME NULL,
  original_created_at   DATETIME NOT NULL,
  synced_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_source_origid (source_id, original_id),
  INDEX idx_original_created (original_created_at),
  INDEX idx_app_platform_time (app_id, platform_id, original_created_at),
  INDEX idx_category (category),
  INDEX idx_sentiment (sentiment),
  INDEX idx_category_status (category_status),
  FULLTEXT INDEX ft_content (content) WITH PARSER ngram
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS sync_cursors (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  source_id     BIGINT UNSIGNED NOT NULL,
  cursor_key    VARCHAR(64) NOT NULL,
  last_original_id VARCHAR(64) NULL,
  last_created_at  DATETIME NULL,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_source_key (source_id, cursor_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS audit_logs (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id     BIGINT UNSIGNED NULL,
  username    VARCHAR(64) NULL,
  action      VARCHAR(64) NOT NULL,
  resource    VARCHAR(64) NULL,
  resource_id VARCHAR(64) NULL,
  ip          VARCHAR(64) NULL,
  detail_json JSON NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_user_time (user_id, created_at),
  INDEX idx_action_time (action, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Dashboard / Panel / Alert tables (reserved for P2/P3)
CREATE TABLE IF NOT EXISTS dashboards (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(128) NOT NULL,
  description   VARCHAR(512) NULL,
  layout_json   JSON NULL,
  created_by    BIGINT UNSIGNED NOT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS panels (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  dashboard_id      BIGINT UNSIGNED NOT NULL,
  name              VARCHAR(128) NOT NULL,
  chart_type        VARCHAR(32) NOT NULL,
  query_config_json JSON NOT NULL,
  position_json     JSON NULL,
  refresh_seconds   INT NOT NULL DEFAULT 60,
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_dashboard (dashboard_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_rules (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name              VARCHAR(128) NOT NULL,
  panel_id          BIGINT UNSIGNED NULL,
  metric_query_json JSON NOT NULL,
  time_window_sec   INT NOT NULL,
  condition_op      VARCHAR(8) NOT NULL,
  threshold         DECIMAL(20,4) NOT NULL,
  level             VARCHAR(16) NOT NULL,
  silence_minutes   INT NOT NULL DEFAULT 30,
  eval_interval_sec INT NOT NULL DEFAULT 60,
  template_id       BIGINT UNSIGNED NULL,
  status            TINYINT NOT NULL DEFAULT 1,
  last_eval_at      DATETIME NULL,
  last_state        VARCHAR(16) NULL,
  consecutive_fires INT NOT NULL DEFAULT 0,
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_channels (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(128) NOT NULL,
  type          VARCHAR(32) NOT NULL,
  webhook_url   VARCHAR(512) NOT NULL,
  secret        VARCHAR(255) NULL,
  status        TINYINT NOT NULL DEFAULT 1,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_rule_channels (
  rule_id    BIGINT UNSIGNED NOT NULL,
  channel_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (rule_id, channel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_templates (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name             VARCHAR(128) NOT NULL,
  content_template TEXT NOT NULL,
  is_default       TINYINT NOT NULL DEFAULT 0,
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_records (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  rule_id             BIGINT UNSIGNED NOT NULL,
  trigger_value       DECIMAL(20,4) NOT NULL,
  threshold           DECIMAL(20,4) NOT NULL,
  level               VARCHAR(16) NOT NULL,
  state               VARCHAR(16) NOT NULL,
  notified_channels   JSON NULL,
  notify_status       VARCHAR(32) NULL,
  notify_detail_json  JSON NULL,
  triggered_at        DATETIME NOT NULL,
  resolved_at         DATETIME NULL,
  INDEX idx_rule_time (rule_id, triggered_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET FOREIGN_KEY_CHECKS=1;
