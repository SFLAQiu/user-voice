package model

import "time"

const (
	LLMProviderEnabled  = 1
	LLMProviderDisabled = 0
)

// LLMProvider stores an OpenAI-compatible LLM API configuration.
// APIKey is AES-256-GCM encrypted before storage; responses mask it as "***".
type LLMProvider struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;uniqueIndex" json:"name"`
	BaseURL   string    `gorm:"size:256" json:"base_url"`
	APIKey    string    `gorm:"size:512" json:"api_key"`
	Model     string    `gorm:"size:128" json:"model"`
	TimeoutMs int       `gorm:"column:timeout_ms;default:30000" json:"timeout_ms"`
	Priority  int       `gorm:"default:1" json:"priority"` // 1 = highest
	Status    int       `gorm:"default:1" json:"status"`   // 1=enabled 0=disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LLMProvider) TableName() string { return "llm_providers" }