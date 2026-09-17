package tool

import (
	"sort"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// Registry is the live ToolSpec catalog. Host, MCP, and WASM all register here.
type Registry struct {
	mu    sync.Mutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(t Tool) {
	if r == nil || t == nil {
		return
	}
	spec := t.Spec()
	if spec.Name == "" {
		return
	}
	r.mu.Lock()
	r.tools[spec.Name] = t
	r.mu.Unlock()
}

func (r *Registry) Unregister(name string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	delete(r.tools, name)
	r.mu.Unlock()
}

func (r *Registry) Get(name string) Tool {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tools[name]
}

func (r *Registry) Specs() []artifact.ToolSpec {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]artifact.ToolSpec, 0, len(names))
	for _, n := range names {
		out = append(out, r.tools[n].Spec())
	}
	return out
}

func (r *Registry) Names() []string {
	specs := r.Specs()
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}

func (r *Registry) Filter(advertise, allowed []string) []artifact.ToolSpec {
	specs := r.Specs()
	if len(advertise) == 0 && len(allowed) == 0 {
		return specs
	}
	want := map[string]bool{}
	if len(advertise) > 0 {
		for _, n := range advertise {
			want[n] = true
		}
	}
	allow := map[string]bool{}
	for _, n := range allowed {
		for _, part := range strings.Fields(n) {
			allow[part] = true
		}
	}
	var out []artifact.ToolSpec
	for _, s := range specs {
		if len(want) > 0 && !want[s.Name] {
			continue
		}
		if len(allow) > 0 && !allow[s.Name] && !alwaysAllowed(s.Name) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func alwaysAllowed(name string) bool {
	switch name {
	case "load_skill", "list_skills", "tool_search", "recall_context", "update_plan", "ask_user":
		return true
	default:
		return false
	}
}

func ParseAllowedTools(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}
