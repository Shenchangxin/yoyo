// Package runtime 提供代理用的工作区工具（WorkspaceTools）、上下文整形和检查点、模型客户端，以及提示拼装。
// 这里没有回合循环：桌面聊天、会话 overlay、编排工单、task 子代理和评测 Harbor 都由 internal/agentloop 的代理驱动，桌面和 Harbor 经 internal/agentloop/desk。
package runtime

import (
	"strings"
)

// isReadonlyTool 按内置名单判断只读。命中的名字可以并行，也可以在流式块结束时预取。名单之外再看 Extra 和 HostAnn。
func isReadonlyTool(name string) bool {
	switch name {
	case "read_file", "list_dir", "glob", "grep", "load_skill",
		"git_status", "git_diff", "recall_context", "tool_search",
		"list_skills", "view_image", "update_plan", "wait", "ask_user",
		"office_query", "office_render", "memory_search", "schedule_list",
		"browser_snapshot", "clipboard_read", "project_list", "connector_read",
		"read_thread", "read_skill_file", "personal_status", "present_choices",
		"read_page", "list_pages":
		return true
	default:
		return false
	}
}

// loadedFrom 返回已加载技能正文，并在有计划或验证提示时附上对应覆盖层。t 为 nil 时返回 nil。
func loadedFrom(t *WorkspaceTools) []string {
	if t == nil {
		return nil
	}
	out := t.loadedBodies()
	if body := planFirstOverlay(t); body != "" {
		out = append(out, body)
	}
	if body := verifyArtifactOverlay(t); body != "" {
		out = append(out, body)
	}
	return out
}

// planTextOf 返回当前计划文本。t 为 nil 时返回空串。
func planTextOf(t *WorkspaceTools) string {
	if t == nil {
		return ""
	}
	t.lock().Lock()
	defer t.lock().Unlock()
	return t.PlanText
}

// readonlyCall 在内置只读名单、Extra.ReadOnly 或 HostAnn.ReadOnly 任一为真时返回 true。web_fetch 和 web_search 不在此列。
func readonlyCall(tools *WorkspaceTools, name string) bool {
	if isReadonlyTool(name) {
		return true
	}
	if tools != nil && tools.Extra != nil {
		if extra, ok := tools.Extra[name]; ok {
			return extra.ReadOnly
		}
	}
	ann := HostAnn(name)
	return ann.ReadOnly
}

// readOnlySubagentCall 给子代理筛工具。readonlyCall 传入 nil，所以只看内置名单和 HostAnn。此外 web_fetch、web_search、reach_status 也返回 true。这三名只在给子代理筛工具时算只读；执行时 desk 仍把 web_fetch 和 web_search 当独占工具（见 desk.exclusiveName）。
func readOnlySubagentCall(name string) bool {
	if readonlyCall(nil, name) {
		return true
	}
	switch name {
	case "web_fetch", "web_search", "reach_status":
		return true
	default:
		return false
	}
}

// PlanOpen 在计划里至少有一条清单、且还有未完成步骤时返回 true。
func PlanOpen(plan string) bool {
	total, open := planSteps(plan)
	return total > 0 && open > 0
}

// planSteps 把计划里的清单行数成总数和未完成数。认得出的完成状态不计入 open。
func planSteps(plan string) (total, open int) {
	for _, line := range strings.Split(plan, "\n") {
		st, ok := planLineStatus(line)
		if !ok {
			continue
		}
		total++
		if !planStatusDone(st) {
			open++
		}
	}
	return total, open
}

// planLineStatus 从清单行取出方括号里的状态。序号点号必须出现在前四个字节内，点号后要有 [status]。对不上时返回空串和 false。
func planLineStatus(line string) (string, bool) {
	line = strings.TrimSpace(line)
	dot := strings.IndexByte(line, '.')
	if dot < 0 || dot > 3 {
		return "", false
	}
	rest := strings.TrimSpace(line[dot+1:])
	if !strings.HasPrefix(rest, "[") {
		return "", false
	}
	end := strings.IndexByte(rest, ']')
	if end < 1 {
		return "", false
	}
	return rest[1:end], true
}

// planStatusDone 把状态收成小写并把空格、连字符换成下划线。complete、completed、done、finished 返回 true，其余返回 false。
func planStatusDone(raw string) bool {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	switch s {
	case "complete", "completed", "done", "finished":
		return true
	default:
		return false
	}
}

// stripSystem 丢掉 system、developer 和 memory 角色，其余原样保留。
func stripSystem(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleSystem || m.Role == RoleDeveloper || m.Role == RoleMemory {
			continue
		}
		out = append(out, m)
	}
	return out
}
