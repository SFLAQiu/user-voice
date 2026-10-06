package model

import "time"

const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"

	UserStatusEnabled  = 1
	UserStatusDisabled = 0
)

type User struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;uniqueIndex" json:"username"`
	PasswordHash string     `gorm:"size:255" json:"-"`
	Role         string     `gorm:"size:16" json:"role"`
	Status       int        `json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 权限分组（非 DB 列，由 service 层填充）
	Groups []PermissionGroup `gorm:"-" json:"groups,omitempty"`
}

func (User) TableName() string { return "users" }

type LoginAttempt struct {
	ID          uint64    `gorm:"primaryKey"`
	Identifier  string    `gorm:"size:128;index:idx_identifier_time,priority:1"`
	AttemptedAt time.Time `gorm:"index:idx_identifier_time,priority:2"`
	Success     int
}

func (LoginAttempt) TableName() string { return "login_attempts" }

type AuditLog struct {
	ID         uint64    `gorm:"primaryKey"`
	UserID     *uint64   `json:"user_id,omitempty"`
	Username   string    `gorm:"size:64" json:"username,omitempty"`
	Action     string    `gorm:"size:64" json:"action"`
	Resource   string    `gorm:"size:64" json:"resource,omitempty"`
	ResourceID string    `gorm:"size:64" json:"resource_id,omitempty"`
	IP         string    `gorm:"size:64" json:"ip,omitempty"`
	DetailJSON JSON      `gorm:"column:detail_json" json:"detail,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
