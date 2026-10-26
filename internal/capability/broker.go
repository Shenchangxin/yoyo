// Package capability 按能力级别审批工具调用，并把工作区读写路径限制在根目录之内。
// Broker 持有始终表和会话授权，Gate 挂起待决审批。身份级别是 SendAsYou、ComputerUse、HighRisk，不能记成 Always。
package capability

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Level 是一项能力的名字，例如 read_workspace 或 shell。
type Level string

const (
	// ReadWorkspace 是读取工作区文件。
	ReadWorkspace Level = "read_workspace"
	// WriteWorkspace 是改写工作区文件。
	WriteWorkspace Level = "write_workspace"
	// SpecifiedPath 是访问调用方点名的路径。
	SpecifiedPath Level = "specified_path"
	// Shell 是执行命令。
	Shell Level = "shell"
	// Network 是出网访问。
	Network Level = "network"
	// HighRisk 是高风险能力。IdentityLevel 把它和 SendAsYou、ComputerUse 一起视为不能记成 Always 的身份级别。
	HighRisk Level = "high_risk"
)

// Decision 是审批结果。
type Decision string

const (
	// Deny 拒绝本次调用。
	Deny Decision = "deny"
	// Once 只放行这一次，不写入会话授权或始终表。
	Once Decision = "once"
	// Session 把该级别记入本会话。
	Session Decision = "session"
	// Always 表示记入始终表。身份级别和 NeverAlwaysAction 命中的动作在 CheckCtx 里改记本会话。
	Always Decision = "always"
)

// Request 是一次能力检查的输入。
type Request struct {
	Level Level `json:"level"`
	// Action 参与 NeverAlwaysAction。review_page、page.save、page.edit 不能记成 Always。
	Action    string `json:"action"`
	Path      string `json:"path"`
	Command   string `json:"command"`
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace"`
	// ForceAsk 为真时 CheckCtx 不读会话缓存，仍然先看始终表。
	ForceAsk bool `json:"force_ask"`
}

// AutoPolicy 是构造 Broker 时写入始终表的级别。
type AutoPolicy struct {
	Allow []Level
}

// Broker 按始终表、会话授权和审批函数决定放行或拒绝。
type Broker struct {
	mu      sync.Mutex
	always  map[Level]bool
	session map[string]map[Level]bool
	// strict 为真表示该会话是 AuthAsk，CheckCtx 跳过始终表。
	strict map[string]bool
	ask    func(ctx context.Context, req Request) (Decision, error)
	policy AutoPolicy
}

// NewBroker 用 policy.Allow 填始终表。ask 为 nil 时换成直接拒绝的函数，错误文本是 capability: no approver。
func NewBroker(policy AutoPolicy, ask func(ctx context.Context, req Request) (Decision, error)) *Broker {
	if ask == nil {
		ask = func(context.Context, Request) (Decision, error) {
			return Deny, fmt.Errorf("capability: no approver")
		}
	}
	always := map[Level]bool{}
	for _, l := range policy.Allow {
		always[l] = true
	}
	return &Broker{
		always:  always,
		session: map[string]map[Level]bool{},
		strict:  map[string]bool{},
		ask:     ask,
		policy:  policy,
	}
}

// SessionLevels 返回该会话已记住的级别。没有会话时返回空切片。遍历 map，顺序不稳定。
func (b *Broker) SessionLevels(sessionID string) []Level {
	b.mu.Lock()
	defer b.mu.Unlock()
	sess := b.session[sessionID]
	out := make([]Level, 0, len(sess))
	for l := range sess {
		out = append(out, l)
	}
	return out
}

// RestoreSession 把 levels 经 DropSticky 过滤后并入该会话，不清除已有授权。HighRisk、SendAsYou、ComputerUse 不会被恢复。
func (b *Broker) RestoreSession(sessionID string, levels []Level) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session[sessionID] == nil {
		b.session[sessionID] = map[Level]bool{}
	}
	for _, l := range DropSticky(levels) {
		b.session[sessionID][l] = true
	}
}

// ClearSession 删掉该会话的授权和 AuthAsk 标记。
func (b *Broker) ClearSession(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.session, sessionID)
	delete(b.strict, sessionID)
}

// ApplyAuthMode 替换该会话的授权，不保留旧的会话级别。b 为 nil 或 sessionID 为空则直接返回。
// mode 先经 ParseAuthMode。只有 AuthAsk 把 strict 标成 true，其他模式把 strict 清掉，CheckCtx 因此重新看始终表。
// 写入会话的是 DropSticky(AuthModeLevels(mode))，身份级别不会留下。AuthAsk 连工作区读写也会再问。
func (b *Broker) ApplyAuthMode(sessionID, mode string) {
	if b == nil || sessionID == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.strict == nil {
		b.strict = map[string]bool{}
	}
	if b.session == nil {
		b.session = map[string]map[Level]bool{}
	}
	mode = ParseAuthMode(mode)
	b.strict[sessionID] = mode == AuthAsk
	next := map[Level]bool{}
	for _, l := range DropSticky(AuthModeLevels(mode)) {
		next[l] = true
	}
	b.session[sessionID] = next
}

// AllowAlways 把级别写入始终表，不看它是不是身份级别。
func (b *Broker) AllowAlways(l Level) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.always[l] = true
}

// Check 用 context.Background 调用 CheckCtx。
func (b *Broker) Check(req Request) error {
	return b.CheckCtx(context.Background(), req)
}

// CheckCtx 执行一次能力检查。ctx 为 nil 时改用 context.Background。
// Workspace 和 Path 都非空，且 workspacePathJail 为真时，先用 WithinWorkspace 判断；不在根内则返回 escapes workspace。
// strict 会话，以及身份级别或 NeverAlwaysAction 命中的动作，不会因始终表直接通过。
// ForceAsk 为假且本会话已有该级别则直接通过，不再询问。否则调用审批函数。
// 审批返回 Always 时，身份级别和 NeverAlwaysAction 只写入本会话，其他级别写入始终表。
// Session 写入本会话。Once 不记录。其余决定返回 denied。
func (b *Broker) CheckCtx(ctx context.Context, req Request) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Workspace != "" && req.Path != "" && workspacePathJail(req) {
		if !WithinWorkspace(req.Workspace, req.Path) {
			return fmt.Errorf("capability: path %q escapes workspace", req.Path)
		}
	}
	// 身份级别和 NeverAlwaysAction 不走始终表。
	promote := NeverAlways(req.Level) || NeverAlwaysAction(req.Action)
	b.mu.Lock()
	strict := b.strict[req.SessionID]
	if !strict && b.always[req.Level] && !promote {
		b.mu.Unlock()
		return nil
	}
	// ForceAsk 不读会话缓存。始终表已经在上面处理。
	if !req.ForceAsk {
		if sess := b.session[req.SessionID]; sess != nil && sess[req.Level] {
			b.mu.Unlock()
			return nil
		}
	}
	ask := b.ask
	b.mu.Unlock()
	dec, err := ask(ctx, req)
	if err != nil {
		return err
	}
	switch dec {
	case Always:
		// 这两类不能记成 Always，只写入本会话。
		if NeverAlways(req.Level) || NeverAlwaysAction(req.Action) {
			b.mu.Lock()
			if b.session[req.SessionID] == nil {
				b.session[req.SessionID] = map[Level]bool{}
			}
			b.session[req.SessionID][req.Level] = true
			b.mu.Unlock()
			return nil
		}
		b.AllowAlways(req.Level)
		return nil
	case Session:
		b.mu.Lock()
		if b.session[req.SessionID] == nil {
			b.session[req.SessionID] = map[Level]bool{}
		}
		b.session[req.SessionID][req.Level] = true
		b.mu.Unlock()
		return nil
	case Once:
		return nil
	default:
		return fmt.Errorf("capability: denied %s", req.Level)
	}
}

// WithinWorkspace 判断 p 是否在 root 之内，或与 root 是同一路径。
// root 和 p 先经 CanonicalizeToolPath，再各自 filepath.Abs，然后 filepath.Rel。相对路径由 Abs 按当前工作目录展开。Abs 或 Rel 失败则返回 false。
// rel 等于 ".."，或以 ".." 加当前平台的 filepath.Separator 开头时返回 false。路径不必已经存在。本函数不调用 filepath.EvalSymlinks。
func WithinWorkspace(root, p string) bool {
	root = CanonicalizeToolPath(root)
	p = CanonicalizeToolPath(p)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(osPathSeparator()))
}

// osPathSeparator 返回 string(filepath.Separator)，供 WithinWorkspace 识别 ".." 前缀。
func osPathSeparator() string {
	return string(filepath.Separator)
}

// LooksLikeURL 在去掉首尾空白并转成小写后，判断字符串是否以 http:// 或 https:// 开头。这样的值是网络地址，不是工作区文件。
func LooksLikeURL(p string) bool {
	s := strings.ToLower(strings.TrimSpace(p))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// workspacePathJail 只对 ReadWorkspace 和 WriteWorkspace 返回 true，并且 Path 不能是 LooksLikeURL。
// 其他级别返回 false。网络和浏览器工具把 URL 放在 Path 里，当成文件关进工作区会挡掉抓取。
func workspacePathJail(req Request) bool {
	switch req.Level {
	case ReadWorkspace, WriteWorkspace:
		return !LooksLikeURL(req.Path)
	default:
		return false
	}
}
