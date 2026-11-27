// .yoyo/hooks.json 的 pre_tool 规则判断时用的载荷。规则的读取和匹配在 hooks_file.go；WorkspaceTools.Call 在执行工具前调用 applyFileHooks。

package runtime

// ToolHook 是一次工具调用的名字、参数和会话。applyFileHooks 匹配到 Deny 规则时把 Deny 设为真并写入 Reason，Call 就不执行这个工具。
type ToolHook struct {
	Name      string
	Arguments string
	SessionID string
	Deny      bool
	Reason    string
}
