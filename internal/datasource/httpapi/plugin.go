// Package httpapi implements a generic HTTP API data source plugin.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/datasource"
)

const Type = "http_api"

// Config is the JSON-serialized plugin config persisted in data_sources.config_json.
type Config struct {
	Endpoint       string                       `json:"endpoint"`
	Method         string                       `json:"method"`
	Headers        map[string]string            `json:"headers"`
	Cookie         string                       `json:"cookie"` // may be ENC:...
	ParamsTemplate map[string]string            `json:"params_template"`
	Iterate        []map[string]any             `json:"iterate"` // variable groups for parameter rendering
	DataPath       string                       `json:"data_path"`
	PagePaginate   PagePaginateConfig           `json:"page_paginate"`
	FieldMapping   map[string]string            `json:"field_mapping"`
	TimeLayout     string                       `json:"time_layout"`    // default: "2006-01-02 15:04:05"
	MaxPages       int                          `json:"max_pages"`      // safety cap
	StopWhenSeen   bool                         `json:"stop_when_seen"` // stop pagination when existing original_id is hit
	Enums          map[string]map[string]string `json:"enums"`          // enum value → label mappings (e.g. app_id: {"1":"AppA"})
}

type PagePaginateConfig struct {
	Param string `json:"param"` // e.g. "to_page"
	Start int    `json:"start"` // e.g. 1
}

type Plugin struct {
	cipher *crypto.Cipher
	client *resty.Client
}

func New(c *crypto.Cipher) *Plugin {
	cli := resty.New().
		SetTimeout(30 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(2 * time.Second)
	return &Plugin{cipher: c, client: cli}
}

func (p *Plugin) Type() string { return Type }

func (p *Plugin) Validate(raw json.RawMessage) error {
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}
	if c.FieldMapping["original_id"] == "" {
		return errors.New("field_mapping.original_id is required")
	}
	if c.FieldMapping["content"] == "" {
		return errors.New("field_mapping.content is required")
	}
	return nil
}

func (p *Plugin) Pull(ctx context.Context, raw json.RawMessage, cursor *datasource.Cursor,
	exists func(string) (bool, error)) (datasource.PullResult, error) {

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return datasource.PullResult{}, err
	}
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.TimeLayout == "" {
		cfg.TimeLayout = "2006-01-02 15:04:05"
	}
	maxPages := cfg.MaxPages
	if maxPages <= 0 {
		maxPages = 50
	}

	cookie, err := p.cipher.Decrypt(cfg.Cookie)
	if err != nil {
		return datasource.PullResult{}, fmt.Errorf("decrypt cookie: %w", err)
	}

	iterations := cfg.Iterate
	if len(iterations) == 0 {
		iterations = []map[string]any{nil}
	}

	var all []datasource.FeedbackItem
	var newestID string
	var newestAt *time.Time

	for _, vars := range iterations {
		page := cfg.PagePaginate.Start
		if page == 0 {
			page = 1
		}
		for pageCount := 0; pageCount < maxPages; pageCount++ {
			params := renderParams(cfg.ParamsTemplate, vars, cfg.PagePaginate.Param, page)
			items, err := p.fetchPage(ctx, cfg, cookie, params)
			if err != nil {
				return datasource.PullResult{}, err
			}
			if len(items) == 0 {
				break
			}
			stop := false
			for _, it := range items {
				ok, err := exists(it.OriginalID)
				if err != nil {
					return datasource.PullResult{}, err
				}
				if ok {
					if cfg.StopWhenSeen {
						stop = true
						break
					}
					continue
				}
				all = append(all, it)
				if newestAt == nil || it.OriginalCreatedAt.After(*newestAt) {
					t := it.OriginalCreatedAt
					newestAt = &t
					newestID = it.OriginalID
				}
			}
			if stop {
				break
			}
			page++
		}
	}

	next := cursor
	if newestAt != nil {
		next = &datasource.Cursor{
			Key:            "default",
			LastOriginalID: newestID,
			LastCreatedAt:  newestAt,
		}
	}
	return datasource.PullResult{Items: all, NextCursor: next}, nil
}

func (p *Plugin) fetchPage(ctx context.Context, cfg Config, cookie string,
	params map[string]string) ([]datasource.FeedbackItem, error) {

	req := p.client.R().SetContext(ctx)
	for k, v := range cfg.Headers {
		req.SetHeader(k, v)
	}
	if cookie != "" {
		req.SetHeader("Cookie", cookie)
	}
	for k, v := range params {
		req.SetQueryParam(k, v)
	}

	resp, err := req.Execute(strings.ToUpper(cfg.Method), cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", cfg.Endpoint, err)
	}
	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode(), truncate(resp.String(), 200))
	}

	var raw map[string]any
	if err := json.Unmarshal(resp.Body(), &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	list, ok := pickPath(raw, cfg.DataPath).([]any)
	if !ok {
		return nil, nil
	}
	out := make([]datasource.FeedbackItem, 0, len(list))
	for _, n := range list {
		m, ok := n.(map[string]any)
		if !ok {
			continue
		}
		it, err := mapToItem(m, cfg.FieldMapping, cfg.Enums, cfg.TimeLayout)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func renderParams(tpl map[string]string, vars map[string]any, pageParam string, page int) map[string]string {
	out := make(map[string]string, len(tpl)+1)
	for k, v := range tpl {
		out[k] = renderOne(v, vars, pageParam, page)
	}
	if pageParam != "" {
		out[pageParam] = strconv.Itoa(page)
	}
	return out
}

func renderOne(s string, vars map[string]any, pageParam string, page int) string {
	out := s
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", fmt.Sprintf("%v", v))
	}
	if pageParam != "" {
		out = strings.ReplaceAll(out, "{{"+pageParam+"}}", strconv.Itoa(page))
		out = strings.ReplaceAll(out, "{{page}}", strconv.Itoa(page))
	}
	return out
}

func pickPath(root any, path string) any {
	if path == "" {
		return root
	}
	cur := root
	for _, seg := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[seg]
	}
	return cur
}

func mapToItem(m map[string]any, mapping map[string]string, enums map[string]map[string]string, layout string) (datasource.FeedbackItem, error) {
	get := func(key string) any { return m[mapping[key]] }
	enumResolve := func(enumKey string, v any) string {
		val := toString(v)
		if e, ok := enums[enumKey]; ok {
			if label, ok := e[val]; ok {
				return label
			}
		}
		return val
	}

	it := datasource.FeedbackItem{
		OriginalID: toString(get("original_id")),
		AppID:      toInt(get("app_id")),
		Platform:   enumResolve("platform_id", get("platform_id")),
		UserID:     toString(get("user_id")),
		UserName:   toString(get("user_name")),
		Content:    toString(get("content")),
		PhoneModel: toString(get("phone_model")),
		AppVersion: normalizeVersion(toString(get("app_version"))),
		ChannelID:  toString(get("channel_id")),
		QQ:         toString(get("qq")),
		FileURL:    toString(get("file_url")),
		Raw:        m,
	}
	if appName := enumResolve("app_id", get("app_id")); appName != "" && it.AppID != 0 {
		it.AppName = appName
	}
	if pid := toIntPtr(get("platform_id")); pid != nil {
		it.PlatformID = pid
	}
	if um := toIntPtr(get("user_mode")); um != nil {
		it.UserMode = um
	}
	if imgs := get("images"); imgs != nil {
		switch v := imgs.(type) {
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok && s != "" {
					it.Images = append(it.Images, s)
				}
			}
		case string:
			if v != "" {
				it.Images = []string{v}
			}
		}
	}
	if vids := get("videos"); vids != nil {
		switch v := vids.(type) {
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok && s != "" {
					it.Videos = append(it.Videos, s)
				}
			}
		case string:
			if v != "" {
				it.Videos = []string{v}
			}
		}
	}
	if ts := toString(get("original_created_at")); ts != "" {
		t, err := time.ParseInLocation(layout, ts, time.Local)
		if err != nil {
			return it, fmt.Errorf("parse time %q: %w", ts, err)
		}
		it.OriginalCreatedAt = t
	} else {
		it.OriginalCreatedAt = time.Now()
	}
	return it, nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func toInt(v any) int {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	}
	return 0
}

func toIntPtr(v any) *int {
	if v == nil {
		return nil
	}
	n := toInt(v)
	return &n
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// normalizeVersion 将版本号每段补零到至少两位。
// "9.7.0" → "09.07.00"，"9.07.0" → "09.07.00"，"10.2.30" → "10.02.30"
func normalizeVersion(v string) string {
	if v == "" {
		return ""
	}
	parts := strings.Split(v, ".")
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			// 非纯数字段保持原样
			continue
		}
		parts[i] = fmt.Sprintf("%02d", n)
	}
	return strings.Join(parts, ".")
}
