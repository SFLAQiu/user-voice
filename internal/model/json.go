package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// JSON is a generic helper for MySQL JSON columns via GORM.
type JSON json.RawMessage

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSON) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = JSON(v)
	default:
		return errors.New("unsupported scan type for JSON")
	}
	return nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("nil JSON pointer")
	}
	*j = append((*j)[:0], data...)
	return nil
}

func JSONFromAny(v any) (JSON, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return JSON(b), nil
}

// ── LocalTime: 本地时间序列化 ──

const localTimeFormat = "2006-01-02 15:04:05"

// LocalTime 包装 time.Time，JSON 序列化为本地时间字符串（不带时区后缀），
// GORM 按 datetime 列读写。零值序列化为 "null"。
type LocalTime time.Time

func (t LocalTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + tt.Format(localTimeFormat) + `"`), nil
}

func (t *LocalTime) UnmarshalJSON(data []byte) error {
	if t == nil {
		return errors.New("nil LocalTime pointer")
	}
	s := strings.Trim(string(data), `"`)
	if s == "null" || s == "" {
		*t = LocalTime(time.Time{})
		return nil
	}
	// 优先解析本地时间格式
	if tt, err := time.ParseInLocation(localTimeFormat, s, time.Local); err == nil {
		*t = LocalTime(tt)
		return nil
	}
	// 兼容 RFC3339（含 Z 后缀），转为本地时区
	if tt, err := time.Parse(time.RFC3339, s); err == nil {
		*t = LocalTime(tt.Local())
		return nil
	}
	return errors.New("unsupported time format: " + s)
}

func (t LocalTime) Value() (driver.Value, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return nil, nil
	}
	return tt, nil
}

func (t *LocalTime) Scan(src any) error {
	if t == nil {
		return errors.New("nil LocalTime pointer")
	}
	if src == nil {
		*t = LocalTime(time.Time{})
		return nil
	}
	tt, ok := src.(time.Time)
	if !ok {
		return errors.New("unsupported scan type for LocalTime")
	}
	*t = LocalTime(tt)
	return nil
}

// Time 返回底层 time.Time，供代码中需要 time.Time 的场景使用。
func (t LocalTime) Time() time.Time {
	return time.Time(t)
}

// NullableLocalTime 是 *LocalTime 的可空版本，JSON 序列化为本地时间字符串或 null。
// GORM 按 datetime 列读写，nil 表示数据库 NULL。
type NullableLocalTime = *LocalTime // 语义别名，nil → null
