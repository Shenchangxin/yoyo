package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
)

type ToolResult struct {
	Content string
	Err     error
}

// ExtraTool is a host-bridged tool (MCP, WASM) sharing the same capability gate.
type ExtraTool struct {
	JSON     ToolJSON
	Call     func(argsJSON string) ToolResult
	ReadOnly bool
}

type WorkspaceTools struct {
	Workspace string
	SessionID string
	Caps      *capability.Broker
	Skills    map[string]string
	Loaded    []string
	Policy    artifact.PolicyPack
	PlanMode  bool
	Ctx       context.Context
	Extra     map[string]ExtraTool
	Spill     *Spill
	Depth     int
	Task      TaskFunc
	PlanText  string
}

func BuiltinToolJSON() []ToolJSON {
	return []ToolJSON{
		fn("read_file", "Read a UTF-8 file under the workspace. Optional offset/limit are 1-based line numbers.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string"},
				"offset": map[string]any{"type": "integer"},
				"limit":  map[string]any{"type": "integer"},
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
		fn("str_replace", "Replace old_str with new_str. Fails if old_str is not unique unless replace_all is true.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":        map[string]any{"type": "string"},
				"old_str":     map[string]any{"type": "string"},
				"new_str":     map[string]any{"type": "string"},
				"replace_all": map[string]any{"type": "boolean"},
			},
			"required": []string{"path", "old_str", "new_str"},
		}),
		fn("list_dir", "List a directory under the workspace.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		}),
		fn("glob", "Find files by glob pattern relative to the workspace (e.g. **/*.go).", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string"},
			},
			"required": []string{"pattern"},
		}),
		fn("grep", "Search file contents with a regex. Optional glob limits which files are scanned.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string"},
				"glob":    map[string]any{"type": "string"},
				"path":    map[string]any{"type": "string"},
			},
			"required": []string{"pattern"},
		}),
		fn("shell", "Run a shell command in the workspace directory. Prefer glob/grep/read_file when possible.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command":     map[string]any{"type": "string"},
				"timeout_sec": map[string]any{"type": "integer"},
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
		fn("apply_patch", "Apply a Codex-style patch (*** Begin Patch / *** Add File or *** Update File / *** End Patch).", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"patch": map[string]any{"type": "string"},
			},
			"required": []string{"patch"},
		}),
		fn("git_status", "Show git status of the workspace.", map[string]any{"type": "object", "properties": map[string]any{}}),
		fn("git_diff", "Show git diff. Optional staged=true for --cached.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"staged": map[string]any{"type": "boolean"},
			},
		}),
		fn("git_commit", "Stage all and commit with a message. Prefer this over shell git.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
			},
			"required": []string{"message"},
		}),
		fn("recall_context", "Retrieve a previously elided tool result by id from the context spill store.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "string"},
			},
			"required": []string{"id"},
		}),
		fn("tool_search", "Look up extra (MCP/WASM) tool schemas by substring. Use when extra tools were deferred.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
			"required": []string{"query"},
		}),
		fn("task", "Run a subagent on a prompt. Returns a summary only (child transcript is isolated). Set isolate=true to copy/worktree the workspace.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt":  map[string]any{"type": "string"},
				"isolate": map[string]any{"type": "boolean"},
			},
			"required": []string{"prompt"},
		}),
		fn("update_plan", "Replace the current task plan with a list of steps and statuses (pending, in_progress, complete).", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"explanation": map[string]any{"type": "string"},
				"plan": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"step":   map[string]any{"type": "string"},
							"status": map[string]any{"type": "string"},
						},
						"required": []string{"step", "status"},
					},
				},
			},
			"required": []string{"plan"},
		}),
		fn("wait", "Pause up to 30 seconds before the next action. Use when polling a command or a file.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"seconds": map[string]any{"type": "integer"},
			},
		}),
		fn("list_skills", "List skill names available via load_skill.", map[string]any{"type": "object", "properties": map[string]any{}}),
		fn("view_image", "Inspect an image file under the workspace (size and type). Prefer this over dumping binary via read_file.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		}),
		fn("web_fetch", "HTTP GET a public https URL and return extracted text. Never used for localhost or private IPs. Requires network approval.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{"type": "string"},
			},
			"required": []string{"url"},
		}),
	}
}

func AllToolJSON(t *WorkspaceTools) []ToolJSON {
	out := BuiltinToolJSON()
	if t != nil && t.Depth > 0 {
		var filtered []ToolJSON
		for _, j := range out {
			name, _ := j.Function["name"].(string)
			if name == "task" {
				continue
			}
			filtered = append(filtered, j)
		}
		out = filtered
	}
	if t == nil {
		return out
	}
	extras := make([]ExtraTool, 0, len(t.Extra))
	for _, extra := range t.Extra {
		extras = append(extras, extra)
	}
	const schemaCap = 8
	if len(extras) <= schemaCap {
		for _, extra := range extras {
			out = append(out, extra.JSON)
		}
		return out
	}
	for _, extra := range extras {
		j := extra.JSON
		if fn, ok := j.Function["description"].(string); ok && len(fn) > 120 {
			cp := map[string]any{}
			for k, v := range j.Function {
				cp[k] = v
			}
			cp["description"] = fn[:117] + "…"
			j.Function = cp
		}
		out = append(out, j)
	}
	return out
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
	if t.PlanMode && !readonlyCall(t, name) {
		return ToolResult{Err: fmt.Errorf("plan mode: write/shell tools are disabled; produce a plan instead")}
	}
	switch name {
	case "read_file":
		return t.readFile(str(args["path"]), intArg(args["offset"]), intArg(args["limit"]))
	case "write_file":
		return t.writeFile(str(args["path"]), str(args["content"]))
	case "str_replace":
		return t.replace(str(args["path"]), str(args["old_str"]), str(args["new_str"]), boolArg(args["replace_all"]))
	case "list_dir":
		p := str(args["path"])
		if p == "" {
			p = "."
		}
		return t.listDir(p)
	case "glob":
		return t.glob(str(args["pattern"]))
	case "grep":
		return t.grep(str(args["pattern"]), str(args["glob"]), str(args["path"]))
	case "shell":
		return t.shell(str(args["command"]), intArg(args["timeout_sec"]))
	case "load_skill":
		return t.loadSkill(str(args["name"]))
	case "apply_patch":
		return t.applyPatch(str(args["patch"]))
	case "git_status":
		return t.git([]string{"status", "--short", "--branch"}, false)
	case "git_diff":
		if boolArg(args["staged"]) {
			return t.git([]string{"diff", "--cached"}, false)
		}
		return t.git([]string{"diff"}, false)
	case "git_commit":
		msg := str(args["message"])
		if msg == "" {
			return ToolResult{Err: fmt.Errorf("empty commit message")}
		}
		if res := t.git([]string{"add", "-A"}, true); res.Err != nil {
			return res
		}
		return t.git([]string{"commit", "-m", msg}, true)
	case "recall_context":
		return t.recall(str(args["id"]))
	case "tool_search":
		return t.toolSearch(str(args["query"]))
	case "task":
		return t.task(str(args["prompt"]), boolArg(args["isolate"]))
	case "update_plan":
		return t.updatePlan(argsJSON)
	case "wait":
		return t.wait(intArg(args["seconds"]))
	case "list_skills":
		return t.listSkills()
	case "view_image":
		return t.viewImage(str(args["path"]))
	case "web_fetch":
		return t.webFetch(str(args["url"]))
	default:
		if t.Extra != nil {
			if extra, ok := t.Extra[name]; ok {
				return extra.Call(argsJSON)
			}
		}
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
	req := capability.Request{
		Level:     level,
		Action:    action,
		Path:      path,
		Command:   cmd,
		SessionID: t.SessionID,
		Workspace: t.Workspace,
		ForceAsk:  policyRequires(t.Policy, level),
	}
	if t.Ctx != nil {
		return t.Caps.CheckCtx(t.Ctx, req)
	}
	return t.Caps.Check(req)
}

func policyRequires(p artifact.PolicyPack, level capability.Level) bool {
	for _, s := range p.RequireApproval {
		if s == string(level) {
			return true
		}
	}
	return p.Mode == "ask" && (level == capability.Shell || level == capability.Network || level == capability.HighRisk)
}

func (t *WorkspaceTools) readFile(rel string, offset, limit int) ToolResult {
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
	if offset > 0 || limit > 0 {
		return ToolResult{Content: numberLines(string(b), offset, limit)}
	}
	if len(b) > maxInlineBytes {
		spill, err := t.spillOverflow(rel, b)
		if err != nil {
			return ToolResult{Err: err}
		}
		head := numberLines(string(b), 1, 80)
		return ToolResult{Content: fmt.Sprintf("%s\n[truncated %d bytes; full file at %s — re-read with offset/limit]\n", head, len(b), spill)}
	}
	return ToolResult{Content: numberLines(string(b), 0, 0)}
}

func numberLines(text string, offset, limit int) string {
	lines := strings.Split(text, "\n")
	start := 0
	if offset > 0 {
		start = offset - 1
		if start > len(lines) {
			start = len(lines)
		}
	}
	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&b, "%6d|%s\n", i+1, lines[i])
	}
	return b.String()
}

func (t *WorkspaceTools) spillOverflow(rel string, b []byte) (string, error) {
	dir := filepath.Join(t.Workspace, ".yoyo", "overflow")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := filepath.Base(rel) + ".txt"
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(".yoyo", "overflow", name)), nil
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

func (t *WorkspaceTools) replace(rel, old, new string, all bool) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "str_replace", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ToolResult{Err: err}
	}
	text := string(b)
	n := strings.Count(text, old)
	if n == 0 {
		return ToolResult{Err: fmt.Errorf("old_str not found")}
	}
	if n > 1 && !all {
		return ToolResult{Err: fmt.Errorf("old_str matched %d times; pass replace_all=true or include more context", n)}
	}
	var next string
	if all {
		next = strings.ReplaceAll(text, old, new)
	} else {
		next = strings.Replace(text, old, new, 1)
	}
	if err := os.WriteFile(p, []byte(next), 0o644); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: fmt.Sprintf("replaced %d occurrence(s) in %s", n, rel)}
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

func (t *WorkspaceTools) glob(pattern string) ToolResult {
	if pattern == "" {
		return ToolResult{Err: fmt.Errorf("empty pattern")}
	}
	if err := t.check(capability.ReadWorkspace, "glob", t.Workspace, ""); err != nil {
		return ToolResult{Err: err}
	}
	matches, err := globWalk(t.Workspace, pattern, 200)
	if err != nil {
		return ToolResult{Err: err}
	}
	if len(matches) == 0 {
		return ToolResult{Content: "no matches"}
	}
	return ToolResult{Content: strings.Join(matches, "\n")}
}

func (t *WorkspaceTools) grep(pattern, globPat, rel string) ToolResult {
	if pattern == "" {
		return ToolResult{Err: fmt.Errorf("empty pattern")}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ToolResult{Err: err}
	}
	root := t.Workspace
	if rel != "" {
		p, err := t.resolve(rel)
		if err != nil {
			return ToolResult{Err: err}
		}
		root = p
	}
	if err := t.check(capability.ReadWorkspace, "grep", root, ""); err != nil {
		return ToolResult{Err: err}
	}
	hits, err := grepWalk(root, t.Workspace, re, globPat, 80)
	if err != nil {
		return ToolResult{Err: err}
	}
	if hits == "" {
		return ToolResult{Content: "no matches"}
	}
	return ToolResult{Content: hits}
}

func (t *WorkspaceTools) shell(command string, timeoutSec int) ToolResult {
	if err := ShellDenied(command, t.Workspace, t.Policy.NetworkAllow); err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.Shell, "shell", t.Workspace, command); err != nil {
		return ToolResult{Err: err}
	}
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	ctx := t.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-lc", command)
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
	for _, n := range t.Loaded {
		if n == name {
			return ToolResult{Content: body}
		}
	}
	t.Loaded = append(t.Loaded, name)
	return ToolResult{Content: body}
}

func (t *WorkspaceTools) recall(id string) ToolResult {
	if t.Spill == nil {
		return ToolResult{Err: fmt.Errorf("no spill store")}
	}
	s, err := t.Spill.Get(id)
	if err != nil {
		return ToolResult{Err: fmt.Errorf("unknown context id %s", id)}
	}
	capped, _ := capText(s, defaultToolResultRunes)
	return ToolResult{Content: capped}
}

func (t *WorkspaceTools) toolSearch(q string) ToolResult {
	if t.Extra == nil {
		return ToolResult{Content: "no extra tools"}
	}
	q = strings.ToLower(q)
	var b strings.Builder
	for name, extra := range t.Extra {
		blob := strings.ToLower(name)
		if d, _ := extra.JSON.Function["description"].(string); d != "" {
			blob += " " + strings.ToLower(d)
		}
		if q != "" && !strings.Contains(blob, q) {
			continue
		}
		raw, _ := json.Marshal(extra.JSON)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if b.Len() == 0 {
		return ToolResult{Content: "no matches"}
	}
	return ToolResult{Content: b.String()}
}

func (t *WorkspaceTools) loadedBodies() []string {
	if t == nil {
		return nil
	}
	var out []string
	for _, name := range t.Loaded {
		if t.Skills == nil {
			continue
		}
		if body, ok := t.Skills[name]; ok {
			out = append(out, "## Skill: "+name+"\n"+body)
		}
	}
	return out
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func intArg(v any) int {
	switch t := v.(type) {
	case nil:
		return 0
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		n, _ := strconv.Atoi(fmt.Sprint(v))
		return n
	}
}

func boolArg(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}
