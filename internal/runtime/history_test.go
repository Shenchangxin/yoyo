package runtime

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestMessagesFromEvents(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeSystem, Payload: map[string]any{"text": "old system"}},
		{Type: trace.TypeUser, Payload: map[string]any{"text": "hello"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "working"}},
		{Type: trace.TypeToolCall, Payload: map[string]any{"id": "1", "name": "write_file", "arguments": "{}"}},
		{Type: trace.TypeToolResult, Payload: map[string]any{"id": "1", "name": "write_file", "content": "wrote x"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "done"}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) < 4 {
		t.Fatalf("%+v", msgs)
	}
	if msgs[0].Role != RoleUser || msgs[0].Content != "hello" {
		t.Fatalf("first %+v", msgs[0])
	}
	for _, m := range msgs {
		if m.Role == RoleSystem {
			t.Fatal("system should be omitted")
		}
	}
}

func TestMessagesFromEventsSkipsDeltas(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "hi"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "He", "delta": true}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "llo", "delta": true}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "Hello"}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 2 {
		t.Fatalf("%+v", msgs)
	}
	if msgs[1].Content != "Hello" {
		t.Fatalf("got %q", msgs[1].Content)
	}
}
