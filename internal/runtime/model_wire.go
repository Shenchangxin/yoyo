package runtime

import "fmt"

// wireMessages projects the internal transcript into OpenAI Chat Completions
// JSON. Internal ToolCall is flat {id,name,arguments}; the wire form is
// {id,type,function:{name,arguments}}. Gateways (DashScope and similar)
// reject the flat shape with "Field required: …tool_calls.N.function".
func wireMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		row := map[string]any{"role": string(m.Role)}
		switch m.Role {
		case RoleTool:
			row["tool_call_id"] = m.ToolCallID
			row["content"] = m.Content
			if m.Name != "" {
				row["name"] = m.Name
			}
		case RoleAssistant:
			calls := wireToolCalls(m.ToolCalls)
			if len(calls) > 0 {
				row["tool_calls"] = calls
				row["content"] = m.Content
			} else {
				row["content"] = m.Content
			}
		default:
			row["content"] = m.Content
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
