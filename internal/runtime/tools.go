// 工作区工具的调度、路径解析、文件读写和 shell。resolve 不调用 filepath.EvalSymlinks，WithinWorkspace 也不调用。宿主名字表在 tools_host.go。子代理预算在 tools_task.go。单次写入正文超过 writeFilePartMax（32 * 1024 字节）由 refuseHugeWrite 拒绝。

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

// ToolResult 是一次工具调用的返回。Content 是给模型的文本，Err 非空表示失败，FileChange 记录改过的文件，Parts 是多模态片段。
type ToolResult struct {
	Content    string
	Err        error
	FileChange *FileChange
	Parts      []ContentPart
}

// FileChange 记录这次写入涉及的路径、统一 diff、增删行数，以及是否新建。
type FileChange struct {
	Paths   []string
	Patch   string
	Added   int
	Removed int
	Created bool
}

// prefetchHit 是流式阶段提前执行的只读结果。在最终 dispatchTools 发出之前不能单独记成一条轨迹。
type prefetchHit struct {
	content   string
	elapsedMs int
	spillID   string
	ok        bool
	change    *FileChange
	parts     []ContentPart
}

// ExtraTool 是宿主桥接的扩展工具，例如 MCP 或 WASM，走同一套能力闸门。Call 是实现。ReadOnly、OpenWorld、Exclusive、Destructive 描述该工具的副作用。
type ExtraTool struct {
	JSON        ToolJSON
	Call        func(argsJSON string) ToolResult
	ReadOnly    bool
	OpenWorld   bool
	Exclusive   bool
	Destructive bool
}

// WorkspaceTools 是一轮会话的工具状态：工作区、能力经纪、技能、已广告的工具和子代理入口。
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
	// ReadOnly 用于 explore 或 qa 子代理：拒绝写入，但不套用计划模式那句「先产出计划」。计划模式和只读不是一回事。
	ReadOnly bool
	// ChatOverlay 为真表示桌面或 CLI 的个人聊天。Harbor 评测保持为假，使 shell、计划用语和工具菜单保持评测稳定。
	ChatOverlay bool
	// OperatorVoice 是聊天叠加层里最近一句真实的操作者原话。
	OperatorVoice string
	// VerifyHint 在聊天带着未完成计划试图结束时为真，用来把 verify-artifact 重新注入工作记忆。
	VerifyHint bool
	// HaltAfter 为真时，这批工具结束就停止本轮。学习测验用它拦住模型自己作答。
	HaltAfter bool
	Ctx       context.Context
	Extra     map[string]ExtraTool
	Spill     *Spill
	// Depth 是子代理嵌套层。0 是父级。达到 MaxTaskDepth 后不能再派 task。
	Depth        int
	Task         TaskFunc
	PlanText     string
	Advertised   []string
	AllowedTools []string
	// DeniedTools 是配置档的拒绝表。聊天可以调用开场 schema 里没有的宿主工具，这张表仍然生效。
	DeniedTools []string
	// ToolJail 为真表示 AllowedTools 是技能前言或配置档的真实允许表。开场 schema 本身不是牢笼：在设置这张表之前，推迟的宿主工具在聊天里仍可调用。
	ToolJail          bool
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
	Pages             pages.Catalog
	OnCapture         func(kind, path, summary string)
	LastBrowser       string
	AllowedConnectors []string
	Takeover          func(question string) (string, error)
	OnLive            func(trace.Event)
	liveCall          string
	liveRound         string
	liveOut           int
	// Packs 是本回合控制器启用的方法论或内容包。
	Packs []PackSession
	// Methodology 为真时，聊天行为改成先加载技能，计划优先的叠加也会让开。
	Methodology bool
	// BootstrapBodies 是包过的会话开始技能正文。聊天路径不再填充它；方法论包改留一行 BootstrapPin。
	BootstrapBodies map[string]string
	// BootstrapPin 是可缓存的一行方法论提示。正文要通过 load_skill 加载。
	BootstrapPin string
	// CatalogLane 是 agent 或 video。agent 车道会藏起 studio、drama、cut 的目录行。
	CatalogLane  string
	pathRules    string
	pathRuleSeen map[string]bool
	// SkillAutoInvoke 是 off、hint 或 on。空串按 on。
	SkillAutoInvoke string
	// SkillCatalogBudget 按字节限制钉住的目录行。零使用默认预算。
	SkillCatalogBudget int
	// SkillHints 是本回合匹配结果，含 hint 和 autoload。
	SkillHints []SkillSuggestion
	// Autoloaded 是宿主按隐式 @skill 注入的那一部分技能名。
	Autoloaded []string
	// repeatScript 拦住下一次会执行该脚本的命令。删除、移动或阅读这条路径不拦截。repeatRefusal 是不执行时返回的工具结果。
	repeatScript string
	// repeatRefusal 是 repeatScript 命中时直接返回的文本。
	repeatRefusal string
}

// BuiltinToolJSON 把 tool.HostSpecs 转成工具 JSON 列表。
func BuiltinToolJSON() []ToolJSON {
	specs := tool.HostSpecs()
	out := make([]ToolJSON, 0, len(specs))
	for _, s := range specs {
		out = append(out, specJSON(s))
	}
	return out
}

// AllToolJSON 组装本回合广告给模型的工具。深度达到 MaxTaskDepth 时去掉 task。ReadOnly 时只留 readOnlySubagentCall 允许的名字。Advertised 或 AllowedTools 非空时按它们过滤，alwaysAdvertise 的名字保留。额外工具超过 8 个或 ChatOverlay 为真时，未启用的推迟并写入溢出或宿主索引。同名只留先出现的宿主 schema。
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
	// schemaCap 是 8。额外工具多于这个数时，未启用的不放进开场 schema。
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
	WriteHostToolIndex(t.Workspace, deferred)
	WriteMCPCatalog(t.Workspace, t.Extra)
	// 网关会因同名工具拒绝整轮。宿主 schema 保留，后出现的同名扩展（例如画布上的 ask_user）丢掉。
	return uniqueToolJSON(out)
}

// uniqueToolJSON 按名字去重，保留第一次出现。空名字丢掉。少于两条时原样返回。
// in 少于两条时原样返回；否则按 function 里的 name 去重，去掉空白后为空的名字丢掉。
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

// fn 用名字、说明和参数对象构造一条 function 类型的 ToolJSON。
// name 放进 function 的 name。
// desc 放进 function 的 description。
// params 放进 function 的 parameters。
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

// writeLikeTool 在名字是 write_file、create_file、apply_patch、str_replace、edit_file、office_create、office_edit 或 cite_sources 时返回真。
// name 是 write_file、create_file、apply_patch、str_replace、edit_file、office_create、office_edit 或 cite_sources 时返回真。
func writeLikeTool(name string) bool {
	switch name {
	case "write_file", "create_file", "apply_patch", "str_replace", "edit_file",
		"office_create", "office_edit", "cite_sources":
		return true
	default:
		return false
	}
}

// Call 是工具入口。写类工具的参数被截断时不执行。未广告、只读子代理或计划模式不允许的调用直接返回错误。已登记的宿主工具走 hostFns，否则走 Extra。
// name 交给 writeLikeTool、toolAllowed、readOnlySubagentCall、readonlyCall 和 ladderBlock，并用来查 hostFns 和 Extra。
// argsJSON 在写类工具上交给 toolArgsTruncated；截断时交给 truncatedWriteErr，否则解析成参数，并接进 ladderBlock 的提示。
func (t *WorkspaceTools) Call(name, argsJSON string) ToolResult {
	if writeLikeTool(name) && toolArgsTruncated(argsJSON) {
		// 截断的载荷不执行。把捞出来的前缀写到磁盘会让下一轮以为已经写完，再对着没写过的文本做 str_replace。模型应改成分段重写。
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

// toolListed 在列表项按空白切开后含有该名字时返回真。
// list 每一项按空白切开后再和 name 比较。
// name 与切开后的某一段相等时返回真。
func toolListed(list []string, name string) bool {
	for _, n := range list {
		for _, part := range strings.Fields(n) {
			if part == name {
				return true
			}
		}
	}
	return false
}

// toolDenied 在 DeniedTools 含有该名字时返回真。工具或名字为空则返回假。
// name 为空时返回假；否则看 DeniedTools 里有没有相等的项。
func (t *WorkspaceTools) toolDenied(name string) bool {
	if t == nil || name == "" {
		return false
	}
	for _, n := range t.DeniedTools {
		if n == name {
			return true
		}
	}
	return false
}

// toolAllowed 先看拒绝表。Extra 里的名字和 alwaysAdvertise 的名字放行。ChatOverlay 且未设 ToolJail 时，已知宿主工具放行。AllowedTools 非空时只认这张表。ChatOverlay 下其余已知宿主工具也放行。有广告表时必须在表里。都没有则放行。
// name 先交给 toolDenied；再看 Extra、alwaysAdvertise、hostToolKnown、AllowedTools 和广告表。
func (t *WorkspaceTools) toolAllowed(name string) bool {
	if t == nil {
		return true
	}
	if t.toolDenied(name) {
		return false
	}
	if t.Extra != nil {
		if _, ok := t.Extra[name]; ok {
			return true
		}
	}
	if alwaysAdvertise(name) {
		return true
	}
	// 开场菜单只是精简 schema，不是调用牢笼。配置档拒绝表和显式允许表 ToolJail 仍然拦截。
	if t.ChatOverlay && hostToolKnown(name) && !t.ToolJail {
		return true
	}
	if len(t.AllowedTools) > 0 {
		return toolListed(t.AllowedTools, name)
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

// advertisedCopy 在锁内复制 Advertised。工具为空时返回 nil。
func (t *WorkspaceTools) advertisedCopy() []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.Advertised...)
}

// alwaysAdvertise 对 load_skill、list_skills、tool_search、recall_context、update_plan、ask_user、read_thread、read_skill_file 返回真。
// name 是 load_skill、list_skills、tool_search、recall_context、update_plan、ask_user、read_thread 或 read_skill_file 时返回真。
func alwaysAdvertise(name string) bool {
	switch name {
	case "load_skill", "list_skills", "tool_search", "recall_context", "update_plan", "ask_user", "read_thread", "read_skill_file":
		return true
	default:
		return false
	}
}

// CommitToolUnlocks 把 ExtraLive 并进 ExtraEnabled 后清空 ExtraLive，并把 pendingHost 交给 unlockAdvertised。
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

// resolve 把相对路径接到 Workspace 上。先 CanonicalizeToolPath，绝对路径保持不变，再 filepath.Clean。WithinWorkspace 拒绝逃出工作区，ForbiddenDiagPath 拒绝诊断面。本函数不调用 filepath.EvalSymlinks，WithinWorkspace 也不调用。
// rel 为空时返回错误；否则交给 CanonicalizeToolPath，非绝对路径再接到 Workspace 上。
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

// check 在 Caps 为空时直接通过。否则按级别、动作、路径和命令做能力检查。policyRequires 为真时 ForceAsk。有工具上下文则用 CheckCtx。
// level 放进能力请求的 Level，并交给 policyRequires。
// action 放进能力请求的 Action。
// path 放进能力请求的 Path。
// cmd 放进能力请求的 Command。
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

// policyRequires 在模式为 bypass 时返回假。模式为 ask 且级别是 Shell、Network、HighRisk、SendAsYou 或 ComputerUse 时返回真。RequireApproval 不在这里变成 ForceAsk，经纪上已有的 Session、Once、Always 授权要保留，所以其余情况返回假。
// level 只在模式为 ask 时和 Shell、Network、HighRisk、SendAsYou、ComputerUse 比较。
func policyRequires(p artifact.PolicyPack, level capability.Level) bool {
	if p.Mode == "bypass" {
		return false
	}
	if p.Mode == "ask" && (level == capability.Shell || level == capability.Network || level == capability.HighRisk || level == capability.SendAsYou || level == capability.ComputerUse) {
		return true
	}
	// RequireApproval 表示第一次调用要进闸门，但这里不把它变成 ForceAsk。经纪上已有的 Session、Once、Always 授权要保留。
	return false
}

// readFile 经 resolve 和 ReadWorkspace 读取文件。指定 offset 或 limit 时走 rangedRead。超过 maxInlineBytes 时把全文写到 .yoyo/overflow，只返回前 80 行和溢出路径。
// rel 交给 resolve；读失败时再交给 missingReadErr，超长时交给 spillOverflow。
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

// missingReadErr 在错误不是不存在时原样返回。不存在时说明路径不存在。路径以 qa/verification.json 或 qa/evidence/run.json 结尾时追加一句：这些文件由 browser_verify 写在游戏工程目录，root 应是该目录而不是 CSS 选择器。父目录能读时最多附上 8 个邻近名字。
// rel 在错误是不存在时写成斜杠路径，放进错误文本，并用来识别 qa/verification.json 和 qa/evidence/run.json。
// abs 取父目录，能读时最多附上 8 个邻近名字。
// err 为 nil 时返回 nil；不是不存在时原样返回。
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

// rangedRead 避免空窗口被当成没有输出的成功。文件为空时写明 0 行。offset 超过总行数时写明实际行数，并指出可读范围。否则交给 numberLines。
// rel 写进空文件或 offset 越界时的说明。
// text 为空时总行数按 0；否则按去掉末尾换行后的换行切开计行，并交给 numberLines。
// limit 连同 offset 交给 numberLines。
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

// numberLines 按 1 起的行号输出。offset 大于 0 时从该行开始。limit 大于 0 时只取这么多行。
// text 按换行切开后编号。
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

// spillOverflow 把超长读到的字节写到工作区 .yoyo/overflow 下，文件名是基名加 .txt，并返回斜杠相对路径。
// rel 取基名，加上 .txt，作为溢出文件名。
// b 写入该文件。
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

// refuseHugeWrite 在 content 的字节数超过 writeFilePartMax 时拒绝。writeFilePartMax 是 32 * 1024。错误文本要求先 create 或 overwrite，再按同样上限 append。
// rel 在超限时写进错误文本。
func refuseHugeWrite(rel, content string) error {
	if len(content) <= writeFilePartMax {
		return nil
	}
	return fmt.Errorf("refusing write_file %s: content is %d bytes (max %d per call). Split the body: first part mode=create or overwrite, each following part mode=append, at most %d bytes per call", rel, len(content), writeFilePartMax, writeFilePartMax)
}

// writeFile 覆盖或新建文件。内容像上下文占位时走 resolveElidedWrite。先过 refuseHugeWrite 和 sealedGameWorldWrite，再 resolve，并以 WriteWorkspace、动作 write_file 审批。与磁盘相同则返回 unchanged。写入前 snapshotBeforeWrite。html 或 htm 会尝试预览。
// rel 在内容像占位时交给 resolveElidedWrite；否则交给 refuseHugeWrite、sealedGameWorldWrite 和 resolve。
// content 像上下文占位时不落盘；否则作为要写入的字节，并交给 refuseHugeWrite。
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

// createFile 只在路径还不存在时写入。这是分段写长文件的第一段，不会覆盖已有内容。已存在则拒绝，并提示改用 overwrite 或 append。通过后交给 writeFile，因此同样受 32 * 1024 字节的单次上限。
// rel 交给 resolve；已存在时写进拒绝文本，否则连同 content 交给 writeFile。
// content 连同 rel 交给 writeFile。
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

// appendFile 把内容追加到已有文件，使长正文不必塞进一次模型输出。上下文占位直接拒绝。先过 refuseHugeWrite，单次仍不得超过 writeFilePartMax（32 * 1024 字节）。文件不存在时当成新建。html 追加被 htmlAppendRefusal 拒绝时不写。
// rel 在内容像占位时写进拒绝文本；否则交给 refuseHugeWrite、sealedGameWorldWrite 和 resolve。
// content 像上下文占位时直接拒绝；否则交给 refuseHugeWrite，通过后追加到已有字节后面。
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

// resolveElidedWrite 不把上下文占位写成文件。路径已有非空文件则返回 unchanged。能从溢出参数里按路径和字段恢复正文时，patch 走 applyPatch，其余走 writeFile。否则拒绝。
// rel 去掉空白后为空则拒绝；否则交给 resolve，已有非空文件则返回 unchanged。
// field 等于 patch 时恢复出的正文走 applyPatch，否则走 writeFile；失败时也写进错误文本。
// recall 连同 rel 和 field 交给 recoverElidedField。
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

// recoverElidedField 先按 recall 标识从溢出里取字段，再扫描以 args 结尾的溢出标识。路径必须一致，正文不能再是占位。
// rel 去掉空白后为空则返回假；否则换成斜杠路径，作为溢出 JSON 里 path 必须一致的值。
// field 去掉空白后为空则返回假；否则作为要从溢出 JSON 取出的键。
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

// spillArgsField 读取溢出 JSON。path 与 wantPath 不一致、字段为空或仍像占位时返回假。
// id 去掉空白，空串则返回假；否则交给 Spill.Get。
// field 作为溢出 JSON 的键取值；取出的字符串为空或仍像占位时返回假。
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

// maybePreviewHTML 在扩展名是 .html 或 .htm 且 Browser 非空时 SetPreview。
// rel 取扩展名，只有 .html 或 .htm 才预览。
// abs 为空时直接返回；否则在扩展名匹配时交给 SetPreview。
func (t *WorkspaceTools) maybePreviewHTML(rel, abs string) {
	if t == nil || t.Browser == nil || abs == "" {
		return
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".html", ".htm":
		t.Browser.SetPreview(abs)
	}
}

// replace 做字符串替换。old 或 new 像上下文占位、old 为空、或路径被密封时拒绝。匹配 0 次返回错误并附上邻近行。多于一次且 replace_all 为假则拒绝。写入前做快照。能力级别是 WriteWorkspace，动作是 str_replace。
// rel 交给 sealedGameWorldWrite 和 resolve；匹配不到或没有变化时写进结果文本。
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

// listDir 列出目录。最多 200 项，超过则提示改用 glob。文件行带字节大小和修改时间。能力级别是 ReadWorkspace。
// rel 交给 resolve，通过后再列出该目录。
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

// glob 在工作区里按模式查找，最多 200 条。空模式返回错误。能力级别是 ReadWorkspace。
// pattern 为空时返回错误；否则连同工作区和 200 交给 globWalk。
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

// grep 在工作区或指定路径里搜索。正则编译失败时按字面量搜索，并在结果前说明。最多 80 条。能力级别是 ReadWorkspace。
// pattern 为空时返回错误；编译失败时按字面量再编译。
// globPat 交给 grepWalk。
// rel 非空时交给 resolve，作为搜索根；为空则用 Workspace。
func (t *WorkspaceTools) grep(pattern, globPat, rel string) ToolResult {
	if pattern == "" {
		return ToolResult{Err: fmt.Errorf("empty pattern")}
	}
	re, compileErr := regexp.Compile(pattern)
	literal := compileErr != nil
	if literal {
		// 贴进来的代码片段常常不是正则。这里若直接报错，模型会原样重试。编译失败就按字面量搜索。
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

// setRepeatBlock 记下要拦住的脚本片段和拒绝时返回的文本。
// script 记到 repeatScript。
// refusal 记到 repeatRefusal。
func (t *WorkspaceTools) setRepeatBlock(script, refusal string) {
	if t == nil {
		return
	}
	t.repeatScript = script
	t.repeatRefusal = refusal
}

// repeatBlocked 在 repeatScript 为空、拒绝文本为空，或脚本是 browser_verify 时返回假。只有会执行该脚本的命令才返回拒绝文本。路径出现在删除、移动、阅读或列目录里仍然执行。
// command 交给 commandRerunsScript，和 repeatScript 比较。
func (t *WorkspaceTools) repeatBlocked(command string) (string, bool) {
	if t == nil || t.repeatScript == "" || t.repeatRefusal == "" || t.repeatScript == "browser_verify" {
		return "", false
	}
	if commandRerunsScript(command, t.repeatScript) {
		return t.repeatRefusal, true
	}
	return "", false
}

// repeatVerifyBlocked 只在 repeatScript 为 browser_verify 且拒绝文本非空时返回该文本。
func (t *WorkspaceTools) repeatVerifyBlocked() (string, bool) {
	if t == nil || t.repeatScript != "browser_verify" || t.repeatRefusal == "" {
		return "", false
	}
	return t.repeatRefusal, true
}

// shell 先看重复拦截，再展开技能脚本、修理工作区路径并去掉多余的 cd。Windows 上把 unix find 转成 glob，聊天叠加层把 cat 或 sed 读文件转成 readFile。超时小于等于 0 时用 60 秒。ChatOverlay 且超时大于 chatShellBlock 时启用空闲和阻塞熔断，熔断后不再给命令加超时上下文。能力级别是 Shell。
// command 先交给 repeatBlocked，再展开技能脚本、修理路径、去掉多余 cd，并作为最终要执行的命令。
// timeoutSec 小于等于 0 时改成 60 秒；聊天叠加层下再交给 chatShellFuse。
func (t *WorkspaceTools) shell(command string, timeoutSec int) ToolResult {
	if msg, ok := t.repeatBlocked(command); ok {
		return ToolResult{Content: msg}
	}
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

// chatShellBlock 是 30 秒，chatShellIdle 是 10 秒。聊天 shell 的熔断用这两个时长。
const (
	// chatShellBlock 是 30 秒。聊天命令的超时大于它才启用熔断，熔断后的阻塞上限也是它。
	chatShellBlock = 30 * time.Second
	// chatShellIdle 是 10 秒。熔断时这么久没有输出就停掉进程。
	chatShellIdle = 10 * time.Second
)

// chatShellFuse 在不是聊天叠加层，或超时不超过 chatShellBlock 时关闭熔断。否则返回 chatShellIdle 和 chatShellBlock。
// overlay 为假时关闭熔断。
// timeout 不超过 chatShellBlock 时关闭熔断。
func chatShellFuse(overlay bool, timeout time.Duration) (idle, block time.Duration, fuse bool) {
	if !overlay || timeout <= chatShellBlock {
		return 0, 0, false
	}
	return chatShellIdle, chatShellBlock, true
}

// stopFusedPID 在 pid 小于等于 0 时返回。Windows 上用 taskkill /T /F。其他系统 FindProcess 后 Kill。
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

// runShell 在 idle 或 block 大于 0，或有输出回调时用 isolation.WaitNotify。否则用 isolation.Run。
// ctx 交给 isolation.WaitNotify 或 isolation.Run。
// cmd 交给 isolation.WaitNotify 或 isolation.Run。
// onChunk 非 nil，或 idle、block 大于 0 时交给 WaitNotify。
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

// setLiveCall 记下当前工具调用 ID、轮次，并把已转发的标准输出字节数清零。
// id 写入 liveCall。
// round 写入 liveRound。
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

// emitStdout 把 shell 的标准输出增量发给 OnLive。单次调用累计最多 8000 字节，超出的丢掉。载荷标成 delta 和 untrusted。
// p 长度为 0 时直接返回；否则按剩余额度截断后放进 OnLive 的 content。
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

// bindShell 按命令形态选择解释器。Windows 上像 PowerShell 或含语句分隔符时走 pwsh。POSIX 语法走 windowsPosixShell，找不到则返回错误。不需要包装时直接执行 argv。Windows 其余情况用 cmd /S /C。非 Windows 需要包装时用 sh -lc。熔断路径的 ctx 为 nil，因此不带超时。
// command 在 Windows 上交给 looksLikePowerShell 和 shellHasStatementSep；像 PowerShell 时连同 workspace 交给 materializePowerShell，POSIX 时拼进 bash -lc。
// posix 为真时在 Windows 上走 windowsPosixShell。
// workspace 在走 PowerShell 时交给 materializePowerShell。
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
		// /S 让带引号的标记在 Go 拼接 Windows 参数时不被拆开，例如 findstr /C:"--- PASS"。
		return newCmd("cmd", "/S", "/C", command), nop, nil
	}
	if !shellNeedsWrapper(command, argv) {
		return newCmd(argv[0], argv[1:]...), nop, nil
	}
	return newCmd("sh", "-lc", command), nop, nil
}

// isExecNotFound 在错误为 nil 时返回假。ErrNotFound，或文本含 executable file not found、file not found、not recognized as an internal 时返回真。
// err 为 nil 时返回假；ErrNotFound，或文本含 executable file not found、file not found、not recognized as an internal 时返回真。
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

// windowsPosixShell 只在 Windows 上查找 Git Bash 或 msys 的 bash。跳过 system32、sysnative、syswow64 和 WindowsApps 里的 WSL 桩。都不可用则返回空串。
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

// usableWindowsPosixShell 拒绝空路径，以及位于 system32、sysnative、syswow64、WindowsApps 或 windows\\system32 下的路径。目标必须存在且不是目录。
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

// rewriteWindowsBash 在 argv 的程序名是 bash 或 sh 且该路径不可用时，换成 windowsPosixShell。找不到则保持原样。
// command 在 argv 为空、程序名不是 bash 或 sh，或找不到可用 shell 时原样返回；换成功后改由 joinShellArgv 重新拼出。
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

// isWSLPosixStub 在 Windows 上判断 bash、sh 或 wsl 是否落在不可用的系统桩路径上。其他系统返回假。
// p 去掉引号后取文件名；不是 bash、sh 或 wsl 时返回假，绝对路径再交给 usableWindowsPosixShell。
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

// loadSkill 先按别名收成规范名。BootstrapBodies 里有包过的正文就直接返回。否则从 Skills 取正文，装饰后加入 Loaded。技能带 AllowedTools 时写入允许表并把 ToolJail 设为真。未知技能返回错误。
// name 先交给 resolveSkillName，再查 BootstrapBodies 和 Skills。
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
			t.ToolJail = true
		}
	}
	return ToolResult{Content: content}
}

// MarkSkillsLoaded 把非空名字记入 Loaded，已存在的不重复加。
// names 去掉空白，空串跳过，已在 Loaded 里的不重复加。
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

// recall 从 Spill 按标识取回文本并分页。没有溢出存储或标识未知时返回错误。
// id 交给 Spill.Get；未知时写进错误文本，成功后连同正文、offset 和 limit 交给 pageSpill。
// offset 连同取回的文本交给 pageSpill。
// limit 连同取回的文本交给 pageSpill。
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

// defaultRecallLines 是 200。recall 未给 limit 时每页这么多行。
const defaultRecallLines = 200

// defaultRecallRunes 是 4000。每一页再用 capText 截到这么多 rune。
const defaultRecallRunes = 4_000

// pageSpill 按行分页。offset 小于等于 0 时从 1 开始。limit 小于等于 0 时用 defaultRecallLines。页文本再截到 defaultRecallRunes。
// id 写进返回文本的 recall id= 那一行。
func pageSpill(id, s string, offset, limit int) string {
	lines := strings.Split(s, "\n")
	if offset <= 0 {
		// 没给 offset 时从第 1 行开始，不把缺参当成错误。
		offset = 1
	}
	if limit <= 0 {
		// 没给 limit 时一页 200 行，页文本再按 defaultRecallRunes（4000 个 rune）截断。
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

// recallNoStub 只对 recall_context 为真，且正文 rune 数不超过 2000。
// name 不是 recall_context 时返回假。
// content 的 rune 数不超过 2000 时，并且 name 是 recall_context，才返回真。
func recallNoStub(name, content string) bool {
	if name != "recall_context" {
		return false
	}
	return utf8.RuneCountInString(content) <= 2000
}

// extraSearchHit 在描述里包含查询，或查询中长度至少 3 的词出现在工具名里时返回真。
// name 用来看 q 里长度至少 3 的词是否出现在工具名里。
// blob 包含 q 时直接返回真。
// q 按空白切开，长度至少 3 的词再去 name 里找。
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

// toolSearch 在 Extra 里按名字和描述匹配，并把命中的 schema 返回。ChatOverlay 下还会搜索尚未广告的宿主工具。聊天里命中的扩展工具记入 ExtraLive，宿主工具记入 pendingHost。非聊天则直接启用扩展工具并 unlockAdvertised。查询已经是当前工具表里的宿主工具且没有扩展命中时，提示直接调用。
// q 去掉空白并转成小写；非空时交给 extraSearchHit，聊天叠加层下还交给 hostToolKnown 和 searchHostSpecs。
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
			if want[name] || alwaysAdvertise(name) || t.toolDenied(name) {
				continue
			}
			if t.ToolJail && !toolListed(t.AllowedTools, name) {
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

// loadedBodies 按 Loaded 的顺序取出技能正文。BootstrapBodies 优先。空正文跳过。其余经 decorateSkillBody。
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

// Pins 返回已加载技能名的副本和当前计划文本。工具为空时两者皆空。
func (t *WorkspaceTools) Pins() (loaded []string, plan string) {
	if t == nil {
		return nil, ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.Loaded...), t.PlanText
}

// str 把 nil 收成空串，其余用 fmt.Sprint。
// v 为 nil 时返回空串，其余交给 fmt.Sprint。
func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// anyStrings 接受 []string 或 []any。其他类型返回 nil。空白项丢掉。
// v 为 nil 或其他类型时返回 nil；是 []string 时原样返回，是 []any 时丢掉去掉空白后为空的项。
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

// intArg 把 nil 收成 0。接受 int、int64、float64、json.Number 和十进制字符串。
// v 为 nil 时返回 0；int、int64、float64 和 json.Number 转成 int，字符串和其他值再按十进制解析。
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

// boolArg 只把布尔真，或字符串 true、1 当成真。其余为假。
// v 是布尔值时原样返回；是字符串时只有 true 或 1 为真，其余类型返回假。
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
