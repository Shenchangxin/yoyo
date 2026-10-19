package browser

import (
	"net/url"
	"strings"
	"testing"
)

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
