package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/feedback/internal/classifier"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

// LLMConfigService manages CRUD for LLM infrastructure configuration
// (circuit breaker, retry mechanism).
type LLMConfigService struct {
	repo *repository.LLMConfigRepo
}

func NewLLMConfigService(repo *repository.LLMConfigRepo) *LLMConfigService {
	return &LLMConfigService{repo: repo}
}

type CreateLLMConfigInput struct {
	Type      string `json:"type" binding:"required,max=32"`
	Name      string `json:"name" binding:"required,max=128"`
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order"`
	Enabled   *int   `json:"enabled"`
	IsDefault *int   `json:"is_default"`
}

type UpdateLLMConfigInput struct {
	Name      *string `json:"name"`
	Content   *string `json:"content"`
	SortOrder *int    `json:"sort_order"`
	Enabled   *int    `json:"enabled"`
	IsDefault *int    `json:"is_default"`
}

func (s *LLMConfigService) List(ctx context.Context) ([]model.LLMConfig, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal("list llm configs", err)
	}
	return rows, nil
}

func (s *LLMConfigService) Create(ctx context.Context, in CreateLLMConfigInput) (*model.LLMConfig, error) {
	existing, err := s.repo.FindByTypeAndName(ctx, in.Type, in.Name)
	if err != nil {
		return nil, apperr.Internal("check llm config", err)
	}
	if existing != nil {
		return nil, apperr.InvalidParam("config already exists: " + in.Type + "/" + in.Name)
	}
	enabled := 1
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	isDefault := 0
	if in.IsDefault != nil {
		isDefault = *in.IsDefault
	}
	c := &model.LLMConfig{
		Type:      in.Type,
		Name:      in.Name,
		Content:   in.Content,
		SortOrder: in.SortOrder,
		Enabled:   enabled,
		IsDefault: isDefault,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, apperr.Internal("create llm config", err)
	}
	return c, nil
}

func (s *LLMConfigService) Update(ctx context.Context, id uint64, in UpdateLLMConfigInput) (*model.LLMConfig, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load llm config", err)
	}
	if c == nil {
		return nil, apperr.NotFound("llm config not found")
	}
	if in.Name != nil {
		existing, err := s.repo.FindByTypeAndName(ctx, c.Type, *in.Name)
		if err != nil {
			return nil, apperr.Internal("check llm config name", err)
		}
		if existing != nil && existing.ID != id {
			return nil, apperr.InvalidParam("config already exists: " + c.Type + "/" + *in.Name)
		}
		c.Name = *in.Name
	}
	if in.Content != nil {
		c.Content = *in.Content
	}
	if in.SortOrder != nil {
		c.SortOrder = *in.SortOrder
	}
	if in.Enabled != nil {
		c.Enabled = *in.Enabled
	}
	if in.IsDefault != nil {
		c.IsDefault = *in.IsDefault
	}
	c.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, apperr.Internal("update llm config", err)
	}
	return c, nil
}

func (s *LLMConfigService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Internal("delete llm config", err)
	}
	return nil
}

// GetCircuitBreakerConfig returns the active circuit breaker configuration.
func (s *LLMConfigService) GetCircuitBreakerConfig(ctx context.Context) (*classifier.CircuitBreakerConfig, error) {
	cfg, err := s.repo.FindDefaultByType(ctx, model.LLMConfigTypeCircuitBreaker)
	if err != nil {
		return nil, apperr.Internal("load circuit breaker config", err)
	}
	if cfg == nil {
		defaultCfg := classifier.DefaultCircuitBreakerConfig()
		return &defaultCfg, nil
	}
	var cbCfg classifier.CircuitBreakerConfig
	if err := json.Unmarshal([]byte(cfg.Content), &cbCfg); err != nil {
		defaultCfg := classifier.DefaultCircuitBreakerConfig()
		return &defaultCfg, nil
	}
	// Fill defaults for zero fields
	if cbCfg.FailureThreshold <= 0 {
		cbCfg.FailureThreshold = 5
	}
	if cbCfg.FailureRateThreshold <= 0 {
		cbCfg.FailureRateThreshold = 0.5
	}
	if cbCfg.WindowSeconds <= 0 {
		cbCfg.WindowSeconds = 60
	}
	if cbCfg.CooldownSeconds <= 0 {
		cbCfg.CooldownSeconds = 30
	}
	if cbCfg.MinRequests <= 0 {
		cbCfg.MinRequests = 3
	}
	return &cbCfg, nil
}

// GetRetryConfig returns the active retry configuration.
func (s *LLMConfigService) GetRetryConfig(ctx context.Context) (*classifier.RetryConfig, error) {
	cfg, err := s.repo.FindDefaultByType(ctx, model.LLMConfigTypeRetry)
	if err != nil {
		return nil, apperr.Internal("load retry config", err)
	}
	if cfg == nil {
		defaultCfg := classifier.DefaultRetryConfig()
		return &defaultCfg, nil
	}
	var retryCfg classifier.RetryConfig
	if err := json.Unmarshal([]byte(cfg.Content), &retryCfg); err != nil {
		defaultCfg := classifier.DefaultRetryConfig()
		return &defaultCfg, nil
	}
	if retryCfg.MaxRetries <= 0 {
		retryCfg.MaxRetries = 3
	}
	if retryCfg.Strategy == "" {
		retryCfg.Strategy = "priority"
	}
	if retryCfg.TimeoutMs <= 0 {
		retryCfg.TimeoutMs = 30000
	}
	return &retryCfg, nil
}

// GetClassifierScheduleConfig returns the active classifier schedule configuration.
func (s *LLMConfigService) GetClassifierScheduleConfig(ctx context.Context) (*classifier.ClassifierScheduleConfig, error) {
	cfg, err := s.repo.FindDefaultByType(ctx, model.LLMConfigTypeClassifier)
	if err != nil {
		return nil, apperr.Internal("load classifier schedule config", err)
	}
	if cfg == nil {
		defaultCfg := classifier.DefaultClassifierScheduleConfig()
		return &defaultCfg, nil
	}
	var scheduleCfg classifier.ClassifierScheduleConfig
	if err := json.Unmarshal([]byte(cfg.Content), &scheduleCfg); err != nil {
		defaultCfg := classifier.DefaultClassifierScheduleConfig()
		return &defaultCfg, nil
	}
	if scheduleCfg.WorkerPoolSize <= 0 {
		scheduleCfg.WorkerPoolSize = 4
	}
	if scheduleCfg.ScanIntervalMs <= 0 {
		scheduleCfg.ScanIntervalMs = 30000
	}
	if scheduleCfg.BatchSize <= 0 {
		scheduleCfg.BatchSize = 10
	}
	if scheduleCfg.RetryIntervalMs <= 0 {
		scheduleCfg.RetryIntervalMs = 600000
	}
	return &scheduleCfg, nil
}
