package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/feedback/internal/classifier"
	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/model"
	"github.com/feedback/internal/pkg/logger"
	"github.com/feedback/internal/queue"
	"github.com/feedback/internal/repository"
)

// ClassifierService periodically scans feedbacks with category_status=0
// and dispatches each to the worker pool for LLM classification.
type ClassifierService struct {
	repo         *repository.FeedbackRepo
	multiClf     *classifier.MultiLLMClassifier
	providerSvc  *LLMProviderService
	llmConfigSvc *LLMConfigService
	logSvc       *ClassificationLogService
	clfConfigSvc *ClassifierConfigService
	cipher       *crypto.Cipher
	pool         *queue.WorkerPool
	processing   sync.Map // tracks feedback IDs currently being classified
	bucketSvc    *MetricBucketService
}

func NewClassifierService(
	repo *repository.FeedbackRepo,
	multiClf *classifier.MultiLLMClassifier,
	providerSvc *LLMProviderService,
	llmConfigSvc *LLMConfigService,
	clfConfigSvc *ClassifierConfigService,
	logSvc *ClassificationLogService,
	cipher *crypto.Cipher,
	bucketSvc *MetricBucketService,
) *ClassifierService {
	scheduleCfg, err := llmConfigSvc.GetClassifierScheduleConfig(context.Background())
	if err != nil {
		logger.L.Warn("load classifier schedule config", zap.Error(err))
		defaultCfg := classifier.DefaultClassifierScheduleConfig()
		scheduleCfg = &defaultCfg
	}
	workers := scheduleCfg.WorkerPoolSize
	if workers <= 0 {
		workers = 4
	}
	return &ClassifierService{
		repo:         repo,
		multiClf:     multiClf,
		providerSvc:  providerSvc,
		llmConfigSvc: llmConfigSvc,
		clfConfigSvc: clfConfigSvc,
		logSvc:       logSvc,
		cipher:       cipher,
		pool:         queue.New(workers, workers*4),
		bucketSvc:    bucketSvc,
	}
}

// InitProviders loads DB providers.
func (s *ClassifierService) InitProviders(ctx context.Context) {
	if err := s.refreshProviders(ctx); err != nil {
		logger.L.Warn("init providers", zap.Error(err))
	}
}

// Start launches the background scan loop and provider refresh loop.
func (s *ClassifierService) Start(ctx context.Context) {
	go s.loop(ctx)
	go s.refreshLoop(ctx, 30*time.Second)
}

// Stop shuts down the worker pool (waits for in-flight jobs).
func (s *ClassifierService) Stop() { s.pool.Stop() }

func (s *ClassifierService) loop(ctx context.Context) {
	scheduleCfg, _ := s.llmConfigSvc.GetClassifierScheduleConfig(ctx)
	interval := time.Duration(scheduleCfg.ScanIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 30 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// 定期重新读取调度配置
			scheduleCfg, _ = s.llmConfigSvc.GetClassifierScheduleConfig(ctx)
			newInterval := time.Duration(scheduleCfg.ScanIntervalMs) * time.Millisecond
			if newInterval > 0 && newInterval != interval {
				interval = newInterval
				t.Reset(interval)
			}
			s.scanAndDispatch(ctx)
		}
	}
}

func (s *ClassifierService) refreshLoop(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.refreshProvidersWithRetry(ctx, 3)
		}
	}
}

// refreshProvidersWithRetry 立即重试刷新，避免 DB 短暂抖动导致 provider 列表长期过时。
func (s *ClassifierService) refreshProvidersWithRetry(ctx context.Context, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		if err := s.refreshProviders(ctx); err != nil {
			logger.L.Warn("refresh providers", zap.Int("attempt", i+1), zap.Error(err))
			if i < maxRetries-1 {
				// 递增等待，同时尊重 context 取消
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(i+1) * time.Second):
				}
				continue
			}
		} else {
			return
		}
	}
}

func (s *ClassifierService) refreshProviders(ctx context.Context) error {
	providers, err := s.providerSvc.ListEnabledDecrypted(ctx)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		logger.L.Info("no LLM providers available, classifier disabled")
		return nil
	}

	cbConfig, err := s.llmConfigSvc.GetCircuitBreakerConfig(ctx)
	if err != nil {
		logger.L.Warn("load circuit breaker config", zap.Error(err))
		defaultCfg := classifier.DefaultCircuitBreakerConfig()
		cbConfig = &defaultCfg
	}
	retryConfig, err := s.llmConfigSvc.GetRetryConfig(ctx)
	if err != nil {
		logger.L.Warn("load retry config", zap.Error(err))
		defaultCfg := classifier.DefaultRetryConfig()
		retryConfig = &defaultCfg
	}

	s.multiClf.UpdateProviders(providers, s.cipher, retryConfig.TimeoutMs)
	s.multiClf.CBSet.UpdateConfig(*cbConfig)
	s.multiClf.UpdateRetryConfig(*retryConfig)

	// 清理已删除 provider 的熔断器残留
	activeIDs := make([]uint64, 0, len(providers))
	for _, p := range providers {
		activeIDs = append(activeIDs, p.ID)
	}
	s.multiClf.CBSet.PruneStale(activeIDs)

	logger.L.Info("providers refreshed",
		zap.Int("count", len(providers)),
		zap.Any("cb_config", cbConfig),
		zap.Any("retry_config", retryConfig))
	return nil
}

func (s *ClassifierService) scanAndDispatch(ctx context.Context) {
	scheduleCfg, _ := s.llmConfigSvc.GetClassifierScheduleConfig(ctx)
	retryInterval := time.Duration(scheduleCfg.RetryIntervalMs) * time.Millisecond
	if retryInterval <= 0 {
		retryInterval = 10 * time.Minute
	}
	if n, err := s.repo.ResetFailedClassifications(ctx, retryInterval); err != nil {
		logger.L.Warn("reset failed classifications", zap.Error(err))
	} else if n > 0 {
		logger.L.Info("reset failed classifications for retry", zap.Int64("count", n))
	}

	batchSize := scheduleCfg.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	rows, err := s.repo.ListPendingClassification(ctx, batchSize)
	if err != nil {
		logger.L.Warn("classifier scan", zap.Error(err))
		return
	}
	for _, fb := range rows {
		fb := fb // capture
		// Skip feedbacks already being classified
		if _, loaded := s.processing.LoadOrStore(fb.ID, true); loaded {
			continue
		}
		s.pool.Submit(func(_ context.Context) {
			s.classifyOne(ctx, fb)
		})
	}
}

func (s *ClassifierService) classifyOne(ctx context.Context, fb model.Feedback) {
	defer s.processing.Delete(fb.ID)

	prompt, activeCfg, err := s.clfConfigSvc.BuildClassificationPrompt(ctx, fb.Content)
	if err != nil {
		logger.L.Warn("build classification prompt", zap.Uint64("id", fb.ID), zap.Error(err))
		s.recordLog(ctx, fb.ID, 1, nil, nil, "", "", nil, err.Error(), model.LogStatusFailed, nil)
		_ = s.repo.MarkClassificationFailed(ctx, fb.ID)
		return
	}

	start := time.Now()
	result, rawResponse, providerID, callErr := s.multiClf.ClassifyWithPrompt(ctx, prompt)
	durationMs := int(time.Since(start).Milliseconds())

	attempt, _ := s.logSvc.NextAttempt(ctx, fb.ID)
	promptConfigID := &activeCfg.PromptConfigID
	if activeCfg.PromptConfigID == 0 {
		promptConfigID = nil
	}
	providerIDPtr := &providerID
	if providerID == 0 {
		providerIDPtr = nil
	}

	if callErr != nil {
		isTimeout := errors.Is(callErr, context.DeadlineExceeded)
		status := model.LogStatusFailed
		errMsg := callErr.Error()
		if isTimeout {
			status = model.LogStatusTimeout
			errMsg = fmtTimeoutMessage(callErr, durationMs)
		}
		logger.L.Warn("classify feedback",
			zap.Uint64("id", fb.ID),
			zap.Uint64("provider_id", providerID),
			zap.Bool("timeout", isTimeout),
			zap.Error(callErr))
		s.recordLog(ctx, fb.ID, attempt, promptConfigID, providerIDPtr, prompt, rawResponse, nil, errMsg, status, &durationMs)
		_ = s.repo.MarkClassificationFailed(ctx, fb.ID)
		return
	}

	confidence := result.Confidence
	parsedResult, _ := marshalParsedResult(result)
	cleanModule := classifier.CleanBusinessModule(result.BusinessModule)

	s.recordLogWithDuration(ctx, fb.ID, attempt, promptConfigID, providerIDPtr, prompt, rawResponse, parsedResult,
		result.Category, cleanModule, result.Sentiment, &confidence, "", model.LogStatusSuccess, durationMs)

	if err := s.repo.UpdateClassification(ctx, fb.ID, result.Category, cleanModule, result.Sentiment, result.Confidence, model.CategoryStatusClassified); err != nil {
		logger.L.Warn("update classification", zap.Uint64("id", fb.ID), zap.Error(err))
	} else {
		if s.bucketSvc != nil {
			oldDims := repository.BucketDimsFromFeedback(fb) // 分类前维度
			// 同步更新内存对象，使 MoveFeedbackBucket 内部计算的 newDims 与 DB 一致
			fb.Category = result.Category
			fb.BusinessModule = cleanModule
			fb.Sentiment = result.Sentiment
			fb.CategoryStatus = model.CategoryStatusClassified
			if err := s.bucketSvc.MoveFeedbackBucket(ctx, fb, oldDims); err != nil {
				logger.L.Warn("metric-bucket: move after classification", zap.Uint64("id", fb.ID), zap.Error(err))
			}
		}
	}
}

func fmtTimeoutMessage(err error, durationMs int) string {
	timeoutSec := float64(durationMs) / 1000
	return fmt.Sprintf("请求超时 (%.1fs): %s", timeoutSec, err.Error())
}

func (s *ClassifierService) recordLog(ctx context.Context, feedbackID uint64, attempt int, promptConfigID *uint64, providerID *uint64, promptUsed, rawResponse string, parsedResult model.JSON, errMsg string, status int, durationMs *int) {
	log := &model.ClassificationLog{
		FeedbackID:     feedbackID,
		Attempt:        attempt,
		PromptConfigID: promptConfigID,
		ProviderID:     providerID,
		PromptUsed:     promptUsed,
		LLMRawResponse: rawResponse,
		ParsedResult:   parsedResult,
		ErrorMessage:   errMsg,
		Status:         status,
		DurationMs:     durationMs,
	}
	if err := s.logSvc.Create(ctx, log); err != nil {
		logger.L.Warn("write classification log", zap.Uint64("feedback_id", feedbackID), zap.Error(err))
	}
}

func (s *ClassifierService) recordLogWithDuration(ctx context.Context, feedbackID uint64, attempt int, promptConfigID *uint64, providerID *uint64, promptUsed, rawResponse string, parsedResult model.JSON, categoryResult, moduleResult, sentimentResult string, confidenceResult *float64, errMsg string, status int, durationMs int) {
	clLog := &model.ClassificationLog{
		FeedbackID:       feedbackID,
		Attempt:          attempt,
		PromptConfigID:   promptConfigID,
		ProviderID:       providerID,
		PromptUsed:       promptUsed,
		LLMRawResponse:   rawResponse,
		ParsedResult:     parsedResult,
		CategoryResult:   categoryResult,
		ModuleResult:     moduleResult,
		SentimentResult:  sentimentResult,
		ConfidenceResult: confidenceResult,
		ErrorMessage:     errMsg,
		Status:           status,
		DurationMs:       &durationMs,
	}
	if err := s.logSvc.Create(ctx, clLog); err != nil {
		logger.L.Warn("write classification log", zap.Uint64("feedback_id", feedbackID), zap.Error(err))
	}
}
