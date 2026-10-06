package model

import "time"

const (
	LLMConfigTypeCircuitBreaker = "circuit_breaker"
	LLMConfigTypeRetry          = "retry"
	LLMConfigTypeClassifier     = "classifier"
)

// LLMConfig stores LLM infrastructure configuration (circuit breaker, retry).
// Content holds a JSON string with the configuration parameters.
type LLMConfig struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"size:32;uniqueIndex:uk_type_name,priority:1" json:"type"`
	Name      string    `gorm:"size:128;uniqueIndex:uk_type_name,priority:2" json:"name"`
	Content   string    `gorm:"type:text" json:"content,omitempty"`
	SortOrder int       `json:"sort_order"`
	Enabled   int       `json:"enabled"`
	IsDefault int       `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LLMConfig) TableName() string { return "llm_configs" }