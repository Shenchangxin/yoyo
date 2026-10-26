// 按名字登记 Tool，并按广告列表和允许列表过滤规格。

package tool

import (
	"sort"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// Registry 是当前的工具规格表。宿主、MCP 和 WASM 都登记到这里。
type Registry struct {
	mu    sync.Mutex
	tools map[string]Tool
}

// NewRegistry 返回一张空表。
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register 按规格名放入工具。接收者、工具或名字为空时直接返回。同名会覆盖。
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

// Unregister 按名字删掉工具。接收者为 nil 时直接返回。
func (r *Registry) Unregister(name string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	delete(r.tools, name)
	r.mu.Unlock()
}

// Get 按名字取出工具。没有或接收者为 nil 时返回 nil。
func (r *Registry) Get(name string) Tool {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tools[name]
}

// Specs 按名字排序返回全部规格。接收者为 nil 时返回 nil。
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

// Names 按 Specs 的顺序返回工具名。
func (r *Registry) Names() []string {
	specs := r.Specs()
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}

// Filter 先按 advertise 留名字，advertise 为空则不限。allowed 非空时，名字必须出现在按空白拆开的允许列表里，或者属于 alwaysAllowed。两个列表都空则返回全部。
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

// alwaysAllowed 对 load_skill、list_skills、tool_search、recall_context、update_plan、ask_user、read_thread、read_skill_file 返回 true。
func alwaysAllowed(name string) bool {
	switch name {
	case "load_skill", "list_skills", "tool_search", "recall_context", "update_plan", "ask_user", "read_thread", "read_skill_file":
		return true
	default:
		return false
	}
}

// ParseAllowedTools 按空白拆允许列表。空白则返回 nil。
func ParseAllowedTools(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}
