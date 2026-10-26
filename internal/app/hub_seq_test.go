// 实时序号、Since，以及通知是否把正文内联。

package app

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestHubAssignsSeqAndSince 确认发布会分配序号，Since 能补到后面的事件，并且通知带上序号。
func TestHubAssignsSeqAndSince(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	h.Publish(trace.Event{SessionID: "s", Type: trace.TypeUser, Payload: map[string]any{"text": "hi"}})
	ev := recv(t, ch)
	if ev.Seq != 1 {
		t.Fatalf("seq %d", ev.Seq)
	}
	h.Publish(trace.Event{SessionID: "s", Type: trace.TypeTurnEnd, Payload: map[string]any{"id": "e"}})
	_ = recv(t, ch)
	got := h.Since(1, "s")
	if len(got) != 1 || got[0].Type != trace.TypeTurnEnd {
		t.Fatalf("%+v", got)
	}
	n := NoticeOf(got[0])
	if n.Seq == 0 || n.Event == nil {
		t.Fatalf("notice %+v", n)
	}
}

// TestNoticeOmitsToolCallAndSettledAssistant 确认工具通知带短字段，过长参数不进通知，已落定的助手正文只留序号。
func TestNoticeOmitsToolCallAndSettledAssistant(t *testing.T) {
	call := NoticeOf(trace.Event{Seq: 2, SessionID: "s", Type: trace.TypeToolCall, Payload: map[string]any{"name": "read_file", "id": "c1", "arguments": `{"path":"a.go"}`}})
	if call.Event == nil || call.Event.Payload["name"] != "read_file" {
		t.Fatalf("compact tool_call must inline, got %+v", call)
	}
	huge := strings.Repeat("x", 8000)
	fat := NoticeOf(trace.Event{Seq: 2, SessionID: "s", Type: trace.TypeToolCall, Payload: map[string]any{
		"name": "write_file", "id": "c2", "path": "a.html", "arguments": `{"path":"a.html","content":"` + huge + `"}`,
	}})
	if fat.Event == nil || fat.Event.Payload["name"] != "write_file" {
		t.Fatalf("fat tool_call must still inline a compact notice, got %+v", fat)
	}
	if _, ok := fat.Event.Payload["arguments"]; ok {
		t.Fatal("fat arguments must not ride the Wails notice")
	}
	settled := NoticeOf(trace.Event{Seq: 2, SessionID: "s", Type: trace.TypeToolResult, Payload: map[string]any{
		"id": "c1", "name": "shell", "content": "hello", "added": 1, "removed": 1,
	}})
	if settled.Event == nil || settled.Event.Payload["delta"] != false {
		t.Fatalf("settled tool_result must inline with delta=false, got %+v", settled.Event)
	}
	letter := NoticeOf(trace.Event{Seq: 3, SessionID: "s", Type: trace.TypeAssistant, Payload: map[string]any{"text": "done", "id": "r1"}})
	if letter.Event != nil {
		t.Fatal("settled assistant must be pull-only")
	}
	delta := NoticeOf(trace.Event{Seq: 4, SessionID: "s", Type: trace.TypeAssistant, Payload: map[string]any{"text": "hi", "delta": true, "id": "r1"}})
	if delta.Event == nil {
		t.Fatal("tiny token slice should inline")
	}
	indexed := NoticeOf(trace.Event{Seq: 4, SessionID: "s", Type: trace.TypeAssistant, Payload: map[string]any{"text": "hi", "delta": true, "id": "r1", "index": 3, "index_from": 1, "block": "text"}})
	if indexed.Frame == nil || indexed.Frame.Index != 3 || indexed.Frame.IndexFrom != 1 || indexed.Frame.Block != "text" {
		t.Fatalf("frame %+v", indexed.Frame)
	}
	body := make([]byte, 4000)
	for i := range body {
		body[i] = 'x'
	}
	large := NoticeOf(trace.Event{Seq: 5, SessionID: "s", Type: trace.TypeAssistant, Payload: map[string]any{"text": string(body), "delta": true, "id": "r1"}})
	if large.Event == nil {
		t.Fatal("token slices must stay inline even when a 50ms window exceeds the byte cap")
	}
	prog := NoticeOf(trace.Event{Seq: 6, SessionID: "s", Type: trace.TypeToolCall, Payload: map[string]any{"name": "write_file", "id": "c1", "progress": true, "bytes": 1200, "path": "a.html", "arguments": `{"path":"a.html"}`}})
	if prog.Event == nil {
		t.Fatal("tiny tool-arg progress must inline")
	}
	stream := NoticeOf(trace.Event{Seq: 7, SessionID: "s", Type: trace.TypeAssistantStream, Payload: map[string]any{
		"kind": "chunk", "attempt_id": "a1", "index": 0, "index_from": 0,
		"chunk": map[string]any{"type": "text-delta", "index": 0, "text": "hi"},
	}})
	if stream.Event == nil || stream.Frame == nil || stream.Frame.Kind != "chunk" || stream.Frame.IndexFrom != 0 {
		t.Fatalf("assistant_stream notice %+v", stream)
	}
	start := NoticeOf(trace.Event{Seq: 8, SessionID: "s", Type: trace.TypeAssistantStream, Payload: map[string]any{
		"kind": "start", "attempt_id": "a1", "turn": 1, "step": 1,
	}})
	if start.Event == nil || start.Frame == nil || start.Frame.Kind != "start" {
		t.Fatalf("assistant_stream start %+v", start)
	}
}

// TestNoticeOmitsLargePayload 确认过大的已落定助手事件不内联，序号仍在。
func TestNoticeOmitsLargePayload(t *testing.T) {
	body := make([]byte, 4000)
	for i := range body {
		body[i] = 'x'
	}
	ev := trace.Event{Seq: 9, SessionID: "s", Type: trace.TypeAssistant, Payload: map[string]any{"text": string(body)}}
	n := NoticeOf(ev)
	if n.Event != nil {
		t.Fatal("expected pull-only notice")
	}
	if n.Seq != 9 {
		t.Fatalf("seq %d", n.Seq)
	}
}
