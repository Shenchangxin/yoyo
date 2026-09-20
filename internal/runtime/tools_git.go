package runtime

import (
	"context"
	"os/exec"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

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

func gitNotRepo(text string, err error) bool {
	blob := text
	if err != nil {
		blob += " " + err.Error()
	}
	return strings.Contains(blob, "not a git repository")
}
