package service

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/query"
	"github.com/feedback/internal/repository"
)

type DashboardService struct {
	dashRepo         *repository.DashboardRepo
	panelRepo        *repository.PanelRepo
	alertRuleRepo    *repository.AlertRuleRepo
	alertChannelRepo *repository.AlertChannelRepo
	alertRecordRepo  *repository.AlertRecordRepo
	db               *gorm.DB
}

func NewDashboardService(
	dr *repository.DashboardRepo,
	pr *repository.PanelRepo,
	arr *repository.AlertRuleRepo,
	acr *repository.AlertChannelRepo,
	arr2 *repository.AlertRecordRepo,
	db *gorm.DB,
) *DashboardService {
	return &DashboardService{dashRepo: dr, panelRepo: pr, alertRuleRepo: arr, alertChannelRepo: acr, alertRecordRepo: arr2, db: db}
}

func (s *DashboardService) List(ctx context.Context) ([]model.Dashboard, error) {
	return s.dashRepo.List(ctx)
}

func (s *DashboardService) Get(ctx context.Context, id uint64) (*model.Dashboard, error) {
	d, err := s.dashRepo.Get(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("dashboard not found")
	}
	return d, nil
}

type CreateDashboardInput struct {
	Name        string          `json:"name" binding:"required,max=128"`
	Description string          `json:"description" binding:"max=512"`
	Layout      json.RawMessage `json:"layout"`
	Filters     json.RawMessage `json:"filters"`
}

func (s *DashboardService) Create(ctx context.Context, in CreateDashboardInput, userID uint64) (*model.Dashboard, error) {
	d := &model.Dashboard{
		Name:        in.Name,
		Description: in.Description,
		LayoutJSON:  model.JSON(in.Layout),
		FiltersJSON: model.JSON(in.Filters),
		CreatedBy:   userID,
	}
	return d, s.dashRepo.Create(ctx, d)
}

type UpdateDashboardInput struct {
	Name           string          `json:"name" binding:"required,max=128"`
	Description    string          `json:"description" binding:"max=512"`
	Layout         json.RawMessage `json:"layout"`
	Filters        json.RawMessage `json:"filters"`
	RebuildAlerts  bool            `json:"rebuild_alerts"` // 篮选变更时重建面板告警规则
}

func (s *DashboardService) Update(ctx context.Context, id uint64, in UpdateDashboardInput) (*model.Dashboard, error) {
	d, err := s.dashRepo.Get(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("dashboard not found")
	}
	d.Name = in.Name
	d.Description = in.Description
	d.LayoutJSON = model.JSON(in.Layout)
	d.FiltersJSON = model.JSON(in.Filters)
	if err := s.dashRepo.Update(ctx, d); err != nil {
		return nil, apperr.Internal("update dashboard", err)
	}

	// 篮选变更时重建面板告警规则
	if in.RebuildAlerts {
		panels, pErr := s.panelRepo.ListByDashboard(ctx, id)
		if pErr != nil {
			fmt.Printf("dashboard: rebuild alert — list panels: %v\n", pErr)
			return d, nil
		}
		for _, p := range panels {
			if len(p.AlertConfigJSON) == 0 || string(p.AlertConfigJSON) == "null" {
				continue
			}
			// 删除旧告警规则及关联记录，然后重新同步
			rule, rErr := s.alertRuleRepo.GetByPanelID(ctx, p.ID)
			if rErr == nil && rule != nil {
				if err := s.alertRecordRepo.DeleteByRuleID(ctx, rule.ID); err != nil {
					fmt.Printf("dashboard: rebuild alert — delete records for rule %d: %v\n", rule.ID, err)
				}
				if err := s.alertRuleRepo.DeleteByPanelID(ctx, p.ID); err != nil {
					fmt.Printf("dashboard: rebuild alert — delete rule for panel %d: %v\n", p.ID, err)
				}
			}
			if err := s.syncPanelAlert(ctx, &p); err != nil {
				fmt.Printf("dashboard: rebuild alert — sync panel %d: %v\n", p.ID, err)
			}
		}
	}

	return d, nil
}

// Delete 删除仪表盘及其所有面板，并级联删除面板关联的告警规则和告警记录。
func (s *DashboardService) Delete(ctx context.Context, id uint64) error {
	// 获取所有面板，级联删除告警规则和记录
	panels, err := s.panelRepo.ListByDashboard(ctx, id)
	if err != nil {
		return s.dashRepo.Delete(ctx, id) // 查不到面板仍继续删除仪表盘
	}
	for _, p := range panels {
		rule, rErr := s.alertRuleRepo.GetByPanelID(ctx, p.ID)
		if rErr == nil && rule != nil {
			if err := s.alertRecordRepo.DeleteByRuleID(ctx, rule.ID); err != nil {
				fmt.Printf("dashboard: delete alert records for panel %d: %v\n", p.ID, err)
			}
			if err := s.alertRuleRepo.DeleteByPanelID(ctx, p.ID); err != nil {
				fmt.Printf("dashboard: delete alert rule for panel %d: %v\n", p.ID, err)
			}
		}
	}
	return s.dashRepo.Delete(ctx, id)
}

// DeleteInfo 返回仪表盘删除前的统计信息，供前端展示二次确认。
type DeleteInfo struct {
	PanelCount    int `json:"panel_count"`
	AlertRuleCount int `json:"alert_rule_count"`
}

// GetDeleteInfo 查询仪表盘下面板数量和告警规则数量。
func (s *DashboardService) GetDeleteInfo(ctx context.Context, dashboardID uint64) (*DeleteInfo, error) {
	panels, err := s.panelRepo.ListByDashboard(ctx, dashboardID)
	if err != nil {
		return nil, err
	}
	info := &DeleteInfo{PanelCount: len(panels)}
	for _, p := range panels {
		rule, rErr := s.alertRuleRepo.GetByPanelID(ctx, p.ID)
		if rErr == nil && rule != nil {
			info.AlertRuleCount++
		}
	}
	return info, nil
}

// HasPanelAlertRules 检查仪表盘是否有面板设置了告警规则。
func (s *DashboardService) HasPanelAlertRules(ctx context.Context, dashboardID uint64) (bool, error) {
	panels, err := s.panelRepo.ListByDashboard(ctx, dashboardID)
	if err != nil {
		return false, err
	}
	for _, p := range panels {
		if len(p.AlertConfigJSON) > 0 && string(p.AlertConfigJSON) != "null" {
			return true, nil
		}
	}
	return false, nil
}

// ── Panel 管理 ──

type CreatePanelInput struct {
	Name           string          `json:"name" binding:"required,max=128"`
	ChartType      string          `json:"chart_type" binding:"required,oneof=line bar pie number"`
	QueryConfig    json.RawMessage `json:"query_config" binding:"required"`
	Position       json.RawMessage `json:"position"`
	RefreshSeconds int             `json:"refresh_seconds"`
	AlertConfig    json.RawMessage `json:"alert_config"`
}

func (s *DashboardService) CreatePanel(ctx context.Context, dashboardID uint64, in CreatePanelInput) (*model.Panel, error) {
	p := &model.Panel{
		DashboardID:     dashboardID,
		Name:            in.Name,
		ChartType:       in.ChartType,
		QueryConfigJSON: model.JSON(in.QueryConfig),
		PositionJSON:    model.JSON(in.Position),
		RefreshSeconds:  in.RefreshSeconds,
		AlertConfigJSON: model.JSON(in.AlertConfig),
	}
	if p.RefreshSeconds <= 0 {
		p.RefreshSeconds = 60
	}
	if err := s.panelRepo.Create(ctx, p); err != nil {
		return nil, apperr.Internal("create panel", err)
	}
	if err := s.syncPanelAlert(ctx, p); err != nil {
		// 告警同步失败不影响面板创建，仅记录日志
		fmt.Printf("dashboard: sync panel alert: %v\n", err)
	}
	return p, nil
}

type UpdatePanelInput struct {
	Name           string          `json:"name" binding:"required,max=128"`
	ChartType      string          `json:"chart_type" binding:"required,oneof=line bar pie number"`
	QueryConfig    json.RawMessage `json:"query_config" binding:"required"`
	Position       json.RawMessage `json:"position"`
	RefreshSeconds int             `json:"refresh_seconds"`
	AlertConfig    json.RawMessage `json:"alert_config"`
}

func (s *DashboardService) UpdatePanel(ctx context.Context, id uint64, in UpdatePanelInput) (*model.Panel, error) {
	p, err := s.panelRepo.Get(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("panel not found")
	}
	p.Name = in.Name
	p.ChartType = in.ChartType
	p.QueryConfigJSON = model.JSON(in.QueryConfig)
	p.PositionJSON = model.JSON(in.Position)
	p.RefreshSeconds = in.RefreshSeconds
	p.AlertConfigJSON = model.JSON(in.AlertConfig)
	if p.RefreshSeconds <= 0 {
		p.RefreshSeconds = 60
	}
	if err := s.panelRepo.Update(ctx, p); err != nil {
		return nil, apperr.Internal("update panel", err)
	}
	if err := s.syncPanelAlert(ctx, p); err != nil {
		fmt.Printf("dashboard: sync panel alert: %v\n", err)
	}
	return p, nil
}

func (s *DashboardService) DeletePanel(ctx context.Context, id uint64) error {
	// 删除面板时同步删除关联告警规则
	if err := s.alertRuleRepo.DeleteByPanelID(ctx, id); err != nil {
		fmt.Printf("dashboard: delete panel alert rule: %v\n", err)
	}
	return s.panelRepo.Delete(ctx, id)
}

// ── 面板告警同步 ──

// syncPanelAlert 将面板告警配置同步到 AlertRule。
// Panel.alert_config_json 是源头，AlertRule 是投影。
func (s *DashboardService) syncPanelAlert(ctx context.Context, panel *model.Panel) error {
	// 无告警配置 → 删除已有规则
	if len(panel.AlertConfigJSON) == 0 || string(panel.AlertConfigJSON) == "null" {
		return s.alertRuleRepo.DeleteByPanelID(ctx, panel.ID)
	}

	// 校验图表类型：仅 line/bar 支持告警
	if panel.ChartType != "line" && panel.ChartType != "bar" {
		return apperr.InvalidParam("pie 和 number 类型面板不支持告警")
	}

	var cfg model.PanelAlertConfig
	if err := json.Unmarshal(panel.AlertConfigJSON, &cfg); err != nil {
		return apperr.InvalidParam(fmt.Sprintf("解析告警配置: %v", err))
	}

	// 查找已有规则
	existing, err := s.alertRuleRepo.GetByPanelID(ctx, panel.ID)
	if err != nil {
		return apperr.Internal("查找面板告警规则", err)
	}

	metricQueryJSON := panel.QueryConfigJSON
	if filterTree := s.loadDashboardFilterTree(ctx, panel.DashboardID); filterTree != nil {
		var queryCfg query.Config
		if err := json.Unmarshal(panel.QueryConfigJSON, &queryCfg); err != nil {
			return apperr.InvalidParam(fmt.Sprintf("解析面板查询配置: %v", err))
		}
		queryCfg.FilterTree = filterTree
		merged, err := json.Marshal(queryCfg)
		if err != nil {
			return apperr.Internal("序列化告警查询配置", err)
		}
		metricQueryJSON = model.JSON(merged)
	}

	if existing == nil {
		// 创建新规则
		rule := &model.AlertRule{
			Name:               panel.Name + " 告警",
			PanelID:            &panel.ID,
			MetricQueryJSON:    metricQueryJSON,
			ConditionOp:        cfg.ConditionOp,
			Threshold:          cfg.Threshold,
			Level:              cfg.Level,
			SilenceMinutes:     cfg.SilenceMinutes,
			EvalIntervalSec:    cfg.EvalIntervalSec,
			PendingDurationSec: cfg.PendingDurationSec,
			TimeWindowSec:      cfg.TimeWindowSec,
			ReduceMode:         cfg.ReduceMode,
			Status:             1,
		}
		if rule.EvalIntervalSec <= 0 {
			rule.EvalIntervalSec = 60
		}
		if rule.SilenceMinutes < 0 {
			rule.SilenceMinutes = 30
		}
		if rule.TimeWindowSec <= 0 {
			rule.TimeWindowSec = 300
		}
		if rule.ReduceMode == "" {
			rule.ReduceMode = "max"
		}
		// avg/sum 与图表数据点割裂，降级为 max
		if rule.ReduceMode != "max" && rule.ReduceMode != "last" {
			rule.ReduceMode = "max"
		}
		if err := s.alertRuleRepo.Create(ctx, rule); err != nil {
			return apperr.Internal("创建面板告警规则", err)
		}
		// 绑定通知渠道
		if len(cfg.ChannelIDs) > 0 {
			return s.alertChannelRepo.SetRuleChannels(ctx, rule.ID, cfg.ChannelIDs)
		}
		return nil
	}

	// 更新已有规则（不动运行态字段）
	existing.ConditionOp = cfg.ConditionOp
	existing.Threshold = cfg.Threshold
	existing.Level = cfg.Level
	existing.SilenceMinutes = cfg.SilenceMinutes
	existing.EvalIntervalSec = cfg.EvalIntervalSec
	existing.PendingDurationSec = cfg.PendingDurationSec
	existing.TimeWindowSec = cfg.TimeWindowSec
	existing.ReduceMode = cfg.ReduceMode
	if existing.TimeWindowSec <= 0 {
		existing.TimeWindowSec = 300
	}
	existing.MetricQueryJSON = metricQueryJSON
	if err := s.alertRuleRepo.Update(ctx, existing); err != nil {
		return apperr.Internal("更新面板告警规则", err)
	}
	// 更新通知渠道绑定
	return s.alertChannelRepo.SetRuleChannels(ctx, existing.ID, cfg.ChannelIDs)
}

// ── 面板告警状态查询 ──

// AlertRuleStateBrief 是告警规则运行态摘要，供前端图表标注使用。
type AlertRuleStateBrief struct {
	RuleID             uint64           `json:"rule_id"`
	LastState          string           `json:"last_state"`
	Threshold          float64          `json:"threshold"`
	ConditionOp        string           `json:"condition_op"`
	Level              string           `json:"level"`
	PendingDurationSec int              `json:"pending_duration_sec"`
	LastEvalAt         *model.LocalTime `json:"last_eval_at,omitempty"`
	ConsecutiveFires   int              `json:"consecutive_fires"`
	ReduceMode         string           `json:"reduce_mode"`
}

// PanelWithAlertState 是面板 + 告警运行态摘要的组合。
type PanelWithAlertState struct {
	model.Panel
	AlertState *AlertRuleStateBrief `json:"alert_state,omitempty"`
}

func (s *DashboardService) GetPanelWithAlertState(ctx context.Context, panelID uint64) (*PanelWithAlertState, error) {
	p, err := s.panelRepo.Get(ctx, panelID)
	if err != nil {
		return nil, apperr.NotFound("panel not found")
	}
	result := &PanelWithAlertState{Panel: *p}

	if len(p.AlertConfigJSON) > 0 && string(p.AlertConfigJSON) != "null" {
		rule, rErr := s.alertRuleRepo.GetByPanelID(ctx, panelID)
		if rErr == nil && rule != nil {
			result.AlertState = &AlertRuleStateBrief{
				RuleID:             rule.ID,
				LastState:          rule.LastState,
				Threshold:          rule.Threshold,
				ConditionOp:        rule.ConditionOp,
				Level:              rule.Level,
				LastEvalAt:         rule.LastEvalAt,
				PendingDurationSec: rule.PendingDurationSec,
				ConsecutiveFires:   rule.ConsecutiveFires,
				ReduceMode:         rule.ReduceMode,
			}
		}
	}
	return result, nil
}

// ── 面板数据查询 ──

// QueryPanel 执行面板查询，合并仪表盘级筛选条件树。
func (s *DashboardService) QueryPanel(ctx context.Context, panelID uint64) ([]map[string]interface{}, error) {
	return s.QueryPanelWithTimeOverride(ctx, panelID, nil)
}

// QueryPanelWithTimeOverride 执行面板查询，支持时间范围覆盖。
// 预聚合查询根据面板 x_dimension.bucket（1m/1h）返回对应粒度数据。
func (s *DashboardService) QueryPanelWithTimeOverride(ctx context.Context, panelID uint64, timeOverride *query.TimeRange) ([]map[string]interface{}, error) {
	p, err := s.panelRepo.Get(ctx, panelID)
	if err != nil {
		return nil, apperr.NotFound("panel not found")
	}

	var cfg query.Config
	if err := json.Unmarshal(p.QueryConfigJSON, &cfg); err != nil {
		return nil, apperr.InvalidParam(fmt.Sprintf("parse query config: %v", err))
	}

	// 时间范围覆盖
	if timeOverride != nil {
		cfg.TimeRange = timeOverride
	}

	// 合并仪表盘级筛选条件树到面板查询配置
	filterTree := s.loadDashboardFilterTree(ctx, p.DashboardID)
	if filterTree != nil {
		cfg.FilterTree = filterTree
	}

	res, err := query.BuildSmartFromConfig(cfg)
	if err != nil {
		return nil, apperr.InvalidParam(fmt.Sprintf("build query: %v", err))
	}
	var rows []map[string]interface{}
	if err := s.db.WithContext(ctx).Raw(res.SQL, res.Args...).Find(&rows).Error; err != nil {
		return nil, apperr.Internal("execute panel query", err)
	}
	// 预聚合查询跳过 ZeroFill：返回稀疏 1m 数据，前端自行填充和聚合
	if !res.PreAgg {
		rows = query.ZeroFill(cfg, rows)
	}
	return rows, nil
}

// loadDashboardFilterTree 解析仪表盘筛选条件为 FilterNode 树。
func (s *DashboardService) loadDashboardFilterTree(ctx context.Context, dashboardID uint64) *query.FilterNode {
	d, err := s.dashRepo.Get(ctx, dashboardID)
	if err != nil || len(d.FiltersJSON) == 0 {
		return nil
	}
	tree, err := query.NormalizeFilterTree(json.RawMessage(d.FiltersJSON))
	if err != nil {
		return nil
	}
	return tree
}

// PanelTemplates 返回预置面板配置模板。
func (s *DashboardService) PanelTemplates() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":       "每日反馈数量趋势",
			"chart_type": "line",
			"query_config": map[string]interface{}{
				"metric":      "count",
				"x_dimension": map[string]interface{}{"field": "original_created_at", "bucket": "1d"},
				"time_range":  map[string]interface{}{"type": "relative", "value": "30d"},
			},
		},
		{
			"name":       "分类分布",
			"chart_type": "pie",
			"query_config": map[string]interface{}{
				"metric":     "count",
				"group_by":   []string{"category"},
				"time_range": map[string]interface{}{"type": "relative", "value": "30d"},
			},
		},
		{
			"name":       "情感倾向分布",
			"chart_type": "bar",
			"query_config": map[string]interface{}{
				"metric":     "count",
				"group_by":   []string{"sentiment"},
				"time_range": map[string]interface{}{"type": "relative", "value": "7d"},
			},
		},
	}
}
