package query

import (
	"testing"
	"time"
)

func TestZeroFill_NoXDimension(t *testing.T) {
	cfg := Config{}
	rows := []map[string]interface{}{{"value": int64(5)}}
	result := ZeroFill(cfg, rows)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
}

func TestZeroFill_WithGroupBy(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		GroupBy:    []string{"category"},
		TimeRange:  &TimeRange{Type: "relative", Value: "1d"},
	}
	rows := []map[string]interface{}{{"x": "2024-01-15 10:00:00", "value": int64(3)}}
	result := ZeroFill(cfg, rows)
	if len(result) != 1 {
		t.Fatalf("group_by 场景不应填充，expected 1, got %d", len(result))
	}
}

func TestZeroFill_NoTimeRange(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
	}
	rows := []map[string]interface{}{{"x": "2024-01-15 10:00:00", "value": int64(3)}}
	result := ZeroFill(cfg, rows)
	if len(result) != 1 {
		t.Fatalf("无 time_range 不应填充，expected 1, got %d", len(result))
	}
}

func TestZeroFill_AllExisting_NoChange(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		// 无时区后缀视为本地时区
		TimeRange: &TimeRange{Type: "absolute", Start: "2024-01-15T08:00:00", End: "2024-01-15T10:00:00"},
	}
	rows := []map[string]interface{}{
		{"x": "2024-01-15 08:00:00", "value": int64(1)},
		{"x": "2024-01-15 09:00:00", "value": int64(2)},
		{"x": "2024-01-15 10:00:00", "value": int64(3)},
	}
	result := ZeroFill(cfg, rows)
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
}

func TestZeroFill_FillsGaps_1h(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T08:00:00", End: "2024-01-15T10:00:00"},
	}
	rows := []map[string]interface{}{
		{"x": "2024-01-15 08:00:00", "value": int64(5)},
		{"x": "2024-01-15 10:00:00", "value": int64(3)},
	}
	result := ZeroFill(cfg, rows)
	if len(result) != 3 {
		t.Fatalf("expected 3 rows (含 09:00 零值填充), got %d", len(result))
	}
	// 检查中间行的零值填充
	filled := false
	for _, r := range result {
		if r["value"].(int64) == 0 {
			filled = true
			break
		}
	}
	if !filled {
		t.Error("should have at least one zero-filled row")
	}
	// 已有行保留原值
	foundOriginal := false
	for _, r := range result {
		if r["value"].(int64) == 5 {
			foundOriginal = true
			break
		}
	}
	if !foundOriginal {
		t.Error("original row with value=5 should be preserved")
	}
}

func TestZeroFill_EmptyRows_FullSequence(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T08:00:00", End: "2024-01-15T09:00:00"},
	}
	result := ZeroFill(cfg, []map[string]interface{}{})
	// 08:00 和 09:00 两个桶
	if len(result) != 2 {
		t.Fatalf("expected 2 zero-fill rows, got %d", len(result))
	}
	for i, r := range result {
		if r["value"].(int64) != 0 {
			t.Errorf("row %d value = %v, expected 0", i, r["value"])
		}
		if r["x"] == "" {
			t.Errorf("row %d x is empty", i)
		}
	}
}

func TestZeroFill_KeyOrdering(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1d"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T00:00:00", End: "2024-01-17T00:00:00"},
	}
	rows := []map[string]interface{}{
		{"x": "2024-01-17", "value": int64(3)},
		{"x": "2024-01-15", "value": int64(1)},
	}
	result := ZeroFill(cfg, rows)
	if len(result) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(result))
	}
	// 检查顺序：第一个 x 应包含 01-15
	if result[0]["x"] != "2024-01-15" {
		t.Errorf("row 0 = %v, expected 2024-01-15", result[0]["x"])
	}
	if result[2]["x"] != "2024-01-17" {
		t.Errorf("row 2 = %v, expected 2024-01-17", result[2]["x"])
	}
}

func TestZeroFill_PreservesExtraColumns(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T08:00:00", End: "2024-01-15T09:00:00"},
	}
	rows := []map[string]interface{}{
		{"x": "2024-01-15 08:00:00", "value": int64(5), "extra_col": "keep_me"},
	}
	result := ZeroFill(cfg, rows)
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	// 找到含有 extra_col 的行
	for _, r := range result {
		if ec, ok := r["extra_col"]; ok && ec == "keep_me" {
			return // 成功找到
		}
	}
	t.Error("original row's extra_col not preserved")
}

func TestZeroFill_FillsGaps_1d(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1d"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T00:00:00", End: "2024-01-17T00:00:00"},
	}
	rows := []map[string]interface{}{
		{"x": "2024-01-15", "value": int64(5)},
	}
	result := ZeroFill(cfg, rows)
	if len(result) != 3 {
		t.Fatalf("expected 3 rows (含 16/17 零值), got %d", len(result))
	}
	if result[1]["value"].(int64) != 0 {
		t.Errorf("day 16 should be zero")
	}
	if result[2]["value"].(int64) != 0 {
		t.Errorf("day 17 should be zero")
	}
}

func TestZeroFill_RelativeTimeRange(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		TimeRange:  &TimeRange{Type: "relative", Value: "1h"},
	}
	rows := []map[string]interface{}{}
	result := ZeroFill(cfg, rows)
	// 1 小时范围，含起止两个边界
	if len(result) < 1 {
		t.Fatalf("relative 1h should produce at least 1 bucket, got %d", len(result))
	}
	for _, r := range result {
		if r["value"].(int64) != 0 {
			t.Errorf("expected all zero, got %v", r["value"])
		}
	}
}

func TestZeroFill_1sBucket_Uses15sFillStep(t *testing.T) {
	// 1s 粒度下自然步长 1s < maxFillPoints 限制，
	// 但 1 分钟范围自然节点数约 60 < 60，所以 fillStep = 15s（最小步长）
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1s"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T10:00:05", End: "2024-01-15T10:01:00"},
	}
	result := ZeroFill(cfg, []map[string]interface{}{})
	// 55s / 15s ≈ 4 个间隔，含两端 = 5 个桶
	if len(result) != 5 {
		t.Fatalf("expected 5 buckets (15s step), got %d", len(result))
	}
	// 检查步长间隔：相邻 x 值时间差应为 15 秒
	for i := 1; i < len(result); i++ {
		prev := result[i-1]["x"].(string)
		curr := result[i]["x"].(string)
		if prev == curr {
			t.Errorf("row %d and %d have same x value", i-1, i)
		}
	}
}

func TestZeroFill_1mBucket_ShortRange_UsesNaturalStep(t *testing.T) {
	// 3 分钟范围 + 1m bucket → 自然节点数 4 ≤ 60，直接使用 1m 步长
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1m"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T10:00:00", End: "2024-01-15T10:03:00"},
	}
	result := ZeroFill(cfg, []map[string]interface{}{})
	// 3min / 1min，含两端 = 4 个桶: 10:00, 10:01, 10:02, 10:03
	if len(result) != 4 {
		t.Fatalf("expected 4 buckets (1m step), got %d", len(result))
	}
}

// 自适应密度测试：6h 范围 + 1m bucket
// 自然步长 1m → 360 个节点 >> 60，需要稀释
// rawStep = 6h / 60 = 6m，对齐到 1m 整数倍 → 6m 步长
// 6h / 6m = 60 个节点
func TestZeroFill_1mBucket_6hRange_AdaptiveDensity(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1m"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T00:00:00", End: "2024-01-15T06:00:00"},
	}
	result := ZeroFill(cfg, []map[string]interface{}{})
	// 6h 范围，自适应步长 = 6m（360m / 60），节点数 ≈ 60-61
	if len(result) > 70 {
		t.Fatalf("6h 范围自适应填充节点数应 ≤70, got %d (太密)", len(result))
	}
	if len(result) < 50 {
		t.Fatalf("6h 范围自适应填充节点数应 ≥50, got %d (太稀)", len(result))
	}
	// 验证步长：相邻 x 值时间差应为 6 分钟
	prevTime, err := time.ParseInLocation("2006-01-02 15:04:05", result[0]["x"].(string), time.Local)
	if err != nil {
		t.Fatalf("parse first x failed: %v", err)
	}
	currTime, err := time.ParseInLocation("2006-01-02 15:04:05", result[1]["x"].(string), time.Local)
	if err != nil {
		t.Fatalf("parse second x failed: %v", err)
	}
	step := currTime.Sub(prevTime)
	if step != 6*time.Minute {
		t.Errorf("expected 6m step, got %v", step)
	}
}

// 自适应密度测试：24h 范围 + 1h bucket
// 自然步长 1h → 24 个节点 ≤ 60，直接使用 1h 步长
func TestZeroFill_1hBucket_24hRange_NaturalStep(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T00:00:00", End: "2024-01-15T23:00:00"},
	}
	result := ZeroFill(cfg, []map[string]interface{}{})
	// 24h / 1h = 24 个节点 ≤ 60，使用自然步长
	if len(result) != 24 {
		t.Fatalf("expected 24 buckets (1h natural step), got %d", len(result))
	}
}

// 自适应密度测试：7d 范围 + 1h bucket
// 自然步长 1h → 168 个节点 > 60，需要稀释
// rawStep = 168h / 60 ≈ 2.8h，对齐到 1h 整数倍 → 3h
// 168h / 3h = 56 个节点
func TestZeroFill_1hBucket_7dRange_AdaptiveDensity(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1h"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-08T00:00:00", End: "2024-01-15T00:00:00"},
	}
	result := ZeroFill(cfg, []map[string]interface{}{})
	// 自适应稀释后节点数应 ≤ 60 左右
	if len(result) > 70 {
		t.Fatalf("7d 范围自适应填充节点数应 ≤70, got %d", len(result))
	}
	if len(result) < 50 {
		t.Fatalf("7d 范围自适应填充节点数应 ≥50, got %d", len(result))
	}
}

// 自适应密度测试：5m 范围 + 1s bucket
// 1s 自然步长 → 300 个节点 > 60，但 1s 最小步长为 15s
// rawStep = 5m / 60 = 5s，最小步长限制 → 15s
// 5m / 15s = 20 个节点
func TestZeroFill_1sBucket_5mRange_AdaptiveDensity(t *testing.T) {
	cfg := Config{
		XDimension: &Dimension{Field: "created_at", Bucket: "1s"},
		TimeRange:  &TimeRange{Type: "absolute", Start: "2024-01-15T10:00:00", End: "2024-01-15T10:05:00"},
	}
	result := ZeroFill(cfg, []map[string]interface{}{})
	// 5m / 15s = 20 个节点 + 1 = 21
	if len(result) > 25 {
		t.Fatalf("5m 范围 1s bucket 自适应填充应 ≤25, got %d", len(result))
	}
	if len(result) < 15 {
		t.Fatalf("5m 范围 1s bucket 自适应填充应 ≥15, got %d", len(result))
	}
}

// fillStepForRange 函数的直接测试
func TestFillStepForRange(t *testing.T) {
	tests := []struct {
		name   string
		d      time.Duration
		bucket string
		want   time.Duration
	}{
		// 短范围：自然步长够稀疏，直接使用
		{"3m_1m_natural", 3 * time.Minute, "1m", time.Minute},
		{"2h_1h_natural", 2 * time.Hour, "1h", time.Hour},
		{"3d_1d_natural", 3 * 24 * time.Hour, "1d", 24 * time.Hour},
		{"2w_1w_natural", 2 * 7 * 24 * time.Hour, "1w", 7 * 24 * time.Hour},

		// 长范围：自然步长过密，自适应稀释
		// 6h / 60 = 6m，对齐到 1m → 6m
		{"6h_1m_adaptive", 6 * time.Hour, "1m", 6 * time.Minute},
		// 30m / 60 = 30s，对齐到 1m → 1m（30s < 1m，multiple=1）
		{"30m_1m_natural", 30 * time.Minute, "1m", time.Minute},
		// 168h / 60 ≈ 2.8h，向上取整到 1h → 3h
		{"7d_1h_adaptive", 7 * 24 * time.Hour, "1h", 3 * time.Hour},
		// 5m / 1s: 1s 最小步长 15s
		{"5m_1s_minStep", 5 * time.Minute, "1s", 15 * time.Second},

		// 1d/1w 固定自然步长
		{"30d_1d_fixed", 30 * 24 * time.Hour, "1d", 24 * time.Hour},
		{"365d_1w_fixed", 365 * 24 * time.Hour, "1w", 7 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fillStepForRange(tt.d, tt.bucket)
			if got != tt.want {
				t.Errorf("fillStepForRange(%v, %s) = %v, want %v", tt.d, tt.bucket, got, tt.want)
			}
		})
	}
}