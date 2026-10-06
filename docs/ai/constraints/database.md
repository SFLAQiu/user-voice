# 数据库 DDL 与 GORM 规范

## 表和字段必须包含 COMMENT

所有 CREATE TABLE 语句必须为每个字段和表本身添加 `COMMENT` 说明。GORM model 中必须使用 `comment` tag 为字段添加注释。

### SQL 示例

```sql
CREATE TABLE `llm_providers` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(100) NOT NULL COMMENT '提供商名称',
  ...
) COMMENT='LLM服务提供商配置表';
```

### Go 示例

```go
type LLMProvider struct {
    ID   uint64 `gorm:"primaryKey;autoIncrement;comment:主键ID"`
    Name string `gorm:"size:100;not null;comment:提供商名称"`
}

func (LLMProvider) TableName() string { return "llm_providers" }
// 表注释通过 migration SQL 的 COMMENT 设置
```
