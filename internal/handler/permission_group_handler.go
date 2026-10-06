package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

type PermissionGroupHandler struct {
	svc *service.PermissionGroupService
}

func NewPermissionGroupHandler(svc *service.PermissionGroupService) *PermissionGroupHandler {
	return &PermissionGroupHandler{svc: svc}
}

func (h *PermissionGroupHandler) List(c *gin.Context) {
	rows, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		response.Fail(c, apperr.Internal("list permission groups", err))
		return
	}
	response.OK(c, rows)
}

func (h *PermissionGroupHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	g, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("get permission group", err))
		return
	}
	if g == nil {
		response.Fail(c, apperr.NotFound("permission group not found"))
		return
	}
	response.OK(c, g)
}

type createPermissionGroupReq struct {
	Name        string          `json:"name" binding:"required,min=1,max=64"`
	Description string          `json:"description" binding:"max=255"`
	Permissions json.RawMessage `json:"permissions" binding:"required"`
}

func (h *PermissionGroupHandler) Create(c *gin.Context) {
	var req createPermissionGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	g, err := h.svc.Create(c.Request.Context(), req.Name, req.Description, model.JSON(req.Permissions))
	if err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	response.OK(c, g)
}

type updatePermissionGroupReq struct {
	Name        string          `json:"name" binding:"required,min=1,max=64"`
	Description string          `json:"description" binding:"max=255"`
	Permissions json.RawMessage `json:"permissions" binding:"required"`
}

func (h *PermissionGroupHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req updatePermissionGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	g, err := h.svc.Update(c.Request.Context(), id, req.Name, req.Description, model.JSON(req.Permissions))
	if err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	response.OK(c, g)
}

func (h *PermissionGroupHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, apperr.Internal("delete permission group", err))
		return
	}
	response.OK(c, gin.H{"ok": true})
}
