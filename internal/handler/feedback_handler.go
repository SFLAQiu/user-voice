package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type FeedbackHandler struct {
	svc *service.FeedbackService
}

func NewFeedbackHandler(svc *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{svc: svc}
}

func (h *FeedbackHandler) List(c *gin.Context) {
	in := service.ListFeedbackInput{
		StartTime:  c.Query("start_time"),
		EndTime:    c.Query("end_time"),
		AppID:      c.Query("app_id"),
		Platform:   c.Query("platform"),
		PlatformID: c.Query("platform_id"),
		Category:       c.Query("category"),
		BusinessModule: c.Query("business_module"),
		Sentiment:      c.Query("sentiment"),
		AppVersion: c.Query("app_version"),
		UserMode:   c.Query("user_mode"),
		UserID:     c.Query("user_id"),
		Keyword:    c.Query("keyword"),
		OrderBy:    c.Query("order_by"),
		OrderDir:   c.Query("order_dir"),
	}
	in.Page = parseIntQ(c, "page", 1)
	in.PageSize = parseIntQ(c, "page_size", 20)

	rows, total, err := h.svc.List(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.Page{
		List:     rows,
		Total:    total,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
}

func (h *FeedbackHandler) Detail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	f, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, f)
}

type updateCategoryReq struct {
	Category       string `json:"category" binding:"required,max=64"`
	BusinessModule string `json:"business_module" binding:"max=64"`
}

func (h *FeedbackHandler) UpdateCategory(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req updateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	if err := h.svc.UpdateCategory(c.Request.Context(), id, req.Category, req.BusinessModule); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func parseIntQ(c *gin.Context, key string, def int) int {
	s := c.Query(key)
	if s == "" {
		return def
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return def
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}
