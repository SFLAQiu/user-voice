package model

import (
	"time"
)

type Dashboard struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	LayoutJSON  JSON      `gorm:"column:layout_json;type:json" json:"layout"`
	FiltersJSON JSON      `gorm:"column:filters_json;type:json" json:"filters"`
	CreatedBy   uint64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Panels      []Panel   `gorm:"-" json:"panels,omitempty"`
}

func (Dashboard) TableName() string { return "dashboards" }

type Panel struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DashboardID     uint64    `json:"dashboard_id"`
	Name            string    `gorm:"size:128;not null" json:"name"`
	ChartType       string    `gorm:"size:32;not null" json:"chart_type"` // line | bar | pie | number
	QueryConfigJSON JSON      `gorm:"column:query_config_json;type:json" json:"query_config"`
	PositionJSON    JSON      `gorm:"column:position_json;type:json" json:"position"`
	RefreshSeconds  int       `gorm:"default:60" json:"refresh_seconds"`
	AlertConfigJSON JSON      `gorm:"column:alert_config_json;type:json" json:"alert_config,omitempty"` // 面板告警配置
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Panel) TableName() string { return "panels" }
