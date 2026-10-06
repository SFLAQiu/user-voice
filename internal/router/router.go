package router

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/feedback/internal/classifier"
	"github.com/feedback/internal/config"
	"github.com/feedback/internal/handler"
	"github.com/feedback/internal/middleware"
	"github.com/feedback/internal/pkg/response"
	"github.com/feedback/internal/repository"
	"github.com/feedback/internal/service"
	"github.com/feedback/internal/setup"
)

type Deps struct {
	Cfg                 *config.Config
	Auth                *service.AuthService
	UserRepo            *repository.UserRepo
	AuditRepo           *repository.AuditRepo
	PermissionSvc       *service.PermissionService
	PermissionGroupSvc  *service.PermissionGroupService
	Feedback            *service.FeedbackService
	DataSource          *service.DataSourceService
	EnumConfig          *service.EnumConfigService
	Dashboard           *service.DashboardService
	Alert               *service.AlertService
	ClassifierConfig    *service.ClassifierConfigService
	ClassificationLog   *service.ClassificationLogService
	LLMProvider         *service.LLMProviderService
	LLMConfig           *service.LLMConfigService
	MultiClassifier     *classifier.MultiLLMClassifier
	OnDSChange          func()
	DimSchema           *service.DimensionSchemaService
	BucketUpdateLogRepo *repository.MetricBucketUpdateLogRepo
}

func New(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery(), middleware.CORS(d.Cfg.CORS.AllowOrigins))

	r.GET("/healthz", func(c *gin.Context) { response.OK(c, gin.H{"status": "ok"}) })
	setup.Register(&r.RouterGroup)

	api := r.Group("/api")

	authH := handler.NewAuthHandler(d.Auth, d.UserRepo)
	authH.SetPermissionService(d.PermissionSvc)
	api.POST("/auth/login", authH.Login)

	authed := api.Group("")
	authed.Use(middleware.JWT(d.Auth, d.PermissionSvc), middleware.Audit(d.AuditRepo))

	authed.GET("/auth/me", authH.Me)
	authed.POST("/auth/logout", authH.Logout)
	authed.GET("/auth/permissions", authH.MyPermissions)

	// 权限组管理
	if d.PermissionGroupSvc != nil {
		pgH := handler.NewPermissionGroupHandler(d.PermissionGroupSvc)
		pg := authed.Group("/permission-groups", middleware.RequirePermission("users", "edit"))
		pg.GET("", pgH.List)
		pg.GET("/:id", pgH.Get)
		pg.POST("", pgH.Create)
		pg.PUT("/:id", pgH.Update)
		pg.DELETE("/:id", pgH.Delete)
	}

	// Feedback
	fbH := handler.NewFeedbackHandler(d.Feedback)
	authed.GET("/feedbacks", middleware.RequirePermission("feedback", "view"), fbH.List)
	authed.GET("/feedbacks/:id", middleware.RequirePermission("feedback", "view"), fbH.Detail)
	authed.PUT("/feedbacks/:id/category", middleware.RequirePermission("feedback", "edit"), fbH.UpdateCategory)
	clfLogH := handler.NewClassificationLogHandler(d.ClassificationLog)
	authed.GET("/feedbacks/:id/classification-logs", middleware.RequirePermission("feedback", "view"), clfLogH.ListByFeedback)

	// Data sources
	dsH := handler.NewDataSourceHandler(d.DataSource, d.OnDSChange)
	authed.GET("/data-sources", middleware.RequirePermission("datasources", "view"), dsH.List)
	authed.GET("/data-sources/:id", middleware.RequirePermission("datasources", "view"), dsH.Get)
	authed.GET("/data-sources/:id/sync-logs", middleware.RequirePermission("datasources", "view"), dsH.ListSyncLogs)
	authed.GET("/data-sources/enums", middleware.RequirePermission("datasources", "view"), dsH.AggregatedEnums)
	dsWrite := authed.Group("", middleware.RequirePermission("datasources", "edit"))
	dsWrite.POST("/data-sources", dsH.Create)
	dsWrite.PUT("/data-sources/:id", dsH.Update)
	dsWrite.DELETE("/data-sources/:id", dsH.Delete)
	dsWrite.POST("/data-sources/:id/sync", dsH.Sync)

	// Enum configs
	enumH := handler.NewEnumConfigHandler(d.EnumConfig)
	authed.GET("/enum-configs/enums", middleware.RequirePermission("enum_configs", "view"), enumH.AggregatedEnums)
	authed.GET("/enum-configs", middleware.RequirePermission("enum_configs", "view"), enumH.List)
	enumWrite := authed.Group("", middleware.RequirePermission("enum_configs", "edit"))
	enumWrite.POST("/enum-configs", enumH.Create)
	enumWrite.PUT("/enum-configs/:id", enumH.Update)
	enumWrite.DELETE("/enum-configs/:id", enumH.Delete)

	// Classifier configs
	clfH := handler.NewClassifierConfigHandler(d.ClassifierConfig)
	authed.GET("/classifier-configs/enums", middleware.RequirePermission("classifier_configs", "view"), clfH.AggregatedEnums)
	authed.GET("/classifier-configs/active", middleware.RequirePermission("classifier_configs", "view"), clfH.ActiveConfig)
	authed.GET("/classifier-configs", middleware.RequirePermission("classifier_configs", "view"), clfH.List)
	clfWrite := authed.Group("", middleware.RequirePermission("classifier_configs", "edit"))
	clfWrite.POST("/classifier-configs", clfH.Create)
	clfWrite.PUT("/classifier-configs/:id", clfH.Update)
	clfWrite.DELETE("/classifier-configs/:id", clfH.Delete)

	// Users
	uH := handler.NewUserHandler(d.UserRepo)
	uH.SetPermissionService(d.PermissionSvc)
	authed.GET("/users", middleware.RequirePermission("users", "view"), uH.List)
	userWrite := authed.Group("", middleware.RequirePermission("users", "edit"))
	userWrite.POST("/users", uH.Create)
	userWrite.PUT("/users/:id/status", uH.SetStatus)

	// 用户权限管理
	authed.GET("/users/:id/permissions", middleware.RequirePermission("users", "view"), uH.GetPermissions)
	authed.GET("/users/:id/groups", middleware.RequirePermission("users", "view"), uH.GetGroups)
	authed.PUT("/users/:id/groups", middleware.RequirePermission("users", "edit"), uH.SetGroups)
	authed.GET("/users/:id/overrides", middleware.RequirePermission("users", "view"), uH.GetOverrides)
	authed.PUT("/users/:id/overrides", middleware.RequirePermission("users", "edit"), uH.SetOverrides)

	// Metric dimension configs
	if d.DimSchema != nil {
		dimH := handler.NewMetricDimensionHandler(d.DimSchema, d.BucketUpdateLogRepo)
		authed.GET("/metric-dimensions", middleware.RequirePermission("metric_dimensions", "view"), dimH.List)
		authed.GET("/metric-dimensions/rebuild-status", middleware.RequirePermission("metric_dimensions", "view"), dimH.RebuildStatus)
		authed.GET("/metric-dimensions/update-logs", middleware.RequirePermission("metric_dimensions", "view"), dimH.ListUpdateLogs)
		dimWrite := authed.Group("", middleware.RequirePermission("metric_dimensions", "edit"))
		dimWrite.POST("/metric-dimensions", dimH.Add)
		dimWrite.DELETE("/metric-dimensions/:field", dimH.Remove)
		dimWrite.POST("/metric-dimensions/backfill", dimH.TriggerBackfill)
	}

	// Dashboards
	if d.Dashboard != nil {
		dashH := handler.NewDashboardHandler(d.Dashboard)
		authed.GET("/dashboards", middleware.RequirePermission("dashboard", "view"), dashH.List)
		authed.GET("/dashboards/:id", middleware.RequirePermission("dashboard", "view"), dashH.Get)
		authed.GET("/dashboards/:id/has-alert-rules", middleware.RequirePermission("dashboard", "view"), dashH.HasAlertRules)
			authed.GET("/dashboards/:id/delete-info", middleware.RequirePermission("dashboard", "view"), dashH.GetDeleteInfo)
		authed.GET("/panels/:panel_id/query", middleware.RequirePermission("dashboard", "view"), dashH.QueryPanel)
		authed.GET("/panels/:panel_id/alert-state", middleware.RequirePermission("dashboard", "view"), dashH.GetPanelAlertState)
		authed.GET("/panels/templates", middleware.RequirePermission("dashboard", "view"), dashH.PanelTemplates)
		dashWrite := authed.Group("", middleware.RequirePermission("dashboard", "edit"))
		dashWrite.POST("/dashboards", dashH.Create)
		dashWrite.PUT("/dashboards/:id", dashH.Update)
		dashWrite.DELETE("/dashboards/:id", dashH.Delete)
		dashWrite.POST("/dashboards/:id/panels", dashH.CreatePanel)
		dashWrite.PUT("/dashboards/:id/panels/:panel_id", dashH.UpdatePanel)
		dashWrite.DELETE("/dashboards/:id/panels/:panel_id", dashH.DeletePanel)
	}

	// 通知渠道 & 告警规则 & 告警记录
	if d.Alert != nil {
		alertH := handler.NewAlertHandler(d.Alert)

		authed.GET("/notification-channels", middleware.RequirePermission("notification_channels", "view"), alertH.ListChannels)
		ncWrite := authed.Group("", middleware.RequirePermission("notification_channels", "edit"))
		ncWrite.POST("/notification-channels", alertH.CreateChannel)
		ncWrite.PUT("/notification-channels/:id", alertH.UpdateChannel)
		ncWrite.DELETE("/notification-channels/:id", alertH.DeleteChannel)
		ncWrite.POST("/notification-channels/:id/test", alertH.TestNotify)

		authed.GET("/alert/rules", middleware.RequirePermission("alert", "view"), alertH.ListRules)
		authed.GET("/alert/records", middleware.RequirePermission("alert", "view"), alertH.ListRecords)
		alertWrite := authed.Group("", middleware.RequirePermission("alert", "edit"))
		alertWrite.POST("/alert/rules", alertH.CreateRule)
		alertWrite.PUT("/alert/rules/:id/status", alertH.ToggleRuleStatus)
		alertWrite.PUT("/alert/rules/:id", alertH.UpdateRule)
		alertWrite.DELETE("/alert/rules/:id", alertH.DeleteRule)
		alertWrite.PUT("/alert/rules/:id/channels", alertH.SetRuleChannels)
	}

	// LLM providers & configs
	if d.LLMProvider != nil && d.LLMConfig != nil && d.MultiClassifier != nil {
		providerH := handler.NewLLMProviderHandler(d.LLMProvider)
		llmConfigH := handler.NewLLMConfigHandler(d.LLMConfig)

		authed.GET("/llm-providers/status", middleware.RequirePermission("llm_providers", "view"), func(c *gin.Context) {
			response.OK(c, d.MultiClassifier.CircuitBreakerStatus())
		})
		authed.GET("/llm-providers", middleware.RequirePermission("llm_providers", "view"), providerH.List)
		llmWrite := authed.Group("", middleware.RequirePermission("llm_providers", "edit"))
		llmWrite.POST("/llm-providers", providerH.Create)
		llmWrite.PUT("/llm-providers/:id", providerH.Update)
		llmWrite.DELETE("/llm-providers/:id", providerH.Delete)

		llmWrite.POST("/llm-providers/test", providerH.TestNew)
		llmWrite.POST("/llm-providers/:id/test", providerH.TestExisting)

		authed.GET("/llm-configs/circuit-breaker", middleware.RequirePermission("llm_providers", "view"), llmConfigH.CircuitBreakerConfig)
		authed.GET("/llm-configs/retry", middleware.RequirePermission("llm_providers", "view"), llmConfigH.RetryConfig)
		authed.GET("/llm-configs", middleware.RequirePermission("llm_providers", "view"), llmConfigH.List)
		llmCfgWrite := authed.Group("", middleware.RequirePermission("llm_providers", "edit"))
		llmCfgWrite.POST("/llm-configs", llmConfigH.Create)
		llmCfgWrite.PUT("/llm-configs/:id", llmConfigH.Update)
		llmCfgWrite.DELETE("/llm-configs/:id", llmConfigH.Delete)
	}

	webDist := "web/dist"
	if _, err := os.Stat(webDist); err == nil {
		r.Static("/assets", webDist+"/assets")
		r.StaticFile("/favicon.svg", webDist+"/favicon.svg")
		r.NoRoute(func(c *gin.Context) {
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"code": 40400, "message": "not found"})
				return
			}
			// 尝试直接返回 dist 根目录下的静态文件（public 产物）
			filePath := webDist + c.Request.URL.Path
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				c.File(filePath)
				return
			}
			c.File(webDist + "/index.html")
		})
	}

	return r
}
