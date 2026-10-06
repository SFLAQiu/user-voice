package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/feedback/internal/middleware"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/repository"
	"github.com/feedback/internal/service"
)

type UserHandler struct {
	users   *repository.UserRepo
	permSvc *service.PermissionService
}

func NewUserHandler(users *repository.UserRepo) *UserHandler {
	return &UserHandler{users: users}
}

// SetPermissionService 注入权限服务（用于用户权限管理接口）。
func (h *UserHandler) SetPermissionService(permSvc *service.PermissionService) {
	h.permSvc = permSvc
}

func (h *UserHandler) List(c *gin.Context) {
	rows, err := h.users.List(c.Request.Context())
	if err != nil {
		response.Fail(c, apperr.Internal("list users", err))
		return
	}
	response.OK(c, rows)
}

type createUserReq struct {
	Username string   `json:"username" binding:"required,min=2,max=64"`
	Password string   `json:"password" binding:"required,min=6,max=128"`
	Role     string   `json:"role" binding:"required,oneof=admin viewer"`
	GroupIDs []uint64 `json:"group_ids"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	existing, err := h.users.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		response.Fail(c, apperr.Internal("lookup", err))
		return
	}
	if existing != nil {
		response.Fail(c, apperr.InvalidParam("username taken"))
		return
	}
	hash, err := service.HashPassword(req.Password)
	if err != nil {
		response.Fail(c, apperr.Internal("hash", err))
		return
	}
	u := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		Role:         req.Role,
		Status:       model.UserStatusEnabled,
	}
	if err := h.users.Create(c.Request.Context(), u); err != nil {
		response.Fail(c, apperr.Internal("create user", err))
		return
	}
	// 分配权限组
	if len(req.GroupIDs) > 0 && h.permSvc != nil {
		_ = h.users.SetGroups(c.Request.Context(), u.ID, req.GroupIDs)
	}
	response.OK(c, u)
}

type setStatusReq struct {
	Status int `json:"status" binding:"oneof=0 1"`
}

func (h *UserHandler) SetStatus(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req setStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	// Prevent self lockout.
	if cur, _ := c.Get(middleware.CtxUserID); cur != nil {
		if uid, ok := cur.(uint64); ok && uid == id && req.Status == model.UserStatusDisabled {
			response.Fail(c, apperr.InvalidParam("cannot disable yourself"))
			return
		}
	}
	if err := h.users.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Fail(c, apperr.Internal("set status", err))
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// GetPermissions 获取用户完整权限信息。
func (h *UserHandler) GetPermissions(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if h.permSvc == nil {
		response.Fail(c, apperr.Internal("permission service not available", nil))
		return
	}
	groups, err := h.users.FindGroupsByUserID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("load groups", err))
		return
	}
	effective, err := h.permSvc.GetEffectivePermissions(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("load effective permissions", err))
		return
	}
	overrides, err := h.users.FindOverridesByUserID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("load overrides", err))
		return
	}
	response.OK(c, gin.H{
		"groups":    groups,
		"effective": effective,
		"overrides": overrides,
	})
}

// GetGroups 获取用户所属的权限组。
func (h *UserHandler) GetGroups(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	groups, err := h.users.FindGroupsByUserID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("load groups", err))
		return
	}
	response.OK(c, groups)
}

type setGroupsReq struct {
	GroupIDs []uint64 `json:"group_ids"`
}

// SetGroups 设置用户的权限组（覆盖式）。
func (h *UserHandler) SetGroups(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req setGroupsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	if err := h.users.SetGroups(c.Request.Context(), id, req.GroupIDs); err != nil {
		response.Fail(c, apperr.Internal("set groups", err))
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// GetOverrides 获取用户权限覆盖列表。
func (h *UserHandler) GetOverrides(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	overrides, err := h.users.FindOverridesByUserID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("load overrides", err))
		return
	}
	response.OK(c, overrides)
}

type setOverridesReq struct {
	Overrides []model.UserPermissionOverride `json:"overrides"`
}

// SetOverrides 设置用户权限覆盖（覆盖式）。
func (h *UserHandler) SetOverrides(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req setOverridesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	if h.permSvc == nil {
		response.Fail(c, apperr.Internal("permission service not available", nil))
		return
	}
	if err := h.permSvc.OverrideRepo().Replace(c.Request.Context(), id, req.Overrides); err != nil {
		response.Fail(c, apperr.Internal("set overrides", err))
		return
	}
	response.OK(c, gin.H{"ok": true})
}
