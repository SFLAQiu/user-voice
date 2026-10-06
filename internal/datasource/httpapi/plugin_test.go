package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/feedback/internal/crypto"
	"github.com/feedback/internal/datasource"
)

func makeCipher(t *testing.T) *crypto.Cipher {
	t.Helper()
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	c, err := crypto.New(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHTTPAPIPullParsesFixture(t *testing.T) {
	fixture := `{
      "data": {
        "feedback_question_list": [
          {"id":"100","app_id":"1","platform_id":"2","user_id":"u1","screen_name":"alice",
           "user_mode":"3","content":"闪退","image":["http://x/y.jpg"],
           "phone_mode":"iPhone15","version_name":"9.7.0","channel_id":"22","created_at":"2026-05-28 10:00:00"},
          {"id":"99","app_id":"1","platform_id":"2","user_id":"u2","screen_name":"bob",
           "content":"广告太多","image":"","phone_mode":"P40","version_name":"9.07.0","channel_id":"6","created_at":"2026-05-28 09:00:00"}
        ]
      }
    }`

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			_, _ = w.Write([]byte(fixture))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"feedback_question_list":[]}}`))
	}))
	defer srv.Close()

	cfg := Config{
		Endpoint: srv.URL,
		Method:   "GET",
		ParamsTemplate: map[string]string{"app_id": "{{app_id}}", "platform_id": "{{platform_id}}"},
		Iterate:      []map[string]any{{"app_id": 1, "platform_id": 2}},
		DataPath:     "data.feedback_question_list",
		PagePaginate: PagePaginateConfig{Param: "to_page", Start: 1},
		FieldMapping: map[string]string{
			"original_id":         "id",
			"app_id":              "app_id",
			"platform_id":         "platform_id",
			"user_id":             "user_id",
			"user_name":           "screen_name",
			"user_mode":           "user_mode",
			"content":             "content",
			"images":              "image",
			"phone_model":         "phone_mode",
			"app_version":         "version_name",
			"channel_id":          "channel_id",
			"original_created_at": "created_at",
		},
		StopWhenSeen: true,
	}
	rawCfg, _ := json.Marshal(cfg)

	p := New(makeCipher(t))
	if err := p.Validate(rawCfg); err != nil {
		t.Fatalf("validate: %v", err)
	}

	res, err := p.Pull(context.Background(), rawCfg, nil, func(string) (bool, error) { return false, nil })
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	if res.Items[0].OriginalID != "100" {
		t.Fatalf("first item id mismatch: %q", res.Items[0].OriginalID)
	}
	if got := res.Items[0].PlatformID; got == nil || *got != 2 {
		t.Fatalf("platform_id not parsed: %v", got)
	}
	if len(res.Items[0].Images) != 1 {
		t.Fatalf("images not parsed: %+v", res.Items[0].Images)
	}
	// 版本号归一化：9.7.0 和 9.07.0 都应变成 09.07.00
	if res.Items[0].AppVersion != "09.07.00" {
		t.Fatalf("item 0 app_version: got %q, want %q", res.Items[0].AppVersion, "09.07.00")
	}
	if res.Items[1].AppVersion != "09.07.00" {
		t.Fatalf("item 1 app_version: got %q, want %q", res.Items[1].AppVersion, "09.07.00")
	}
}

func TestHTTPAPIStopsWhenSeen(t *testing.T) {
	fixture := `{"data":{"feedback_question_list":[
        {"id":"100","content":"a","created_at":"2026-01-01 00:00:00"}
    ]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	cfg := Config{
		Endpoint:     srv.URL,
		DataPath:     "data.feedback_question_list",
		PagePaginate: PagePaginateConfig{Param: "page", Start: 1},
		FieldMapping: map[string]string{
			"original_id":         "id",
			"content":             "content",
			"original_created_at": "created_at",
		},
		StopWhenSeen: true,
		MaxPages:     3,
	}
	rawCfg, _ := json.Marshal(cfg)

	p := New(makeCipher(t))
	called := 0
	exists := func(string) (bool, error) { called++; return true, nil }
	res, err := p.Pull(context.Background(), rawCfg, nil, exists)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("expected 0 new items, got %d", len(res.Items))
	}
	if called != 1 {
		t.Fatalf("expected exists to be called once before stopping, got %d", called)
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct{ in, want string }{
		{"9.7.0", "09.07.00"},
		{"9.07.0", "09.07.00"},
		{"10.2.30", "10.02.30"},
		{"9.7", "09.07"},
		{"", ""},
		{"1", "01"},
		{"v2.3.4", "v2.03.04"},
		{"2.3.4-beta", "02.03.4-beta"},
	}
	for _, tt := range tests {
		got := normalizeVersion(tt.in)
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

var _ datasource.Plugin = (*Plugin)(nil)
