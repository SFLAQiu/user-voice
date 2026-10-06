package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/service"
)

const (
	CtxUserID      = "user_id"
	CtxUsername    = "username"
	CtxRole        = "role"
	CtxPermissions = "permissions"
)

// JWT extracts and verifies the Authorization header.
func JWT(auth *service.AuthService, permSvc *service.PermissionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			response.Fail(c, apperr.Unauthorized("missing token"))
			c.Abort()
			return
		}
		claims, err := auth.ParseToken(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			response.Fail(c, err)
			c.Abort()
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxRole, claims.Role)

		// 计算并缓存用户有效权限到 Context
		if permSvc != nil {
			perms, err := permSvc.GetEffectivePermissions(c.Request.Context(), claims.UserID)
			if err == nil {
				c.Set(CtxPermissions, perms)
			}
		}

		c.Next()
	}
}

// RequireRole enforces a role match (admin can pass anywhere; viewer only matches "viewer").
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		rs, _ := role.(string)
		if rs == "admin" {
			c.Next()
			return
		}
		if _, ok := allowed[rs]; !ok {
			response.Fail(c, apperr.Forbidden("权限不足"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequirePermission 检查用户是否拥有指定页面+操作的权限。
// 超管（role=admin）直接通过。
func RequirePermission(pageKey, permKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		if rs, ok := role.(string); ok && rs == "admin" {
			c.Next()
			return
		}
		perms, _ := c.Get(CtxPermissions)
		pmap, _ := perms.(map[string][]string)
		for _, p := range pmap[pageKey] {
			if p == permKey {
				c.Next()
				return
			}
		}
		response.Fail(c, apperr.Forbidden("权限不足"))
		c.Abort()
	}
}
