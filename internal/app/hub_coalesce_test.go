package app

import (
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestHubCoalescesAssistantDeltas(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeAssistant,
		Payload: map[string]any{"text": "he", "delta": true, "id": "s:r1", "round": "s:r1"},
	})
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeAssistant,
		Payload: map[string]any{"text": "llo", "delta": true, "id": "s:r1", "round": "s:r1"},
	})
	h.FlushCoalesce()
	ev := recv(t, ch)
	if got, _ := ev.Payload["text"].(string); got != "hello" {
		t.Fatalf("coalesced text %q", got)
	}
	if delta, _ := ev.Payload["delta"].(bool); !delta {
		t.Fatal("delta flag dropped")
	}
	select {
	case extra := <-ch:
		t.Fatalf("extra event %+v", extra)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestHubFlushesOnTypeChange(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeAssistant,
		Payload: map[string]any{"text": "hi", "delta": true, "id": "s:r1"},
	})
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeToolCall,
		Payload: map[string]any{"name": "shell", "id": "c1"},
	})
	asst := recv(t, ch)
	if got, _ := asst.Payload["text"].(string); got != "hi" {
		t.Fatalf("flushed assistant %q", got)
	}
	call := recv(t, ch)
	if call.Type != trace.TypeToolCall {
		t.Fatalf("want tool_call got %s", call.Type)
	}
}

func TestHubDoesNotMergeDifferentRounds(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeAssistant,
		Payload: map[string]any{"text": "a", "delta": true, "id": "s:r1", "round": "s:r1"},
	})
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeAssistant,
		Payload: map[string]any{"text": "b", "delta": true, "id": "s:r2", "round": "s:r2"},
	})
	h.FlushCoalesce()
	first := recv(t, ch)
	second := recv(t, ch)
	a, _ := first.Payload["text"].(string)
	b, _ := second.Payload["text"].(string)
	if a != "a" || b != "b" {
		t.Fatalf("rounds mixed %q %q", a, b)
	}
}

func TestHubCoalescesShellStdout(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeToolResult,
		Payload: map[string]any{"content": "hel", "delta": true, "id": "c1", "name": "shell"},
	})
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeToolResult,
		Payload: map[string]any{"content": "lo", "delta": true, "id": "c1", "name": "shell"},
	})
	h.FlushCoalesce()
	ev := recv(t, ch)
	if got, _ := ev.Payload["content"].(string); got != "hello" {
		t.Fatalf("stdout %q", got)
	}
}

func recv(t *testing.T, ch <-chan trace.Event) trace.Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatal("timeout")
		return trace.Event{}
	}
}
