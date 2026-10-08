package app

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

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

func TestNoticeOmitsToolCallAndSettledAssistant(t *testing.T) {
	call := NoticeOf(trace.Event{Seq: 2, SessionID: "s", Type: trace.TypeToolCall, Payload: map[string]any{"name": "read_file", "arguments": `{"path":"a.go"}`}})
	if call.Event != nil {
		t.Fatal("tool_call must be pull-only")
	}
	letter := NoticeOf(trace.Event{Seq: 3, SessionID: "s", Type: trace.TypeAssistant, Payload: map[string]any{"text": "done", "id": "r1"}})
	if letter.Event != nil {
		t.Fatal("settled assistant must be pull-only")
	}
	delta := NoticeOf(trace.Event{Seq: 4, SessionID: "s", Type: trace.TypeAssistant, Payload: map[string]any{"text": "hi", "delta": true, "id": "r1"}})
	if delta.Event == nil {
		t.Fatal("tiny token slice should inline")
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
}

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
