package capability

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

type Level string

const (
	ReadWorkspace  Level = "read_workspace"
	WriteWorkspace Level = "write_workspace"
	SpecifiedPath  Level = "specified_path"
	Shell          Level = "shell"
	Network        Level = "network"
	HighRisk       Level = "high_risk"
)

type Decision string

const (
	Deny    Decision = "deny"
	Once    Decision = "once"
	Session Decision = "session"
	Always  Decision = "always"
)

type Request struct {
	Level     Level  `json:"level"`
	Action    string `json:"action"`
	Path      string `json:"path"`
	Command   string `json:"command"`
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace"`
	ForceAsk  bool   `json:"force_ask"`
}

type AutoPolicy struct {
	Allow []Level
}

type Broker struct {
	mu      sync.Mutex
	always  map[Level]bool
	session map[string]map[Level]bool
	strict  map[string]bool
	ask     func(ctx context.Context, req Request) (Decision, error)
	policy  AutoPolicy
}

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

func (b *Broker) ClearSession(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.session, sessionID)
	delete(b.strict, sessionID)
}

// ApplyAuthMode replaces this session's grant set. Ask skips the always-map
// so even workspace reads/writes prompt. Identity levels stay ForceAsk.
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

func (b *Broker) AllowAlways(l Level) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.always[l] = true
}

func (b *Broker) Check(req Request) error {
	return b.CheckCtx(context.Background(), req)
}

func (b *Broker) CheckCtx(ctx context.Context, req Request) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Workspace != "" && req.Path != "" && workspacePathJail(req) {
		if !WithinWorkspace(req.Workspace, req.Path) {
			return fmt.Errorf("capability: path %q escapes workspace", req.Path)
		}
	}
	b.mu.Lock()
	strict := b.strict[req.SessionID]
	if !strict && b.always[req.Level] {
		b.mu.Unlock()
		return nil
	}
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
		if NeverAlways(req.Level) {
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

func osPathSeparator() string {
	return string(filepath.Separator)
}

// LooksLikeURL is a network locator, not a workspace file path.
func LooksLikeURL(p string) bool {
	s := strings.ToLower(strings.TrimSpace(p))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// workspacePathJail is only for workspace file levels. Network/browser
// tools put URLs in Path; jailing those as files blocks every fetch.
func workspacePathJail(req Request) bool {
	switch req.Level {
	case ReadWorkspace, WriteWorkspace:
		return !LooksLikeURL(req.Path)
	default:
		return false
	}
}
