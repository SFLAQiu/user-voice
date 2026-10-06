package model

import "time"

type AlertRule struct {
	ID               uint64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string      `gorm:"size:128;not null" json:"name"`
	PanelID          *uint64     `json:"panel_id,omitempty"`
	MetricQueryJSON  JSON        `gorm:"column:metric_query_json;type:json" json:"metric_query"`
	TimeWindowSec    int         `json:"time_window_sec"`
	ConditionOp      string      `gorm:"size:8;not null" json:"condition_op"` // > >= < <= =
	Threshold        float64     `gorm:"type:decimal(20,4)" json:"threshold"`
	Level            string      `gorm:"size:16;not null" json:"level"` // info | warning | critical
	SilenceMinutes   int         `gorm:"default:30" json:"silence_minutes"`
	EvalIntervalSec  int         `gorm:"default:60" json:"eval_interval_sec"`
	PendingDurationSec int       `gorm:"default:0;comment:待确认时长（秒）：指标突破阈值后需持续高于阈值的时间，0=立即告警" json:"pending_duration_sec"`
	ReduceMode         string    `gorm:"size:16;default:'max';comment:告警缩减模式：max|last" json:"reduce_mode"`
	TemplateID         *uint64   `json:"template_id,omitempty"`
	Status           int         `gorm:"default:1" json:"status"` // 1 enabled 0 disabled
	LastEvalAt       *LocalTime  `json:"last_eval_at,omitempty"`
	LastState        string      `gorm:"size:16" json:"last_state"`
	ConsecutiveFires int         `gorm:"default:0" json:"consecutive_fires"`
	LastNotifyAt     *LocalTime  `json:"last_notify_at,omitempty"`
	CreatedAt        LocalTime   `json:"created_at"`
	UpdatedAt        LocalTime   `json:"updated_at"`
}

func (AlertRule) TableName() string { return "alert_rules" }

type AlertChannel struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string    `gorm:"size:128;not null;uniqueIndex:uk_name" json:"name"`
	Type       string    `gorm:"size:32;not null" json:"type"` // wecom | feishu | dingtalk | webhook
	WebhookURL string    `gorm:"size:512;not null" json:"webhook_url"`
	Secret     string    `gorm:"size:255" json:"secret,omitempty"`
	Status     int       `gorm:"default:1" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (AlertChannel) TableName() string { return "alert_channels" }

type AlertTemplate struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"size:128;not null" json:"name"`
	ContentTemplate string    `gorm:"type:text;not null" json:"content_template"`
	IsDefault       int       `json:"is_default"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (AlertTemplate) TableName() string { return "alert_templates" }

type AlertRecord struct {
	ID               uint64      `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleID           uint64      `json:"rule_id"`
	TriggerValue     float64     `gorm:"type:decimal(20,4)" json:"trigger_value"`
	Threshold        float64     `gorm:"type:decimal(20,4)" json:"threshold"`
	Level            string      `gorm:"size:16;not null" json:"level"`
	State            string      `gorm:"size:16;not null" json:"state"` // pending | firing | resolved
	NotifiedChannels JSON        `gorm:"column:notified_channels;type:json" json:"notified_channels"`
	NotifyStatus     string      `gorm:"size:32" json:"notify_status"`
	NotifyDetailJSON JSON        `gorm:"column:notify_detail_json;type:json" json:"notify_detail,omitempty"`
	TriggeredAt      LocalTime   `json:"triggered_at"`
	MetricTimestamp   *LocalTime `gorm:"column:metric_timestamp;type:datetime" json:"metric_timestamp,omitempty"`
	ResolvedAt       *LocalTime  `json:"resolved_at,omitempty"`
}

// AlertRuleChannel is the join table for rule-channel associations.
type AlertRuleChannel struct {
	RuleID    uint64 `gorm:"column:rule_id;primaryKey"`
	ChannelID uint64 `gorm:"column:channel_id;primaryKey"`
}

func (AlertRuleChannel) TableName() string { return "alert_rule_channels" }

func (AlertRecord) TableName() string { return "alert_records" }
