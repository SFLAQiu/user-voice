package handler

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type EnumConfigHandler struct {
	svc *service.EnumConfigService
}

func NewEnumConfigHandler(svc *service.EnumConfigService) *EnumConfigHandler {
	return &EnumConfigHandler{svc: svc}
}

func (h *EnumConfigHandler) List(c *gin.Context) {
	rows, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *EnumConfigHandler) Create(c *gin.Context) {
	var in service.CreateEnumConfigInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	ec, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ec)
}

func (h *EnumConfigHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var in service.UpdateEnumConfigInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	ec, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ec)
}

func (h *EnumConfigHandler) Delete(c *gin.Context) {
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

func (h *EnumConfigHandler) AggregatedEnums(c *gin.Context) {
	enums, err := h.svc.AggregatedMap(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, enums)
}