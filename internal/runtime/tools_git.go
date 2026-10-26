// 工作区里的 git 子命令。git_status、git_diff、git_commit 经 tools_host.go 调用这里。不解析 diff，hunk 选择在 hunks.go。

package runtime

import (
	"context"
	"os/exec"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// git 在工作区目录执行 git。write 为 false 时能力级别是 ReadWorkspace，为 true 时是 WriteWorkspace，动作名是 git。输出含 not a git repository 时不当成错误，返回该说明。
// args 用空格拼进能力检查的命令文本，并作为 git 的参数传给 exec.CommandContext。
func (t *WorkspaceTools) git(args []string, write bool) ToolResult {
	level := capability.ReadWorkspace
	if write {
		level = capability.WriteWorkspace
	}
	if err := t.check(level, "git", t.Workspace, strings.Join(args, " ")); err != nil {
		return ToolResult{Err: err}
	}
	ctx := t.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	hideConsole(cmd)
	cmd.Dir = t.Workspace
	out, err := cmd.CombinedOutput()
	text := string(out)
	if err != nil {
		if gitNotRepo(text, err) {
			return ToolResult{Content: "not a git repository"}
		}
		if text == "" {
			text = err.Error()
		}
		return ToolResult{Content: text, Err: err}
	}
	if strings.TrimSpace(text) == "" {
		return ToolResult{Content: "ok"}
	}
	return ToolResult{Content: text}
}

// gitNotRepo 在合并后的输出或错误文本里查找 not a git repository。
// text 先放进待查文本。
// err 非 nil 时把 Error() 接在 text 后面，再查找 not a git repository。
func gitNotRepo(text string, err error) bool {
	blob := text
	if err != nil {
		blob += " " + err.Error()
	}
	return strings.Contains(blob, "not a git repository")
}
