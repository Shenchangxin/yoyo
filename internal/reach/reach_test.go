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

func TestNextRotatesSourcesAndKeepsTheDefaultTopic(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	lines := []string{"开源模型", "短剧"}
	got, ok := Next(Log{}, lines, now)
	if !ok || got.Kind != KindHN || !strings.Contains(got.Query, "AI") || !strings.Contains(got.Query, "开源模型") {
		t.Fatalf("%#v %v", got, ok)
	}
	if _, ok := Next(Log{Next: now.Add(time.Hour).Format(time.RFC3339)}, lines, now); ok {
		t.Fatal("not due")
	}
	if _, ok := Next(Log{Pending: &Item{Title: "待说", URL: "https://example.com"}}, lines, now); ok {
		t.Fatal("pending blocks")
	}
	cat := Catalog()
	bili := -1
	seen := map[string]bool{}
	for i, src := range cat {
		if src.ID == "" || src.Name == "" || src.Kind == "" || seen[src.ID] {
			t.Fatalf("source %#v", src)
		}
		seen[src.ID] = true
		if src.Kind == KindFeed && !strings.HasPrefix(src.URL, "https://") {
			t.Fatalf("feed url %#v", src)
		}
		if src.ID == "bilibili" {
			bili = i
		}
	}
	if len(cat) < 12 || bili < 0 {
		t.Fatalf("catalog %d bili %d", len(cat), bili)
	}
	video, ok := Next(Log{Cursor: bili}, nil, now)
	topic := DefaultTopics[bili%len(DefaultTopics)]
	if !ok || video.Kind != KindWeb || !strings.Contains(video.Query, "site:bilibili.com") || !strings.Contains(video.Query, topic) {
		t.Fatalf("bilibili %#v", video)
	}
}

func TestPaceAndAllowanceFollowHowRecentlyTheUserSpoke(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	if PaceOf(time.Time{}, false, now) != PaceQuiet || PaceOf(now.Add(-5*time.Hour), false, now) != PaceQuiet {
		t.Fatal("quiet")
	}
	if PaceOf(now.Add(-time.Hour), false, now) != PaceSteady {
		t.Fatal("steady")
	}
	if PaceOf(now.Add(-5*time.Minute), false, now) != PaceBusy || PaceOf(time.Time{}, true, now) != PaceBusy {
		t.Fatal("busy")
	}
	if !Hold(now.Add(-time.Minute), false, now) || Hold(now.Add(-30*time.Minute), false, now) || !Hold(time.Time{}, true, now) {
		t.Fatal("hold")
	}
	quiet, busy := AllowanceFor(PaceQuiet), AllowanceFor(PaceBusy)
	if quiet.Daily != 12 || quiet.Speak != 35*time.Minute || busy.Daily != 6 || busy.Speak != 90*time.Minute || busy.Probe <= quiet.Probe {
		t.Fatalf("quiet %#v busy %#v", quiet, busy)
	}
	if ProbeAfter(PaceQuiet, now) < quiet.Probe || ProbeAfter(PaceBusy, now) < busy.Probe {
		t.Fatal("probe gap")
	}
}

func TestPreferDropsOffTopicAndRememberCapsSkips(t *testing.T) {
	items := []Item{
		{Title: "周末市集", URL: "https://example.com/a", Snippet: "手作"},
		{Title: "新的智能体框架", URL: "https://example.com/b", Snippet: "Agent"},
	}
	got := Prefer(items, false, "AI", "")
	if len(got) != 1 || got[0].URL != "https://example.com/b" {
		t.Fatalf("%#v", got)
	}
	if mail := Prefer([]Item{{Title: "email weekly", URL: "https://example.com/m"}}, false, "AI", ""); len(mail) != 0 {
		t.Fatalf("email is not AI: %#v", mail)
	}
	kept := Prefer(items, true, "AI", "周末")
	if len(kept) != 2 || kept[0].URL != "https://example.com/a" {
		t.Fatalf("%#v", kept)
	}
	ids := Remember(nil, "reach:a")
	ids = Remember(ids, "reach:a")
	if len(ids) != 1 {
		t.Fatalf("%#v", ids)
	}
	for i := 0; i < 205; i++ {
		ids = Remember(ids, "reach:"+strings.Repeat("n", i+1))
	}
	if len(ids) != 200 || ids[0] == "reach:n" {
		t.Fatalf("len %d first %s", len(ids), ids[0])
	}
}

func TestParseExaKeepsPublicLinks(t *testing.T) {
	text := "Title: Step 5\nURL: https://example.com/step\nPublished: N/A\n\n免费预览\nTitle: Local\nURL: http://127.0.0.1/x\n"
	items := ParseExa(text)
	if len(items) != 1 || items[0].URL != "https://example.com/step" || items[0].Snippet != "免费预览" {
		t.Fatalf("%#v", items)
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
