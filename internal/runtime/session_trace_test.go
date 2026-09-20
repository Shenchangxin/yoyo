package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestProjectTraceSkipsDeltasAndKeepsCalls(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	evs := []trace.Event{
		{TS: ts, Type: trace.TypeUser, Source: "ui", Payload: map[string]any{"text": "list files", "id": "u1"}},
		{TS: ts.Add(time.Millisecond), Type: trace.TypeAssistant, Payload: map[string]any{"text": "hi", "delta": true, "id": "s:r1"}},
		{TS: ts.Add(20 * time.Millisecond), Type: trace.TypeAssistant, Payload: map[string]any{"text": "I will list them.", "id": "s:r1", "round": "s:r1"}},
		{TS: ts.Add(30 * time.Millisecond), Type: trace.TypeToolCall, Payload: map[string]any{"id": "c1", "name": "shell", "arguments": `{"cmd":"ls"}`, "round": "s:r1"}},
		{TS: ts.Add(80 * time.Millisecond), Type: trace.TypeToolResult, Payload: map[string]any{
			"id": "c1", "name": "shell",
			"content":    "[elided tool_result id=c1 name=shell bytes=4096 — call recall_context with this id]\nhead",
			"spill_id":   "c1",
			"bytes":      4096,
			"elapsed_ms": 12,
			"round":      "s:r1",
		}},
		{TS: ts.Add(90 * time.Millisecond), Type: trace.TypeCompact, Payload: map[string]any{"kind": "shape", "note": "ok", "tokens": 1800}},
		{TS: ts.Add(100 * time.Millisecond), Type: trace.TypeTurnEnd, Payload: map[string]any{"ok": true}},
	}
	view := ProjectTrace(evs)
	if view.Stats.Events != 7 || view.Stats.Deltas != 1 || view.Stats.ToolCalls != 1 || view.Stats.Users != 1 {
		t.Fatalf("stats %+v", view.Stats)
	}
	if view.Stats.Tokens != 1800 {
		t.Fatalf("tokens %d", view.Stats.Tokens)
	}
	if view.Stats.DurationMs != 100 {
		t.Fatalf("duration %d", view.Stats.DurationMs)
	}
	if len(view.Events) != 6 {
		t.Fatalf("projected %d", len(view.Events))
	}
	var sawSpill bool
	for _, ev := range view.Events {
		if ev.Type == "tool_result" {
			if ev.SpillID != "c1" || ev.Bytes != 4096 || ev.ElapsedMs != 12 {
				t.Fatalf("tool_result %+v", ev)
			}
			if ev.Lane != "exec" {
				t.Fatalf("lane %s", ev.Lane)
			}
			sawSpill = true
		}
		if ev.Type == "assistant" && strings.Contains(ev.Summary, "hi") && ev.Summary != "I will list them." {
			t.Fatalf("delta leaked: %s", ev.Summary)
		}
	}
	if !sawSpill {
		t.Fatal("missing tool_result")
	}
}

func TestProjectTraceRecoversSpillIDFromStub(t *testing.T) {
	evs := []trace.Event{{
		Type: trace.TypeToolResult,
		Payload: map[string]any{
			"name":    "read_file",
			"id":      "old",
			"content": "[elided tool_result id=blob9 name=read_file bytes=88 — call recall_context with this id]\npreview",
		},
	}}
	view := ProjectTrace(evs)
	if len(view.Events) != 1 || view.Events[0].SpillID != "blob9" || view.Events[0].Bytes != 88 {
		t.Fatalf("%+v", view.Events)
	}
}

func TestIngestReturnsSpillID(t *testing.T) {
	sp := NewSpill(t.TempDir())
	full := strings.Repeat("TAIL", 20_000)
	preview, id := ingestToolResult(sp, "call-1", "shell", full, 0, false)
	if id != "call-1" {
		t.Fatalf("id %q", id)
	}
	if !strings.Contains(preview, "elided") {
		t.Fatalf("preview %s", preview[:min(80, len(preview))])
	}
	got, err := sp.Get("call-1")
	if err != nil || got != full {
		t.Fatalf("spill err=%v len=%d", err, len(got))
	}
}
