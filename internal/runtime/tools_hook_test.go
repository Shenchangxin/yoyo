// 锁住工作区 .yoyo/hooks.json 的 pre_tool 规则在 WorkspaceTools.Call 里生效：匹配到 Deny 时工具不执行。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileHookDeniesTool 确认 FileHooks 里匹配 write_file 的 Deny 规则让 Call 返回规则里的原因，文件没有写出来。
// 桌面、编排工单和评测 Harbor 都经 agentloop/desk 调 Call，这道拦截只在这里，不依赖事件总线。
func TestFileHookDeniesTool(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{
		Workspace: dir,
		FileHooks: []FileHook{{Match: "write_file", Deny: true, Reason: "blocked by test hook"}},
	}
	res := tools.Call("write_file", `{"path":"x.txt","content":"no"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "blocked by test hook") {
		t.Fatalf("want hook denial, got content=%q err=%v", res.Content, res.Err)
	}
	if _, err := os.Stat(filepath.Join(dir, "x.txt")); err == nil {
		t.Fatal("write should have been denied")
	}
}
