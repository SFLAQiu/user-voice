package model

import "time"

const (
	DataSourceStatusEnabled  = 1
	DataSourceStatusDisabled = 0

	SyncStatusSuccess = "success"
	SyncStatusFailed  = "failed"
	SyncStatusRunning = "running"
)

type DataSource struct {
	ID             uint64     `gorm:"primaryKey" json:"id"`
	Name           string     `gorm:"size:128;uniqueIndex" json:"name"`
	Type           string     `gorm:"size:32" json:"type"`
	ConfigJSON     JSON       `gorm:"column:config_json" json:"config"`
	SyncCron       string     `gorm:"size:64" json:"sync_cron"`
	Status         int        `json:"status"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus string     `gorm:"size:32" json:"last_sync_status,omitempty"`
	LastSyncError  string     `gorm:"type:text" json:"last_sync_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (DataSource) TableName() string { return "data_sources" }

type SyncCursor struct {
	ID             uint64 `gorm:"primaryKey"`
	SourceID       uint64 `gorm:"uniqueIndex:uk_source_key,priority:1"`
	CursorKey      string `gorm:"size:64;uniqueIndex:uk_source_key,priority:2"`
	LastOriginalID string `gorm:"size:64"`
	LastCreatedAt  *time.Time
	UpdatedAt      time.Time
}

func (SyncCursor) TableName() string { return "sync_cursors" }

type SyncLog struct {
	ID            uint64     `gorm:"primaryKey" json:"id"`
	SourceID      uint64     `json:"source_id"`
	Status        string     `gorm:"size:32" json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	FetchedCount  int        `json:"fetched_count"`
	InsertedCount int        `json:"inserted_count"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (SyncLog) TableName() string { return "sync_logs" }
