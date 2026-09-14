package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

type ToolResult struct {
	Content string
	Err     error
}

type WorkspaceTools struct {
	Workspace string
	SessionID string
	Caps      *capability.Broker
	Skills    map[string]string
}

func BuiltinToolJSON() []ToolJSON {
	return []ToolJSON{
		fn("read_file", "Read a UTF-8 file under the workspace.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		}),
		fn("write_file", "Write a UTF-8 file under the workspace.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
			},
			"required": []string{"path", "content"},
		}),
		fn("str_replace", "Replace the first occurrence of old_str with new_str in a file.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"old_str": map[string]any{"type": "string"},
				"new_str": map[string]any{"type": "string"},
			},
			"required": []string{"path", "old_str", "new_str"},
		}),
		fn("list_dir", "List a directory under the workspace.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		}),
		fn("shell", "Run a shell command in the workspace directory.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
			},
			"required": []string{"command"},
		}),
		fn("load_skill", "Load a skill body by name into context.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
			"required": []string{"name"},
		}),
	}
}

func fn(name, desc string, params map[string]any) ToolJSON {
	return ToolJSON{
		Type: "function",
		Function: map[string]any{
			"name":        name,
			"description": desc,
			"parameters":  params,
		},
	}
}

func (t *WorkspaceTools) Call(name, argsJSON string) ToolResult {
	var args map[string]any
	if argsJSON != "" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}
	if args == nil {
		args = map[string]any{}
	}
	switch name {
	case "read_file":
		return t.readFile(str(args["path"]))
	case "write_file":
		return t.writeFile(str(args["path"]), str(args["content"]))
	case "str_replace":
		return t.replace(str(args["path"]), str(args["old_str"]), str(args["new_str"]))
	case "list_dir":
		p := str(args["path"])
		if p == "" {
			p = "."
		}
		return t.listDir(p)
	case "shell":
		return t.shell(str(args["command"]))
	case "load_skill":
		return t.loadSkill(str(args["name"]))
	default:
		return ToolResult{Err: fmt.Errorf("unknown tool %s", name)}
	}
}

func (t *WorkspaceTools) resolve(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	p := rel
	if !filepath.IsAbs(p) {
		p = filepath.Join(t.Workspace, rel)
	}
	p = filepath.Clean(p)
	if !capability.WithinWorkspace(t.Workspace, p) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return p, nil
}

func (t *WorkspaceTools) check(level capability.Level, action, path, cmd string) error {
	if t.Caps == nil {
		return nil
	}
	return t.Caps.Check(capability.Request{
		Level:     level,
		Action:    action,
		Path:      path,
		Command:   cmd,
		SessionID: t.SessionID,
		Workspace: t.Workspace,
	})
}

func (t *WorkspaceTools) readFile(rel string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.ReadWorkspace, "read_file", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: string(b)}
}

func (t *WorkspaceTools) writeFile(rel, content string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "write_file", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return ToolResult{Err: err}
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "wrote " + rel}
}

func (t *WorkspaceTools) replace(rel, old, new string) ToolResult {
	res := t.readFile(rel)
	if res.Err != nil {
		return res
	}
	if !strings.Contains(res.Content, old) {
		return ToolResult{Err: fmt.Errorf("old_str not found")}
	}
	return t.writeFile(rel, strings.Replace(res.Content, old, new, 1))
}

func (t *WorkspaceTools) listDir(rel string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.ReadWorkspace, "list_dir", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	ents, err := os.ReadDir(p)
	if err != nil {
		return ToolResult{Err: err}
	}
	var b strings.Builder
	for _, e := range ents {
		if e.IsDir() {
			b.WriteString("d ")
		} else {
			b.WriteString("f ")
		}
		b.WriteString(e.Name())
		b.WriteByte('\n')
	}
	return ToolResult{Content: b.String()}
}

func (t *WorkspaceTools) shell(command string) ToolResult {
	if err := t.check(capability.Shell, "shell", t.Workspace, command); err != nil {
		return ToolResult{Err: err}
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-lc", command)
	}
	cmd.Dir = t.Workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ToolResult{Content: string(out), Err: err}
	}
	return ToolResult{Content: string(out)}
}

func (t *WorkspaceTools) loadSkill(name string) ToolResult {
	if t.Skills == nil {
		return ToolResult{Err: fmt.Errorf("no skills")}
	}
	body, ok := t.Skills[name]
	if !ok {
		return ToolResult{Err: fmt.Errorf("unknown skill %s", name)}
	}
	return ToolResult{Content: body}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
