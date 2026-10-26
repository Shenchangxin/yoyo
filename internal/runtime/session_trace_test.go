// shape 行计入 Projections，不算 Compactions。delta 不进时间线。裁剪后的统计仍是全量。

package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestProjectTraceSkipsDeltasAndKeepsCalls 确认 delta 不进时间线，工具调用保留，shape 的 token 写入统计但不算压缩。
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

// TestTrimTraceViewDropsNoiseAndTails 确认裁剪后统计仍是全长，压缩为 1，时间线丢掉 file_change 且不超过保留条数。
func TestTrimTraceViewDropsNoiseAndTails(t *testing.T) {
	var evs []trace.Event
	evs = append(evs, trace.Event{Type: trace.TypeUser, Payload: map[string]any{"text": "go"}})
	for i := 0; i < 40; i++ {
		evs = append(evs, trace.Event{Type: trace.TypeFileChange, Payload: map[string]any{"path": "x"}})
		evs = append(evs, trace.Event{Type: trace.TypeCompact, Payload: map[string]any{"kind": "shape", "tokens": 9}})
		evs = append(evs, trace.Event{Type: trace.TypeToolCall, Payload: map[string]any{"id": "c", "name": "read_file", "arguments": "{}"}})
	}
	evs = append(evs, trace.Event{Type: trace.TypeCompact, Payload: map[string]any{"kind": "checkpoint", "tokens": 9}})
	view := TrimTraceView(ProjectTrace(evs), 10)
	if view.Stats.Events != len(evs) {
		t.Fatalf("stats must stay full: %d != %d", view.Stats.Events, len(evs))
	}
	// shape 行是投影抽样，不是压缩。只有 checkpoint 才计入压缩。
	if view.Stats.Compactions != 1 || view.Stats.Projections != 40 {
		t.Fatalf("compactions %d projections %d", view.Stats.Compactions, view.Stats.Projections)
	}
	if len(view.Events) > 10 {
		t.Fatalf("timeline %d", len(view.Events))
	}
	for _, ev := range view.Events {
		if ev.Type == "file_change" || (ev.Type == "compaction" && ev.Name != "checkpoint") {
			t.Fatalf("noise leaked %+v", ev)
		}
	}
}

// TestProjectTraceRecoversSpillIDFromStub 确认从省略正文恢复 spill id blob9 和字节数 88。
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

// TestProjectTraceUsageCarriesLatencyAndCache 确认用量摘要含 1240ms 和 cache 80%。
func TestProjectTraceUsageCarriesLatencyAndCache(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
	view := ProjectTrace([]trace.Event{{
		TS:     ts,
		Type:   trace.TypeUsage,
		Source: "model",
		Payload: map[string]any{
			"round":             "s1:r2",
			"prompt_tokens":     200,
			"completion_tokens": 40,
			"cached_tokens":     160,
			"cache_reported":    true,
			"elapsed_ms":        1240,
			"ttft_ms":           180,
		},
	}})
	if len(view.Events) != 1 {
		t.Fatalf("events %d", len(view.Events))
	}
	ev := view.Events[0]
	if ev.Type != "model_usage" || ev.PromptTokens != 200 || ev.CompletionTokens != 40 || ev.CachedTokens != 160 {
		t.Fatalf("tokens %+v", ev)
	}
	if ev.ElapsedMs != 1240 || ev.TtftMs != 180 || !ev.CacheReported {
		t.Fatalf("timing %+v", ev)
	}
	if !strings.Contains(ev.Summary, "1240ms") || !strings.Contains(ev.Summary, "cache 80%") {
		t.Fatalf("summary %q", ev.Summary)
	}
}

// TestIngestReturnsSpillID 确认超长工具结果被省略，spill 里仍能取回全文。
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
