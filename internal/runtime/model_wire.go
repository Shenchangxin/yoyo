// 把内部消息收成 OpenAI Chat Completions 的 JSON。
// 不发 HTTP。OpenAIClient.doChat 调用 wireMessages。

package runtime

import "fmt"

// wireContent 在没有 Parts 时返回 Content 字符串。有 Parts 时收成 text 与 image_url 数组；结果为空则退回 Content。
func wireContent(m Message) any {
	if len(m.Parts) == 0 {
		return m.Content
	}
	parts := make([]map[string]any, 0, len(m.Parts)+1)
	if m.Content != "" {
		parts = append(parts, map[string]any{"type": "text", "text": m.Content})
	}
	for _, p := range m.Parts {
		if p.Type == "image_url" && p.ImageURL != "" {
			parts = append(parts, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": p.ImageURL},
			})
			continue
		}
		if p.Text != "" {
			parts = append(parts, map[string]any{"type": "text", "text": p.Text})
		}
	}
	if len(parts) == 0 {
		return m.Content
	}
	return parts
}

// wireMessages 把内部记录投影成 OpenAI Chat Completions 的 JSON。内部 ToolCall 是扁平的 {id,name,arguments}；线上是 {id,type,function:{name,arguments}}。网关（DashScope 一类）会以 “Field required: …tool_calls.N.function” 拒绝扁平形状。
// RoleDeveloper 写成 name=developer 的 system。RoleMemory 写成 name=working_memory 的 user。
func wireMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		row := map[string]any{"role": string(m.Role)}
		switch m.Role {
		case RoleTool:
			row["tool_call_id"] = m.ToolCallID
			row["content"] = wireContent(m)
			if m.Name != "" {
				row["name"] = m.Name
			}
		case RoleDeveloper:
			row["role"] = "system"
			row["name"] = "developer"
			row["content"] = wireContent(m)
		case RoleMemory:
			row["role"] = "user"
			row["name"] = "working_memory"
			row["content"] = wireContent(m)
		case RoleAssistant:
			calls := wireToolCalls(m.ToolCalls)
			if len(calls) > 0 {
				row["tool_calls"] = calls
				row["content"] = wireContent(m)
			} else {
				row["content"] = wireContent(m)
			}
		default:
			row["content"] = wireContent(m)
			if m.Name != "" {
				row["name"] = m.Name
			}
		}
		out = append(out, row)
	}
	return out
}

// wireToolCalls 把工具调用嵌进 function。名字为空的条目丢掉。参数为空时写成 {}。ID 为空时用 call_<名字>_<下标>。
func wireToolCalls(calls []ToolCall) []map[string]any {
	if len(calls) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(calls))
	for i, tc := range calls {
		name := tc.Name
		if name == "" {
			continue
		}
		args := tc.Arguments
		if args == "" {
			args = "{}"
		}
		id := tc.ID
		if id == "" {
			id = fmt.Sprintf("call_%s_%d", name, i)
		}
		out = append(out, map[string]any{
			"id":   id,
			"type": "function",
			"function": map[string]any{
				"name":      name,
				"arguments": args,
			},
		})
	}
	return out
}
