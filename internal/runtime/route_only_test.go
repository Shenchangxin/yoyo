package runtime

import "testing"

// TestRouteOnlyHidesShell 锁住路由会话只广告 Extra，shell 和 tool_search 都不能调用。
func TestRouteOnlyHidesShell(t *testing.T) {
	tools := &WorkspaceTools{
		RouteOnly: true,
		Extra: map[string]ExtraTool{
			"task_route": {JSON: ToolJSON{Type: "function", Function: map[string]any{"name": "task_route"}}},
		},
	}
	for _, tool := range AllToolJSON(tools) {
		name, _ := tool.Function["name"].(string)
		if name == "shell" || name == "tool_search" || name == "write_file" {
			t.Fatal(name)
		}
	}
	if !tools.toolAllowed("task_route") || tools.toolAllowed("shell") || tools.toolAllowed("tool_search") {
		t.Fatal("filter")
	}
}
