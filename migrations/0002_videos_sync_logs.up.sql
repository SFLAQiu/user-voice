-- Add videos column to feedbacks (supports video URL collection from external APIs)
ALTER TABLE feedbacks ADD COLUMN videos JSON NULL AFTER images;

-- Create sync_logs table for historical sync execution tracking
CREATE TABLE IF NOT EXISTS sync_logs (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  source_id       BIGINT UNSIGNED NOT NULL,
  status          VARCHAR(32) NOT NULL DEFAULT 'running'   COMMENT 'running|success|failed',
  started_at      DATETIME NOT NULL,
  finished_at     DATETIME NULL,
  fetched_count   INT NOT NULL DEFAULT 0,
  inserted_count  INT NOT NULL DEFAULT 0,
  error_message   TEXT NULL,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_source_time (source_id, started_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;