package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/feedback/internal/middleware"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/repository"
	"github.com/feedback/internal/service"
)

type AuthHandler struct {
	svc     *service.AuthService
	users   *repository.UserRepo
	permSvc *service.PermissionService
}

func NewAuthHandler(svc *service.AuthService, users *repository.UserRepo) *AuthHandler {
	return &AuthHandler{svc: svc, users: users}
}

// SetPermissionService 注入权限服务（用于 MyPermissions 接口）。
func (h *AuthHandler) SetPermissionService(permSvc *service.PermissionService) {
	h.permSvc = permSvc
}

type loginReq struct {
	Username string `json:"username" binding:"required,min=2,max=64"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperr.InvalidParam(err.Error()))
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, res)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Stateless JWT: client just drops the token. Audit logs the event.
	response.OK(c, gin.H{"ok": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get(middleware.CtxUserID)
	id, _ := uid.(uint64)
	u, err := h.users.FindByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("load user", err))
		return
	}
	if u == nil {
		response.Fail(c, apperr.NotFound("user not found"))
		return
	}
	response.OK(c, u)
}

// MyPermissions 返回当前登录用户的有效权限。
func (h *AuthHandler) MyPermissions(c *gin.Context) {
	uid, _ := c.Get(middleware.CtxUserID)
	id, _ := uid.(uint64)
	if h.permSvc == nil {
		response.OK(c, gin.H{"permissions": gin.H{}})
		return
	}
	perms, err := h.permSvc.GetEffectivePermissions(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, apperr.Internal("load permissions", err))
		return
	}
	if perms == nil {
		perms = map[string][]string{}
	}
	response.OK(c, gin.H{
		"permissions": perms,
	})
}
