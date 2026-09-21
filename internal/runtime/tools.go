package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/browser"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/computeruse"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/isolation"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/schedule"
	"github.com/Shenchangxin/yoyo/internal/tool"
)

type ToolResult struct {
	Content    string
	Err        error
	FileChange *FileChange
	Parts      []ContentPart
}

type FileChange struct {
	Paths []string
	Patch string
}

// prefetchHit is a stream-time readonly execution that must not be traced
// until the final dispatchTools path emits once.
type prefetchHit struct {
	content   string
	elapsedMs int
	spillID   string
	ok        bool
	change    *FileChange
	parts     []ContentPart
}

// ExtraTool is a host-bridged tool (MCP, WASM) sharing the same capability gate.
type ExtraTool struct {
	JSON        ToolJSON
	Call        func(argsJSON string) ToolResult
	ReadOnly    bool
	OpenWorld   bool
	Exclusive   bool
	Destructive bool
}

type WorkspaceTools struct {
	Workspace    string
	Home         string
	SessionID    string
	Caps         *capability.Broker
	Skills       map[string]string
	SkillDirs    map[string]string
	Loaded       []string
	ExtraEnabled []string
	ExtraLive    []string
	pendingHost  []string
	taskProfile  string
	Policy       artifact.PolicyPack
	PlanMode     bool
	// ChatOverlay is desktop/CLI personal chat (I6). Harbor eval leaves it
	// false so shell, plan language, and the tool menu stay eval-stable.
	ChatOverlay bool
	// OperatorVoice is the latest real operator utterance for chat overlays.
	OperatorVoice string
	// VerifyHint is set when chat tries to end_turn with an open plan so
	// verify-artifact is re-injected into working memory (I2/I5).
	VerifyHint   bool
	Ctx          context.Context
	Extra        map[string]ExtraTool
	Spill        *Spill
	Depth        int
	Task         TaskFunc
	PlanText     string
	Advertised   []string
	AllowedTools []string
	AskUser      func(question string) (string, error)
	SkillMeta    map[string]artifact.Skill
	prefetch     map[string]prefetchHit
	mu           sync.Mutex
	MaxParallel  int
	SearchAPI    func(query string) (string, error)
	Sessions     func(id string) []Message
	Memory       *memory.Store
	Schedule     *schedule.Service
	Projects     *project.Store
	Connectors   *connector.Broker
	Browser      *browser.Host
	Computer     *computeruse.Host
	Inbox        *inbox.Store
	LastBrowser  string
}

func BuiltinToolJSON() []ToolJSON {
	specs := tool.HostSpecs()
	out := make([]ToolJSON, 0, len(specs))
	for _, s := range specs {
		out = append(out, specJSON(s))
	}
	return out
}

func AllToolJSON(t *WorkspaceTools) []ToolJSON {
	out := BuiltinToolJSON()
	if t != nil && t.Depth >= MaxTaskDepth {
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
	advertised := t.advertisedCopy()
	if len(advertised) > 0 || len(t.AllowedTools) > 0 {
		allow := map[string]bool{}
		for _, n := range t.AllowedTools {
			for _, part := range strings.Fields(n) {
				allow[part] = true
			}
		}
		want := map[string]bool{}
		for _, n := range advertised {
			want[n] = true
		}
		var filtered []ToolJSON
		for _, j := range out {
			name, _ := j.Function["name"].(string)
			if len(want) > 0 && !want[name] && !alwaysAdvertise(name) {
				continue
			}
			if len(allow) > 0 && !allow[name] && !alwaysAdvertise(name) {
				continue
			}
			filtered = append(filtered, j)
		}
		out = filtered
	}
	names := make([]string, 0, len(t.Extra))
	for name := range t.Extra {
		names = append(names, name)
	}
	sort.Strings(names)
	const schemaCap = 8
	enabled := map[string]bool{}
	t.mu.Lock()
	for _, n := range t.ExtraEnabled {
		enabled[n] = true
	}
	t.mu.Unlock()
	var deferred []string
	freezeExtra := t.ChatOverlay
	for _, name := range names {
		if freezeExtra || len(names) > schemaCap {
			if enabled[name] {
				out = append(out, t.Extra[name].JSON)
				continue
			}
			deferred = append(deferred, name)
			continue
		}
		out = append(out, t.Extra[name].JSON)
	}
	if t.Spill != nil {
		for _, name := range deferred {
			extra, ok := t.Extra[name]
			if !ok {
				continue
			}
			raw, _ := json.Marshal(extra.JSON)
			t.Spill.Put("mcp-"+sanitizeID(name), string(raw))
		}
	}
	if t.ChatOverlay {
		deferred = append(deferred, deferredHostTools(advertised)...)
	}
	if len(deferred) > 0 {
		for i := range out {
			name, _ := out[i].Function["name"].(string)
			if name != "tool_search" {
				continue
			}
			desc, _ := out[i].Function["description"].(string)
			cp := map[string]any{}
			for k, v := range out[i].Function {
				cp[k] = v
			}
			cp["description"] = desc + " Deferred extra tools: " + strings.Join(deferred, ", ") + "."
			out[i].Function = cp
			break
		}
	}
	WriteMCPCatalog(t.Workspace, t.Extra)
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
	if t != nil && !t.toolAllowed(name) {
		return ToolResult{Err: fmt.Errorf("tool %s is not advertised for this harness/skill", name)}
	}
	if t != nil && t.PlanMode && !readonlyCall(t, name) {
		return ToolResult{Err: fmt.Errorf("plan mode: write/shell tools are disabled; produce a plan instead")}
	}
	if fn := hostFns[name]; fn != nil {
		return fn(t, args, argsJSON)
	}
	if t != nil && t.Extra != nil {
		if extra, ok := t.Extra[name]; ok {
			return extra.Call(argsJSON)
		}
	}
	return ToolResult{Err: fmt.Errorf("unknown tool %s", name)}
}

func (t *WorkspaceTools) toolAllowed(name string) bool {
	if t == nil {
		return true
	}
	if t.Extra != nil {
		if _, ok := t.Extra[name]; ok {
			return true
		}
	}
	if alwaysAdvertise(name) {
		return true
	}
	if len(t.AllowedTools) > 0 {
		for _, n := range t.AllowedTools {
			for _, part := range strings.Fields(n) {
				if part == name {
					return true
				}
			}
		}
		return false
	}
	if t.ChatOverlay {
		return hostToolKnown(name)
	}
	advertised := t.advertisedCopy()
	if len(advertised) > 0 {
		for _, n := range advertised {
			if n == name {
				return true
			}
		}
		return false
	}
	return true
}

func (t *WorkspaceTools) advertisedCopy() []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.Advertised...)
}

func alwaysAdvertise(name string) bool {
	switch name {
	case "load_skill", "list_skills", "tool_search", "recall_context", "update_plan", "ask_user", "read_thread":
		return true
	default:
		return false
	}
}

func (t *WorkspaceTools) CommitToolUnlocks() {
	if t == nil {
		return
	}
	t.mu.Lock()
	have := map[string]bool{}
	for _, n := range t.ExtraEnabled {
		have[n] = true
	}
	for _, n := range t.ExtraLive {
		if n == "" || have[n] {
			continue
		}
		t.ExtraEnabled = append(t.ExtraEnabled, n)
		have[n] = true
	}
	t.ExtraLive = nil
	host := append([]string(nil), t.pendingHost...)
	t.pendingHost = nil
	t.mu.Unlock()
	t.unlockAdvertised(host)
}

func (t *WorkspaceTools) resolve(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	p := capability.CanonicalizeToolPath(rel)
	if !filepath.IsAbs(p) {
		p = filepath.Join(t.Workspace, p)
	}
	p = filepath.Clean(p)
	if !capability.WithinWorkspace(t.Workspace, p) {
		return "", fmt.Errorf("path escapes workspace")
	}
	if capability.ForbiddenDiagPath(t.Home, p) {
		return "", fmt.Errorf("path denied: diagnostic plane")
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
	if p.Mode == "bypass" {
		return false
	}
	if p.Mode == "ask" && (level == capability.Shell || level == capability.Network || level == capability.HighRisk || level == capability.SendAsYou || level == capability.ComputerUse) {
		return true
	}
	// RequireApproval means the first call must enter the gate. It is not
	// ForceAsk: Session/Once/Always grants on the broker must stick.
	return false
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
	want := []byte(content)
	if prev, err := os.ReadFile(p); err == nil && bytes.Equal(prev, want) {
		return ToolResult{Content: "unchanged " + rel}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return ToolResult{Err: err}
	}
	if err := os.WriteFile(p, want, 0o644); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "wrote " + rel, FileChange: &FileChange{Paths: []string{rel}}}
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
		return ToolResult{Err: fmt.Errorf("old_str not found in %s\n%s", rel, fileContextHint(text, old))}
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
	if next == text {
		return ToolResult{Content: "unchanged " + rel}
	}
	if err := os.WriteFile(p, []byte(next), 0o644); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: fmt.Sprintf("replaced %d occurrence(s) in %s", n, rel), FileChange: &FileChange{Paths: []string{rel}}}
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
	n := 0
	for _, e := range ents {
		if n >= 200 {
			b.WriteString("…[truncated; use glob for more]\n")
			break
		}
		if e.IsDir() {
			b.WriteString("d ")
		} else {
			b.WriteString("f ")
		}
		b.WriteString(e.Name())
		b.WriteByte('\n')
		n++
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
	command = t.expandSkillScripts(command)
	if t != nil && t.ChatOverlay {
		if rel, offset, limit, ok := parseShellRead(command); ok {
			res := t.readFile(rel, offset, limit)
			if res.Err != nil {
				res.Content = "read source with read_file/grep/glob, not shell cat/sed. path=" + rel
				return res
			}
			prefix := "redirected shell read to read_file (do not cat source) path=" + rel + "\n"
			res.Content = prefix + res.Content
			return res
		}
	}
	argv := SplitShellArgv(command)
	extra := t.skillDirRoots()
	if err := ShellDenied(command, t.Workspace, t.Policy.NetworkAllow, extra...); err != nil {
		return ToolResult{Err: err}
	}
	if err := DenyArgvPaths(argv, t.Workspace, extra...); err != nil {
		return ToolResult{Err: err}
	}
	if err := DenyHomePlanes(argv, t.Home); err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.Shell, "shell", t.Workspace, command); err != nil {
		return ToolResult{Err: err}
	}
	posix := looksPosixUnix(command)
	rewriteNote := ""
	if runtime.GOOS == "windows" && cmdStartWouldHang(command) {
		if next, ok := rewriteCmdStart(command); ok {
			rewriteNote = "rewrote cmd start to Start-Process (captured stdout would hang)\n"
			command = next
			argv = SplitShellArgv(command)
		} else {
			return ToolResult{Err: cmdStartHangError()}
		}
	}
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	parent := t.Ctx
	if parent == nil {
		parent = context.Background()
	}
	timeout := time.Duration(timeoutSec) * time.Second
	idle, block, fuse := chatShellFuse(t != nil && t.ChatOverlay, timeout)
	var ctx context.Context
	var cancel context.CancelFunc
	if fuse {
		ctx = parent
	} else {
		ctx, cancel = context.WithTimeout(parent, timeout)
		defer cancel()
	}
	bindCtx := ctx
	if fuse {
		bindCtx = nil
	}
	cmd, err := bindShell(bindCtx, command, argv, posix)
	if err != nil {
		return ToolResult{Err: err}
	}
	cmd.Dir = t.Workspace
	out := runShell(ctx, cmd, idle, block)
	if out.Err != nil && runtime.GOOS == "windows" && !posix && !looksPosixUnix(command) && !shellNeedsWrapper(command, argv) && isExecNotFound(out.Err) {
		if fuse {
			cmd = exec.Command("cmd", "/C", command)
		} else {
			cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		}
		cmd.Dir = t.Workspace
		out = runShell(ctx, cmd, idle, block)
	}
	content := rewriteNote + string(out.Output)
	if out.Background {
		content = fmt.Sprintf("still running pid=%d; not killed (idle/block fuse)\n%s", out.PID, content)
		return ToolResult{Content: content}
	}
	if out.Err != nil {
		return ToolResult{Content: content, Err: out.Err}
	}
	return ToolResult{Content: content}
}

const (
	chatShellBlock = 30 * time.Second
	chatShellIdle  = 10 * time.Second
)

func chatShellFuse(overlay bool, timeout time.Duration) (idle, block time.Duration, fuse bool) {
	if !overlay || timeout <= chatShellBlock {
		return 0, 0, false
	}
	return chatShellIdle, chatShellBlock, true
}

func runShell(ctx context.Context, cmd *exec.Cmd, idle, block time.Duration) isolation.Outcome {
	if idle > 0 || block > 0 {
		return isolation.Wait(ctx, cmd, idle, block)
	}
	out, err := isolation.Run(ctx, cmd)
	return isolation.Outcome{Output: out, Err: err}
}

func bindShell(ctx context.Context, command string, argv []string, posix bool) (*exec.Cmd, error) {
	newCmd := func(name string, args ...string) *exec.Cmd {
		if ctx != nil {
			return exec.CommandContext(ctx, name, args...)
		}
		return exec.Command(name, args...)
	}
	if runtime.GOOS == "windows" {
		if posix || looksPosixUnix(command) {
			sh := windowsPosixShell()
			if sh == "" {
				return nil, fmt.Errorf("unix shell syntax on Windows and bash is not on PATH; use list_dir/glob/grep or cmd builtins")
			}
			return newCmd(sh, "-lc", "set +m; trap '' HUP; "+command), nil
		}
		if !shellNeedsWrapper(command, argv) {
			return newCmd(argv[0], argv[1:]...), nil
		}
		return newCmd("cmd", "/C", command), nil
	}
	if !shellNeedsWrapper(command, argv) {
		return newCmd(argv[0], argv[1:]...), nil
	}
	return newCmd("sh", "-lc", command), nil
}

func isExecNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "executable file not found") || strings.Contains(s, "file not found") || strings.Contains(s, "not recognized as an internal")
}

func windowsPosixShell() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	var candidates []string
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates,
			filepath.Join(pf, "Git", "bin", "bash.exe"),
			filepath.Join(pf, "Git", "usr", "bin", "bash.exe"),
		)
	}
	if pf86 := os.Getenv("ProgramFiles(x86)"); pf86 != "" {
		candidates = append(candidates, filepath.Join(pf86, "Git", "bin", "bash.exe"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		candidates = append(candidates, filepath.Join(local, "Programs", "Git", "bin", "bash.exe"))
	}
	candidates = append(candidates, `C:\msys64\usr\bin\bash.exe`)
	for _, n := range []string{"bash.exe", "bash"} {
		if p, err := exec.LookPath(n); err == nil {
			candidates = append(candidates, p)
		}
	}
	for _, p := range candidates {
		if usableWindowsPosixShell(p) {
			return p
		}
	}
	return ""
}

func usableWindowsPosixShell(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	low := strings.ToLower(filepath.Clean(p))
	if strings.Contains(low, `\system32\`) || strings.Contains(low, `/system32/`) {
		return false
	}
	if strings.Contains(low, `\windowsapps\`) || strings.Contains(low, `\windows\system32`) {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func (t *WorkspaceTools) loadSkill(name string) ToolResult {
	if t.Skills == nil {
		return ToolResult{Err: fmt.Errorf("no skills")}
	}
	body, ok := t.Skills[name]
	if !ok {
		return ToolResult{Err: fmt.Errorf("unknown skill %s", name)}
	}
	content := t.decorateSkillBody(name, body)
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, n := range t.Loaded {
		if n == name {
			return ToolResult{Content: content}
		}
	}
	t.Loaded = append(t.Loaded, name)
	if t.SkillMeta != nil {
		if sk, ok := t.SkillMeta[name]; ok && strings.TrimSpace(sk.AllowedTools) != "" {
			t.AllowedTools = strings.Fields(sk.AllowedTools)
		}
	}
	return ToolResult{Content: content}
}

func (t *WorkspaceTools) recall(id string, offset, limit int) ToolResult {
	if t.Spill == nil {
		return ToolResult{Err: fmt.Errorf("no spill store")}
	}
	s, err := t.Spill.Get(id)
	if err != nil {
		return ToolResult{Err: fmt.Errorf("unknown context id %s", id)}
	}
	return ToolResult{Content: pageSpill(id, s, offset, limit)}
}

const defaultRecallLines = 200
const defaultRecallRunes = 4_000

func pageSpill(id, s string, offset, limit int) string {
	lines := strings.Split(s, "\n")
	if offset <= 0 {
		offset = 1
	}
	if limit <= 0 {
		limit = defaultRecallLines
	}
	start := offset - 1
	if start > len(lines) {
		start = len(lines)
	}
	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}
	page := strings.Join(lines[start:end], "\n")
	capped, trunc := capText(page, defaultRecallRunes)
	var b strings.Builder
	fmt.Fprintf(&b, "recall id=%s lines=%d-%d of %d\n", id, start+1, end, len(lines))
	b.WriteString(capped)
	if trunc {
		b.WriteString("\n[page capped — pass offset/limit to page further]")
	}
	return b.String()
}

func recallNoStub(name, content string) bool {
	if name != "recall_context" {
		return false
	}
	return utf8.RuneCountInString(content) <= 2000
}

func (t *WorkspaceTools) toolSearch(q string) ToolResult {
	q = strings.ToLower(strings.TrimSpace(q))
	var b strings.Builder
	var matched []string
	if t != nil && t.Extra != nil {
		for name, extra := range t.Extra {
			blob := strings.ToLower(name)
			if d, _ := extra.JSON.Function["description"].(string); d != "" {
				blob += " " + strings.ToLower(d)
			}
			if q != "" && !strings.Contains(blob, q) {
				continue
			}
			matched = append(matched, name)
			raw, _ := json.Marshal(extra.JSON)
			b.Write(raw)
			b.WriteByte('\n')
		}
	}
	var hostHits []string
	if t != nil && t.ChatOverlay {
		want := map[string]bool{}
		for _, n := range t.advertisedCopy() {
			want[n] = true
		}
		for _, name := range searchHostSpecs(q) {
			if want[name] || alwaysAdvertise(name) {
				continue
			}
			js, ok := hostSpecJSON(name)
			if !ok {
				continue
			}
			hostHits = append(hostHits, name)
			raw, _ := json.Marshal(js)
			b.Write(raw)
			b.WriteByte('\n')
		}
	}
	if b.Len() == 0 {
		if t == nil || (t.Extra == nil && !t.ChatOverlay) {
			return ToolResult{Content: "no extra tools"}
		}
		return ToolResult{Content: "no matches"}
	}
	if t != nil && t.Extra != nil {
		t.mu.Lock()
		for _, name := range matched {
			if t.ChatOverlay {
				seen := false
				for _, n := range t.ExtraLive {
					if n == name {
						seen = true
						break
					}
				}
				if !seen {
					t.ExtraLive = append(t.ExtraLive, name)
				}
			} else {
				seen := false
				for _, n := range t.ExtraEnabled {
					if n == name {
						seen = true
						break
					}
				}
				if !seen {
					t.ExtraEnabled = append(t.ExtraEnabled, name)
				}
			}
		}
		t.mu.Unlock()
	}
	if t != nil && t.ChatOverlay && len(hostHits) > 0 {
		t.mu.Lock()
		have := map[string]bool{}
		for _, n := range t.pendingHost {
			have[n] = true
		}
		for _, n := range hostHits {
			if n == "" || have[n] {
				continue
			}
			t.pendingHost = append(t.pendingHost, n)
			have[n] = true
		}
		t.mu.Unlock()
	} else {
		t.unlockAdvertised(hostHits)
	}
	return ToolResult{Content: b.String()}
}

func (t *WorkspaceTools) loadedBodies() []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
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

func (t *WorkspaceTools) Pins() (loaded []string, plan string) {
	if t == nil {
		return nil, ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.Loaded...), t.PlanText
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func anyStrings(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			s := strings.TrimSpace(fmt.Sprint(x))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
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
