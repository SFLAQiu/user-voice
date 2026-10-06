// Package datasource defines the plugin contract for feedback data sources.
package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// FeedbackItem is the normalized payload yielded by every plugin.
type FeedbackItem struct {
	OriginalID        string
	AppID             int
	AppName           string
	Platform          string
	PlatformID        *int
	UserID            string
	UserName          string
	UserMode          *int
	Content           string
	Images            []string
	Videos            []string
	PhoneModel        string
	AppVersion        string
	ChannelID         string
	QQ                string
	FileURL           string
	OriginalCreatedAt time.Time
	Raw               map[string]any
}

// Cursor allows incremental sync. Plugins may store an opaque key per logical stream.
type Cursor struct {
	Key            string
	LastOriginalID string
	LastCreatedAt  *time.Time
}

// Plugin is the contract every data source must satisfy.
type Plugin interface {
	Type() string
	Validate(config json.RawMessage) error
	// Pull fetches items newer than the provided cursor. The returned next cursor must
	// reflect the latest item seen. The skipFn lets the scheduler stop pagination early
	// when an already-imported id is encountered.
	Pull(ctx context.Context, raw json.RawMessage, cursor *Cursor,
		exists func(originalID string) (bool, error)) (PullResult, error)
}

// PullResult is the outcome of one Pull invocation.
type PullResult struct {
	Items      []FeedbackItem
	NextCursor *Cursor
}

// Registry holds installed plugins by type.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[p.Type()] = p
}

func (r *Registry) Get(typ string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[typ]
	if !ok {
		return nil, fmt.Errorf("unknown data source plugin: %s", typ)
	}
	return p, nil
}

func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.plugins))
	for k := range r.plugins {
		out = append(out, k)
	}
	return out
}
