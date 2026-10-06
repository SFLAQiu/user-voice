package model

import "time"

const (
	LogStatusPending = 0
	LogStatusSuccess = 1
	LogStatusFailed  = 2
	LogStatusTimeout = 3
)

type ClassificationLog struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	FeedbackID       uint64    `gorm:"index:idx_feedback_id" json:"feedback_id"`
	Attempt          int       `json:"attempt"`
	PromptConfigID   *uint64   `json:"prompt_config_id,omitempty"`
	ProviderID       *uint64   `gorm:"column:provider_id;index:idx_provider_id" json:"provider_id,omitempty"`
	ProviderName     string    `gorm:"-" json:"provider_name,omitempty"`
	PromptUsed       string    `gorm:"type:text" json:"prompt_used,omitempty"`
	LLMRawResponse   string    `gorm:"type:text;column:llm_raw_response" json:"llm_raw_response,omitempty"`
	ParsedResult     JSON      `gorm:"column:parsed_result" json:"parsed_result,omitempty"`
	CategoryResult   string    `gorm:"size:64" json:"category_result,omitempty"`
	ModuleResult     string    `gorm:"size:64;column:module_result" json:"module_result,omitempty"`
	SentimentResult  string    `gorm:"size:16" json:"sentiment_result,omitempty"`
	ConfidenceResult *float64  `gorm:"column:confidence_result;type:decimal(4,3)" json:"confidence_result,omitempty"`
	ErrorMessage     string    `gorm:"type:text" json:"error_message,omitempty"`
	Status           int       `gorm:"index:idx_status_time,priority:1" json:"status"`
	DurationMs       *int      `gorm:"column:duration_ms" json:"duration_ms,omitempty"`
	CreatedAt        time.Time `gorm:"index:idx_status_time,priority:2" json:"created_at"`
}

func (ClassificationLog) TableName() string { return "classification_logs" }
