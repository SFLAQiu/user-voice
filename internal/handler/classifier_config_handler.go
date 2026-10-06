package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type ClassifierConfigHandler struct {
	svc *service.ClassifierConfigService
}

func NewClassifierConfigHandler(svc *service.ClassifierConfigService) *ClassifierConfigHandler {
	return &ClassifierConfigHandler{svc: svc}
}

func (h *ClassifierConfigHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *ClassifierConfigHandler) Create(c *gin.Context) {
	var in service.CreateClassifierConfigInput
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

func (h *ClassifierConfigHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.UpdateClassifierConfigInput
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

func (h *ClassifierConfigHandler) Delete(c *gin.Context) {
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

func (h *ClassifierConfigHandler) ActiveConfig(c *gin.Context) {
	cfg, err := h.svc.GetActiveConfig(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, cfg)
}

func (h *ClassifierConfigHandler) AggregatedEnums(c *gin.Context) {
	enums, err := h.svc.AggregatedEnums(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, enums)
}