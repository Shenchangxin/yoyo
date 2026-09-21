package runtime

import "encoding/json"

func toolsJSONTokens(tools []ToolJSON) int {
	if len(tools) == 0 {
		return 0
	}
	b, err := json.Marshal(tools)
	if err != nil {
		return 0
	}
	return estimateTokens(string(b))
}

func fillLedger(rep *ShapeReport, prefix, dynamic string, tools []ToolJSON, window int) {
	if rep == nil {
		return
	}
	rep.PrefixTokens = estimateTokens(prefix)
	rep.DynamicTokens = estimateTokens(dynamic)
	rep.SchemaTokens = toolsJSONTokens(tools)
	rep.Window = window
	if rep.DynamicAt == "" {
		rep.DynamicAt = "tail"
	}
}
