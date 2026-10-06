package classifier

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/pkg/logger"

	"go.uber.org/zap"
)

// ProviderClient wraps an OpenAI client with metadata and circuit breaker.
type ProviderClient struct {
	ProviderID uint64
	Name       string
	Client     *openai.Client
	Model      string
	Timeout    time.Duration
	Priority   int
	CB         *CircuitBreaker
}

// ClassifierScheduleConfig controls the classifier's scan/dispatch behavior.
type ClassifierScheduleConfig struct {
	WorkerPoolSize int `json:"worker_pool_size"` // 并发 worker 数量
	ScanIntervalMs int `json:"scan_interval_ms"` // 扫描间隔（毫秒）
	BatchSize      int `json:"batch_size"`        // 每次扫描拉取条数
	RetryIntervalMs int `json:"retry_interval_ms"` // 失败重置间隔（毫秒）
}

// DefaultClassifierScheduleConfig returns sensible defaults.
func DefaultClassifierScheduleConfig() ClassifierScheduleConfig {
	return ClassifierScheduleConfig{
		WorkerPoolSize:  4,
		ScanIntervalMs:  30000,
		BatchSize:       10,
		RetryIntervalMs: 600000,
	}
}

// RetryConfig controls fallback behavior across providers.
type RetryConfig struct {
	MaxRetries int    `json:"max_retries"` // max number of providers to try
	Strategy   string `json:"strategy"`    // "priority" or "random"
	TimeoutMs  int    `json:"timeout_ms"`  // global default timeout per LLM request (ms)
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		Strategy:   "priority",
		TimeoutMs:  30000,
	}
}

// LLMProviderDecrypted is the DB row with the API key decrypted for client creation.
type LLMProviderDecrypted struct {
	ID        uint64
	Name      string
	BaseURL   string
	APIKey    string // plaintext after decryption
	Model     string
	TimeoutMs int
	Priority  int
	Status    int
}

// MultiLLMClassifier tries multiple providers with circuit breaker and retry.
type MultiLLMClassifier struct {
	mu      sync.RWMutex
	clients []ProviderClient // sorted by priority
	CBSet   *CircuitBreakerSet
	retry   RetryConfig
}

// NewMultiLLMClassifier creates a multi-provider classifier.
func NewMultiLLMClassifier(cbSet *CircuitBreakerSet, retry RetryConfig) *MultiLLMClassifier {
	return &MultiLLMClassifier{
		CBSet: cbSet,
		retry: retry,
	}
}

// UpdateProviders refreshes the sorted provider client list.
// globalTimeoutMs is used as fallback when a provider has no explicit timeout.
func (m *MultiLLMClassifier) UpdateProviders(providers []LLMProviderDecrypted, cipher *crypto.Cipher, globalTimeoutMs int) error {
	clients := make([]ProviderClient, 0, len(providers))
	for _, p := range providers {
		clientCfg := openai.DefaultConfig(p.APIKey)
		if p.BaseURL != "" {
			clientCfg.BaseURL = p.BaseURL
		}
		timeoutMs := p.TimeoutMs
		if timeoutMs <= 0 {
			timeoutMs = globalTimeoutMs
		}
		if timeoutMs <= 0 {
			timeoutMs = 30000
		}
		timeout := time.Duration(timeoutMs) * time.Millisecond
		cb := m.CBSet.Get(p.ID)
		clients = append(clients, ProviderClient{
			ProviderID: p.ID,
			Name:       p.Name,
			Client:     openai.NewClientWithConfig(clientCfg),
			Model:      p.Model,
			Timeout:    timeout,
			Priority:   p.Priority,
			CB:         cb,
		})
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Priority < clients[j].Priority
	})

	m.mu.Lock()
	m.clients = clients
	m.mu.Unlock()
	return nil
}

// UpdateRetryConfig updates the retry configuration.
func (m *MultiLLMClassifier) UpdateRetryConfig(retry RetryConfig) {
	m.mu.Lock()
	m.retry = retry
	m.mu.Unlock()
}

// ClassifyWithPrompt tries providers in order, skipping circuit-broken ones,
// up to max_retries attempts. Returns which provider was used.
func (m *MultiLLMClassifier) ClassifyWithPrompt(ctx context.Context, prompt string) (Result, string, uint64, error) {
	m.mu.RLock()
	candidates := m.availableCandidates()
	retry := m.retry
	m.mu.RUnlock()

	if len(candidates) == 0 {
		return Result{}, "", 0, fmt.Errorf("no available LLM providers")
	}

	if retry.Strategy == "random" {
		shuffleClients(candidates)
	}

	maxAttempts := retry.MaxRetries
	if maxAttempts <= 0 {
		maxAttempts = len(candidates)
	}
	if maxAttempts > len(candidates) {
		maxAttempts = len(candidates)
	}

	var lastErr error
	var lastProviderID uint64
	for i := 0; i < maxAttempts; i++ {
		pc := candidates[i]
		result, raw, err := pc.executeWithCB(ctx, prompt)
		if err == nil {
			return result, raw, pc.ProviderID, nil
		}
		lastErr = err
		lastProviderID = pc.ProviderID
		logger.L.Warn("provider failed, trying next",
			zap.Uint64("provider_id", pc.ProviderID),
			zap.String("provider_name", pc.Name),
			zap.Error(err))
	}
	return Result{}, "", lastProviderID, fmt.Errorf("all %d provider attempts failed: %w", maxAttempts, lastErr)
}

// availableCandidates returns providers whose circuit breaker is not open.
func (m *MultiLLMClassifier) availableCandidates() []ProviderClient {
	var out []ProviderClient
	for _, pc := range m.clients {
		if pc.CB.State() == StateOpen {
			continue
		}
		out = append(out, pc)
	}
	return out
}

// executeWithCB wraps the LLM call through the circuit breaker.
func (pc *ProviderClient) executeWithCB(ctx context.Context, prompt string) (Result, string, error) {
	rawResult, cbErr := pc.CB.Execute(func() (interface{}, error) {
		timeout := pc.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resp, err := pc.Client.CreateChatCompletion(callCtx, openai.ChatCompletionRequest{
			Model: pc.Model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
			Temperature: 0.2,
		})
		if err != nil {
			return nil, fmt.Errorf("llm call: %w", err)
		}
		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("llm: empty choices")
		}
		return resp.Choices[0].Message.Content, nil
	})
	if cbErr != nil {
		return Result{}, "", cbErr
	}
	rawText, ok := rawResult.(string)
	if !ok {
		return Result{}, "", fmt.Errorf("unexpected result type from circuit breaker")
	}
	result, parseErr := parseResult(rawText)
	return result, rawText, parseErr
}

// CBStatusSummary is a snapshot for the status API.
type CBStatusSummary struct {
	ProviderID          uint64 `json:"provider_id"`
	ProviderName        string `json:"provider_name"`
	State               string `json:"state"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	TotalFailures       int    `json:"total_failures"`
	TotalSuccesses      int    `json:"total_successes"`
	Requests            int    `json:"requests"`
}

// CircuitBreakerStatus returns status summaries for all known providers.
func (m *MultiLLMClassifier) CircuitBreakerStatus() []CBStatusSummary {
	m.mu.RLock()
	clients := m.clients
	m.mu.RUnlock()

	allStats := m.CBSet.AllStats()
	out := make([]CBStatusSummary, 0, len(clients))
	for _, pc := range clients {
		stats, ok := allStats[pc.ProviderID]
		if !ok {
			out = append(out, CBStatusSummary{
				ProviderID:   pc.ProviderID,
				ProviderName: pc.Name,
				State:        StateClosed.String(),
			})
			continue
		}
		out = append(out, CBStatusSummary{
			ProviderID:          pc.ProviderID,
			ProviderName:        pc.Name,
			State:               stats.State.String(),
			ConsecutiveFailures: stats.ConsecutiveFailures,
			TotalFailures:       stats.TotalFailures,
			TotalSuccesses:      stats.TotalSuccesses,
			Requests:            stats.Requests,
		})
	}
	return out
}

func shuffleClients(clients []ProviderClient) {
	rand.Shuffle(len(clients), func(i, j int) {
		clients[i], clients[j] = clients[j], clients[i]
	})
}
