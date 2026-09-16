package runtime

import "strings"

// Legalize makes an OpenAI Chat Completions transcript valid:
// every assistant.tool_calls id has a following role=tool message, empty
// tool bodies get an explicit anchor, and orphan tool messages are kept as
// user text so stubbed bytes are not dropped from a projection.
//
// User/orphan rows are never inserted between an assistant.tool_calls set
// and its matching role=tool messages.
func Legalize(msgs []Message) []Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]Message, 0, len(msgs)+4)
	pending := map[string]ToolCall{}
	var order []string
	var held []string

	flushOpen := func() {
		for _, id := range order {
			tc, ok := pending[id]
			if !ok {
				continue
			}
			name := tc.Name
			if name == "" {
				name = "tool"
			}
			out = append(out, Message{
				Role:       RoleTool,
				ToolCallID: id,
				Name:       name,
				Content:    emptyToolAnchor(name),
			})
		}
		pending = map[string]ToolCall{}
		order = nil
		for _, text := range held {
			if strings.TrimSpace(text) != "" {
				out = append(out, Message{Role: RoleUser, Content: text})
			}
		}
		held = nil
	}

	openPair := func() bool {
		return len(pending) > 0 || len(order) > 0
	}

	for _, m := range msgs {
		switch m.Role {
		case RoleTool:
			name := m.Name
			if name == "" {
				name = "tool"
			}
			content := m.Content
			if strings.TrimSpace(content) == "" {
				content = emptyToolAnchor(name)
			}
			if _, ok := pending[m.ToolCallID]; ok {
				out = append(out, Message{
					Role:       RoleTool,
					ToolCallID: m.ToolCallID,
					Name:       name,
					Content:    content,
				})
				delete(pending, m.ToolCallID)
				continue
			}
			if openPair() {
				held = append(held, content)
				continue
			}
			out = append(out, Message{Role: RoleUser, Content: content})
		case RoleAssistant:
			flushOpen()
			asst := m
			if n := len(m.ToolCalls); n > 0 {
				calls := make([]ToolCall, n)
				copy(calls, m.ToolCalls)
				for i := range calls {
					if calls[i].ID == "" {
						calls[i].ID = "tc" + itoa(i+1)
					}
				}
				asst.ToolCalls = calls
			}
			out = append(out, asst)
			if len(asst.ToolCalls) == 0 {
				continue
			}
			pending = map[string]ToolCall{}
			order = make([]string, 0, len(asst.ToolCalls))
			for _, tc := range asst.ToolCalls {
				pending[tc.ID] = tc
				order = append(order, tc.ID)
			}
		default:
			if openPair() && m.Role == RoleUser {
				held = append(held, m.Content)
				continue
			}
			flushOpen()
			out = append(out, m)
		}
	}
	flushOpen()
	return out
}

func pairingBoundary(msgs []Message, cut int) int {
	if cut < 1 {
		return 1
	}
	if cut >= len(msgs) {
		return len(msgs)
	}
	for cut > 1 && msgs[cut].Role == RoleTool {
		cut--
	}
	if cut > 1 && msgs[cut-1].Role == RoleAssistant && len(msgs[cut-1].ToolCalls) > 0 {
		cut--
	}
	for cut > 1 && msgs[cut].Role == RoleTool {
		cut--
	}
	return cut
}

func emptyToolAnchor(name string) string {
	if name == "" {
		name = "tool"
	}
	return "(" + name + " completed with no output)"
}
