SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS llm_providers (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name        VARCHAR(128) NOT NULL UNIQUE,
  base_url    VARCHAR(256) NOT NULL,
  api_key     VARCHAR(512) NOT NULL COMMENT 'AES-256-GCM encrypted',
  model       VARCHAR(128) NOT NULL,
  timeout_ms  INT NOT NULL DEFAULT 30000,
  priority    INT NOT NULL DEFAULT 1 COMMENT '1=highest',
  status      TINYINT NOT NULL DEFAULT 1 COMMENT '1=enabled 0=disabled',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS llm_configs (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  type        VARCHAR(32) NOT NULL,
  name        VARCHAR(128) NOT NULL,
  content     TEXT NULL,
  sort_order  INT NOT NULL DEFAULT 0,
  enabled     TINYINT NOT NULL DEFAULT 1,
  is_default  TINYINT NOT NULL DEFAULT 0,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_type_name (type, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE classification_logs ADD COLUMN provider_id BIGINT UNSIGNED NULL COMMENT '使用的 llm_providers.id' AFTER prompt_config_id;
ALTER TABLE classification_logs ADD INDEX idx_provider_id (provider_id);

-- Seed default circuit breaker config
INSERT INTO llm_configs (type, name, content, sort_order, enabled, is_default) VALUES
('circuit_breaker', '默认熔断配置', '{"failure_threshold":5,"failure_rate_threshold":0.5,"window_seconds":60,"cooldown_seconds":30,"min_requests":3}', 0, 1, 1);

-- Seed default retry config
INSERT INTO llm_configs (type, name, content, sort_order, enabled, is_default) VALUES
('retry', '默认重试配置', '{"max_retries":3,"strategy":"priority"}', 0, 1, 1);