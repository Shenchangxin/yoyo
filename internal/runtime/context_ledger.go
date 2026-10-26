// 把前缀、动态尾和工具 schema 的估算 token 写进 ShapeReport。计量和 compact 事件读这些字段。
// 不决定裁哪些消息。DynamicAt 为空时写成 tail，表示动态块在历史后面。

package runtime

import "encoding/json"

// toolsJSONTokens 把工具 schema 编成 JSON 再估算 token。列表为空或编码失败时返回 0。
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

// fillLedger 写入前缀、动态尾、schema 和窗口。DynamicAt 为空时写成 tail。
// rep 为 nil 时直接返回。
// prefix 交给 estimateTokens，写入 PrefixTokens。
// dynamic 交给 estimateTokens，写入 DynamicTokens。
// tools 交给 toolsJSONTokens。
// window 写入 Window。
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
