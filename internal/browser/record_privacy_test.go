// 本文件锁住密钥启发式和地址查询打码。由 go test 调用。

package browser

import (
	"net/url"
	"strings"
	"testing"
)

// TestRedactRecordValueUsesSecretHeuristic 锁住像密钥的值会打码，普通邮箱原样留下。
func TestRedactRecordValueUsesSecretHeuristic(t *testing.T) {
	got := redactRecordValue(RecordTarget{Name: "email"}, "sk-live-secret")
	if got != "{{sensitive_value}}" {
		t.Fatalf("api key: %q", got)
	}
	got = redactRecordValue(RecordTarget{Type: "password"}, "hunter2")
	if got != "{{sensitive_value}}" {
		t.Fatalf("password field: %q", got)
	}
	got = redactRecordValue(RecordTarget{Name: "email"}, "user@example.com")
	if got != "user@example.com" {
		t.Fatalf("plain email: %q", got)
	}
}

// TestRedactRecordURLMasksTokens 锁住查询里的 code 变成 ***，其他参数保留。
func TestRedactRecordURLMasksTokens(t *testing.T) {
	got := redactRecordURL("https://example.com/cb?code=abc&keep=1")
	if strings.Contains(got, "code=abc") {
		t.Fatalf("oauth code leaked: %q", got)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("code") != "***" || u.Query().Get("keep") != "1" {
		t.Fatalf("%q", got)
	}
}
