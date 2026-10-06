package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/feedback/internal/alert/notifier"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

type AlertService struct {
	ruleRepo    *repository.AlertRuleRepo
	channelRepo *repository.AlertChannelRepo
	recordRepo  *repository.AlertRecordRepo
	panelRepo   *repository.PanelRepo
	dashRepo    *repository.DashboardRepo
}

func NewAlertService(
	rr *repository.AlertRuleRepo,
	cr *repository.AlertChannelRepo,
	rec *repository.AlertRecordRepo,
	pr *repository.PanelRepo,
	dr *repository.DashboardRepo,
) *AlertService {
	return &AlertService{ruleRepo: rr, channelRepo: cr, recordRepo: rec, panelRepo: pr, dashRepo: dr}
}

type AlertRuleInput struct {
	Name               string          `json:"name" binding:"required,max=128"`
	PanelID            *uint64         `json:"panel_id"` // 仅解析，CreateRule/UpdateRule 拒绝非空值
	MetricQuery        json.RawMessage `json:"metric_query" binding:"required"`
	TimeWindowSec      int             `json:"time_window_sec"`
	ConditionOp        string          `json:"condition_op" binding:"required,oneof=> >= < <= ="`
	Threshold          float64         `json:"threshold"`
	Level              string          `json:"level" binding:"required,oneof=info warning critical"`
	SilenceMinutes     int             `json:"silence_minutes"`
	EvalIntervalSec    int             `json:"eval_interval_sec"`
	PendingDurationSec int             `json:"pending_duration_sec"`
	ReduceMode         string          `json:"reduce_mode" binding:"omitempty,oneof=max last"`
	Status             int             `json:"status"`
	ChannelIDs         []uint64        `json:"channel_ids"`
}

type AlertRuleWithChannels struct {
	model.AlertRule
	ChannelIDs    []uint64 `json:"channel_ids"`
	PanelName     string   `json:"panel_name,omitempty"`    // 面板告警规则的面板名称
	DashboardID   uint64   `json:"dashboard_id,omitempty"`  // 面板所属仪表盘ID，用于前端跳转
	DashboardName string   `json:"dashboard_name,omitempty"` // 面板所属仪表盘名称
}

func (s *AlertService) ListRules(ctx context.Context, filterType string) ([]AlertRuleWithChannels, error) {
	rules, err := s.ruleRepo.ListByType(ctx, filterType)
	if err != nil {
		return nil, err
	}
	result := make([]AlertRuleWithChannels, len(rules))
	ruleIDs := make([]uint64, len(rules))
	panelIDs := make([]uint64, 0)
	for i, r := range rules {
		result[i].AlertRule = r
		ruleIDs[i] = r.ID
		if r.PanelID != nil && *r.PanelID > 0 {
			panelIDs = append(panelIDs, *r.PanelID)
		}
	}
	// 批量获取通知渠道绑定
	chMap, err := s.channelRepo.BatchGetRuleChannelIDs(ctx, ruleIDs)
	if err != nil {
		chMap = map[uint64][]uint64{}
	}
	for i := range result {
		result[i].ChannelIDs = chMap[result[i].ID]
	}
	// 批量获取面板名称及仪表盘名称（仅面板告警规则需要）
	if len(panelIDs) > 0 {
		panels, pErr := s.panelRepo.ListByIDs(ctx, panelIDs)
		if pErr == nil {
			nameMap := make(map[uint64]string, len(panels))
			dashIDMap := make(map[uint64]uint64, len(panels))
			for _, p := range panels {
				nameMap[p.ID] = p.Name
				dashIDMap[p.ID] = p.DashboardID
			}
			// 批量获取仪表盘名称
			dashIDs := make([]uint64, 0, len(panels))
			for _, dID := range dashIDMap {
				dashIDs = append(dashIDs, dID)
			}
			dashNameMap := make(map[uint64]string)
			if len(dashIDs) > 0 && s.dashRepo != nil {
				dashes, dErr := s.dashRepo.ListByIDs(ctx, dashIDs)
				if dErr == nil {
					for _, d := range dashes {
						dashNameMap[d.ID] = d.Name
					}
				}
			}
			for i := range result {
				if result[i].PanelID != nil {
					result[i].PanelName = nameMap[*result[i].PanelID]
					result[i].DashboardID = dashIDMap[*result[i].PanelID]
					result[i].DashboardName = dashNameMap[dashIDMap[*result[i].PanelID]]
				}
			}
		}
	}
	return result, nil
}

func (s *AlertService) CreateRule(ctx context.Context, in AlertRuleInput) (*model.AlertRule, error) {
	// 面板告警规则必须通过面板配置创建，不可直接设置 panel_id
	if in.PanelID != nil && *in.PanelID != 0 {
		return nil, apperr.InvalidParam("面板告警规则需通过面板配置创建，不可直接设置 panel_id")
	}
	rule := &model.AlertRule{
		Name:               in.Name,
		MetricQueryJSON:    model.JSON(in.MetricQuery),
		TimeWindowSec:      in.TimeWindowSec,
		ConditionOp:        in.ConditionOp,
		Threshold:          in.Threshold,
		Level:              in.Level,
		SilenceMinutes:     in.SilenceMinutes,
		EvalIntervalSec:    in.EvalIntervalSec,
		PendingDurationSec: in.PendingDurationSec,
		ReduceMode:         in.ReduceMode,
		Status:             in.Status,
	}
	if rule.Status == 0 {
		rule.Status = 1
	}
	if rule.EvalIntervalSec <= 0 {
		rule.EvalIntervalSec = 60
	}
	if rule.SilenceMinutes <= 0 {
		rule.SilenceMinutes = 30
	}
	if rule.TimeWindowSec <= 0 {
		rule.TimeWindowSec = 300
	}
	if rule.ReduceMode == "" {
		rule.ReduceMode = "max"
	}
	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, apperr.Internal("create rule", err)
	}
	if len(in.ChannelIDs) > 0 {
		if err := s.channelRepo.SetRuleChannels(ctx, rule.ID, in.ChannelIDs); err != nil {
			return nil, apperr.Internal("set rule channels", err)
		}
	}
	return rule, nil
}

func (s *AlertService) UpdateRule(ctx context.Context, id uint64, in AlertRuleInput) (*model.AlertRule, error) {
	// 面板告警规则必须通过面板配置修改
	if in.PanelID != nil && *in.PanelID != 0 {
		return nil, apperr.InvalidParam("面板告警规则需通过面板配置修改，不可直接设置 panel_id")
	}
	rule, err := s.ruleRepo.Get(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("rule not found")
	}
	// 面板关联的规则不允许通过此接口编辑
	if rule.PanelID != nil && *rule.PanelID > 0 {
		return nil, apperr.InvalidParam("面板告警规则需通过面板配置修改，不可在告警管理页面直接编辑")
	}
	rule.Name = in.Name
	rule.MetricQueryJSON = model.JSON(in.MetricQuery)
	rule.TimeWindowSec = in.TimeWindowSec
	rule.ConditionOp = in.ConditionOp
	rule.Threshold = in.Threshold
	rule.Level = in.Level
	rule.SilenceMinutes = in.SilenceMinutes
	rule.EvalIntervalSec = in.EvalIntervalSec
	rule.PendingDurationSec = in.PendingDurationSec
	rule.ReduceMode = in.ReduceMode
	if in.Status != 0 {
		rule.Status = in.Status
	}
	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, apperr.Internal("update rule", err)
	}
	if in.ChannelIDs != nil {
		if err := s.channelRepo.SetRuleChannels(ctx, id, in.ChannelIDs); err != nil {
			return nil, apperr.Internal("set rule channels", err)
		}
	}
	return rule, nil
}

func (s *AlertService) ToggleRuleStatus(ctx context.Context, id uint64, status int) (*model.AlertRule, error) {
	rule, err := s.ruleRepo.Get(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("rule not found")
	}
	rule.Status = status
	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, apperr.Internal("toggle rule status", err)
	}
	return rule, nil
}

func (s *AlertService) DeleteRule(ctx context.Context, id uint64) error {
	rule, err := s.ruleRepo.Get(ctx, id)
	if err != nil {
		return apperr.NotFound("rule not found")
	}
	// 面板告警规则的删除需同步清理面板 alert_config_json
	if rule.PanelID != nil && *rule.PanelID > 0 {
		panel, pErr := s.panelRepo.Get(ctx, *rule.PanelID)
		if pErr == nil && panel != nil {
			panel.AlertConfigJSON = model.JSON{}
			s.panelRepo.Update(ctx, panel)
		}
	}
	return s.ruleRepo.Delete(ctx, id)
}

type AlertChannelInput struct {
	Name       string `json:"name" binding:"required,max=128"`
	Type       string `json:"type" binding:"required,oneof=wecom feishu dingtalk webhook"`
	WebhookURL string `json:"webhook_url" binding:"required,max=512"`
	Secret     string `json:"secret" binding:"max=255"`
	Status     int    `json:"status"`
}

func (s *AlertService) ListChannels(ctx context.Context) ([]model.AlertChannel, error) {
	return s.channelRepo.List(ctx)
}

func (s *AlertService) CreateChannel(ctx context.Context, in AlertChannelInput) (*model.AlertChannel, error) {
	ch := &model.AlertChannel{
		Name:       in.Name,
		Type:       in.Type,
		WebhookURL: in.WebhookURL,
		Secret:     in.Secret,
		Status:     in.Status,
	}
	if ch.Status == 0 {
		ch.Status = 1
	}
	return ch, s.channelRepo.Create(ctx, ch)
}

func (s *AlertService) UpdateChannel(ctx context.Context, id uint64, in AlertChannelInput) (*model.AlertChannel, error) {
	ch, err := s.channelRepo.Get(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("channel not found")
	}
	ch.Name = in.Name
	ch.Type = in.Type
	ch.WebhookURL = in.WebhookURL
	ch.Secret = in.Secret
	if in.Status != 0 {
		ch.Status = in.Status
	}
	return ch, s.channelRepo.Update(ctx, ch)
}

func (s *AlertService) DeleteChannel(ctx context.Context, id uint64) error {
	return s.channelRepo.Delete(ctx, id)
}

func (s *AlertService) SetRuleChannels(ctx context.Context, ruleID uint64, channelIDs []uint64) error {
	return s.channelRepo.SetRuleChannels(ctx, ruleID, channelIDs)
}

func (s *AlertService) ListRecords(ctx context.Context, ruleID uint64, limit int) ([]model.AlertRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.recordRepo.List(ctx, ruleID, limit)
}

// ListRecordsByTimeRange 查询指定时间范围内的告警记录
func (s *AlertService) ListRecordsByTimeRange(ctx context.Context, ruleID uint64, startAt, endAt time.Time, limit int) ([]model.AlertRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.recordRepo.ListByTimeRange(ctx, ruleID, startAt, endAt, limit)
}

func (s *AlertService) TestNotify(ctx context.Context, channelID uint64, msg string) error {
	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return apperr.NotFound("channel not found")
	}
	n, ok := notifier.Registry[ch.Type]
	if !ok {
		return apperr.InvalidParam(fmt.Sprintf("unsupported channel type: %s", ch.Type))
	}
	return n.Send(ctx, ch.WebhookURL, ch.Secret, msg)
}
