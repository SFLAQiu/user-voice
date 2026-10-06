package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/pkg/response"

	apperr "github.com/feedback/internal/pkg/errors"
)

// Recovery turns panics into 500 responses without leaking stack to clients.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.L.Error("panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())))
				response.Fail(c, apperr.New(apperr.CodeInternal, "internal server error"))
				c.Abort()
			}
		}()
		c.Next()
	}
}

// CORS allows the configured origins; credentials disabled by default.
func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
