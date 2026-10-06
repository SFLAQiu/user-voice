package model

import "time"

const (
	CategoryStatusPending    = 0
	CategoryStatusClassified = 1
	CategoryStatusFailed     = 2
	CategoryStatusManual     = 3

	SentimentPositive = "positive"
	SentimentNeutral  = "neutral"
	SentimentNegative = "negative"
)

type Feedback struct {
	ID                 uint64     `gorm:"primaryKey" json:"id"`
	SourceID           uint64     `json:"source_id"`
	OriginalID         string     `gorm:"size:64;uniqueIndex:uk_source_origid,priority:2" json:"original_id"`
	AppID              int        `json:"app_id"`
	AppName            string     `gorm:"size:64" json:"app_name,omitempty"`
	Platform           string     `gorm:"size:32" json:"platform,omitempty"`
	PlatformID         *int       `json:"platform_id,omitempty"`
	UserID             string     `gorm:"size:64" json:"user_id,omitempty"`
	UserName           string     `gorm:"size:128" json:"user_name,omitempty"`
	UserMode           *int       `json:"user_mode,omitempty"`
	Content            string     `gorm:"type:text" json:"content"`
	Images             JSON       `gorm:"column:images" json:"images,omitempty"`
	Videos             JSON       `gorm:"column:videos" json:"videos,omitempty"`
	PhoneModel         string     `gorm:"size:64" json:"phone_model,omitempty"`
	AppVersion         string     `gorm:"size:32" json:"app_version,omitempty"`
	ChannelID          string     `gorm:"size:32" json:"channel_id,omitempty"`
	QQ                 string     `gorm:"size:32;column:qq" json:"qq,omitempty"`
	FileURL            string     `gorm:"type:text;column:file_url" json:"file_url,omitempty"`
	RawJSON            JSON       `gorm:"column:raw_json" json:"-"`
	Category           string     `gorm:"size:64" json:"category,omitempty"`
	BusinessModule     string     `gorm:"size:64" json:"business_module,omitempty"`
	CategoryConfidence *float64   `gorm:"column:category_confidence" json:"category_confidence,omitempty"`
	CategoryStatus     int        `json:"category_status"`
	Sentiment          string     `gorm:"size:16" json:"sentiment,omitempty"`
	ClassifiedAt       *time.Time `json:"classified_at,omitempty"`
	OriginalCreatedAt  time.Time  `json:"original_created_at"`
	SyncedAt           time.Time  `json:"synced_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (Feedback) TableName() string { return "feedbacks" }

// Sortable feedback columns (whitelist).
var FeedbackSortableColumns = map[string]string{
	"original_created_at": "original_created_at",
	"id":                  "id",
	"synced_at":           "synced_at",
}
