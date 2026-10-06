package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/feedback/internal/middleware"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/query"
	"github.com/feedback/internal/service"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *DashboardHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	d, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

func (h *DashboardHandler) Create(c *gin.Context) {
	var in service.CreateDashboardInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	uid, _ := c.Get(middleware.CtxUserID)
	userID, _ := uid.(uint64)
	d, err := h.svc.Create(c.Request.Context(), in, userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

func (h *DashboardHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.UpdateDashboardInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	d, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, d)
}

func (h *DashboardHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// Panel handlers

func (h *DashboardHandler) CreatePanel(c *gin.Context) {
	dashID, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.CreatePanelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	p, err := h.svc.CreatePanel(c.Request.Context(), dashID, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *DashboardHandler) UpdatePanel(c *gin.Context) {
	id, err := parseUintParam(c, "panel_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.UpdatePanelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	p, err := h.svc.UpdatePanel(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *DashboardHandler) DeletePanel(c *gin.Context) {
	id, err := parseUintParam(c, "panel_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.DeletePanel(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *DashboardHandler) QueryPanel(c *gin.Context) {
	id, err := parseUintParam(c, "panel_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	overrideTimeRange := parseTimeRangeOverride(c)
	rows, err := h.svc.QueryPanelWithTimeOverride(c.Request.Context(), id, overrideTimeRange)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

// parseTimeRangeOverride 从 query params 解析可选的时间范围覆盖参数。
// 参数名：time_range_type, time_range_value, time_range_start, time_range_end
func parseTimeRangeOverride(c *gin.Context) *query.TimeRange {
	trType := c.Query("time_range_type")
	if trType == "" {
		return nil
	}
	tr := &query.TimeRange{Type: trType}
	switch trType {
	case "relative":
		tr.Value = c.Query("time_range_value")
		if tr.Value == "" {
			tr.Value = "1d"
		}
	case "absolute":
		tr.Start = c.Query("time_range_start")
		tr.End = c.Query("time_range_end")
	}
	return tr
}

func (h *DashboardHandler) PanelTemplates(c *gin.Context) {
	response.OK(c, h.svc.PanelTemplates())
}

func (h *DashboardHandler) GetPanelAlertState(c *gin.Context) {
	id, err := parseUintParam(c, "panel_id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	state, err := h.svc.GetPanelWithAlertState(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, state)
}

func (h *DashboardHandler) HasAlertRules(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	has, err := h.svc.HasPanelAlertRules(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"has_alert_rules": has})
}

func (h *DashboardHandler) GetDeleteInfo(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	info, err := h.svc.GetDeleteInfo(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, info)
}
