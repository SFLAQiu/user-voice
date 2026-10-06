package query

// Package query builds safe, parameterized SQL for panel data queries.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config mirrors the JSON schema for panel query configuration.
// count metric + pre-agg time buckets, or sum metric for number panels.
type Config struct {
	Metric          string      `json:"metric"`            // count | sum
	XDimension      *Dimension  `json:"x_dimension"`       // must be original_created_at for pre-agg
	GroupBy         []string    `json:"group_by"`
	Filters         []Filter    `json:"filters"`           // panel-level, flat AND (legacy)
	PanelFilterTree *FilterNode `json:"panel_filter_tree"` // panel-level, recursive AND/OR
	FilterTree      *FilterNode `json:"filter_tree"`       // dashboard-level, recursive AND/OR
	TimeRange       *TimeRange  `json:"time_range"`
	Limit           int         `json:"limit"`
}

type Dimension struct {
	Field  string `json:"field"`
	Bucket string `json:"bucket"` // 1h | 1d | 1w
}

type Filter struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

type TimeRange struct {
	Type  string `json:"type"`  // relative | absolute
	Value string `json:"value"` // "7d" | "30d" | etc.
	Start string `json:"start"` // ISO8601 for absolute
	End   string `json:"end"`
}

// allowedFields is the whitelist of columns that can be referenced in queries.
// Only includes columns present in metric_time_buckets pre-agg table.
var allowedFields = map[string]string{
	"app_id":              "app_id",
	"platform_id":         "platform_id",
	"app_version":         "app_version",
	"category":            "category",
	"business_module":     "business_module",
	"sentiment":           "sentiment",
	"category_status":     "category_status",
	"original_created_at": "original_created_at",
}

var allowedOps = map[string]string{
	"=": "=", "!=": "!=", ">": ">", ">=": ">=", "<": "<", "<=": "<=",
	"in": "IN", "not in": "NOT IN", "like": "LIKE",
}

var bucketFormats = map[string]string{
	"1m": "%Y-%m-%d %H:%i:00",
	"1h": "%Y-%m-%d %H:00:00",
	"1d": "%Y-%m-%d",
}

// BuildResult holds the final SQL and args.
type BuildResult struct {
	SQL    string
	Args   []interface{}
	PreAgg bool   // true if query comes from metric_time_buckets
	Bucket string // granularity of returned data: "1m" for pre-agg, or cfg.XDimension.Bucket for raw
}

// Build constructs a SELECT query for the given panel config.
func Build(raw json.RawMessage) (BuildResult, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return BuildResult{}, fmt.Errorf("parse query config: %w", err)
	}
	return build(cfg)
}

// BuildFromConfig constructs a SELECT query from an already-parsed Config.
func BuildFromConfig(cfg Config) (BuildResult, error) {
	return build(cfg)
}

func build(cfg Config) (BuildResult, error) {
	var args []interface{}

	// --- SELECT clause ---
	metric, err := buildMetric(cfg)
	if err != nil {
		return BuildResult{}, err
	}

	selectParts := []string{metric + " AS value"}

	if cfg.XDimension != nil {
		xCol, ok := allowedFields[cfg.XDimension.Field]
		if !ok {
			return BuildResult{}, fmt.Errorf("unknown x_dimension field: %s", cfg.XDimension.Field)
		}
		format, ok := bucketFormats[cfg.XDimension.Bucket]
		if !ok {
			// default to day
			format = "%Y-%m-%d"
		}
		selectParts = append(selectParts, fmt.Sprintf("DATE_FORMAT(%s, ?) AS x", xCol))
		args = append(args, format)
	}

	for _, g := range cfg.GroupBy {
		col, ok := allowedFields[g]
		if !ok {
			return BuildResult{}, fmt.Errorf("unknown group_by field: %s", g)
		}
		selectParts = append(selectParts, col+" AS "+col)
	}

	// --- FROM ---
	sql := "SELECT " + strings.Join(selectParts, ", ") + " FROM feedbacks"

	// --- WHERE ---
	where, wargs, err := buildWhere(cfg)
	if err != nil {
		return BuildResult{}, err
	}
	if where != "" {
		sql += " WHERE " + where
		args = append(args, wargs...)
	}

	// --- GROUP BY ---
	var groupParts []string
	if cfg.XDimension != nil {
		groupParts = append(groupParts, "x")
	}
	for _, g := range cfg.GroupBy {
		groupParts = append(groupParts, allowedFields[g])
	}
	if len(groupParts) > 0 {
		sql += " GROUP BY " + strings.Join(groupParts, ", ")
	}

	// --- ORDER BY ---
	if cfg.XDimension != nil {
		sql += " ORDER BY x ASC"
	}

	// --- LIMIT ---
	limit := cfg.Limit
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	sql += fmt.Sprintf(" LIMIT %d", limit)

	return BuildResult{SQL: sql, Args: args}, nil
}

func buildMetric(cfg Config) (string, error) {
	switch cfg.Metric {
	case "count", "":
		return "COUNT(*)", nil
	default:
		return "", fmt.Errorf("unsupported metric: %s (only count is supported)", cfg.Metric)
	}
}

// ParseAbsoluteTime 将前端发送的时间字符串解析为本地时区的 time.Time。
// 前端 dayjs.toISOString() 输出 UTC 时间（含 Z 后缀），MySQL driver 使用 loc=Local，
// 需要将 UTC 时刻转换为本地时区表达，确保 driver 正确将 UTC 时刻写入 MySQL。
func ParseAbsoluteTime(s string) (time.Time, error) {
	// 优先尝试 RFC3339（前端 dayjs().toISOString() 格式，含 UTC Z 后缀）
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local(), nil
	}
	// 兼容无时区后缀的 ISO8601 格式，视为本地时区
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t, nil
	}
	// 兼容 MySQL DATETIME 格式，视为本地时区
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", s)
}

func buildWhere(cfg Config) (string, []interface{}, error) {
	var clauses []string
	var args []interface{}

	// Time range filter
	if cfg.TimeRange != nil {
		tr := cfg.TimeRange
		field := "original_created_at"
		switch tr.Type {
		case "relative":
			d, err := ParseDuration(tr.Value)
			if err != nil {
				return "", nil, fmt.Errorf("invalid time_range value: %w", err)
			}
			clauses = append(clauses, field+" >= ?")
			args = append(args, time.Now().Add(-d))
		case "absolute":
			if tr.Start != "" {
				startTime, err := ParseAbsoluteTime(tr.Start)
				if err != nil {
					return "", nil, fmt.Errorf("invalid time_range_start: %w", err)
				}
				clauses = append(clauses, field+" >= ?")
				args = append(args, startTime)
			}
			if tr.End != "" {
				endTime, err := ParseAbsoluteTime(tr.End)
				if err != nil {
					return "", nil, fmt.Errorf("invalid time_range_end: %w", err)
				}
				clauses = append(clauses, field+" <= ?")
				args = append(args, endTime)
			}
		}
	}

	// Flat filters (panel-level, always AND)
	for _, f := range cfg.Filters {
		col, ok := allowedFields[f.Field]
		if !ok {
			return "", nil, fmt.Errorf("unknown filter field: %s", f.Field)
		}
		op, ok := allowedOps[strings.ToLower(f.Op)]
		if !ok {
			return "", nil, fmt.Errorf("unsupported op: %s", f.Op)
		}
		switch strings.ToLower(f.Op) {
		case "in", "not in":
			vals, ok := toSlice(f.Value)
			if !ok {
				return "", nil, fmt.Errorf("filter %s %s expects array value", f.Field, f.Op)
			}
			placeholders := strings.Repeat("?,", len(vals))
			placeholders = placeholders[:len(placeholders)-1]
			clauses = append(clauses, fmt.Sprintf("%s %s (%s)", col, op, placeholders))
			args = append(args, vals...)
		default:
			clauses = append(clauses, col+" "+op+" ?")
			args = append(args, f.Value)
		}
	}

	// PanelFilterTree (panel-level, recursive AND/OR)
	if cfg.PanelFilterTree != nil {
		treeSQL, treeArgs, err := BuildFilterNodeSQL(*cfg.PanelFilterTree)
		if err != nil {
			return "", nil, fmt.Errorf("build panel filter tree: %w", err)
		}
		if treeSQL != "" {
			clauses = append(clauses, treeSQL)
			args = append(args, treeArgs...)
		}
	}

	// FilterTree (dashboard-level, recursive AND/OR)
	if cfg.FilterTree != nil {
		treeSQL, treeArgs, err := BuildFilterNodeSQL(*cfg.FilterTree)
		if err != nil {
			return "", nil, fmt.Errorf("build filter tree: %w", err)
		}
		if treeSQL != "" {
			clauses = append(clauses, treeSQL)
			args = append(args, treeArgs...)
		}
	}

	return strings.Join(clauses, " AND "), args, nil
}

func toSlice(v interface{}) ([]interface{}, bool) {
	switch s := v.(type) {
	case []interface{}:
		return s, true
	case []string:
		out := make([]interface{}, len(s))
		for i, x := range s {
			out[i] = x
		}
		return out, true
	}
	return nil, false
}

func ParseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("too short")
	}
	n := 0
	for _, c := range s[:len(s)-1] {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-numeric")
		}
		n = n*10 + int(c-'0')
	}
	switch s[len(s)-1] {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 's':
		return time.Duration(n) * time.Second, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %c", s[len(s)-1])
	}
}

// FormatDuration 将秒数转为查询引擎可识别的相对时间字符串。
// 如 300 → "5m", 3600 → "1h", 86400 → "1d"。
func FormatDuration(sec int) string {
	if sec <= 0 {
		return "5m"
	}
	switch {
	case sec%86400 == 0:
		return fmt.Sprintf("%dd", sec/86400)
	case sec%3600 == 0:
		return fmt.Sprintf("%dh", sec/3600)
	case sec%60 == 0:
		return fmt.Sprintf("%dm", sec/60)
	default:
		return fmt.Sprintf("%ds", sec)
	}
}

// AutoBucket 根据时间范围长度自动选择最优分桶粒度。
// 规则参考 Grafana：范围越小粒度越细，使数据点控制在合理区间。
func AutoBucket(tr *TimeRange) string {
	if tr == nil {
		return "1d"
	}
	var d time.Duration
	switch tr.Type {
	case "relative":
		parsed, err := ParseDuration(tr.Value)
		if err != nil || parsed <= 0 {
			d = 24 * time.Hour
		} else {
			d = parsed
		}
	case "absolute":
		if tr.Start != "" && tr.End != "" {
			start, sErr := time.Parse(time.RFC3339, tr.Start)
			end, eErr := time.Parse(time.RFC3339, tr.End)
			if sErr == nil && eErr == nil && end.After(start) {
				d = end.Sub(start)
			}
		}
	}
	if d <= 0 {
		d = 24 * time.Hour
	}

	switch {
	case d <= 5*time.Minute:
		return "1s"
	case d <= 30*time.Minute:
		return "1m"
	case d <= 6*time.Hour:
		return "1m"
	case d <= 24*time.Hour:
		return "1h"
	case d <= 7*24*time.Hour:
		return "1h"
	case d <= 30*24*time.Hour:
		return "1d"
	default:
		return "1w"
	}
}

// bucketGoLayouts maps bucket strings to Go time.Parse layouts
// for parsing the x-column output of MySQL DATE_FORMAT.
var bucketGoLayouts = map[string]string{
	"1s": "2006-01-02 15:04:05",
	"1m": "2006-01-02 15:04:05",
	"1h": "2006-01-02 15:04:05",
	"1d": "2006-01-02",
}

// ParseBucketTime 将 DATE_FORMAT 产生的 x 列字符串解析回 *time.Time。
// 1w 桶（%Y-%u 格式如 "2026-23"）手动计算 ISO 周对应周一 00:00。
// x 为空或解析失败返回 (nil, nil)；bucket 不识别返回 error。
func ParseBucketTime(x interface{}, bucket string) (*time.Time, error) {
	if x == nil {
		return nil, nil
	}

	var s string
	switch v := x.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case time.Time:
		t := v.In(time.Local)
		return &t, nil
	default:
		s = fmt.Sprintf("%v", x)
	}
	if s == "" {
		return nil, nil
	}

	if bucket == "1w" {
		return parseISOWeek(s)
	}

	layout, ok := bucketGoLayouts[bucket]
	if !ok {
		return nil, fmt.Errorf("unknown bucket: %s", bucket)
	}
	t, err := time.ParseInLocation(layout, s, time.Local)
	if err != nil {
		return nil, nil // 解析失败不算错误，返回 nil 降级
	}
	return &t, nil
}

// parseISOWeek 解析 "2026-23" 格式的 ISO 周，返回该周周一 00:00 本地时间。
func parseISOWeek(s string) (*time.Time, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return nil, nil
	}
	year, err1 := strconv.Atoi(parts[0])
	week, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	// 找到该年 1 月 1 日后第一个周一，然后加 (week-1)*7 天
	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	daysToMonday := (int(time.Monday) - int(jan1.Weekday()) + 7) % 7
	if daysToMonday == 0 && week > 1 && jan1.Weekday() != time.Monday {
		daysToMonday = 0
	}
	// 1 月 1 日所在周是 ISO 周的第几天（周一=1，周日=7）
	jan1Weekday := int(jan1.Weekday())
	if jan1Weekday == 0 {
		jan1Weekday = 7
	}
	// 首周从第一个周一开始
	firstMonday := jan1.AddDate(0, 0, (8-jan1Weekday)%7)
	result := firstMonday.AddDate(0, 0, (week-1)*7)
	return &result, nil
}
