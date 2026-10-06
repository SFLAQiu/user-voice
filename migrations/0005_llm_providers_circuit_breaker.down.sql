SET NAMES utf8mb4;

DELETE FROM llm_configs WHERE type IN ('circuit_breaker', 'retry');
ALTER TABLE classification_logs DROP INDEX idx_provider_id;
ALTER TABLE classification_logs DROP COLUMN provider_id;
DROP TABLE IF EXISTS llm_configs;
DROP TABLE IF EXISTS llm_providers;