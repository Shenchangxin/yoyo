package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// MessagesFromEvents rebuilds an OpenAI-style transcript from a session
// trajectory so Send() can continue a conversation. The live system prompt
// is assembled separately from the active harness — old system events are
// ignored so a promoted snapshot takes effect on the next turn.
//
// A TypeCompact event with kind=checkpoint is a rebuild origin: summary +
// retained tail replace everything before it. JSONL is never truncated.
func MessagesFromEvents(evs []trace.Event) []Message {
	start := 0
	var head []Message
	for i, ev := range evs {
		if ev.Type != trace.TypeCompact {
			continue
		}
		kind, _ := ev.Payload["kind"].(string)
		if kind != "checkpoint" {
			continue
		}
		start = i + 1
		head = nil
		if sum, _ := ev.Payload["summary"].(string); strings.TrimSpace(sum) != "" {
			head = append(head, Message{
				Role:    RoleUser,
				Content: "Context checkpoint (untrusted working memory; pins were reassembled separately):\n" + sum,
			})
		}
		if tail := messagesFromAny(ev.Payload["tail"]); len(tail) > 0 {
			head = append(head, stripSystem(tail)...)
		}
	}
	var out []Message
	var pending Message
	flush := func() {
		if pending.Role == "" {
			return
		}
		if pending.Content == "" && len(pending.ToolCalls) == 0 {
			pending = Message{}
			return
		}
		out = append(out, pending)
		pending = Message{}
	}
	for _, ev := range evs[start:] {
		if delta, _ := ev.Payload["delta"].(bool); delta {
			continue
		}
		switch ev.Type {
		case trace.TypeUser:
			flush()
			text, _ := ev.Payload["text"].(string)
			if text != "" {
				out = append(out, Message{Role: RoleUser, Content: text})
			}
		case trace.TypeAssistant:
			flush()
			text, _ := ev.Payload["text"].(string)
			pending = Message{Role: RoleAssistant, Content: text}
		case trace.TypeToolCall:
			if pending.Role != RoleAssistant {
				flush()
				pending = Message{Role: RoleAssistant}
			}
			id, _ := ev.Payload["id"].(string)
			name, _ := ev.Payload["name"].(string)
			args, _ := ev.Payload["arguments"].(string)
			pending.ToolCalls = append(pending.ToolCalls, ToolCall{ID: id, Name: name, Arguments: args})
		case trace.TypeToolResult:
			flush()
			id, _ := ev.Payload["id"].(string)
			name, _ := ev.Payload["name"].(string)
			content, _ := ev.Payload["content"].(string)
			out = append(out, Message{Role: RoleTool, ToolCallID: id, Name: name, Content: content})
		}
	}
	flush()
	if len(head) == 0 {
		return out
	}
	return append(head, out...)
}
