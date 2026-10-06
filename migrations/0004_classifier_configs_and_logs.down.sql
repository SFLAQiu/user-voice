SET NAMES utf8mb4;

DROP TABLE IF EXISTS classification_logs;
DROP TABLE IF EXISTS classifier_configs;
ALTER TABLE feedbacks DROP INDEX idx_business_module;
ALTER TABLE feedbacks DROP COLUMN business_module;