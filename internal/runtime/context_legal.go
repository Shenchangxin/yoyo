// 把投影整理成 OpenAI 聊天补全能接受的消息序列。kernel.prompt、projectChat、MeasureContext 和 retainKeepTokens 会调用。
// 只返回新切片。不改入参，也不写 JSONL。

package runtime

import "strings"

// Legalize 整理发给模型的投影。每个 assistant.tool_calls 的 id 后面都要有 role=tool；缺结果时补 toolOutcomeUnknown，空正文换成 emptyToolAnchor。对不上调用的 tool 改成用户文本，占位字节不会从投影里丢掉。
// 一组 tool_calls 还没配齐时，用户消息和孤立 tool 先暂存，配齐后再追加，避免插进配对中间。空的 tool call id 写成 tc 加序号。返回新切片，不修改入参，也不写 JSONL。
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
				Content:    toolOutcomeUnknown,
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
				// 空的 role=tool 正文换成一句完成说明，避免投影里出现空工具消息。
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
				// 这一组 tool_calls 还没配齐。对不上的工具正文先暂存，配齐后再作为用户文本追加，不插进配对中间。
				held = append(held, content)
				continue
			}
			// 没有未完成的配对时，对不上调用的 tool 改成用户文本。字节不从投影里丢掉。
			out = append(out, Message{Role: RoleUser, Content: content})
		case RoleAssistant:
			flushOpen()
			asst := m
			if n := len(m.ToolCalls); n > 0 {
				calls := make([]ToolCall, n)
				copy(calls, m.ToolCalls)
				for i := range calls {
					if calls[i].ID == "" {
						// 空的 tool call id 写成 tc 加从 1 起的序号。只改投影副本，不回写 JSONL。
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

// pairingBoundary 把剪切点退到不拆开 assistant.tool_calls 和紧随的 role=tool。cut 小于 1 时返回 1，达到或超过长度时返回长度。
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

// emptyToolAnchor 给空的工具正文一个明确句子，避免投影里出现空的 role=tool。
func emptyToolAnchor(name string) string {
	if name == "" {
		name = "tool"
	}
	return "(" + name + " completed with no output)"
}

// toolOutcomeUnknown 用于调用已经记下、但结果还不知道的情况。不把它说成成功的空完成。尚未派发的调用用 toolAbortedBeforeDispatch。
const toolOutcomeUnknown = "ERROR: TOOL_OUTCOME_UNKNOWN: this call was recorded but its outcome is unknown. Retry only if it is read-only or idempotent. If it may have changed files or state, check that state or ask the user before retrying."

// toolAbortedBeforeDispatch 表示调用在派发前被中止。
const toolAbortedBeforeDispatch = "ERROR: ABORTED_BEFORE_DISPATCH: tool call aborted before dispatch. Retry it if it is still needed."
