package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/query"
	"github.com/feedback/internal/service"
)

type AlertHandler struct {
	svc *service.AlertService
}

func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// --- Rules ---

func (h *AlertHandler) ListRules(c *gin.Context) {
	filterType := c.Query("type") // "custom" | "panel" | 空=全部
	rules, err := h.svc.ListRules(c.Request.Context(), filterType)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rules)
}

func (h *AlertHandler) CreateRule(c *gin.Context) {
	var in service.AlertRuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	r, err := h.svc.CreateRule(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, r)
}

func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.AlertRuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	r, err := h.svc.UpdateRule(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, r)
}

func (h *AlertHandler) ToggleRuleStatus(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var body struct {
		Status *int `json:"status" binding:"required,oneof=0 1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	r, err := h.svc.ToggleRuleStatus(c.Request.Context(), id, *body.Status)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, r)
}

func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.DeleteRule(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// --- Channels ---

func (h *AlertHandler) ListChannels(c *gin.Context) {
	channels, err := h.svc.ListChannels(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, channels)
}

type channelInput struct {
	Name       string `json:"name" binding:"required,max=128"`
	Type       string `json:"type" binding:"required,oneof=wecom feishu dingtalk webhook"`
	WebhookURL string `json:"webhook_url" binding:"required,max=512"`
	Secret     string `json:"secret" binding:"max=255"`
	Status     int    `json:"status"`
}

func (h *AlertHandler) CreateChannel(c *gin.Context) {
	var in channelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	ch, err := h.svc.CreateChannel(c.Request.Context(), service.AlertChannelInput(in))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ch)
}

func (h *AlertHandler) UpdateChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in channelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	ch, err := h.svc.UpdateChannel(c.Request.Context(), id, service.AlertChannelInput(in))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ch)
}

func (h *AlertHandler) DeleteChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.DeleteChannel(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// --- Records ---

func (h *AlertHandler) ListRecords(c *gin.Context) {
	ruleID := uint64(0)
	if s := c.Query("rule_id"); s != "" {
		if n, err := strconv.ParseUint(s, 10, 64); err == nil {
			ruleID = n
		}
	}
	limit := parseIntQ(c, "limit", 50)

	// 时间范围过滤（本地时间字符串或 RFC3339 均可）
	startAtStr := c.Query("start_at")
	endAtStr := c.Query("end_at")
	if startAtStr != "" && endAtStr != "" {
		startAt, sErr := query.ParseAbsoluteTime(startAtStr)
		endAt, eErr := query.ParseAbsoluteTime(endAtStr)
		if sErr == nil && eErr == nil && endAt.After(startAt) {
			records, err := h.svc.ListRecordsByTimeRange(c.Request.Context(), ruleID, startAt, endAt, limit)
			if err != nil {
				response.Fail(c, err)
				return
			}
			response.OK(c, records)
			return
		}
	}

	records, err := h.svc.ListRecords(c.Request.Context(), ruleID, limit)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, records)
}

// SetRuleChannels binds channels to a rule.
func (h *AlertHandler) SetRuleChannels(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var body struct {
		ChannelIDs []uint64 `json:"channel_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	if err := h.svc.SetRuleChannels(c.Request.Context(), id, body.ChannelIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// TestNotify sends a test message to a channel.
func (h *AlertHandler) TestNotify(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = json.Unmarshal([]byte(`{}`), &body)
	}
	if body.Message == "" {
		body.Message = "测试通知：告警平台连接正常"
	}
	if err := h.svc.TestNotify(c.Request.Context(), id, body.Message); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
