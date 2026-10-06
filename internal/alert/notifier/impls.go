package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WeCom struct{}

func (WeCom) Type() string { return "wecom" }

func (WeCom) Send(ctx context.Context, webhookURL, _ string, msg string) error {
	body, _ := json.Marshal(map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{"content": msg},
	})
	return post(ctx, webhookURL, body)
}

type Feishu struct{}

func (Feishu) Type() string { return "feishu" }

func (Feishu) Send(ctx context.Context, webhookURL, _ string, msg string) error {
	body, _ := json.Marshal(map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"elements": []map[string]interface{}{
				{"tag": "markdown", "content": msg},
			},
		},
	})
	return post(ctx, webhookURL, body)
}

type DingTalk struct{}

func (DingTalk) Type() string { return "dingtalk" }

func (DingTalk) Send(ctx context.Context, webhookURL, _ string, msg string) error {
	body, _ := json.Marshal(map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{"text": msg, "title": "告警通知"},
	})
	return post(ctx, webhookURL, body)
}

// Webhook is a generic HTTP POST sender.
type Webhook struct{}

func (Webhook) Type() string { return "webhook" }

func (Webhook) Send(ctx context.Context, webhookURL, _ string, msg string) error {
	body, _ := json.Marshal(map[string]string{"message": msg})
	return post(ctx, webhookURL, body)
}

// Registry holds all notifier implementations.
var Registry = map[string]Notifier{
	"wecom":    WeCom{},
	"feishu":   Feishu{},
	"dingtalk": DingTalk{},
	"webhook":  Webhook{},
}

func post(ctx context.Context, url string, body []byte) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}
