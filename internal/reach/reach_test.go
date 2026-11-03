package reach

import (
	"strings"
	"testing"
	"time"
)

func TestLinesDropSecretsAndKeepAShortProfile(t *testing.T) {
	got := Lines([]string{"开源模型", "sk-token-abc", "https://example.com", "开源模型", "a", strings.Repeat("长", 50)})
	if len(got) != 1 || got[0] != "开源模型" {
		t.Fatalf("%#v", got)
	}
}

func TestParseEnvelopeAndChooseSkipsSeen(t *testing.T) {
	raw := `noise
{"ok":true,"items":[{"title":"Step 5","url":"https://example.com/step","text":"免费到 25 日"},{"title":"nope","url":"http://insecure.example"}]}`
	items := ParseEnvelope(raw)
	if len(items) != 2 {
		t.Fatalf("%#v", items)
	}
	seen := map[string]bool{}
	first := Choose(items, seen)
	if first == nil || first.URL != "https://example.com/step" {
		t.Fatalf("%#v", first)
	}
	seen[ID(first.URL)] = true
	if Choose(items, seen) != nil {
		t.Fatal("seen link should not return")
	}
	if _, ok := Emit(first, seen); ok {
		t.Fatal("emit seen")
	}
	fact, ok := Emit(first, nil)
	if !ok || !strings.Contains(fact.Detail, "https://example.com/step") || fact.Code != "reach" {
		t.Fatalf("%#v", fact)
	}
}

func TestPlanWaitsAndRotates(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	lines := []string{"开源模型", "短剧"}
	q, ok := Plan(Log{}, lines, now)
	if !ok || q != "开源模型" {
		t.Fatalf("%s %v", q, ok)
	}
	if _, ok := Plan(Log{Next: now.Add(time.Hour).Format(time.RFC3339)}, lines, now); ok {
		t.Fatal("not due")
	}
	if _, ok := Plan(Log{}, nil, now); ok {
		t.Fatal("no profile")
	}
	log := Log{Cursor: 1}
	q, ok = Plan(log, lines, now)
	if !ok || q != "短剧" {
		t.Fatalf("rotate %s", q)
	}
}

func TestParseHost(t *testing.T) {
	text := "1. Step 5\n   https://example.com/step\n   免费预览\n2. Local\n   http://127.0.0.1/x\n"
	items := ParseHost(text)
	if len(items) != 2 || items[0].URL != "https://example.com/step" || items[0].Snippet != "免费预览" {
		t.Fatalf("%#v", items)
	}
	if Choose(items, nil).URL != "https://example.com/step" {
		t.Fatal("http dropped")
	}
}
