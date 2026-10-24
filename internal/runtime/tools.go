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
	"github.com/Shenchangxin/yoyo/internal/pages"
	"github.com/Shenchangxin/yoyo/internal/personal"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/pwsh"
	"github.com/Shenchangxin/yoyo/internal/schedule"
	"github.com/Shenchangxin/yoyo/internal/tool"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type ToolResult struct {
	Content    string
	Err        error
	FileChange *FileChange
	Parts      []ContentPart
}

type FileChange struct {
	Paths   []string
	Patch   string
	Added   int
	Removed int
	Created bool
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
	Policy       artifact.PolicyPack
	PlanMode     bool
	// ReadOnly is explore/qa subagents: deny writes without the plan-mode
	// persona ("produce a plan"). Plan mode and read-only are not the same.
	ReadOnly bool
	// ChatOverlay is desktop/CLI personal chat (I6). Harbor eval leaves it
	// false so shell, plan language, and the tool menu stay eval-stable.
	ChatOverlay bool
	// OperatorVoice is the latest real operator utterance for chat overlays.
	OperatorVoice string
	// VerifyHint is set when chat tries to end_turn with an open plan so
	// verify-artifact is re-injected into working memory (I2/I5).
	VerifyHint        bool
	Ctx               context.Context
	Extra             map[string]ExtraTool
	Spill             *Spill
	Depth             int
	Task              TaskFunc
	PlanText          string
	Advertised        []string
	AllowedTools      []string
	AskUser           func(question string) (string, error)
	SkillMeta         map[string]artifact.Skill
	prefetch          map[string]prefetchHit
	mu                sync.Mutex
	MaxParallel       int
	SearchAPI         func(query string) (string, error)
	Sessions          func(id string) []Message
	ListThreads       func() []ThreadRef
	SendThread        func(id, text string) error
	Memory            *memory.Store
	Schedule          *schedule.Service
	Projects          *project.Store
	Connectors        *connector.Broker
	Personal          *personal.Engine
	Browser           *browser.Host
	Computer          *computeruse.Host
	Inbox             *inbox.Store
	Pages             *pages.Store
	OnCapture         func(kind, path, summary string)
	LastBrowser       string
	AllowedConnectors []string
	Takeover          func(question string) (string, error)
	OnLive            func(trace.Event)
	liveCall          string
	liveRound         string
	liveOut           int
	// Packs are enabled methodology/content packs for this controller turn.
	Packs []PackSession
	// Methodology yields write-now chat conduct and plan-first overlay.
	Methodology bool
	// BootstrapBodies are wrapped session-start skill texts (not the catalog body).
	BootstrapBodies map[string]string
	// SkillAutoInvoke is off | hint | on. Empty means on.
	SkillAutoInvoke string
	// SkillCatalogBudget caps catalog lines in pins (bytes). Zero uses default.
	SkillCatalogBudget int
	// SkillHints are this-turn matcher results (hint + autoload).
	SkillHints []SkillSuggestion
	// Autoloaded is the subset the host injected as implicit @skill.
	Autoloaded []string
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
	if t != nil && t.ReadOnly {
		var filtered []ToolJSON
		for _, j := range out {
			name, _ := j.Function["name"].(string)
			if !readOnlySubagentCall(name) {
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
	// Gateways reject the whole turn when a name appears twice. Host schemas
	// stay; a later extra with the same name (canvas ask_user) is dropped.
	return uniqueToolJSON(out)
}

func uniqueToolJSON(in []ToolJSON) []ToolJSON {
	if len(in) < 2 {
		return in
	}
	seen := make(map[string]bool, len(in))
	out := make([]ToolJSON, 0, len(in))
	for _, j := range in {
		name, _ := j.Function["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
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

func writeLikeTool(name string) bool {
	switch name {
	case "write_file", "create_file", "apply_patch", "str_replace", "edit_file",
		"office_create", "office_edit", "cite_sources":
		return true
	default:
		return false
	}
}

func (t *WorkspaceTools) Call(name, argsJSON string) ToolResult {
	if writeLikeTool(name) && toolArgsTruncated(argsJSON) {
		// A cut-off payload is never executed. A salvaged prefix on disk looks
		// finished to the next round and invites str_replace over text that was
		// never written. The model resends the same work in parts instead.
		return ToolResult{Err: truncatedWriteErr(argsJSON)}
	}
	args := unwrapToolArgs(parseToolArgs(argsJSON))
	if args == nil {
		args = map[string]any{}
	}
	if t != nil && !t.toolAllowed(name) {
		return ToolResult{Err: fmt.Errorf("tool %s is not advertised for this harness/skill", name)}
	}
	if t != nil && t.ReadOnly && !readOnlySubagentCall(name) {
		return ToolResult{Err: fmt.Errorf("read-only subagent: write/shell tools are disabled; return findings in the final letter")}
	}
	if t != nil && t.PlanMode && !readonlyCall(t, name) {
		return ToolResult{Err: fmt.Errorf("plan mode: write/shell tools are disabled; produce a plan instead")}
	}
	rawHint := str(args["url"]) + " " + str(args["app"]) + " " + str(args["op"]) + " " + str(args["detail"])
	if t != nil {
		if err := t.ladderBlock(name, strings.TrimSpace(rawHint+" "+argsJSON)); err != nil {
			return ToolResult{Err: err}
		}
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
	case "load_skill", "list_skills", "tool_search", "recall_context", "update_plan", "ask_user", "read_thread", "read_skill_file":
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
		return ToolResult{Err: missingReadErr(rel, p, err)}
	}
	if offset > 0 || limit > 0 {
		return ToolResult{Content: rangedRead(rel, string(b), offset, limit)}
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

func missingReadErr(rel, abs string, err error) error {
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	rel = filepath.ToSlash(rel)
	msg := fmt.Sprintf("read_file %s: path does not exist", rel)
	slash := strings.ToLower(rel)
	if strings.HasSuffix(slash, "qa/verification.json") || strings.HasSuffix(slash, "qa/evidence/run.json") {
		msg += ". browser_verify writes those under the game project directory inferred from the HTML path — pass root as that directory, not a CSS selector"
	}
	parent := filepath.Dir(abs)
	ents, e := os.ReadDir(parent)
	if e != nil {
		return fmt.Errorf("%s", msg)
	}
	names := make([]string, 0, 8)
	for i, ent := range ents {
		if i >= 8 {
			names = append(names, "…")
			break
		}
		n := ent.Name()
		if ent.IsDir() {
			n += "/"
		}
		names = append(names, n)
	}
	if len(names) > 0 {
		msg += ". nearby: " + strings.Join(names, ", ")
	}
	return fmt.Errorf("%s", msg)
}

// rangedRead keeps an empty window from reading as a silent success. A model
// that cannot see the file length re-reads past EOF; session 3b9f1bbb660e4236
// issued six such reads (offsets 270-296 of a 264-line file) and each came
// back as "completed with no output", so the turn spun. Name the real extent.
func rangedRead(rel, text string, offset, limit int) string {
	total := 0
	if text != "" {
		total = len(strings.Split(strings.TrimSuffix(text, "\n"), "\n"))
	}
	if total == 0 {
		return fmt.Sprintf("%s is empty (0 lines).", rel)
	}
	if offset > total {
		return fmt.Sprintf("%s has %d lines; offset %d is past the end. Read offset 1-%d, or omit offset to start from the top.", rel, total, offset, total)
	}
	return numberLines(text, offset, limit)
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

func refuseHugeWrite(rel, content string) error {
	if len(content) <= writeFilePartMax {
		return nil
	}
	return fmt.Errorf("refusing write_file %s: content is %d bytes (max %d per call). Split the body: first part mode=create or overwrite, each following part mode=append, at most %d bytes per call", rel, len(content), writeFilePartMax, writeFilePartMax)
}

func (t *WorkspaceTools) writeFile(rel, content string) ToolResult {
	if looksLikeContextStub(content) {
		return t.resolveElidedWrite(rel, "content", "")
	}
	if err := refuseHugeWrite(rel, content); err != nil {
		return ToolResult{Err: err}
	}
	if err := sealedGameWorldWrite(rel); err != nil {
		return ToolResult{Err: err}
	}
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "write_file", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	want := []byte(content)
	prev, err := os.ReadFile(p)
	created := err != nil
	if err == nil && bytes.Equal(prev, want) {
		return ToolResult{Content: "unchanged " + rel}
	}
	if created {
		prev = nil
	}
	t.snapshotBeforeWrite(rel, p)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return ToolResult{Err: err}
	}
	if err := os.WriteFile(p, want, 0o644); err != nil {
		return ToolResult{Err: err}
	}
	t.maybePreviewHTML(rel, p)
	out := "wrote " + rel
	if w := htmlOverwriteWarning(rel, string(want)); w != "" {
		out += "\n" + w
	}
	return ToolResult{Content: out, FileChange: fileChangeOf(rel, prev, want, created)}
}

// createFile writes only when the path is new. It is the explicit form of
// "start a long file" and never clobbers existing work.
func (t *WorkspaceTools) createFile(rel, content string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if _, err := os.Stat(p); err == nil {
		return ToolResult{Err: fmt.Errorf("refusing write_file mode=create: %s already exists; use mode=overwrite to replace it or mode=append to extend it", rel)}
	}
	return t.writeFile(rel, content)
}

// appendFile extends a file in parts. Long bodies go through here so no single
// tool call has to carry the whole file inside one model output budget.
func (t *WorkspaceTools) appendFile(rel, content string) ToolResult {
	if looksLikeContextStub(content) {
		return ToolResult{Err: fmt.Errorf("refusing to append a context stub to %s; send the real next part", rel)}
	}
	if err := refuseHugeWrite(rel, content); err != nil {
		return ToolResult{Err: err}
	}
	if err := sealedGameWorldWrite(rel); err != nil {
		return ToolResult{Err: err}
	}
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "write_file", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	prev, err := os.ReadFile(p)
	created := false
	if err != nil {
		if !os.IsNotExist(err) {
			return ToolResult{Err: err}
		}
		created = true
		prev = nil
	}
	want := make([]byte, 0, len(prev)+len(content))
	want = append(want, prev...)
	want = append(want, content...)
	if msg := htmlAppendRefusal(rel, string(want)); msg != "" {
		return ToolResult{Err: fmt.Errorf("%s", msg)}
	}
	t.snapshotBeforeWrite(rel, p)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return ToolResult{Err: err}
	}
	if err := os.WriteFile(p, want, 0o644); err != nil {
		return ToolResult{Err: err}
	}
	t.maybePreviewHTML(rel, p)
	return ToolResult{
		Content:    fmt.Sprintf("appended %d bytes to %s (file is now %d bytes)", len(content), rel, len(want)),
		FileChange: fileChangeOf(rel, prev, want, created),
	}
}

func (t *WorkspaceTools) resolveElidedWrite(rel, field, recall string) ToolResult {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return ToolResult{Err: fmt.Errorf("refusing to write a context stub; missing path")}
	}
	if p, err := t.resolve(rel); err == nil {
		if st, e := os.Stat(p); e == nil && !st.IsDir() && st.Size() > 0 {
			return ToolResult{Content: fmt.Sprintf("unchanged %s (%d bytes already on disk). The elided placeholder is not file content — use str_replace or apply_patch.", rel, st.Size())}
		}
	}
	if body, ok := t.recoverElidedField(rel, field, recall); ok {
		if field == "patch" {
			return t.applyPatch(body)
		}
		return t.writeFile(rel, body)
	}
	return ToolResult{Err: fmt.Errorf("refusing to write a context stub for %s; the file is not on disk. Write the real %s — never paste _elided/omitted placeholders", rel, field)}
}

func (t *WorkspaceTools) recoverElidedField(rel, field, recall string) (string, bool) {
	if t == nil || t.Spill == nil || strings.TrimSpace(rel) == "" || strings.TrimSpace(field) == "" {
		return "", false
	}
	want := filepath.ToSlash(rel)
	if body, ok := t.spillArgsField(recall, want, field); ok {
		return body, true
	}
	for _, id := range t.Spill.ListIDs() {
		if !strings.HasSuffix(strings.ToLower(id), "args") {
			continue
		}
		if body, ok := t.spillArgsField(id, want, field); ok {
			return body, true
		}
	}
	return "", false
}

func (t *WorkspaceTools) spillArgsField(id, wantPath, field string) (string, bool) {
	id = strings.TrimSpace(id)
	if t == nil || t.Spill == nil || id == "" {
		return "", false
	}
	raw, err := t.Spill.Get(id)
	if err != nil || looksLikeContextStub(raw) {
		return "", false
	}
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return "", false
	}
	path, _ := m["path"].(string)
	if filepath.ToSlash(path) != wantPath {
		return "", false
	}
	body, _ := m[field].(string)
	if body == "" || looksLikeContextStub(body) {
		return "", false
	}
	return body, true
}

func (t *WorkspaceTools) maybePreviewHTML(rel, abs string) {
	if t == nil || t.Browser == nil || abs == "" {
		return
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".html", ".htm":
		t.Browser.SetPreview(abs)
	}
}

func (t *WorkspaceTools) replace(rel, old, new string, all bool) ToolResult {
	if looksLikeContextStub(old) || looksLikeContextStub(new) {
		return ToolResult{Err: fmt.Errorf("refusing to write a context stub; read_file that path or recall_context the spill id instead of rewriting from memory")}
	}
	if err := sealedGameWorldWrite(rel); err != nil {
		return ToolResult{Err: err}
	}
	if old == "" {
		return ToolResult{Err: fmt.Errorf("old_str must not be empty; quote the exact existing text to replace")}
	}
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
	t.snapshotBeforeWrite(rel, p)
	if err := os.WriteFile(p, []byte(next), 0o644); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{
		Content:    fmt.Sprintf("replaced %d occurrence(s) in %s", n, rel),
		FileChange: fileChangeOf(rel, []byte(text), []byte(next), false),
	}
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
			b.WriteString(e.Name())
		} else {
			b.WriteString("f ")
			b.WriteString(e.Name())
			if info, err := e.Info(); err == nil {
				fmt.Fprintf(&b, "  %d  %s", info.Size(), info.ModTime().Format("2006-01-02T15:04"))
			}
		}
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
	re, compileErr := regexp.Compile(pattern)
	literal := compileErr != nil
	if literal {
		// Pasted code fragments (`})();`, unbalanced `)`) are not regexes. A hard
		// error here looped the model on the same bad pattern; search the text.
		re = regexp.MustCompile(regexp.QuoteMeta(pattern))
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
	note := ""
	if literal {
		note = "pattern is not a valid regex; matched it as literal text.\n"
	}
	if hits == "" {
		return ToolResult{Content: note + "no matches"}
	}
	return ToolResult{Content: note + hits}
}

func (t *WorkspaceTools) shell(command string, timeoutSec int) ToolResult {
	command = t.expandSkillScripts(command)
	ws := ""
	if t != nil {
		ws = t.Workspace
	}
	command = repairWorkspacePathEscapes(command, ws)
	command = stripRedundantWorkspaceCd(command, ws)
	if runtime.GOOS == "windows" {
		command = normalizeWindowsQuotes(command)
	}
	command, cleanupInline := rewriteInlineInterpreters(command, ws)
	defer cleanupInline()
	if runtime.GOOS == "windows" && t != nil {
		if pat, ok := parseUnixFindGlob(command); ok {
			res := t.glob(pat)
			prefix := "redirected unix find to glob (Windows FIND.exe is a text search) pattern=" + pat + "\n"
			if res.Err != nil {
				res.Content = prefix + res.Err.Error()
				return res
			}
			res.Content = prefix + res.Content
			return res
		}
		if unixFindMisuse(command) {
			return ToolResult{Err: fmt.Errorf("Windows FIND.exe is a text search, not unix find(1). Use glob (pattern *.pdf) or grep for content")}
		}
	}
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
	if runtime.GOOS == "windows" {
		command, argv = rewriteWindowsBash(command, argv)
	}
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
	rewriteNote := ""
	if runtime.GOOS == "windows" && windowsPosixShell() == "" {
		if next, ok := stripUnixOutputPipes(command); ok {
			rewriteNote = "dropped unix head/tail pipe (no bash on PATH); output is already captured\n"
			command = next
			argv = SplitShellArgv(command)
		}
	}
	posix := looksPosixUnix(command)
	if runtime.GOOS == "windows" && cmdStartWouldHang(command) {
		if next, ok := rewriteAnyStart(command); ok {
			rewriteNote += "rewrote cmd start to Start-Process (captured stdout would hang)\n"
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
	cmd, cleanup, err := bindShell(bindCtx, command, argv, posix, ws)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return ToolResult{Err: err}
	}
	cmd.Dir = t.Workspace
	out := runShell(ctx, cmd, idle, block, t.emitStdout)
	if out.Err != nil && runtime.GOOS == "windows" && !posix && !looksPosixUnix(command) && !shellNeedsWrapper(command, argv) && isExecNotFound(out.Err) {
		if fuse {
			cmd = exec.Command("cmd", "/S", "/C", command)
		} else {
			cmd = exec.CommandContext(ctx, "cmd", "/S", "/C", command)
		}
		cmd.Dir = t.Workspace
		out = runShell(ctx, cmd, idle, block, t.emitStdout)
	}
	content := rewriteNote + string(out.Output)
	if out.Background {
		stopFusedPID(out.PID)
		content = fmt.Sprintf("stopped pid=%d after idle/block fuse (no output). Do not poll with wait/tasklist. For public HTTP use web_fetch; if https returns 406, retry the http URL.\n%s", out.PID, content)
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

func stopFusedPID(pid int) {
	if pid <= 0 {
		return
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
		hideConsole(cmd)
		_ = cmd.Run()
		return
	}
	p, err := os.FindProcess(pid)
	if err == nil {
		_ = p.Kill()
	}
}

func runShell(ctx context.Context, cmd *exec.Cmd, idle, block time.Duration, onChunk func([]byte)) isolation.Outcome {
	if idle > 0 || block > 0 {
		return isolation.WaitNotify(ctx, cmd, idle, block, onChunk)
	}
	if onChunk != nil {
		return isolation.WaitNotify(ctx, cmd, 0, 0, onChunk)
	}
	out, err := isolation.Run(ctx, cmd)
	return isolation.Outcome{Output: out, Err: err}
}

func (t *WorkspaceTools) setLiveCall(id, round string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.liveCall = id
	t.liveRound = round
	t.liveOut = 0
	t.mu.Unlock()
}

func (t *WorkspaceTools) emitStdout(p []byte) {
	if t == nil || t.OnLive == nil || len(p) == 0 {
		return
	}
	t.mu.Lock()
	id := t.liveCall
	round := t.liveRound
	session := t.SessionID
	if id == "" || t.liveOut >= 8_000 {
		t.mu.Unlock()
		return
	}
	remain := 8_000 - t.liveOut
	if len(p) > remain {
		p = p[:remain]
	}
	t.liveOut += len(p)
	t.mu.Unlock()
	payload := map[string]any{
		"id": id, "name": "shell", "content": string(p), "delta": true, "untrusted": true,
	}
	if round != "" {
		payload["round"] = round
	}
	t.OnLive(trace.Event{
		Type:      trace.TypeToolResult,
		SessionID: session,
		Source:    "tool",
		Payload:   payload,
	})
}

func bindShell(ctx context.Context, command string, argv []string, posix bool, workspace string) (*exec.Cmd, func(), error) {
	nop := func() {}
	newCmd := func(name string, args ...string) *exec.Cmd {
		if ctx != nil {
			return exec.CommandContext(ctx, name, args...)
		}
		return exec.Command(name, args...)
	}
	if runtime.GOOS == "windows" {
		if looksLikePowerShell(command) || shellHasStatementSep(command) {
			exe := pwsh.Look()
			if exe == "" {
				exe = pwsh.Fallback()
			}
			name, args, cleanup, err := materializePowerShell(exe, command, workspace)
			if err != nil {
				return nil, nop, err
			}
			return newCmd(name, args...), cleanup, nil
		}
		if posix || looksPosixUnix(command) {
			sh := windowsPosixShell()
			if sh == "" {
				return nil, nop, fmt.Errorf("unix shell syntax on Windows and bash is not on PATH; use list_dir/glob/grep or cmd builtins")
			}
			return newCmd(sh, "-lc", "set +m; trap '' HUP; "+command), nop, nil
		}
		if !shellNeedsWrapper(command, argv) {
			return newCmd(argv[0], argv[1:]...), nop, nil
		}
		// /S keeps quoted tokens (findstr /C:"--- PASS") from being split
		// when Go's Windows argv joining wraps the command.
		return newCmd("cmd", "/S", "/C", command), nop, nil
	}
	if !shellNeedsWrapper(command, argv) {
		return newCmd(argv[0], argv[1:]...), nop, nil
	}
	return newCmd("sh", "-lc", command), nop, nil
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
	if strings.Contains(low, `\sysnative\`) || strings.Contains(low, `/sysnative/`) {
		return false
	}
	if strings.Contains(low, `\syswow64\`) || strings.Contains(low, `/syswow64/`) {
		return false
	}
	if strings.Contains(low, `\windowsapps\`) || strings.Contains(low, `\windows\system32`) {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func rewriteWindowsBash(command string, argv []string) (string, []string) {
	if len(argv) == 0 {
		return command, argv
	}
	name := strings.ToLower(filepath.Base(argv[0]))
	name = strings.TrimSuffix(name, ".exe")
	if name != "bash" && name != "sh" {
		return command, argv
	}
	if usableWindowsPosixShell(argv[0]) {
		return command, argv
	}
	sh := windowsPosixShell()
	if sh == "" {
		return command, argv
	}
	argv = append([]string{sh}, argv[1:]...)
	return joinShellArgv(argv), argv
}

func isWSLPosixStub(p string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	p = strings.Trim(p, `"'`)
	base := strings.ToLower(filepath.Base(p))
	name := strings.TrimSuffix(base, ".exe")
	if name != "bash" && name != "sh" && name != "wsl" {
		return false
	}
	abs := p
	if !filepath.IsAbs(p) {
		if found := lookPath(p); found != "" {
			abs = found
		} else if found := lookPath(name + ".exe"); found != "" {
			abs = found
		} else {
			return false
		}
	}
	return !usableWindowsPosixShell(abs)
}

func (t *WorkspaceTools) loadSkill(name string) ToolResult {
	name = t.resolveSkillName(name)
	if t.Skills == nil && (t.BootstrapBodies == nil || t.BootstrapBodies[name] == "") {
		return ToolResult{Err: fmt.Errorf("no skills")}
	}
	t.mu.Lock()
	if t.BootstrapBodies != nil {
		if wrapped := strings.TrimSpace(t.BootstrapBodies[name]); wrapped != "" {
			t.mu.Unlock()
			return ToolResult{Content: wrapped}
		}
	}
	t.mu.Unlock()
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

func (t *WorkspaceTools) MarkSkillsLoaded(names ...string) {
	if t == nil {
		return
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		t.mu.Lock()
		dup := false
		for _, n := range t.Loaded {
			if n == name {
				dup = true
				break
			}
		}
		if !dup {
			t.Loaded = append(t.Loaded, name)
		}
		t.mu.Unlock()
	}
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

func extraSearchHit(name, blob, q string) bool {
	if strings.Contains(blob, q) {
		return true
	}
	for _, tok := range strings.Fields(q) {
		if len(tok) >= 3 && strings.Contains(name, tok) {
			return true
		}
	}
	return false
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
			if q != "" && !extraSearchHit(name, blob, q) {
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
		if hostToolKnown(q) && (want[q] || alwaysAdvertise(q)) && b.Len() == 0 {
			return ToolResult{Content: q + " is already in your tool list — call it directly. tool_search is for deferred extra (MCP/WASM) tools."}
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
	names := append([]string(nil), t.Loaded...)
	skills := t.Skills
	boot := t.BootstrapBodies
	t.mu.Unlock()
	var out []string
	for _, name := range names {
		body := ""
		if boot != nil {
			if b, ok := boot[name]; ok {
				out = append(out, "## Skill: "+name+"\n"+b)
				continue
			}
		}
		if skills != nil {
			body = skills[name]
		}
		if strings.TrimSpace(body) == "" {
			continue
		}
		out = append(out, "## Skill: "+name+"\n"+t.decorateSkillBody(name, body))
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
