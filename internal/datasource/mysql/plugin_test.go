package mysql

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/feedback/internal/datasource"
)

// 保留插件核心纯逻辑回归：校验、标识符防注入、行映射。
func TestValidate(t *testing.T) {
	p := New(nil)
	ok := json.RawMessage(`{"host":"h","username":"u","database":"d","table":"fb","time_field":"created_at","field_mapping":{"original_id":"id","content":"txt","original_created_at":"ctime"}}`)
	if err := p.Validate(ok); err != nil {
		t.Fatalf("expect valid, got %v", err)
	}
	if err := p.Validate(json.RawMessage(`{"host":"h"}`)); err == nil {
		t.Fatal("expect invalid config to fail")
	}
}

func TestQuoteIdentRejectsInjection(t *testing.T) {
	if q, _ := quoteIdent("fb_2024"); q != "`fb_2024`" {
		t.Fatalf("unexpected quote: %s", q)
	}
	if _, err := quoteIdent("fb`); DROP TABLE users;--"); err == nil {
		t.Fatal("identifier with backtick must be rejected")
	}
}

func TestRowToItemMapsColumns(t *testing.T) {
	now := time.Now()
	row := map[string]any{"id": int64(7), "txt": "hello", "ctime": now}
	mapping := map[string]string{"original_id": "id", "content": "txt", "original_created_at": "ctime"}
	it, err := rowToItem(row, mapping, nil)
	if err != nil {
		t.Fatal(err)
	}
	if it.OriginalID != "7" || it.Content != "hello" || !it.OriginalCreatedAt.Equal(now) {
		t.Fatalf("bad item: %+v", it)
	}
}

// 编译期契约校验
var _ datasource.Plugin = (*Plugin)(nil)
