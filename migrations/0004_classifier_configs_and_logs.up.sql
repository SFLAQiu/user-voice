SET NAMES utf8mb4;

-- Classification prompt / category / business_module configs
CREATE TABLE IF NOT EXISTS classifier_configs (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  type        VARCHAR(32) NOT NULL COMMENT 'prompt | category | business_module',
  name        VARCHAR(128) NOT NULL,
  content     TEXT NOT NULL COMMENT 'prompt模板(type=prompt) 或描述(type=category/module)',
  sort_order  INT NOT NULL DEFAULT 0,
  enabled     TINYINT NOT NULL DEFAULT 1,
  is_default  TINYINT NOT NULL DEFAULT 0 COMMENT '仅prompt类型: 默认使用的模板',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_type_name (type, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Classification attempt logs
CREATE TABLE IF NOT EXISTS classification_logs (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  feedback_id       BIGINT UNSIGNED NOT NULL,
  attempt           INT NOT NULL DEFAULT 1 COMMENT '第几次尝试',
  prompt_config_id  BIGINT UNSIGNED NULL COMMENT '使用的classifier_configs.id',
  prompt_used       TEXT NULL COMMENT '实际发送给LLM的完整prompt',
  llm_raw_response  TEXT NULL COMMENT 'LLM原始输出',
  parsed_result     JSON NULL COMMENT '解析后的JSON结果',
  category_result   VARCHAR(64) NULL,
  module_result     VARCHAR(64) NULL,
  sentiment_result  VARCHAR(16) NULL,
  confidence_result DECIMAL(4,3) NULL,
  error_message     TEXT NULL,
  status            TINYINT NOT NULL COMMENT '0待处理 1成功 2失败',
  duration_ms       INT NULL COMMENT 'LLM调用耗时(ms)',
  created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_feedback_id (feedback_id),
  INDEX idx_status_time (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Add business_module column to feedbacks
ALTER TABLE feedbacks ADD COLUMN business_module VARCHAR(64) NULL COMMENT '归属业务模块' AFTER category;
ALTER TABLE feedbacks ADD INDEX idx_business_module (business_module);

-- Seed data: default prompt template
INSERT INTO classifier_configs (type, name, content, sort_order, enabled, is_default) VALUES
('prompt', '默认分类提示词', '你是产品反馈分类助手。请根据反馈内容输出严格 JSON，不要输出任何其他文字：\n{\n  \"category\": \"<分类>\",\n  \"business_module\": \"<业务模块>\",\n  \"confidence\": <0到1的小数>,\n  \"sentiment\": \"positive|neutral|negative\"\n}\n可选分类：{categories}\n可选业务模块：{business_modules}\n\n反馈内容：{content}', 0, 1, 1);

-- Seed data: categories (from current config.yaml)
INSERT INTO classifier_configs (type, name, content, sort_order, enabled, is_default) VALUES
('category', '功能问题', '', 1, 1, 0),
('category', '体验问题', '', 2, 1, 0),
('category', '内容问题', '', 3, 1, 0),
('category', '商业化问题', '', 4, 1, 0),
('category', '数据问题', '', 5, 1, 0),
('category', '兼容性问题', '', 6, 1, 0),
('category', '账号问题', '', 7, 1, 0),
('category', '建议需求', '', 8, 1, 0),
('category', '其他', '', 9, 1, 0);

-- Seed data: business modules（通用 demo 数据，可按实际业务在「分类配置」页面调整）
INSERT INTO classifier_configs (type, name, content, sort_order, enabled, is_default) VALUES
('business_module', '账号体系', '注册登录、账号安全、资料管理等', 1, 1, 0),
('business_module', '消息通知', '推送、站内信、订阅提醒等', 2, 1, 0),
('business_module', '支付订单', '支付流程、订单管理、退款等', 3, 1, 0),
('business_module', '内容浏览', '内容展示、搜索、推荐等', 4, 1, 0),
('business_module', '上传存储', '文件上传、图片视频存储等', 5, 1, 0),
('business_module', '数据统计', '报表、数据分析、导出等', 6, 1, 0),
('business_module', '社交互动', '评论、点赞、分享等', 7, 1, 0),
('business_module', '设置中心', '偏好设置、隐私设置、通用配置等', 8, 1, 0),
('business_module', '其他', '暂无法归类的模块', 9, 1, 0);