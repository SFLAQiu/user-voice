package alert

// Package alert implements the alert evaluation engine.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"text/template"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/feedback/internal/alert/notifier"
	"github.com/feedback/internal/model"
	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/query"
	"github.com/feedback/internal/repository"
)

// Engine periodically evaluates alert rules.
type Engine struct {
	ruleRepo    *repository.AlertRuleRepo
	channelRepo *repository.AlertChannelRepo
	recordRepo  *repository.AlertRecordRepo
	panelRepo   *repository.PanelRepo
	dashRepo    *repository.DashboardRepo
	db          *gorm.DB
	interval    time.Duration
	mu          sync.Mutex
}

// New creates an Engine that scans rules every interval.
func New(
	ruleRepo *repository.AlertRuleRepo,
	channelRepo *repository.AlertChannelRepo,
	recordRepo *repository.AlertRecordRepo,
	panelRepo *repository.PanelRepo,
	dashRepo *repository.DashboardRepo,
	db *gorm.DB,
	interval time.Duration,
) *Engine {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Engine{
		ruleRepo:    ruleRepo,
		channelRepo: channelRepo,
		recordRepo:  recordRepo,
		panelRepo:   panelRepo,
		dashRepo:    dashRepo,
		db:          db,
		interval:    interval,
	}
}

// Start runs the evaluation loop until ctx is cancelled.
func (e *Engine) Start(ctx context.Context) {
	go e.loop(ctx)
}

func (e *Engine) loop(ctx context.Context) {
	t := time.NewTicker(e.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.evalAll(ctx)
		}
	}
}

func (e *Engine) evalAll(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.ruleRepo.ListEnabled(ctx)
	if err != nil {
		logger.L.Warn("alert: list rules", zap.Error(err))
		return
	}
	for _, rule := range rules {
		if err := e.evalOne(ctx, rule); err != nil {
			logger.L.Warn("alert: eval rule", zap.Uint64("rule_id", rule.ID), zap.Error(err))
		}
	}
}

func (e *Engine) evalOne(ctx context.Context, rule model.AlertRule) error {
	// Check if it's time to evaluate.
	if rule.LastEvalAt != nil {
		elapsed := time.Since(rule.LastEvalAt.Time())
		if elapsed < time.Duration(rule.EvalIntervalSec)*time.Second {
			return nil
		}
	}

	value, metricTimestamp, err := e.computeMetric(ctx, rule)
	if err != nil {
		return fmt.Errorf("compute metric: %w", err)
	}

	firing := compare(value, rule.ConditionOp, rule.Threshold)
	now := time.Now()

	// ── 状态流转：ok → pending → firing → resolved ──
	// pending_duration_sec > 0 时启用 Grafana 式待确认机制，
	// 指标突破阈值先进入 pending（预警），持续高于阈值超过待确认时长才转为 firing。
	// pending_duration_sec = 0 时与原行为一致（立即 firing）。
	state := "ok"
	fires := 0
	switch {
	case firing && rule.LastState == "ok":
		fires = 1
		if rule.PendingDurationSec > 0 {
			state = "pending"
		} else {
			state = "firing"
		}
	case firing && rule.LastState == "pending":
		fires = rule.ConsecutiveFires + 1
		elapsedSec := fires * rule.EvalIntervalSec
		if elapsedSec >= rule.PendingDurationSec {
			state = "firing"
		} else {
			state = "pending"
		}
	case firing && (rule.LastState == "firing" || rule.LastState == "resolved"):
		fires = rule.ConsecutiveFires + 1
		state = "firing"
	case !firing && rule.LastState == "firing":
		state = "resolved"
	case !firing && rule.LastState == "pending":
		state = "ok"
	default:
		state = "ok"
	}

	// ── 通知判断 ──
	// pending → ok：不通知（仅预警，未真正告警）
	// pending → firing：首次告警通知
	// firing → resolved：恢复通知
	// 仍在 firing 且超过静默期：重复通知
	shouldNotify := false
	switch {
	case state == "firing" && rule.LastState == "pending":
		shouldNotify = true
	case state == "firing" && rule.LastState != "firing":
		shouldNotify = true
	case state == "firing" && rule.LastState == "firing":
		if rule.LastNotifyAt != nil {
			if time.Since(rule.LastNotifyAt.Time()) >= time.Duration(rule.SilenceMinutes)*time.Minute {
				shouldNotify = true
			}
		} else {
			shouldNotify = true
		}
	case state == "resolved":
		shouldNotify = true
	}

	if err := e.ruleRepo.UpdateEvalState(ctx, rule.ID, state, fires, now); err != nil {
		return fmt.Errorf("update eval state: %w", err)
	}

	// 厺重：同一 metric_timestamp 不重复创建 firing/pending 记录。
	// 只有新的触发桶才产出新记录。
	shouldCreateRecord := true
	if metricTimestamp != nil && (state == "firing" || state == "pending") {
		exists, err := e.recordRepo.HasFiringRecordByMetricTs(ctx, rule.ID, *metricTimestamp)
		if err != nil {
			logger.L.Warn("alert: check duplicate metric_ts", zap.Uint64("rule_id", rule.ID), zap.Error(err))
		} else if exists {
			shouldCreateRecord = false
		}
	}

	if state == "pending" && shouldCreateRecord {
		e.createPendingRecord(ctx, rule, value, metricTimestamp)
	}

	if shouldNotify {
		if shouldCreateRecord {
			e.notify(ctx, rule, value, state, metricTimestamp)
		} else {
			if err := e.ruleRepo.UpdateLastNotifyAt(ctx, rule.ID, now); err != nil {
				logger.L.Warn("alert: update last_notify_at", zap.Uint64("rule_id", rule.ID), zap.Error(err))
			}
		}
	}
	return nil
}

// computeMetric 将告警规则的 MetricQueryJSON 解析为查询配置，
// 所有告警始终查询 metric_time_buckets 预聚合表，确保与图表数据源一致。
// 粒度（1m/1h）从面板 x_dimension.bucket 读取，与图表一致。
// 重构：直接用 SQL 选出 reduce_mode 对应的单条触发桶（max/last），
// 不再通过应用层遍历/零填充/类型转换，彻底消除触发值与图表不一致的问题。
func (e *Engine) computeMetric(ctx context.Context, rule model.AlertRule) (float64, *time.Time, error) {
	var cfg query.Config
	if err := json.Unmarshal(rule.MetricQueryJSON, &cfg); err != nil {
		return 0, nil, fmt.Errorf("parse query config: %w", err)
	}

	// 用 TimeWindowSec 覆盖 time_range（语义：评估窗口时长）
	windowSec := rule.TimeWindowSec
	if windowSec <= 0 {
		windowSec = 300
	}
	cfg.TimeRange = &query.TimeRange{
		Type:  "relative",
		Value: query.FormatDuration(windowSec),
	}

	// 强制告警走预聚合表配置
	cfg.Metric = ""
	cfg.GroupBy = nil
	if cfg.XDimension == nil {
		cfg.XDimension = &query.Dimension{
			Field:  "original_created_at",
			Bucket: "1m",
		}
	}

	// 合并仪表盘级筛选条件树，与面板图表查询口径一致。
	if rule.PanelID != nil && e.dashRepo != nil {
		panel, pErr := e.panelRepo.Get(ctx, *rule.PanelID)
		if pErr == nil && panel != nil {
			dash, dErr := e.dashRepo.Get(ctx, panel.DashboardID)
			if dErr == nil && dash != nil && len(dash.FiltersJSON) > 0 {
				tree, tErr := query.NormalizeFilterTree(json.RawMessage(dash.FiltersJSON))
				if tErr == nil && tree != nil {
					cfg.FilterTree = tree
				}
			}
		}
	}

	// 确定 reduce 模式
	mode := rule.ReduceMode
	if mode == "" {
		mode = "max"
	}
	if mode != "max" && mode != "last" {
		mode = "max"
	}

	// 直接用 SQL 选出触发桶（单条结果），避免应用层遍历/类型转换
	sqlStr, args, err := query.BuildPreAggReduce(cfg, mode)
	if err != nil {
		return 0, nil, fmt.Errorf("build reduce query: %w", err)
	}

	var result struct {
		AggValue   float64   `gorm:"column:agg_value"`
		BucketTime time.Time `gorm:"column:bucket_time"`
	}
	if err := e.db.WithContext(ctx).Raw(sqlStr, args...).Scan(&result).Error; err != nil {
		return 0, nil, fmt.Errorf("execute reduce query: %w", err)
	}

	// 无数据（bucket_time 零值）
	if result.BucketTime.IsZero() {
		return 0, nil, nil
	}

	metricTs := result.BucketTime.In(time.Local)
	logger.L.Debug("alert: computeMetric result",
		zap.Uint64("rule_id", rule.ID),
		zap.String("mode", mode),
		zap.Float64("value", result.AggValue),
		zap.Time("metric_ts", metricTs),
		zap.String("sql", sqlStr),
	)

	return result.AggValue, &metricTs, nil
}

// metricTimestampLocal 将 *time.Time 转为 *model.LocalTime，nil 保持 nil。
func metricTimestampLocal(t *time.Time) *model.LocalTime {
	if t == nil {
		return nil
	}
	lt := model.LocalTime(*t)
	return &lt
}

func compare(val float64, op string, threshold float64) bool {
	switch op {
	case ">":
		return val > threshold
	case ">=":
		return val >= threshold
	case "<":
		return val < threshold
	case "<=":
		return val <= threshold
	case "=", "==":
		return val == threshold
	}
	return false
}

func (e *Engine) notify(ctx context.Context, rule model.AlertRule, value float64, state string, metricTimestamp *time.Time) {
	channels, err := e.channelRepo.GetRuleChannels(ctx, rule.ID)
	if err != nil {
		logger.L.Warn("alert: get channels", zap.Uint64("rule_id", rule.ID), zap.Error(err))
	}

	msg := e.renderMessage(ctx, rule, value, state)
	notifiedIDs := make([]uint64, 0, len(channels))
	notifyStatus := "no_channels"

	for _, ch := range channels {
		n, ok := notifier.Registry[ch.Type]
		if !ok {
			logger.L.Warn("alert: unknown notifier type", zap.String("type", ch.Type))
			continue
		}
		if err := n.Send(ctx, ch.WebhookURL, ch.Secret, msg); err != nil {
			logger.L.Warn("alert: send notification", zap.Uint64("channel_id", ch.ID), zap.Error(err))
			notifyStatus = "partial_fail"
			continue
		}
		notifiedIDs = append(notifiedIDs, ch.ID)
	}

	if len(notifiedIDs) > 0 && notifyStatus != "partial_fail" {
		notifyStatus = "ok"
	}
	if len(notifiedIDs) > 0 && notifyStatus == "partial_fail" {
		notifyStatus = "partial_fail"
	}

	// Update last_notify_at so silence logic works correctly.
	now := time.Now()
	if err := e.ruleRepo.UpdateLastNotifyAt(ctx, rule.ID, now); err != nil {
		logger.L.Warn("alert: update last_notify_at", zap.Uint64("rule_id", rule.ID), zap.Error(err))
	}

	chIDsJSON, _ := json.Marshal(notifiedIDs)
	rec := &model.AlertRecord{
		RuleID:           rule.ID,
		TriggerValue:     value,
		Threshold:        rule.Threshold,
		Level:            rule.Level,
		State:            state,
		NotifiedChannels: model.JSON(chIDsJSON),
		NotifyStatus:     notifyStatus,
		TriggeredAt:      model.LocalTime(now),
		MetricTimestamp:  metricTimestampLocal(metricTimestamp),
	}
	// resolved 或 ok 记录填充 ResolvedAt
	if state == "resolved" || state == "ok" {
		lt := model.LocalTime(now)
		rec.ResolvedAt = &lt
	}
	if err := e.recordRepo.Create(ctx, rec); err != nil {
		logger.L.Warn("alert: create record", zap.Error(err))
	}
}

// createPendingRecord 创建预警记录（不发送通知，仅用于前端渲染预警线）
func (e *Engine) createPendingRecord(ctx context.Context, rule model.AlertRule, value float64, metricTimestamp *time.Time) {
	now := time.Now()
	rec := &model.AlertRecord{
		RuleID:           rule.ID,
		TriggerValue:     value,
		Threshold:        rule.Threshold,
		Level:            rule.Level,
		State:            "pending",
		NotifiedChannels: model.JSON([]byte("[]")),
		NotifyStatus:     "no_channels",
		TriggeredAt:      model.LocalTime(now),
		MetricTimestamp:  metricTimestampLocal(metricTimestamp),
	}
	if err := e.recordRepo.Create(ctx, rec); err != nil {
		logger.L.Warn("alert: create pending record", zap.Uint64("rule_id", rule.ID), zap.Error(err))
	}
}

const defaultTpl = `### {{.TitleIcon}} {{.TitleText}} {{.Name}}

**当前值**: {{printf "%.2f" .Value}} {{.Op}} **阈值 {{printf "%.2f" .Threshold}}**

**状态**: {{.StateLabel}}

**时间**: {{.Time}}`

var levelLabels = map[string]string{"info": "信息", "warning": "警告", "critical": "严重"}
var stateLabels = map[string]string{"ok": "正常", "firing": "触发中", "pending": "待评估", "resolved": "已恢复"}

func (e *Engine) renderMessage(ctx context.Context, rule model.AlertRule, value float64, state string) string {
	tpl, err := template.New("").Parse(defaultTpl)
	if err != nil {
		return fmt.Sprintf("Alert: %s value=%.2f state=%s", rule.Name, value, state)
	}

	titleIcon := "💚"
	titleText := "恢复正常"
	if state == "firing" {
		titleIcon = "💔"
		titleText = levelLabels[rule.Level] + "告警"
	} else if state == "pending" {
		titleIcon = "⚠️"
		titleText = "预警"
	}

	// 面板告警规则在标题中加入仪表盘名称
	namePrefix := ""
	if rule.PanelID != nil && e.panelRepo != nil && e.dashRepo != nil {
		panel, pErr := e.panelRepo.Get(ctx, *rule.PanelID)
		if pErr == nil && panel != nil {
			dash, dErr := e.dashRepo.Get(ctx, panel.DashboardID)
			if dErr == nil && dash != nil {
				namePrefix = "【仪表盘: " + dash.Name + "】"
			}
		}
	}

	data := map[string]interface{}{
		"TitleIcon":  titleIcon,
		"TitleText":  titleText,
		"Name":       namePrefix + rule.Name,
		"Value":      value,
		"Op":         rule.ConditionOp,
		"Threshold":  rule.Threshold,
		"StateLabel": stateLabels[state],
		"Time":       time.Now().Format("2006-01-02 15:04:05"),
	}
	var buf bytes.Buffer
	_ = tpl.Execute(&buf, data)
	return buf.String()
}
