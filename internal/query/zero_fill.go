package query

import (
	"fmt"
	"time"
)

// maxFillPoints 是零值填充的目标最大节点数。
// 填充密度按此上限自适应：范围越长步长越粗。
const maxFillPoints = 60

// fillStepForRange 根据时间范围长度和查询 bucket 粒度，
// 自适应计算零值填充步长，使填充节点数不超过 maxFillPoints。
// 规则：范围越长步长越粗；1s bucket 最小步长 15s；1d/1w 保持自然步长。
func fillStepForRange(d time.Duration, bucket string) time.Duration {
	// 1d 和 1w bucket 的步长固定为自然步长（天/周），不需要稀释
	natural := bucketStep(bucket)
	if bucket == "1d" || bucket == "1w" {
		return natural
	}

	// 计算按自然步长会产生多少节点
	naturalCount := int(d/natural) + 1
	if naturalCount <= maxFillPoints {
		// 自然步长已够稀疏，直接使用
		// 但 1s bucket 下最小步长为 15s
		if natural < 15*time.Second {
			return 15 * time.Second
		}
		return natural
	}

	// 自然步长过密，按 maxFillPoints 计算稀释步长
	// 步长 = 范围长度 / 目标节点数，然后向上对齐到 bucket 的整数倍
	rawStep := d / time.Duration(maxFillPoints)
	// 对齐到自然步长的整数倍（保证填充 x 值与 SQL DATE_FORMAT 输出可匹配）
	// 向上取整：remainder > 0 时 multiple 加 1，确保节点数 ≤ maxFillPoints
	multiple := rawStep / natural
	if rawStep%natural > 0 {
		multiple++
	}
	step := natural * multiple

	// 1s bucket 下最小步长 15s
	if bucket == "1s" && step < 15*time.Second {
		step = 15 * time.Second
	}
	return step
}

// truncateToFillStep 将时间 t 向下对齐到填充步长边界。
// 对于自适应步长（非自然步长的整数倍），使用 Truncate。
// 对于 1d/1w bucket，使用本地日期截断。
func truncateToFillStep(t time.Time, bucket string, step time.Duration) time.Time {
	switch bucket {
	case "1d":
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	case "1w":
		y, w := t.ISOWeek()
		return isoWeekStart(y, w, t.Location())
	default:
		return t.Truncate(step)
	}
}

// bucketStep 返回 bucket 对应的时间步长。
func bucketStep(bucket string) time.Duration {
	switch bucket {
	case "1s":
		return time.Second
	case "1m":
		return time.Minute
	case "1h":
		return time.Hour
	case "1d":
		return 24 * time.Hour
	case "1w":
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

// isoWeekStart 返回指定 ISO 周的周一 00:00:00。
func isoWeekStart(year, week int, loc *time.Location) time.Time {
	t := time.Date(year, 1, 4, 0, 0, 0, 0, loc)
	_, w := t.ISOWeek()
	diff := week - w
	return t.AddDate(0, 0, diff*7).Truncate(24 * time.Hour)
}

// formatBucketKey 将时间格式化为与 MySQL DATE_FORMAT 输出一致的字符串，
// 确保与 SQL 查询结果的 x 字段可精确匹配。
// bucket 决定格式精度，而非填充步长。
func formatBucketKey(t time.Time, bucket string) string {
	switch bucket {
	case "1s":
		return t.Format("2006-01-02 15:04:05")
	case "1m":
		return t.Truncate(time.Minute).Format("2006-01-02 15:04:05")
	case "1h":
		return t.Truncate(time.Hour).Format("2006-01-02 15:04:05")
	case "1d":
		return t.Format("2006-01-02")
	case "1w":
		y, w := t.ISOWeek()
		return fmt.Sprintf("%04d-%02d", y, w)
	default:
		return t.Format("2006-01-02")
	}
}

// ZeroFill 对趋势图查询结果填充缺失时间桶的零值。
// 仅当 cfg.XDimension 不为空且 cfg.GroupBy 为空时生效（饼图等分组场景不处理）。
// 填充密度根据时间范围自适应：范围越长步长越粗，目标约 maxFillPoints 个节点。
func ZeroFill(cfg Config, rows []map[string]interface{}) []map[string]interface{} {
	if cfg.XDimension == nil || len(cfg.GroupBy) > 0 || cfg.TimeRange == nil {
		return rows
	}

	startTime, endTime, ok := resolveTimeRange(cfg.TimeRange)
	if !ok {
		return rows
	}

	bucket := cfg.XDimension.Bucket
	d := endTime.Sub(startTime)
	step := fillStepForRange(d, bucket)

	// 对齐起始时间到填充步长边界
	startTime = truncateToFillStep(startTime, bucket, step)

	// 生成填充时间点序列
	keys := make([]string, 0, maxFillPoints+1)
	for t := startTime; !t.After(endTime); t = t.Add(step) {
		keys = append(keys, formatBucketKey(t, bucket))
	}

	// 构建已有行索引
	rowByX := make(map[string]map[string]interface{}, len(rows))
	for _, r := range rows {
		if x, ok := r["x"]; ok {
			rowByX[fmt.Sprint(x)] = r
		}
	}

	// 合并：已有行保留，缺失行补零
	out := make([]map[string]interface{}, 0, len(keys)+len(rows))
	for _, k := range keys {
		if r, ok := rowByX[k]; ok {
			out = append(out, r)
		} else {
			out = append(out, map[string]interface{}{
				"x":     k,
				"value": int64(0),
			})
		}
	}
	return out
}

// resolveTimeRange 从 TimeRange 解析出起止时间。
func resolveTimeRange(tr *TimeRange) (time.Time, time.Time, bool) {
	switch tr.Type {
	case "relative":
		d, err := ParseDuration(tr.Value)
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		endTime := time.Now()
		return endTime.Add(-d), endTime, true
	case "absolute":
		var start, end time.Time
		var err error
		if tr.Start != "" {
			start, err = ParseAbsoluteTime(tr.Start)
			if err != nil {
				return time.Time{}, time.Time{}, false
			}
		}
		if tr.End != "" {
			end, err = ParseAbsoluteTime(tr.End)
			if err != nil {
				return time.Time{}, time.Time{}, false
			}
		}
		if start.IsZero() || end.IsZero() {
			return time.Time{}, time.Time{}, false
		}
		return start, end, true
	default:
		return time.Time{}, time.Time{}, false
	}
}

// ZeroFill1m 对 1m 粒度预聚合数据逐分钟零填充（无自适应稀释）。
// 用于 pre-agg 查询：始终填充时间范围内的每个分钟，确保图表节点连续无间断。
// 跳过 group_by 场景（多维度组合的零填充过于复杂，ECharts connectNulls 处理间断）。
func ZeroFill1m(cfg Config, rows []map[string]interface{}) []map[string]interface{} {
	if cfg.XDimension == nil || len(cfg.GroupBy) > 0 || cfg.TimeRange == nil {
		return rows
	}

	startTime, endTime, ok := resolveTimeRange(cfg.TimeRange)
	if !ok {
		return rows
	}

	// 向下对齐到整分钟
	startTime = startTime.Truncate(time.Minute)

	// 逐分钟生成填充时间点
	const fillFormat = "2006-01-02 15:04:05"
	keys := make([]string, 0, int(endTime.Sub(startTime)/time.Minute)+2)
	for t := startTime; !t.After(endTime); t = t.Add(time.Minute) {
		keys = append(keys, t.Format(fillFormat))
	}

	// 构建已有行索引（x 为 DATE_FORMAT 输出的 datetime 字符串）
	rowByX := make(map[string]map[string]interface{}, len(rows))
	for _, r := range rows {
		if x, ok := r["x"]; ok {
			rowByX[fmt.Sprint(x)] = r
		}
	}

	// 合并：已有行保留，缺失分钟补零
	out := make([]map[string]interface{}, 0, len(keys)+len(rows))
	for _, k := range keys {
		if r, ok := rowByX[k]; ok {
			out = append(out, r)
		} else {
			out = append(out, map[string]interface{}{
				"x":     k,
				"value": int64(0),
			})
		}
	}
	return out
}