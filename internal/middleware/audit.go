package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/feedback/internal/model"
	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/repository"
)

// Audit logs all non-GET requests with a body snippet (capped).
func Audit(repo *repository.AuditRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || strings.HasPrefix(c.FullPath(), "/healthz") {
			c.Next()
			return
		}
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		c.Next()

		var uid *uint64
		if v, ok := c.Get(CtxUserID); ok {
			if id, ok2 := v.(uint64); ok2 {
				uid = &id
			}
		}
		username, _ := c.Get(CtxUsername)
		us, _ := username.(string)

		detail := map[string]any{
			"status": c.Writer.Status(),
			"path":   c.FullPath(),
			"method": c.Request.Method,
		}
		if len(body) > 0 && len(body) <= 2048 {
			var pretty any
			if err := json.Unmarshal(body, &pretty); err == nil {
				detail["body"] = pretty
			}
		}
		detJSON, _ := json.Marshal(detail)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := repo.Create(ctx, &model.AuditLog{
			UserID:     uid,
			Username:   us,
			Action:     c.Request.Method + " " + c.FullPath(),
			Resource:   c.FullPath(),
			IP:         c.ClientIP(),
			DetailJSON: detJSON,
			CreatedAt:  time.Now(),
		})
		if err != nil {
			logger.L.Warn("audit write failed", zap.Error(err))
		}
	}
}
