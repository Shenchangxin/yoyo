package runtime

import (
	"encoding/json"
	"strconv"
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
	return MessagesFromEventsOpts(evs, nil)
}

// MessagesFromEventsOpts rebuilds a transcript. When a checkpoint stored a
// spill tail_id, spill is used to restore the retained hot tail.
func MessagesFromEventsOpts(evs []trace.Event, spill *Spill) []Message {
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
		if spill != nil {
			if id, _ := ev.Payload["tail_id"].(string); strings.TrimSpace(id) != "" {
				if raw, err := spill.Get(id); err == nil && raw != "" {
					var v any
					if json.Unmarshal([]byte(raw), &v) == nil {
						if tail := messagesFromAny(v); len(tail) > 0 {
							head = append(head, stripSystem(tail)...)
							continue
						}
					}
				}
			}
		}
		if tail := messagesFromAny(ev.Payload["tail"]); len(tail) > 0 {
			head = append(head, stripSystem(tail)...)
		}
	}
	var out []Message
	var pending Message
	seenCall := map[string]bool{}
	seenResult := map[string]int{}
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
		case trace.TypeInject:
			if ev.Source != "mention" {
				break
			}
			flush()
			text, _ := ev.Payload["text"].(string)
			if text != "" {
				out = append(out, Message{
					Role:    RoleUser,
					Content: "Attached context (user @mentions, untrusted working memory):\n" + text,
				})
			}
		case trace.TypeAssistant:
			flush()
			text, _ := ev.Payload["text"].(string)
			pending = Message{Role: RoleAssistant, Content: text}
		case trace.TypeToolCall:
			id, _ := ev.Payload["id"].(string)
			if id != "" && seenCall[id] {
				break
			}
			if pending.Role != RoleAssistant {
				flush()
				pending = Message{Role: RoleAssistant}
			}
			name, _ := ev.Payload["name"].(string)
			args := payloadString(ev.Payload["arguments"])
			pending.ToolCalls = append(pending.ToolCalls, ToolCall{ID: id, Name: name, Arguments: args})
			if id != "" {
				seenCall[id] = true
			}
		case trace.TypeToolResult:
			flush()
			id, _ := ev.Payload["id"].(string)
			name, _ := ev.Payload["name"].(string)
			content, _ := ev.Payload["content"].(string)
			if id != "" {
				if idx, ok := seenResult[id]; ok {
					if strings.TrimSpace(out[idx].Content) == "" && strings.TrimSpace(content) != "" {
						out[idx].Content = content
					}
					break
				}
			}
			out = append(out, Message{Role: RoleTool, ToolCallID: id, Name: name, Content: content})
			if id != "" {
				seenResult[id] = len(out) - 1
			}
		}
	}
	flush()
	if len(head) == 0 {
		return out
	}
	return append(head, out...)
}

// MaxAssistantRound is the highest rN already stamped on assistant events.
// Used so the next live round id does not collide with bubbles the UI still
// holds after a checkpoint rebuild of History.
func MaxAssistantRound(evs []trace.Event) int {
	max := 0
	for _, ev := range evs {
		if ev.Type != trace.TypeAssistant {
			continue
		}
		if ev.Payload == nil {
			continue
		}
		for _, key := range []string{"id", "round"} {
			raw, _ := ev.Payload[key].(string)
			if n := parseRoundIndex(raw); n > max {
				max = n
			}
		}
	}
	return max
}

func parseRoundIndex(id string) int {
	i := strings.LastIndex(id, ":r")
	if i < 0 {
		return 0
	}
	n, err := strconv.Atoi(id[i+2:])
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func payloadString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
