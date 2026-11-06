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
	noted := *first
	noted.Note = "跟你记过的短剧对得上。\nhttps://example.com/step\n最小的下一步：有空再看。现在忽略也没有任何影响。"
	got, ok := Emit(&noted, nil)
	if !ok || !strings.Contains(got.Detail, "现在忽略也没有任何影响") || strings.Contains(got.Detail, "免费到 25 日") {
		t.Fatalf("%#v", got)
	}
}

func TestNextRotatesNewsAndHot(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	lines := []string{"开源模型", "短剧"}
	got, ok := Next(Log{}, lines, now)
	if !ok || got.Channel != "web" || got.Query != "开源模型" {
		t.Fatalf("%#v %v", got, ok)
	}
	if _, ok := Next(Log{Next: now.Add(time.Hour).Format(time.RFC3339)}, lines, now); ok {
		t.Fatal("not due")
	}
	hot, ok := Next(Log{Cursor: 1}, nil, now)
	if !ok || hot.Channel != "bilibili" || hot.Action != "hot" {
		t.Fatalf("hot %#v", hot)
	}
	news, ok := Next(Log{Cursor: 3}, lines, now)
	if !ok || news.Query != "短剧 新闻" {
		t.Fatalf("news %#v", news)
	}
	social := Command(Probe{Channel: "twitter", Action: "search", Query: "热点"})
	if !strings.Contains(strings.Join(social, " "), "public-web") {
		t.Fatalf("%#v", social)
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
