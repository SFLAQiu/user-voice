package model

import "time"

type EnumConfig struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	Field     string     `gorm:"size:64;uniqueIndex" json:"field"`
	Labels    JSON       `gorm:"column:labels" json:"labels"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (EnumConfig) TableName() string { return "enum_configs" }