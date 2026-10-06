package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/feedback/internal/alert"
	"github.com/feedback/internal/classifier"
	"github.com/feedback/internal/config"
	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/datasource"
	"github.com/feedback/internal/datasource/httpapi"
	"github.com/feedback/internal/datasource/mysql"
	"github.com/feedback/internal/db"
	"github.com/feedback/internal/middleware"
	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/repository"
	"github.com/feedback/internal/router"
	"github.com/feedback/internal/scheduler"
	"github.com/feedback/internal/service"
	"github.com/feedback/internal/setup"
	"github.com/gin-gonic/gin"
)

// startSetupServer 以最小模式监听 :8080，仅暴露初始化向导接口，完成后退出（由部署方拉起进入正式模式）。
func startSetupServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery())
	setup.RegisterFull(r)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "setup-mode"}) })
	// 配置已落盘但服务未重启时，业务路由不存在；给出明确提示而非笼统 404
	r.NoRoute(func(c *gin.Context) {
		if setup.Done() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": 50300, "message": "初始化已完成，请重启服务后刷新页面"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 40400, "message": "not found"})
	})

	srv := &http.Server{Addr: ":8080", Handler: r, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
	go func() {
		log.Printf("[setup] 未检测到配置文件，初始化向导已启动：请访问 http://<host>:8080 完成初始化")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("setup server listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func main() {
	cfgPath := flag.String("config", "", "配置文件路径（默认 configs/config.yaml）")
	flag.Parse()

	resolvedCfgPath := *cfgPath
	if resolvedCfgPath == "" {
		resolvedCfgPath = setup.ConfigPath()
	}

	// 未初始化：以最小模式启动，只提供网页初始化向导 API
	if !setup.Done() {
		startSetupServer()
		return
	}

	cfg, err := config.Load(resolvedCfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := logger.Init(cfg.Log.Level, cfg.Log.File); err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	cipher, err := crypto.New(cfg.Encryption.Key)
	if err != nil {
		logger.L.Fatal("init cipher", zap.Error(err))
	}

	gdb, err := db.Open(cfg.Database)
	if err != nil {
		logger.L.Fatal("open db", zap.Error(err))
	}

	// Repositories
	userRepo := repository.NewUserRepo(gdb)
	loginRepo := repository.NewLoginAttemptRepo(gdb)
	auditRepo := repository.NewAuditRepo(gdb)
	dsRepo := repository.NewDataSourceRepo(gdb)
	cursorRepo := repository.NewSyncCursorRepo(gdb)
	fbRepo := repository.NewFeedbackRepo(gdb)
	syncLogRepo := repository.NewSyncLogRepo(gdb)
	enumConfigRepo := repository.NewEnumConfigRepo(gdb)
	clfConfigRepo := repository.NewClassifierConfigRepo(gdb)
	clfLogRepo := repository.NewClassificationLogRepo(gdb)
	llmProviderRepo := repository.NewLLMProviderRepo(gdb)
	llmConfigRepo := repository.NewLLMConfigRepo(gdb)
	alertRuleRepo := repository.NewAlertRuleRepo(gdb)
	alertChannelRepo := repository.NewAlertChannelRepo(gdb)
	alertRecordRepo := repository.NewAlertRecordRepo(gdb)
	permGroupRepo := repository.NewPermissionGroupRepo(gdb)
	permOverrideRepo := repository.NewUserPermissionOverridesRepo(gdb)

	// Metric time bucket repos
	metricBucketRepo := repository.NewMetricTimeBucketRepo(gdb)
	metricDimConfigRepo := repository.NewMetricDimensionConfigRepo(gdb)
	metricBucketUpdateLogRepo := repository.NewMetricBucketUpdateLogRepo(gdb)

	// Services
	authSvc := service.NewAuthService(userRepo, loginRepo, cfg.JWT, cfg.LoginLimit)
	permSvc := service.NewPermissionService(permGroupRepo, userRepo, permOverrideRepo)
	permGroupSvc := service.NewPermissionGroupService(permGroupRepo)

	// Metric bucket service (time grid)
	metricBucketSvc := service.NewMetricBucketService(metricBucketRepo, metricBucketUpdateLogRepo, gdb)
	metricBucketSvc.InitVersion(context.Background())
	fbSvc := service.NewFeedbackService(fbRepo, metricBucketSvc)
	dimSchemaSvc := service.NewDimensionSchemaService(metricDimConfigRepo, metricBucketRepo, metricBucketSvc, metricBucketUpdateLogRepo, gdb)

	// Plugins
	reg := datasource.NewRegistry()
	reg.Register(httpapi.New(cipher))
	reg.Register(mysql.New(cipher))

	enumConfigSvc := service.NewEnumConfigService(enumConfigRepo)
	clfConfigSvc := service.NewClassifierConfigService(clfConfigRepo)
	clfLogSvc := service.NewClassificationLogService(clfLogRepo, llmProviderRepo)
	llmProviderSvc := service.NewLLMProviderService(llmProviderRepo, cipher)
	llmConfigSvc := service.NewLLMConfigService(llmConfigRepo)

	dsSvc := service.NewDataSourceService(dsRepo, cursorRepo, fbRepo, syncLogRepo, enumConfigSvc, reg, cipher, metricBucketSvc)

	// Multi-LLM Classifier with circuit breaker and retry
	cbSet := classifier.NewCircuitBreakerSet(classifier.DefaultCircuitBreakerConfig())
	multiClf := classifier.NewMultiLLMClassifier(cbSet, classifier.DefaultRetryConfig())
	clfSvc := service.NewClassifierService(fbRepo, multiClf, llmProviderSvc, llmConfigSvc, clfConfigSvc, clfLogSvc, cipher, metricBucketSvc)
	clfCtx, clfCancel := context.WithCancel(context.Background())
	defer clfCancel()

	clfSvc.InitProviders(clfCtx)
	clfSvc.Start(clfCtx)
	defer clfSvc.Stop()

	// Alert & Dashboard repos
	dashRepo := repository.NewDashboardRepo(gdb)
	panelRepo := repository.NewPanelRepo(gdb)

	// Dashboard
	dashSvc := service.NewDashboardService(dashRepo, panelRepo, alertRuleRepo, alertChannelRepo, alertRecordRepo, gdb)

	// Alert
	alertSvc := service.NewAlertService(alertRuleRepo, alertChannelRepo, alertRecordRepo, panelRepo, dashRepo)
	alertEngine := alert.New(alertRuleRepo, alertChannelRepo, alertRecordRepo, panelRepo, dashRepo, gdb, time.Minute)
	alertCtx, alertCancel := context.WithCancel(context.Background())
	defer alertCancel()
	alertEngine.Start(alertCtx)

	// Seed admin
	if err := service.EnsureAdminSeed(context.Background(), userRepo, cfg.Admin); err != nil {
		logger.L.Warn("admin seed", zap.Error(err))
	}

	// 初始化默认权限组并迁移现有用户
	if err := service.EnsureDefaultPermissionGroups(context.Background(), permGroupRepo, userRepo, permOverrideRepo); err != nil {
		logger.L.Warn("permission group seed", zap.Error(err))
	}

	// Scheduler
	sched := scheduler.New(gdb, dsRepo, dsSvc)
	if err := sched.Start(context.Background()); err != nil {
		logger.L.Fatal("start scheduler", zap.Error(err))
	}
	defer sched.Stop()

	reloadFn := func() {
		if err := sched.Reload(context.Background()); err != nil {
			logger.L.Warn("scheduler reload", zap.Error(err))
		}
	}

	// Router
	r := router.New(router.Deps{
		Cfg:                 cfg,
		Auth:                authSvc,
		UserRepo:            userRepo,
		AuditRepo:           auditRepo,
		PermissionSvc:       permSvc,
		PermissionGroupSvc:  permGroupSvc,
		Feedback:            fbSvc,
		DataSource:          dsSvc,
		EnumConfig:          enumConfigSvc,
		Dashboard:           dashSvc,
		Alert:               alertSvc,
		ClassifierConfig:    clfConfigSvc,
		ClassificationLog:   clfLogSvc,
		LLMProvider:         llmProviderSvc,
		LLMConfig:           llmConfigSvc,
		MultiClassifier:     multiClf,
		DimSchema:           dimSchemaSvc,
		BucketUpdateLogRepo: metricBucketUpdateLogRepo,
		OnDSChange:          reloadFn,
	})

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.L.Info("server starting", zap.String("addr", cfg.Server.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal("listen", zap.Error(err))
		}
	}()

	// Periodic login_attempts cleanup
	go cleanupLoop(context.Background(), loginRepo, cfg.LoginLimit.Window)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.L.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func cleanupLoop(ctx context.Context, repo *repository.LoginAttemptRepo, window time.Duration) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = repo.Cleanup(ctx, time.Now().Add(-2*window))
		}
	}
}
