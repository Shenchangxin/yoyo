package runtime

import (
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/tool"
)

// ChatCoreTools is the schema sent on personal chat (I6). HostSpecs stay in
// CAS; deferred names are recovered with tool_search. Harbor eval keeps its
// own Advertised list.
var ChatCoreTools = []string{
	"read_file", "write_file", "str_replace", "apply_patch",
	"list_dir", "glob", "grep", "shell",
	"git_status", "git_diff", "git_commit",
	"task", "wait", "view_image",
	"web_fetch", "web_search",
}

// ApplyChatToolMenu projects the harness catalog onto the coding core.
// alwaysAdvertise tools stay visible via AllToolJSON even if omitted here.
func ApplyChatToolMenu(have []string) []string {
	core := map[string]bool{}
	for _, n := range ChatCoreTools {
		core[n] = true
	}
	seen := map[string]bool{}
	var out []string
	add := func(n string) {
		if n == "" || seen[n] {
			return
		}
		seen[n] = true
		out = append(out, n)
	}
	if len(have) == 0 {
		for _, n := range ChatCoreTools {
			add(n)
		}
		return out
	}
	for _, n := range have {
		if core[n] || alwaysAdvertise(n) {
			add(n)
		}
	}
	for _, n := range ChatCoreTools {
		add(n)
	}
	return out
}

func deferredHostTools(advertised []string) []string {
	want := map[string]bool{}
	for _, n := range advertised {
		want[n] = true
	}
	var out []string
	for _, s := range tool.HostSpecs() {
		if want[s.Name] || alwaysAdvertise(s.Name) {
			continue
		}
		out = append(out, s.Name)
	}
	sort.Strings(out)
	return out
}

func hostToolKnown(name string) bool {
	for _, s := range tool.HostSpecs() {
		if s.Name == name {
			return true
		}
	}
	return false
}

func (t *WorkspaceTools) unlockAdvertised(names []string) {
	if t == nil || len(names) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	have := map[string]bool{}
	for _, n := range t.Advertised {
		have[n] = true
	}
	for _, n := range names {
		if n == "" || have[n] {
			continue
		}
		t.Advertised = append(t.Advertised, n)
		have[n] = true
	}
}

func searchHostSpecs(q string) []string {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	var out []string
	for _, s := range tool.HostSpecs() {
		blob := strings.ToLower(s.Name + " " + s.Description)
		if strings.Contains(blob, q) {
			out = append(out, s.Name)
		}
	}
	sort.Strings(out)
	return out
}

func hostSpecJSON(name string) (ToolJSON, bool) {
	for _, s := range tool.HostSpecs() {
		if s.Name == name {
			return specJSON(s), true
		}
	}
	return ToolJSON{}, false
}
