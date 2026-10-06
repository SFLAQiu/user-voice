package query

// Package query 增加预聚合查询路由。
// BuildSmart/BuildSmartFromConfig 对查询自动路由：
//   count metric + XDimension → BuildPreAgg（时间序列，按桶分组）
//   sum metric + XDimension → BuildPreAggTotal（单值汇总，不按时间分组）

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// supportedBuckets 预聚合查询支持的粒度集合。
var supportedBuckets = map[string]bool{"1m": true, "1h": true, "1d": true}

// preAggAllowedFields 预聚合查询允许引用的 metric_time_buckets 列。
var preAggAllowedFields = map[string]string{
	"app_id":          "app_id",
	"platform_id":     "platform_id",
	"app_version":     "app_version",
	"category":        "category",
	"business_module": "business_module",
	"sentiment":       "sentiment",
	"category_status": "category_status",
}

// BuildSmart 解析 JSON 配置后自动路由到最佳查询路径。
func BuildSmart(raw json.RawMessage) (BuildResult, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return BuildResult{}, fmt.Errorf("parse query config: %w", err)
	}
	return BuildSmartFromConfig(cfg)
}

// BuildSmartFromConfig 根据 Config 自动路由查询路径。
// count + XDimension(1m/1h/1d) → BuildPreAgg (时间序列)
// sum + XDimension(1m) → BuildPreAggTotal (单值汇总)
// bucket 不在支持范围 → 明确报错，避免静默降级导致数据不一致
func BuildSmartFromConfig(cfg Config) (BuildResult, error) {
	// sum metric → 单值汇总查询
	if canUsePreAggTotal(cfg) {
		return BuildPreAggTotal(cfg)
	}
	if canUsePreAgg(cfg) {
		return BuildPreAgg(cfg)
	}
	// count metric + x_dimension 存在但 bucket 不支持 → 报错而非静默 fallback
	if cfg.XDimension != nil && (cfg.Metric == "count" || cfg.Metric == "") {
		if !supportedBuckets[cfg.XDimension.Bucket] {
			return BuildResult{}, fmt.Errorf("unsupported bucket: %s (only 1m, 1h, 1d are supported)", cfg.XDimension.Bucket)
		}
	}
	return build(cfg)
}

// canUsePreAgg 判断是否路由到预聚合时间序列查询。
// 条件：metric=count（含空），x_dimension 存在且 bucket 为支持的粒度（1m/1h/1d）。
func canUsePreAgg(cfg Config) bool {
	if cfg.Metric != "count" && cfg.Metric != "" {
		return false
	}
	if cfg.XDimension == nil {
		return false
	}
	return supportedBuckets[cfg.XDimension.Bucket]
}

// canUsePreAggTotal 判断是否路由到预聚合单值汇总查询。
// 条件：metric=sum，x_dimension 存在且 bucket 为支持的粒度（1m/1h/1d）。
func canUsePreAggTotal(cfg Config) bool {
	if cfg.Metric != "sum" {
		return false
	}
	if cfg.XDimension == nil {
		return false
	}
	return supportedBuckets[cfg.XDimension.Bucket]
}

// BuildPreAggTotal 从 metric_time_buckets 预聚合表构建单值汇总查询。
// 对指定时间范围和粒度求 SUM(feedback_count)，不按时间分组，返回单个总数。
func BuildPreAggTotal(cfg Config) (BuildResult, error) {
	granularity := resolveGranularity(cfg)

	var args []interface{}

	// --- SELECT ---
	sql := "SELECT SUM(feedback_count) AS value FROM metric_time_buckets"

	// --- WHERE ---
	where, wargs, err := buildPreAggWhere(cfg, granularity)
	if err != nil {
		return BuildResult{}, err
	}
	if where != "" {
		sql += " WHERE " + where
		args = append(args, wargs...)
	}

	return BuildResult{SQL: sql, Args: args, PreAgg: true, Bucket: granularity}, nil
}

// BuildPreAgg 从 metric_time_buckets 预聚合表构建查询。
// 根据 cfg.XDimension.Bucket 选择 granularity（1m/1h/1d），返回对应粒度的数据。
// DATE_FORMAT 按粒度选择合适的输出格式，前端直接绘制图表。
func BuildPreAgg(cfg Config) (BuildResult, error) {
	// 确定 granularity：从 x_dimension.bucket 读取
	granularity := resolveGranularity(cfg)

	// DATE_FORMAT 按粒度选择：1m/1h → datetime，1d → date only
	var dateFormat string
	switch granularity {
	case "1d":
		dateFormat = "'%Y-%m-%d'"
	default:
		dateFormat = "'%Y-%m-%d %H:%i:%s'"
	}

	var args []interface{}

	// --- SELECT ---
	selectParts := []string{
		"SUM(feedback_count) AS value",
		fmt.Sprintf("DATE_FORMAT(bucket_time, %s) AS x", dateFormat),
	}

	// group_by 维度列
	for _, g := range cfg.GroupBy {
		col, ok := preAggAllowedFields[g]
		if !ok {
			return build(cfg)
		}
		selectParts = append(selectParts, col+" AS "+col)
	}

	// --- FROM ---
	sql := "SELECT " + strings.Join(selectParts, ", ") + " FROM metric_time_buckets"

	// --- WHERE ---
	where, wargs, err := buildPreAggWhere(cfg, granularity)
	if err != nil {
		return BuildResult{}, err
	}
	if where != "" {
		sql += " WHERE " + where
		args = append(args, wargs...)
	}

	// --- GROUP BY ---
	// 按 bucket_time + 维度列分组（粒度由 granularity 决定）
	groupParts := []string{"bucket_time"}
	for _, g := range cfg.GroupBy {
		col, ok := preAggAllowedFields[g]
		if !ok {
			return build(cfg)
		}
		groupParts = append(groupParts, col)
	}
	sql += " GROUP BY " + strings.Join(groupParts, ", ")

	// --- ORDER BY ---
	sql += " ORDER BY bucket_time ASC"

	// --- LIMIT ---
	// 1m 数据可能较多，提高上限；前端按需聚合
	limit := cfg.Limit
	if limit <= 0 || limit > 50000 {
		limit = 50000
	}
	sql += fmt.Sprintf(" LIMIT %d", limit)

	return BuildResult{SQL: sql, Args: args, PreAgg: true, Bucket: granularity}, nil
}

// resolveGranularity 从 Config 的 x_dimension.bucket 确定 granularity，默认 1m。
func resolveGranularity(cfg Config) string {
	if cfg.XDimension != nil && supportedBuckets[cfg.XDimension.Bucket] {
		return cfg.XDimension.Bucket
	}
	return "1m"
}

// truncateToGranularity 将时间截断到与查询粒度匹配的精度。
// 1m → 截断到分钟，1h → 截断到小时，1d → 截断到本地时区午夜。
func truncateToGranularity(t time.Time, granularity string) time.Time {
	switch granularity {
	case "1h":
		return t.Truncate(time.Hour)
	case "1d":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	default:
		return t.Truncate(time.Minute)
	}
}

// buildPreAggWhere 构建预聚合表的 WHERE 条件。
func buildPreAggWhere(cfg Config, granularity string) (string, []interface{}, error) {
	var clauses []string
	var args []interface{}

	// 固定条件：只读指定粒度基础行 + 当前活跃维度版本
	clauses = append(clauses, "granularity = ?")
	args = append(args, granularity)
	clauses = append(clauses, "dimension_schema_version = (SELECT active_dimension_version FROM metric_bucket_configs WHERE id = 1)")

	// 时间范围
	if cfg.TimeRange != nil {
		tr := cfg.TimeRange
		switch tr.Type {
		case "relative":
			d, err := ParseDuration(tr.Value)
			if err != nil {
				return "", nil, fmt.Errorf("invalid time_range value: %w", err)
			}
			clauses = append(clauses, "bucket_time >= ?")
			args = append(args, truncateToGranularity(time.Now().Add(-d), granularity))
		case "absolute":
			if tr.Start != "" {
				startTime, err := ParseAbsoluteTime(tr.Start)
				if err != nil {
					return "", nil, fmt.Errorf("invalid time_range_start: %w", err)
				}
				clauses = append(clauses, "bucket_time >= ?")
				args = append(args, truncateToGranularity(startTime, granularity))
			}
			if tr.End != "" {
				endTime, err := ParseAbsoluteTime(tr.End)
				if err != nil {
					return "", nil, fmt.Errorf("invalid time_range_end: %w", err)
				}
				clauses = append(clauses, "bucket_time <= ?")
				args = append(args, truncateToGranularity(endTime, granularity))
			}
		}
	}

	// 维度过滤
	hasCategoryFilter := false
	hasSentimentFilter := false
	for _, f := range cfg.Filters {
		col, ok := preAggAllowedFields[f.Field]
		if !ok {
			return "", nil, fmt.Errorf("field %s not available in pre-aggregation", f.Field)
		}
		if f.Field == "category" || f.Field == "business_module" {
			hasCategoryFilter = true
		}
		if f.Field == "sentiment" {
			hasSentimentFilter = true
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
		treeSQL, treeArgs, err := BuildFilterNodeSQLWithFields(*cfg.PanelFilterTree, preAggAllowedFields)
		if err != nil {
			return "", nil, fmt.Errorf("build panel filter tree: %w", err)
		}
		if treeSQL != "" {
			clauses = append(clauses, treeSQL)
			args = append(args, treeArgs...)
		}
		trackPreAggFilters(*cfg.PanelFilterTree, &hasCategoryFilter, &hasSentimentFilter)
	}

	// FilterTree (dashboard-level, 使用 preAggAllowedFields 校验)
	if cfg.FilterTree != nil {
		treeSQL, treeArgs, err := BuildFilterNodeSQLWithFields(*cfg.FilterTree, preAggAllowedFields)
		if err != nil {
			return "", nil, fmt.Errorf("build dashboard filter tree: %w", err)
		}
		if treeSQL != "" {
			clauses = append(clauses, treeSQL)
			args = append(args, treeArgs...)
		}
		trackPreAggFilters(*cfg.FilterTree, &hasCategoryFilter, &hasSentimentFilter)
	}

	// category_status 过滤策略
	if hasCategoryFilter || hasSentimentFilter {
		clauses = append(clauses, "category_status IN (1, 3)")
	}

	return strings.Join(clauses, " AND "), args, nil
}

// ReduceResult 是 BuildPreAggReduce 的返回结果：单条触发桶。
type ReduceResult struct {
	Value      float64
	BucketTime time.Time
}

// BuildPreAggReduce 直接用 SQL 选出 reduce_mode 对应的单条触发桶。
// 完全绕过应用层遍历/零填充/类型转换，一次 SQL 拿到 trigger_value + metric_timestamp。
//
// max: 窗口内 SUM(feedback_count) 最大的桶（并列取最新桶）
// last: 窗口内最近一条有数据的桶
//
// granularity: 从 cfg.XDimension.Bucket 读取，默认 1m。
// 返回 nil 表示窗口内无数据。
func BuildPreAggReduce(cfg Config, mode string) (string, []interface{}, error) {
	granularity := resolveGranularity(cfg)

	where, args, err := buildPreAggWhere(cfg, granularity)
	if err != nil {
		return "", nil, err
	}

	// 基础子查询：按 bucket_time 聚合，排除空行
	baseWhere := where
	if baseWhere != "" {
		baseWhere += " AND feedback_count > 0"
	} else {
		baseWhere = "feedback_count > 0"
	}

	var orderBy string
	switch mode {
	case "last":
		orderBy = "bucket_time DESC"
	default: // max
		orderBy = "agg_value DESC, bucket_time DESC"
	}

	sql := fmt.Sprintf(`SELECT SUM(feedback_count) AS agg_value, bucket_time
FROM metric_time_buckets
WHERE %s
GROUP BY bucket_time
HAVING SUM(feedback_count) > 0
ORDER BY %s
LIMIT 1`, baseWhere, orderBy)

	return sql, args, nil
}
