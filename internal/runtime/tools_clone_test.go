// 锁住子代理拿到的工具集私有状态是独立副本，父子各自写入不会互相影响。

package runtime

import "testing"

// TestCloneIsolatesPrivateState 确认 clone 之后：路径规则和待解锁宿主工具继承下来，但 map 和 slice 是独立副本；
// 正在执行的 shell 调用不继承；父子各自有一把锁。
func TestCloneIsolatesPrivateState(t *testing.T) {
	parent := &WorkspaceTools{Workspace: t.TempDir()}
	parent.pendingHost = []string{"browser_open"}
	parent.pathRules = "rule-a"
	parent.pathRuleSeen = map[string]bool{"a/AGENTS.md": true}
	parent.liveCall = "call-1"

	child := parent.clone()
	if child.pathRules != "rule-a" || !child.pathRuleSeen["a/AGENTS.md"] {
		t.Fatalf("child must inherit path rules: %q %v", child.pathRules, child.pathRuleSeen)
	}
	if len(child.pendingHost) != 1 || child.pendingHost[0] != "browser_open" {
		t.Fatalf("child must inherit pending host tools: %v", child.pendingHost)
	}
	child.pathRuleSeen["b/AGENTS.md"] = true
	if parent.pathRuleSeen["b/AGENTS.md"] {
		t.Fatal("child wrote into the parent's pathRuleSeen map")
	}
	child.pendingHost[0] = "changed"
	if parent.pendingHost[0] != "browser_open" {
		t.Fatal("child shares the parent's pendingHost backing array")
	}
	if child.liveCall != "" {
		t.Fatalf("child must not inherit the parent's live shell call: %q", child.liveCall)
	}
	if child.lock() == parent.lock() {
		t.Fatal("child must get its own lock")
	}
}
