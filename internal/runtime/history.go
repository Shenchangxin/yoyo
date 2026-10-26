// 从会话轨迹重建发给模型的消息。实时系统提示由当前 harness 另装，不在这里复原。
// 会话开跑前用 MessagesFromEventsOpts 收成历史，再交给 MaybeCheckpoint。读别的线程时也会调用。JSONL 的截断不在这个文件。

package runtime

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// MessagesFromEvents 从会话轨迹重建发给模型的消息，供下一轮继续。实时系统提示由当前 harness 另装，旧的 system 事件被忽略，升级后的快照下一轮才生效。
// kind 为 checkpoint 的 TypeCompact 是重建起点：摘要加保留的尾部替换它之前的一切。JSONL 本身不会被截短。spill 传 nil。
func MessagesFromEvents(evs []trace.Event) []Message {
	return MessagesFromEventsOpts(evs, nil)
}

// MessagesFromEventsOpts 同样重建消息。检查点若存了 tail_id 且 spill 能取回非空正文，用这份热尾，不再读 payload 里的 tail。
// delta 为真的事件跳过。prompt-update 和 repeat-tool-reminder 在正文非空白时原样变成用户消息。mention 贴进尚未被别的消息隔开的上一条用户消息，否则先攒着，最后仍没有用户消息就单独成条。其他来源的注入丢弃。
// 同一工具 id 的再次 tool_call 丢弃。同一 id 的再次 tool_result 只在原结果正文为空白、新正文非空白时补上。用户文本为空但带了 parts 时，文本写成 (image attached)。
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
	pendingMention := ""
	lastUser := -1
	seenCall := map[string]bool{}
	seenResult := map[string]int{}
	applyMention := func(user string) string {
		text := AttachMention(user, pendingMention)
		pendingMention = ""
		return text
	}
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
			parts := partsFromPayload(ev.Payload["parts"])
			if text != "" || len(parts) > 0 {
				if text == "" {
					text = "(image attached)"
				}
				out = append(out, Message{Role: RoleUser, Content: applyMention(text), Parts: parts})
				lastUser = len(out) - 1
			}
		case trace.TypeInject:
			src := ev.Source
			if src == "" {
				if name, _ := ev.Payload["name"].(string); name != "" {
					src = name
				}
			}
			text, _ := ev.Payload["text"].(string)
			if src == "prompt-update" || src == "repeat-tool-reminder" {
				if strings.TrimSpace(text) == "" {
					break
				}
				flush()
				out = append(out, Message{Role: RoleUser, Content: text})
				break
			}
			if src != "mention" {
				break
			}
			if strings.TrimSpace(text) == "" {
				break
			}
			if lastUser >= 0 && lastUser == len(out)-1 && out[lastUser].Role == RoleUser {
				out[lastUser].Content = AttachMention(out[lastUser].Content, text)
				break
			}
			if pendingMention != "" {
				pendingMention += "\n" + text
			} else {
				pendingMention = text
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
	if pendingMention != "" {
		out = append(out, Message{Role: RoleUser, Content: AttachMention("", pendingMention)})
	}
	if len(head) == 0 {
		return closeUnanswered(out)
	}
	return closeUnanswered(append(head, out...))
}

// closeUnanswered 给紧跟在助手消息后面、却没有对应结果的工具调用补一条未知结果。调用已经记进轨迹，所以结果是未知，而不是还没开始。空 id 的调用也会补上。只查看紧邻的一串工具消息。
func closeUnanswered(msgs []Message) []Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]Message, 0, len(msgs)+2)
	for i := 0; i < len(msgs); i++ {
		m := msgs[i]
		out = append(out, m)
		if m.Role != RoleAssistant || len(m.ToolCalls) == 0 {
			continue
		}
		have := map[string]bool{}
		j := i + 1
		for j < len(msgs) && msgs[j].Role == RoleTool {
			have[msgs[j].ToolCallID] = true
			out = append(out, msgs[j])
			j++
		}
		for _, tc := range m.ToolCalls {
			if tc.ID != "" && have[tc.ID] {
				continue
			}
			name := tc.Name
			if name == "" {
				name = "tool"
			}
			out = append(out, Message{
				Role:       RoleTool,
				ToolCallID: tc.ID,
				Name:       name,
				Content:    toolOutcomeUnknown,
			})
		}
		i = j - 1
	}
	return out
}

// MaxAssistantRound 返回助手事件的 id 或 round 字段里已经出现过的最大 rN。下一次实时回合用更大的序号，避免检查点重建历史后和界面上还在的气泡撞号。没有助手事件时返回 0。
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

// parseRoundIndex 取最后一个 :r 后面的非负整数。没有 :r、解析失败或是负数时返回 0。
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

// partsFromPayload 把 payload 里的 parts 解成内容片段。类型、图片地址和文本都空的片段丢掉。解不出来或结果为空时返回 nil。
func partsFromPayload(v any) []ContentPart {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var parts []ContentPart
	if json.Unmarshal(raw, &parts) != nil {
		return nil
	}
	out := parts[:0]
	for _, p := range parts {
		if p.Type == "" && p.ImageURL == "" && p.Text == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// payloadString 把参数收成字符串。nil 返回空串，已经是字符串时原样返回，其他值做 JSON，失败时返回空串。
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

// DropTrailingUser 从重建出的历史末尾去掉刚落盘的那条操作者消息，好让 MaybeCheckpoint 仍看到之前的尾巴，再由 seed 重新附上。
// text 规范化后为空、历史为空、末尾不是用户消息，或去掉 mention 后缀后仍对不上，则原样返回。
func DropTrailingUser(hist []Message, text string) []Message {
	text = NormalizeUserText(text)
	if text == "" || len(hist) == 0 {
		return hist
	}
	last := hist[len(hist)-1]
	if last.Role != RoleUser {
		return hist
	}
	body := last.Content
	if i := strings.Index(body, "\n\n"+mentionUserPrefix); i >= 0 {
		body = body[:i]
	}
	if NormalizeUserText(body) != text {
		return hist
	}
	return hist[:len(hist)-1]
}
