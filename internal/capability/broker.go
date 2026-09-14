package capability

import (
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
	Level     Level
	Action    string
	Path      string
	Command   string
	SessionID string
	Workspace string
}

type AutoPolicy struct {
	Allow []Level
}

type Broker struct {
	mu      sync.Mutex
	always  map[Level]bool
	session map[string]map[Level]bool
	ask     func(Request) (Decision, error)
	policy  AutoPolicy
}

func NewBroker(policy AutoPolicy, ask func(Request) (Decision, error)) *Broker {
	if ask == nil {
		ask = func(Request) (Decision, error) { return Deny, fmt.Errorf("capability: no approver") }
	}
	always := map[Level]bool{}
	for _, l := range policy.Allow {
		always[l] = true
	}
	return &Broker{
		always:  always,
		session: map[string]map[Level]bool{},
		ask:     ask,
		policy:  policy,
	}
}

func (b *Broker) AllowAlways(l Level) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.always[l] = true
}

func (b *Broker) Check(req Request) error {
	if req.Workspace != "" && req.Path != "" {
		if !WithinWorkspace(req.Workspace, req.Path) && req.Level != SpecifiedPath && req.Level != HighRisk {
			return fmt.Errorf("capability: path %q escapes workspace", req.Path)
		}
	}
	b.mu.Lock()
	if b.always[req.Level] {
		b.mu.Unlock()
		return nil
	}
	if sess := b.session[req.SessionID]; sess != nil && sess[req.Level] {
		b.mu.Unlock()
		return nil
	}
	ask := b.ask
	b.mu.Unlock()
	dec, err := ask(req)
	if err != nil {
		return err
	}
	switch dec {
	case Always:
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
