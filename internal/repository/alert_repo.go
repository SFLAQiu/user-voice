package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
)

type AlertRuleRepo struct{ db *gorm.DB }

func NewAlertRuleRepo(db *gorm.DB) *AlertRuleRepo { return &AlertRuleRepo{db: db} }

func (r *AlertRuleRepo) List(ctx context.Context) ([]model.AlertRule, error) {
	var rows []model.AlertRule
	return rows, r.db.WithContext(ctx).Order("id DESC").Find(&rows).Error
}

func (r *AlertRuleRepo) ListEnabled(ctx context.Context) ([]model.AlertRule, error) {
	var rows []model.AlertRule
	return rows, r.db.WithContext(ctx).Where("status = 1").Find(&rows).Error
}

// GetByPanelID 查询面板关联的告警规则，不存在时返回 nil。
func (r *AlertRuleRepo) GetByPanelID(ctx context.Context, panelID uint64) (*model.AlertRule, error) {
	var row model.AlertRule
	err := r.db.WithContext(ctx).Where("panel_id = ?", panelID).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// DeleteByPanelID 删除面板关联的告警规则。
func (r *AlertRuleRepo) DeleteByPanelID(ctx context.Context, panelID uint64) error {
	return r.db.WithContext(ctx).Where("panel_id = ?", panelID).Delete(&model.AlertRule{}).Error
}

// ListByType 按来源类型过滤告警规则：custom=自定义(panel_id IS NULL), panel=面板告警(panel_id IS NOT NULL), 空=全部。
func (r *AlertRuleRepo) ListByType(ctx context.Context, filterType string) ([]model.AlertRule, error) {
	var rows []model.AlertRule
	q := r.db.WithContext(ctx).Order("id DESC")
	switch filterType {
	case "custom":
		q = q.Where("panel_id IS NULL")
	case "panel":
		q = q.Where("panel_id IS NOT NULL")
	}
	return rows, q.Find(&rows).Error
}

func (r *AlertRuleRepo) Get(ctx context.Context, id uint64) (*model.AlertRule, error) {
	var row model.AlertRule
	return &row, r.db.WithContext(ctx).First(&row, id).Error
}

func (r *AlertRuleRepo) Create(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *AlertRuleRepo) Update(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *AlertRuleRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.AlertRule{}, id).Error
}

// UpdateEvalState persists evaluation result after each engine run.
func (r *AlertRuleRepo) UpdateEvalState(ctx context.Context, id uint64, state string, fires int, evalAt time.Time) error {
	return r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_eval_at":      evalAt,
		"last_state":        state,
		"consecutive_fires": fires,
	}).Error
}

// UpdateLastNotifyAt sets the timestamp of the most recent notification.
func (r *AlertRuleRepo) UpdateLastNotifyAt(ctx context.Context, id uint64, t time.Time) error {
	return r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("id = ?", id).
		Update("last_notify_at", t).Error
}

type AlertChannelRepo struct{ db *gorm.DB }

func NewAlertChannelRepo(db *gorm.DB) *AlertChannelRepo { return &AlertChannelRepo{db: db} }

func (r *AlertChannelRepo) List(ctx context.Context) ([]model.AlertChannel, error) {
	var rows []model.AlertChannel
	return rows, r.db.WithContext(ctx).Order("id DESC").Find(&rows).Error
}

func (r *AlertChannelRepo) Get(ctx context.Context, id uint64) (*model.AlertChannel, error) {
	var row model.AlertChannel
	return &row, r.db.WithContext(ctx).First(&row, id).Error
}

func (r *AlertChannelRepo) Create(ctx context.Context, ch *model.AlertChannel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *AlertChannelRepo) Update(ctx context.Context, ch *model.AlertChannel) error {
	return r.db.WithContext(ctx).Save(ch).Error
}

func (r *AlertChannelRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.AlertChannel{}, id).Error
}

// GetRuleChannelIDs returns the channel IDs bound to a rule.
func (r *AlertChannelRepo) GetRuleChannelIDs(ctx context.Context, ruleID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).Model(&model.AlertRuleChannel{}).
		Where("rule_id = ?", ruleID).Pluck("channel_id", &ids).Error
	return ids, err
}

// BatchGetRuleChannelIDs returns channel IDs for multiple rules as a map[ruleID][]channelID.
func (r *AlertChannelRepo) BatchGetRuleChannelIDs(ctx context.Context, ruleIDs []uint64) (map[uint64][]uint64, error) {
	if len(ruleIDs) == 0 {
		return map[uint64][]uint64{}, nil
	}
	var rows []model.AlertRuleChannel
	err := r.db.WithContext(ctx).Where("rule_id IN ?", ruleIDs).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint64][]uint64, len(ruleIDs))
	for _, r := range rows {
		m[r.RuleID] = append(m[r.RuleID], r.ChannelID)
	}
	return m, nil
}

// GetRuleChannels fetches channels bound to a rule via the join table.
func (r *AlertChannelRepo) GetRuleChannels(ctx context.Context, ruleID uint64) ([]model.AlertChannel, error) {
	var rows []model.AlertChannel
	err := r.db.WithContext(ctx).Raw(`
		SELECT ac.* FROM alert_channels ac
		INNER JOIN alert_rule_channels arc ON arc.channel_id = ac.id
		WHERE arc.rule_id = ?`, ruleID).Scan(&rows).Error
	return rows, err
}

// SetRuleChannels replaces the channel bindings for a rule.
func (r *AlertChannelRepo) SetRuleChannels(ctx context.Context, ruleID uint64, channelIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("rule_id = ?", ruleID).Delete(&model.AlertRuleChannel{}).Error; err != nil {
			return err
		}
		for _, cid := range channelIDs {
			if err := tx.Create(&model.AlertRuleChannel{RuleID: ruleID, ChannelID: cid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type AlertRecordRepo struct{ db *gorm.DB }

func NewAlertRecordRepo(db *gorm.DB) *AlertRecordRepo { return &AlertRecordRepo{db: db} }

func (r *AlertRecordRepo) Create(ctx context.Context, rec *model.AlertRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *AlertRecordRepo) List(ctx context.Context, ruleID uint64, limit int) ([]model.AlertRecord, error) {
	var rows []model.AlertRecord
	q := r.db.WithContext(ctx).Order("triggered_at DESC").Limit(limit)
	if ruleID > 0 {
		q = q.Where("rule_id = ?", ruleID)
	}
	return rows, q.Find(&rows).Error
}

// ListByTimeRange 查询指定时间范围内的告警记录
func (r *AlertRecordRepo) ListByTimeRange(ctx context.Context, ruleID uint64, startAt, endAt time.Time, limit int) ([]model.AlertRecord, error) {
	var rows []model.AlertRecord
	q := r.db.WithContext(ctx).Order("COALESCE(metric_timestamp, triggered_at) ASC").Limit(limit)
	if ruleID > 0 {
		q = q.Where("rule_id = ?", ruleID)
	}
	// 告警线定位规则：
	// 1) firing/pending 必须有 metric_timestamp，且按 metric_timestamp 过滤（严格对齐指标桶时间）
	// 2) resolved/ok 允许回退 triggered_at，避免丢失恢复事件
	q = q.Where(
		"((state IN ? AND metric_timestamp IS NOT NULL AND metric_timestamp >= ? AND metric_timestamp <= ?) OR (state IN ? AND COALESCE(metric_timestamp, triggered_at) >= ? AND COALESCE(metric_timestamp, triggered_at) <= ?))",
		[]string{"firing", "pending"}, startAt, endAt,
		[]string{"resolved", "ok"}, startAt, endAt,
	)
	return rows, q.Find(&rows).Error
}

// HasFiringRecordByMetricTs 判断某规则是否已有相同 metric_timestamp 的 firing/pending 记录。
// 用于去重：同一触发桶不应产生多条告警记录。
func (r *AlertRecordRepo) HasFiringRecordByMetricTs(ctx context.Context, ruleID uint64, metricTs time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AlertRecord{}).
		Where("rule_id = ? AND metric_timestamp = ? AND state IN ?", ruleID, metricTs, []string{"firing", "pending"}).
		Limit(1).Count(&count).Error
	return count > 0, err
}

// DeleteByRuleID 删除指定规则的所有告警记录。
func (r *AlertRecordRepo) DeleteByRuleID(ctx context.Context, ruleID uint64) error {
	return r.db.WithContext(ctx).Where("rule_id = ?", ruleID).Delete(&model.AlertRecord{}).Error
}

// DeleteByRuleIDs 批量删除多条规则的告警记录。
func (r *AlertRecordRepo) DeleteByRuleIDs(ctx context.Context, ruleIDs []uint64) error {
	if len(ruleIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("rule_id IN ?", ruleIDs).Delete(&model.AlertRecord{}).Error
}
