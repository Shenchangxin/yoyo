// 锁住轨迹重建：忽略 system 和 delta，折叠 mention，工具 id 去重，没有结果的调用补上未知结果。

package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestMessagesFromEvents 确认重建从用户消息开始，system 事件不进入结果。
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

// TestMessagesFromEventsTwoTurns 确认两轮用户和助手按原顺序保留四条消息。
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

// TestMessagesFromEventsSkipsDeltas 确认 delta 为真的助手片段被跳过，只留下最终的 Hello。
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

// TestMessagesFromEventsIncludesMentionInject 确认 mention 贴进后面的用户消息，skill 注入被丢掉。
func TestMessagesFromEventsIncludesMentionInject(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeInject, Source: "mention", Payload: map[string]any{"text": "README.md"}},
		{Type: trace.TypeUser, Payload: map[string]any{"text": "summarize"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "ok", "id": "s:r1"}},
		{Type: trace.TypeInject, Source: "skill", Payload: map[string]any{"text": "skill body"}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 2 {
		t.Fatalf("%+v", msgs)
	}
	if msgs[0].Role != RoleUser || !strings.Contains(msgs[0].Content, "README.md") || !strings.HasPrefix(msgs[0].Content, "summarize") {
		t.Fatalf("inject %+v", msgs[0])
	}
	if msgs[1].Content != "ok" {
		t.Fatalf("asst %+v", msgs[1])
	}
}

// TestMessagesFromEventsFoldsMentionAfterUser 确认用户原文仍在前面，随后的 mention 技能附在同一条用户消息上。
func TestMessagesFromEventsFoldsMentionAfterUser(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "帮我查论文"}},
		{Type: trace.TypeInject, Source: "mention", Payload: map[string]any{"text": "## @skill:arxiv-watcher\nUse scripts"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "好", "id": "s:r1"}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 2 {
		t.Fatalf("%+v", msgs)
	}
	if !strings.HasPrefix(msgs[0].Content, "帮我查论文") {
		t.Fatalf("operator must lead: %q", msgs[0].Content)
	}
	if !strings.Contains(msgs[0].Content, "arxiv-watcher") {
		t.Fatalf("skill dropped: %q", msgs[0].Content)
	}
}

// TestMaxAssistantRound 确认 id 为 s:r1、round 为 s:r12 时最大回合号是 12。
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

// TestMessagesFromEventsDedupesToolIDs 确认同一个工具 id 的再次调用和再次结果只留一份。
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

// TestMessagesFromEventsClosesUnansweredTool 确认为没有结果的 shell 调用补上含 TOOL_OUTCOME_UNKNOWN 的工具消息。
func TestMessagesFromEventsClosesUnansweredTool(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "edit it"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "", "id": "s:r1"}},
		{Type: trace.TypeToolCall, Payload: map[string]any{"id": "c1", "name": "str_replace", "arguments": `{"path":"a.go"}`}},
		{Type: trace.TypeToolCall, Payload: map[string]any{"id": "c2", "name": "shell", "arguments": `{"command":"go test"}`}},
		{Type: trace.TypeToolResult, Payload: map[string]any{"id": "c1", "name": "str_replace", "content": "edited"}},
	}
	msgs := MessagesFromEvents(evs)
	var shell string
	for _, m := range msgs {
		if m.Role == RoleTool && m.ToolCallID == "c2" {
			shell = m.Content
		}
	}
	if !strings.Contains(shell, "TOOL_OUTCOME_UNKNOWN") {
		t.Fatalf("missing result %q msgs=%+v", shell, msgs)
	}
}

// TestMessagesFromEventsKeepsRepeatReminder 确认 repeat-tool-reminder 原样变成下一条用户消息，并且算控制行。
func TestMessagesFromEventsKeepsRepeatReminder(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "继续"}},
		{Type: trace.TypeInject, Source: "repeat-tool-reminder", Payload: map[string]any{"text": repeatGentleZH}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 2 || msgs[1].Content != repeatGentleZH || !isControlUser(msgs[1].Content) {
		t.Fatalf("%+v", msgs)
	}
}

// TestMessagesFromEventsKeepsImageParts 确认用户事件上的图片 parts 留在重建出的消息里。
func TestMessagesFromEventsKeepsImageParts(t *testing.T) {
	evs := []trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{
			"text":  "look",
			"parts": []map[string]any{{"type": "image_url", "image_url": "data:image/png;base64,xx"}},
		}},
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 1 || len(msgs[0].Parts) != 1 || msgs[0].Parts[0].ImageURL == "" {
		t.Fatalf("%+v", msgs)
	}
}
