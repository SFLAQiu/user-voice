package service

import (
	"context"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/feedback/internal/classifier"
	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/model"
	apperr "github.com/feedback/internal/pkg/errors"
	"github.com/feedback/internal/repository"
)

const maskedKey = "***"

// LLMProviderService manages CRUD for LLM provider configurations.
type LLMProviderService struct {
	repo   *repository.LLMProviderRepo
	cipher *crypto.Cipher
}

func NewLLMProviderService(repo *repository.LLMProviderRepo, cipher *crypto.Cipher) *LLMProviderService {
	return &LLMProviderService{repo: repo, cipher: cipher}
}

type CreateLLMProviderInput struct {
	Name      string `json:"name" binding:"required,max=128"`
	BaseURL   string `json:"base_url" binding:"required,max=256"`
	APIKey    string `json:"api_key" binding:"required,max=512"`
	Model     string `json:"model" binding:"required,max=128"`
	TimeoutMs *int   `json:"timeout_ms"`
	Priority  *int   `json:"priority"`
	Status    *int   `json:"status"`
}

type UpdateLLMProviderInput struct {
	Name      *string `json:"name"`
	BaseURL   *string `json:"base_url"`
	APIKey    *string `json:"api_key"`
	Model     *string `json:"model"`
	TimeoutMs *int    `json:"timeout_ms"`
	Priority  *int    `json:"priority"`
	Status    *int    `json:"status"`
}

func (s *LLMProviderService) Create(ctx context.Context, in CreateLLMProviderInput) (*model.LLMProvider, error) {
	existing, err := s.repo.FindByName(ctx, in.Name)
	if err != nil {
		return nil, apperr.Internal("check provider name", err)
	}
	if existing != nil {
		return nil, apperr.InvalidParam("provider name already exists: " + in.Name)
	}

	encKey, err := s.cipher.Encrypt(in.APIKey)
	if err != nil {
		return nil, apperr.Internal("encrypt api key", err)
	}

	timeoutMs := 30000
	if in.TimeoutMs != nil {
		timeoutMs = *in.TimeoutMs
	}
	priority := 1
	if in.Priority != nil {
		priority = *in.Priority
	}
	status := model.LLMProviderEnabled
	if in.Status != nil {
		status = *in.Status
	}

	p := &model.LLMProvider{
		Name:      in.Name,
		BaseURL:   in.BaseURL,
		APIKey:    encKey,
		Model:     in.Model,
		TimeoutMs: timeoutMs,
		Priority:  priority,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, apperr.Internal("create llm provider", err)
	}
	p.APIKey = maskedKey
	return p, nil
}

func (s *LLMProviderService) Update(ctx context.Context, id uint64, in UpdateLLMProviderInput) (*model.LLMProvider, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load llm provider", err)
	}
	if p == nil {
		return nil, apperr.NotFound("llm provider not found")
	}

	if in.Name != nil {
		existing, err := s.repo.FindByName(ctx, *in.Name)
		if err != nil {
			return nil, apperr.Internal("check provider name", err)
		}
		if existing != nil && existing.ID != id {
			return nil, apperr.InvalidParam("provider name already exists: " + *in.Name)
		}
		p.Name = *in.Name
	}
	if in.BaseURL != nil {
		p.BaseURL = *in.BaseURL
	}
	if in.APIKey != nil && *in.APIKey != "" && *in.APIKey != maskedKey {
		encKey, err := s.cipher.Encrypt(*in.APIKey)
		if err != nil {
			return nil, apperr.Internal("encrypt api key", err)
		}
		p.APIKey = encKey
	}
	if in.Model != nil {
		p.Model = *in.Model
	}
	if in.TimeoutMs != nil {
		p.TimeoutMs = *in.TimeoutMs
	}
	if in.Priority != nil {
		p.Priority = *in.Priority
	}
	if in.Status != nil {
		p.Status = *in.Status
	}
	p.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, apperr.Internal("update llm provider", err)
	}
	p.APIKey = maskedKey
	return p, nil
}

func (s *LLMProviderService) Delete(ctx context.Context, id uint64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Internal("delete llm provider", err)
	}
	return nil
}

func (s *LLMProviderService) List(ctx context.Context) ([]model.LLMProvider, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperr.Internal("list llm providers", err)
	}
	for i := range rows {
		rows[i].APIKey = maskedKey
	}
	return rows, nil
}

func (s *LLMProviderService) Get(ctx context.Context, id uint64) (*model.LLMProvider, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load llm provider", err)
	}
	if p == nil {
		return nil, apperr.NotFound("llm provider not found")
	}
	p.APIKey = maskedKey
	return p, nil
}

// ListEnabledDecrypted returns all enabled providers with decrypted API keys.
// This is an internal method for the classifier — never expose via API.
func (s *LLMProviderService) ListEnabledDecrypted(ctx context.Context) ([]classifier.LLMProviderDecrypted, error) {
	rows, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return nil, apperr.Internal("list enabled llm providers", err)
	}
	out := make([]classifier.LLMProviderDecrypted, 0, len(rows))
	for _, p := range rows {
		decKey, err := s.cipher.Decrypt(p.APIKey)
		if err != nil {
			return nil, apperr.Internal("decrypt api key for provider "+p.Name, err)
		}
		out = append(out, classifier.LLMProviderDecrypted{
			ID:        p.ID,
			Name:      p.Name,
			BaseURL:   p.BaseURL,
			APIKey:    decKey,
			Model:     p.Model,
			TimeoutMs: p.TimeoutMs,
			Priority:  p.Priority,
			Status:    p.Status,
		})
	}
	return out, nil
}

// Cipher returns the cipher instance for seed operations.
func (s *LLMProviderService) Cipher() *crypto.Cipher { return s.cipher }

// Repo returns the repository for seed operations.
func (s *LLMProviderService) Repo() *repository.LLMProviderRepo { return s.repo }

// TestConnectionInput is the input for testing a new LLM provider connection.
type TestConnectionInput struct {
	BaseURL   string `json:"base_url" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
	Model     string `json:"model" binding:"required"`
	TimeoutMs *int   `json:"timeout_ms"`
}

// TestConnectionResult is the result of a LLM provider connection test.
type TestConnectionResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms"`
	Model     string `json:"model"`
}

// TestConnection tests a new LLM provider configuration by sending a minimal chat completion request.
func (s *LLMProviderService) TestConnection(ctx context.Context, in TestConnectionInput) (*TestConnectionResult, error) {
	timeoutMs := 30000
	if in.TimeoutMs != nil && *in.TimeoutMs > 0 {
		timeoutMs = *in.TimeoutMs
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond

	clientCfg := openai.DefaultConfig(in.APIKey)
	if in.BaseURL != "" {
		clientCfg.BaseURL = in.BaseURL
	}
	client := openai.NewClientWithConfig(clientCfg)

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	resp, err := client.CreateChatCompletion(callCtx, openai.ChatCompletionRequest{
		Model:       in.Model,
		MaxTokens:   5,
		Temperature: 0.1,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: "Hi"},
		},
	})
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		return &TestConnectionResult{
			Success:   false,
			Message:   "连接失败: " + err.Error(),
			LatencyMs: latencyMs,
			Model:     in.Model,
		}, nil
	}
	if len(resp.Choices) == 0 {
		return &TestConnectionResult{
			Success:   false,
			Message:   "连接失败: LLM 返回空响应",
			LatencyMs: latencyMs,
			Model:     in.Model,
		}, nil
	}
	return &TestConnectionResult{
		Success:   true,
		Message:   "连接成功",
		LatencyMs: latencyMs,
		Model:     in.Model,
	}, nil
}

// TestExistingProvider tests an existing provider by loading its config from DB.
func (s *LLMProviderService) TestExistingProvider(ctx context.Context, id uint64) (*TestConnectionResult, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal("load llm provider", err)
	}
	if p == nil {
		return nil, apperr.NotFound("llm provider not found")
	}
	decKey, err := s.cipher.Decrypt(p.APIKey)
	if err != nil {
		return nil, apperr.Internal("decrypt api key", err)
	}
	in := TestConnectionInput{
		BaseURL:   p.BaseURL,
		APIKey:    decKey,
		Model:     p.Model,
		TimeoutMs: &p.TimeoutMs,
	}
	return s.TestConnection(ctx, in)
}