package app

import (
	"fmt"
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
}

func TestHubCoalesceKeepsIndexRange(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeAssistant,
		Payload: map[string]any{"text": "he", "delta": true, "id": "s:r1", "round": "s:r1", "index": 1, "block": "text"},
	})
	h.Publish(trace.Event{
		SessionID: "s", Type: trace.TypeAssistant,
		Payload: map[string]any{"text": "llo", "delta": true, "id": "s:r1", "round": "s:r1", "index": 2, "block": "text"},
	})
	h.FlushCoalesce()
	ev := recv(t, ch)
	if got, _ := ev.Payload["text"].(string); got != "hello" {
		t.Fatalf("coalesced text %q", got)
	}
	if payloadInt(ev.Payload, "index") != 2 || payloadInt(ev.Payload, "index_from") != 1 {
		t.Fatalf("index range %+v", ev.Payload)
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

func TestHubCoalescesAssistantStreamChunks(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	chunk := func(index int, text string) trace.Event {
		return trace.Event{
			SessionID: "s", Type: trace.TypeAssistantStream,
			Payload: map[string]any{
				"kind": "chunk", "attempt_id": "a1", "round": "s:r1", "id": "s:r1",
				"index": index, "chunk": map[string]any{"type": "text-delta", "index": 0, "text": text},
			},
		}
	}
	h.Publish(chunk(0, "he"))
	h.Publish(chunk(1, "llo"))
	h.FlushCoalesce()
	ev := recv(t, ch)
	body := payloadMap(ev.Payload["chunk"])
	got, _ := body["text"].(string)
	if got != "hello" {
		t.Fatalf("coalesced stream %q payload %+v", got, ev.Payload)
	}
	if payloadInt(ev.Payload, "index") != 1 || payloadInt(ev.Payload, "index_from") != 0 {
		t.Fatalf("index range %+v", ev.Payload)
	}
	n := NoticeOf(ev)
	if n.Event == nil || n.Frame == nil || n.Frame.Index != 1 || n.Frame.IndexFrom != 0 {
		t.Fatalf("notice frame %+v", n.Frame)
	}
	if n.Frame.Chunk["text"] != "hello" {
		t.Fatalf("frame chunk %+v", n.Frame.Chunk)
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

func TestHubDropsLiveWhenSubscriberBlocked(t *testing.T) {
	h := NewHub()
	ch, off := h.Subscribe("s")
	defer off()
	start := time.Now()
	for i := 0; i < 8200; i++ {
		h.Publish(trace.Event{
			SessionID: "s", Type: trace.TypeToolCall,
			Payload: map[string]any{"name": "write_file", "id": fmt.Sprintf("%d", i)},
		})
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("Publish blocked on slow subscriber")
	}
	h.Publish(trace.Event{SessionID: "s", Type: trace.TypeTurnEnd, Payload: map[string]any{"id": "end"}})
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type == trace.TypeTurnEnd {
				return
			}
		case <-deadline:
			t.Fatal("turn_end never arrived")
		}
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
