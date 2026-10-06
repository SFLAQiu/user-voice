// Package mysql 实现从 MySQL 表增量拉取反馈数据的插件。
package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/datasource"
)

const Type = "mysql"

// Config 为 data_sources.config_json 中持久化的插件配置。
type Config struct {
	Host         string            `json:"host"`
	Port         int               `json:"port"`
	Username     string            `json:"username"`
	Password     string            `json:"password"` // 保存时加密为 ENC:...
	Database     string            `json:"database"`
	Table        string            `json:"table"`
	TimeField    string            `json:"time_field"`    // 增量游标时间列
	BatchLimit   int               `json:"batch_limit"`   // 单次同步最大条数，默认 1000
	FieldMapping map[string]string `json:"field_mapping"` // 系统字段 → 源表列名
	Enums        map[string]map[string]string `json:"enums"`
}

// Plugin 满足 datasource.Plugin 契约。
type Plugin struct {
	cipher *crypto.Cipher
}

func New(c *crypto.Cipher) *Plugin { return &Plugin{cipher: c} }

func (p *Plugin) Type() string { return Type }

func (p *Plugin) Validate(raw json.RawMessage) error {
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Database == "" {
		return errors.New("database is required")
	}
	if c.Table == "" {
		return errors.New("table is required")
	}
	if c.TimeField == "" {
		return errors.New("time_field is required")
	}
	if c.FieldMapping["original_id"] == "" {
		return errors.New("field_mapping.original_id is required")
	}
	if c.FieldMapping["content"] == "" {
		return errors.New("field_mapping.content is required")
	}
	if c.FieldMapping["original_created_at"] == "" {
		return errors.New("field_mapping.original_created_at is required")
	}
	return nil
}

func (p *Plugin) Pull(ctx context.Context, raw json.RawMessage, cursor *datasource.Cursor,
	exists func(string) (bool, error)) (datasource.PullResult, error) {

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return datasource.PullResult{}, err
	}
	if cfg.Port == 0 {
		cfg.Port = 3306
	}
	if cfg.BatchLimit <= 0 {
		cfg.BatchLimit = 1000
	}

	pwd, err := p.cipher.Decrypt(cfg.Password)
	if err != nil {
		return datasource.PullResult{}, fmt.Errorf("decrypt password: %w", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&timeout=10s",
		cfg.Username, pwd, cfg.Host, cfg.Port, cfg.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return datasource.PullResult{}, fmt.Errorf("connect mysql: %w", err)
	}
	sqldb, _ := db.DB()
	defer sqldb.Close()

	// 时间游标：只拉取游标之后的记录，按时间升序，限制单批数量
	tableIdent, err := quoteIdent(cfg.Table)
	if err != nil {
		return datasource.PullResult{}, err
	}
	query := db.Table(tableIdent)
	if cursor != nil && cursor.LastCreatedAt != nil {
		query = query.Where(fmt.Sprintf("%s > ?", mustQuote(cfg.TimeField)), *cursor.LastCreatedAt)
	}

	rows := make([]map[string]any, 0, cfg.BatchLimit)
	if err := query.
		Order(mustQuote(cfg.TimeField) + " ASC").
		Limit(cfg.BatchLimit).
		Find(&rows).Error; err != nil {
		return datasource.PullResult{}, fmt.Errorf("query %s: %w", cfg.Table, err)
	}

	items := make([]datasource.FeedbackItem, 0, len(rows))
	next := &datasource.Cursor{Key: "default"}
	if cursor != nil && cursor.LastCreatedAt != nil {
		t := *cursor.LastCreatedAt
		next.LastCreatedAt = &t
	}
	for _, row := range rows {
		it, err := rowToItem(row, cfg.FieldMapping, cfg.Enums)
		if err != nil {
			return datasource.PullResult{}, err
		}
		items = append(items, it)
		// 游标推进到本批最新时间
		if next.LastCreatedAt == nil || it.OriginalCreatedAt.After(*next.LastCreatedAt) {
			t := it.OriginalCreatedAt
			next.LastCreatedAt = &t
		}
	}
	// 本批条数不足上限说明已追平，游标可安全保存
	return datasource.PullResult{Items: items, NextCursor: next}, nil
}

func rowToItem(row map[string]any, mapping map[string]string, enums map[string]map[string]string) (datasource.FeedbackItem, error) {
	get := func(key string) any {
		col, ok := mapping[key]
		if !ok || col == "" {
			return nil
		}
		return row[col]
	}
	enumResolve := func(enumKey string, v any) string {
		val := toString(v)
		if e, ok := enums[enumKey]; ok {
			if label, ok := e[val]; ok {
				return label
			}
		}
		return val
	}

	it := datasource.FeedbackItem{
		OriginalID: toString(get("original_id")),
		AppID:      toInt(get("app_id")),
		Platform:   enumResolve("platform_id", get("platform_id")),
		UserID:     toString(get("user_id")),
		UserName:   toString(get("user_name")),
		Content:    toString(get("content")),
		PhoneModel: toString(get("phone_model")),
		AppVersion: toString(get("app_version")),
		ChannelID:  toString(get("channel_id")),
		QQ:         toString(get("qq")),
		FileURL:    toString(get("file_url")),
		Raw:        row,
	}
	if appName := enumResolve("app_id", get("app_id")); appName != "" && it.AppID != 0 {
		it.AppName = appName
	}
	if pid := toIntPtr(get("platform_id")); pid != nil {
		it.PlatformID = pid
	}
	if um := toIntPtr(get("user_mode")); um != nil {
		it.UserMode = um
	}
	for _, key := range []string{"images", "videos"} {
		if v := toString(get(key)); v != "" {
			var arr []string
			if err := json.Unmarshal([]byte(v), &arr); err == nil && len(arr) > 0 {
				if key == "images" {
					it.Images = arr
				} else {
					it.Videos = arr
				}
			}
		}
	}
	switch t := get("original_created_at").(type) {
	case time.Time:
		it.OriginalCreatedAt = t
	case *time.Time:
		if t != nil {
			it.OriginalCreatedAt = *t
		} else {
			it.OriginalCreatedAt = time.Now()
		}
	case string:
		for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02"} {
			if ts, err := time.ParseInLocation(layout, t, time.Local); err == nil {
				it.OriginalCreatedAt = ts
				break
			}
		}
	}
	if it.OriginalCreatedAt.IsZero() {
		it.OriginalCreatedAt = time.Now()
	}
	return it, nil
}

// mustQuote 校验并包裹标识符，失败 panic（仅在插件内部对已校验字段使用）。
func mustQuote(name string) string {
	q, err := quoteIdent(name)
	if err != nil {
		panic(err)
	}
	return q
}

// quoteIdent 防止表名/列名注入：仅允许字母数字下划线与点（库名.表名），含其他字符直接报错。
func quoteIdent(name string) (string, error) {
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.') {
			return "", fmt.Errorf("invalid identifier %q", name)
		}
	}
	if name == "" {
		return "", errors.New("empty identifier")
	}
	return "`" + name + "`", nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func toInt(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return int(t)
	case int:
		return t
	case uint64:
		return int(t)
	case float64:
		return int(t)
	case []byte:
		var n int
		fmt.Sscanf(string(t), "%d", &n)
		return n
	default:
		return 0
	}
}

func toIntPtr(v any) *int {
	if v == nil {
		return nil
	}
	if n := toInt(v); n != 0 {
		return &n
	}
	return nil
}
