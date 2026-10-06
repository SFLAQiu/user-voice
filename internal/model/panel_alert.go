package model

// PanelAlertConfig 存储在 panels.alert_config_json 中，作为面板告警的源头配置。
// 当面板保存告警配置时，自动同步创建/更新 AlertRule（panel_id 关联）。
type PanelAlertConfig struct {
	Threshold          float64  `json:"threshold"`           // 告警阈值
	ConditionOp        string   `json:"condition_op"`        // 比较操作符：> >= < <= =
	Level              string   `json:"level"`               // 告警级别：info | warning | critical
	SilenceMinutes     int      `json:"silence_minutes"`     // 静默时间（分钟）
	EvalIntervalSec    int      `json:"eval_interval_sec"`   // 评估间隔（秒）
	PendingDurationSec int      `json:"pending_duration_sec"` // 待确认时长（秒），0=立即告警
	TimeWindowSec      int      `json:"time_window_sec"`     // 告警查询的时间窗口（秒），0=默认300（5分钟）
	ReduceMode         string   `json:"reduce_mode"`         // 告警缩减模式：max | avg | last | sum，默认 max
	ChannelIDs         []uint64 `json:"channel_ids"`         // 通知渠道 ID 列表
}