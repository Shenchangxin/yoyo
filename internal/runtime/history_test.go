package runtime

import (
	"strings"
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

func TestMessagesFromEventsTwoTurns(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "first"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "one", "id": "s:r1"}},
		{Type: trace.TypeUser, Payload: map[string]any{"text": "second"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "two", "id": "s:r2"}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 4 {
		t.Fatalf("%+v", msgs)
	}
	if msgs[0].Content != "first" || msgs[1].Content != "one" || msgs[2].Content != "second" || msgs[3].Content != "two" {
		t.Fatalf("%+v", msgs)
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

func TestMessagesFromEventsIncludesMentionInject(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeInject, Source: "mention", Payload: map[string]any{"text": "README.md"}},
		{Type: trace.TypeUser, Payload: map[string]any{"text": "summarize"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "ok", "id": "s:r1"}},
		{Type: trace.TypeInject, Source: "skill", Payload: map[string]any{"text": "skill body"}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 3 {
		t.Fatalf("%+v", msgs)
	}
	if msgs[0].Role != RoleUser || !strings.Contains(msgs[0].Content, "README.md") {
		t.Fatalf("inject %+v", msgs[0])
	}
	if msgs[1].Content != "summarize" {
		t.Fatalf("user %+v", msgs[1])
	}
}

func TestMaxAssistantRound(t *testing.T) {
	n := MaxAssistantRound([]trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "hi"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"id": "s:r1"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"round": "s:r12"}},
	})
	if n != 12 {
		t.Fatalf("got %d", n)
	}
}

func TestMessagesFromEventsDedupesToolIDs(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "read it"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "", "id": "s:r1"}},
		{Type: trace.TypeToolCall, Payload: map[string]any{"id": "c1", "name": "read_file", "arguments": `{"path":"a.go"}`}},
		{Type: trace.TypeToolResult, Payload: map[string]any{"id": "c1", "name": "read_file", "content": "package a", "elapsed_ms": 4}},
		{Type: trace.TypeToolCall, Payload: map[string]any{"id": "c1", "name": "read_file", "arguments": `{"path":"a.go"}`}},
		{Type: trace.TypeToolResult, Payload: map[string]any{"id": "c1", "name": "read_file", "content": "package a"}},
	}
	msgs := MessagesFromEvents(evs)
	calls, results := 0, 0
	for _, m := range msgs {
		if m.Role == RoleAssistant {
			calls += len(m.ToolCalls)
		}
		if m.Role == RoleTool {
			results++
			if m.Content != "package a" {
				t.Fatalf("content %q", m.Content)
			}
		}
	}
	if calls != 1 || results != 1 {
		t.Fatalf("calls=%d results=%d msgs=%+v", calls, results, msgs)
	}
}
