package runtime

import (
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/tool"
)

// ChatCoreTools is the schema sent on every personal chat. Everything else
// is a name in .yoyo/tools/INDEX.md until tool_search, and joins the tools
// array only at the next checkpoint. Harbor keeps its own Advertised list.
var ChatCoreTools = []string{
	"read_file", "write_file", "str_replace",
	"list_dir", "glob", "grep", "shell",
	"git_status", "git_diff",
	"task",
	"web_fetch", "web_search",
	"run_skill_script", "read_skill_file",
}

// ChatBrowserCore is the browser surface that stays in the first pack.
// The other browser_* tools are steps inside browser_act, or they come
// back through tool_search after load_skill browser-use.
var ChatBrowserCore = []string{
	"browser_open", "browser_snapshot", "browser_act",
}

// ChatDeferredTools stay out of the first schema pack. Calling them still
// works after tool_search; the schema is not in the opening tools array.
var ChatDeferredTools = []string{
	"apply_patch", "git_commit", "wait", "view_image", "reach_status",
	"office_create", "office_edit", "office_query", "office_render",
	"browser_click", "browser_type", "browser_fill", "browser_scroll",
	"browser_hover", "browser_select", "browser_press", "browser_download",
	"browser_screenshot", "browser_verify", "browser_takeover", "browser_tabs",
	"browser_profile", "browser_record", "browser_cookies",
	"connector_read", "connector_draft", "connector_send",
	"computer_act",
	"delegate_work", "complete_work", "answer_work", "create_goal",
	"watch_page", "present_choices", "personal_status", "remember_fact",
	"read_page", "list_pages", "review_page",
}

// ApplyChatToolMenu projects the harness catalog onto the opening schema.
// Deferred families are never pulled back in, even if a snapshot lists them.
// alwaysAdvertise tools stay visible via AllToolJSON even if omitted here.
func ApplyChatToolMenu(have []string) []string {
	core := map[string]bool{}
	for _, n := range ChatCoreTools {
		core[n] = true
	}
	for _, n := range ChatBrowserCore {
		core[n] = true
	}
	deferred := map[string]bool{}
	for _, n := range ChatDeferredTools {
		deferred[n] = true
	}
	seen := map[string]bool{}
	var out []string
	add := func(n string) {
		if n == "" || seen[n] || deferred[n] {
			return
		}
		seen[n] = true
		out = append(out, n)
	}
	for _, n := range have {
		if core[n] || alwaysAdvertise(n) {
			add(n)
		}
	}
	for _, n := range ChatCoreTools {
		add(n)
	}
	for _, n := range ChatBrowserCore {
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
