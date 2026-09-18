package runtime

import "fmt"

// wireMessages projects the internal transcript into OpenAI Chat Completions
// JSON. Internal ToolCall is flat {id,name,arguments}; the wire form is
// {id,type,function:{name,arguments}}. Gateways (DashScope and similar)
// reject the flat shape with "Field required: …tool_calls.N.function".
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
