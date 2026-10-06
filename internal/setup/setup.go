// Package setup 网页初始化向导：未初始化时收集数据库、密钥、管理员账号，写盘并完成种子。
package setup

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/feedback/internal/config"
	"github.com/feedback/internal/db"
	"github.com/feedback/internal/pkg/response"
)

// ConfigPath 返回配置文件路径；独立函数便于复用与测试。
func ConfigPath() string { return "configs/config.yaml" }

// Done 报告初始化是否已完成（配置文件存在即视为完成）。
func Done() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

// Register 挂载初始化向导路由（/api/setup/*，无需鉴权）。
func Register(r gin.IRouter) {
	h := &handler{}
	g := r.Group("/api/setup")
	g.GET("/status", h.status)
}

// RegisterFull 挂载完整向导路由（仅未初始化的向导模式使用）。
func RegisterFull(r gin.IRouter) {
	Register(r)
	h := &handler{}
	g := r.Group("/api/setup")
	g.POST("/test-db", h.testDB)
	g.GET("/secrets", h.genSecrets)
	g.POST("/finish", h.finish)
}

type handler struct{}

func (h *handler) status(c *gin.Context) {
	response.OK(c, gin.H{"initialized": Done()})
}

type dbReq struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
	Name     string `json:"name" binding:"required"`
}

func (r dbReq) dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		r.User, r.Password, r.Host, r.Port, r.Name)
}

// testDB 测试数据库连通并顺带执行迁移（等价于真正初始化时的动作，成功即说明该库可用）。
func (h *handler) testDB(c *gin.Context) {
	var req dbReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := db.Migrate(req.dsn()); err != nil {
		response.OK(c, gin.H{"ok": false, "message": err.Error()})
		return
	}
	response.OK(c, gin.H{"ok": true, "message": "连接成功，表结构已就绪"})
}

// genSecrets 生成随机 JWT secret 与 AES key，前端预填可改。
func (h *handler) genSecrets(c *gin.Context) {
	response.OK(c, gin.H{"jwt_secret": randHex(32), "encryption_key": randBase64(32)})
}

type finishReq struct {
	Host          string `json:"host" binding:"required"`
	Port          int    `json:"port" binding:"required,min=1,max=65535"`
	User          string `json:"user" binding:"required"`
	Password      string `json:"password"`
	Name          string `json:"name" binding:"required"`
	JWTSecret     string `json:"jwt_secret" binding:"required,min=16"`
	EncryptionKey string `json:"encryption_key" binding:"required"`
	AdminUsername string `json:"admin_username" binding:"required,min=2,max=64"`
	AdminPassword string `json:"admin_password" binding:"required,min=6,max=128"`
}

// finish 校验并写出配置文件，随后服务重启加载正式配置（部署方式决定重启手段）。
func (h *handler) finish(c *gin.Context) {
	var req finishReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	cfgPath := ConfigPath()
	if err := writeConfig(req, cfgPath); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"config_path": cfgPath, "message": "初始化完成，请重启服务生效"})
}

// writeConfig 校验迁移与配置合法性后落盘。
func writeConfig(req finishReq, cfgPath string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		req.User, req.Password, req.Host, req.Port, req.Name)
	// 写盘前先真实验证：迁移成功才落盘，避免写出连不上的配置
	if err := db.Migrate(dsn); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	if err := config.Validate(req.JWTSecret, req.EncryptionKey); err != nil {
		return fmt.Errorf("配置校验失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cfgPath, []byte(renderYAML(req, dsn)), 0o600)
}

func renderYAML(req finishReq, dsn string) string {
	return fmt.Sprintf(`server:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

database:
  dsn: "%s"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 1h

jwt:
  secret: "%s"
  expire: 8h

encryption:
  # 32-byte base64 (openssl rand -base64 32)
  key: "%s"

admin:
  username: "%s"
  password: "%s"

login_limit:
  max_attempts: 5
  window: 5m

log:
  level: "info"
  file: ""

cors:
  allow_origins:
    - "http://localhost:5173"
`, dsn, req.JWTSecret, req.EncryptionKey, req.AdminUsername, req.AdminPassword)
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func randBase64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
