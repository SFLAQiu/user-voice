package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type LLMConfigHandler struct {
	svc *service.LLMConfigService
}

func NewLLMConfigHandler(svc *service.LLMConfigService) *LLMConfigHandler {
	return &LLMConfigHandler{svc: svc}
}

func (h *LLMConfigHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *LLMConfigHandler) Create(c *gin.Context) {
	var in service.CreateLLMConfigInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	cfg, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cfg)
}

func (h *LLMConfigHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.UpdateLLMConfigInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	cfg, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cfg)
}

func (h *LLMConfigHandler) Delete(c *gin.Context) {
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

func (h *LLMConfigHandler) CircuitBreakerConfig(c *gin.Context) {
	cfg, err := h.svc.GetCircuitBreakerConfig(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cfg)
}

func (h *LLMConfigHandler) RetryConfig(c *gin.Context) {
	cfg, err := h.svc.GetRetryConfig(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cfg)
}