-- 权限组管理

CREATE TABLE IF NOT EXISTS permission_groups (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  name        VARCHAR(64) NOT NULL COMMENT '权限组名称',
  description VARCHAR(255) COMMENT '描述',
  permissions JSON NOT NULL COMMENT '权限定义：{page_key: [perm_key, ...]}',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='权限组表';

CREATE TABLE IF NOT EXISTS user_permission_groups (
  user_id   BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  group_id  BIGINT UNSIGNED NOT NULL COMMENT '权限组ID',
  PRIMARY KEY (user_id, group_id),
  INDEX idx_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户权限组关联表';

CREATE TABLE IF NOT EXISTS user_permission_overrides (
  user_id     BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  page_key    VARCHAR(64) NOT NULL COMMENT '页面标识',
  perm_key    VARCHAR(64) NOT NULL COMMENT '权限标识',
  grant_type  TINYINT NOT NULL DEFAULT 1 COMMENT '1=授权, 0=拒绝',
  PRIMARY KEY (user_id, page_key, perm_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户权限覆盖表';
