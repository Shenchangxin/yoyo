// 聊天车道首包工具名单：核心工具、浏览器核心，以及推迟到 tool_search 的名字。
// 不执行工具。ApplyChatToolMenu 决定开场 schema。Harbor 仍用自己的 Advertised 列表。

package runtime

import (
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/tool"
)

// ChatCoreTools 是每次个人聊天都发送的模式。其余名字留在 .yoyo/tools/INDEX.md，直到 tool_search，并只在下一次检查点进入 tools 数组。Harbor 保留自己的 Advertised 列表。
var ChatCoreTools = []string{
	"read_file", "write_file", "str_replace",
	"list_dir", "glob", "grep", "shell",
	"git_status", "git_diff",
	"task",
	"web_fetch", "web_search",
	"run_skill_script", "read_skill_file",
}

// ChatBrowserCore 是留在第一包里的浏览器面。其它 browser_* 工具是 browser_act 里的步骤，或在 load_skill browser-use 之后经 tool_search 回来。
var ChatBrowserCore = []string{
	"browser_open", "browser_snapshot", "browser_act",
}

// ChatDeferredTools 不进第一包模式。tool_search 之后仍可调用；开场的 tools 数组里没有它们的模式。
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

// ApplyChatToolMenu 把驾驭目录投影到开场模式。推迟的家族即使快照里有也不拉回来。alwaysAdvertise 的工具即使这里省略，仍经 AllToolJSON 保持可见。
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

// deferredHostTools 返回宿主规格里尚未广告、且不是 alwaysAdvertise 的名字，按字母排序。
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

// hostToolKnown 判断名字是不是宿主工具规格。
func hostToolKnown(name string) bool {
	for _, s := range tool.HostSpecs() {
		if s.Name == name {
			return true
		}
	}
	return false
}

// unlockAdvertised 把名字并进 Advertised，已有的不重复。
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

// searchHostSpecs 在宿主工具的名字和描述里做不区分大小写的子串搜索。查询为空返回 nil。
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

// hostSpecJSON 返回同名宿主工具的 JSON 规格。没有则返回 false。
func hostSpecJSON(name string) (ToolJSON, bool) {
	for _, s := range tool.HostSpecs() {
		if s.Name == name {
			return specJSON(s), true
		}
	}
	return ToolJSON{}, false
}
