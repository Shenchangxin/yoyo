// 会话工作笔记和用户原文的整理。笔记记目标、文件、决定和错误，不是 ACE playbook。
// 检查点、计量和 seed 会读它。控制行和「继续」类短句不改已经记下的目标。

package runtime

import (
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// 运行时插进用户角色的催促、计划和附件标记。isControlUser 把它们排除出笔记目标。
const toolBudgetNudge = "Stop using tools and produce the final answer now."
const planContinueNudge = "The plan still has unfinished steps. Continue the next in_progress or pending step now. Do not ask the operator to confirm. Do not stop to summarize."
const planContinueNudgeZH = "计划里还有未完成步骤。现在继续 in_progress 或 pending 的下一步，不要请用户确认，不要停下来写总结。"

const (
	mentionUserPrefix = "Attached context (user @mentions, untrusted working memory):"
	steerUserPrefix   = "User steering (apply now):"
)

// AttachMention 把操作者原话放在前面，注入的提及或技能正文跟在后面。注入为空时原样返回 user。
func AttachMention(user, inject string) string {
	user = strings.TrimSpace(user)
	inject = strings.TrimSpace(inject)
	if inject == "" {
		return user
	}
	block := mentionUserPrefix + "\n" + inject
	if user == "" {
		return block
	}
	return user + "\n\n" + block
}

// SessionNotes 是这一轮任务的工作记忆，不是 ACE playbook。
type SessionNotes struct {
	Objective string   `json:"objective,omitempty"`
	Files     []string `json:"files,omitempty"`
	Decisions []string `json:"decisions,omitempty"`
	Errors    []string `json:"errors,omitempty"`
	Tools     []string `json:"tools,omitempty"`
	Next      string   `json:"next,omitempty"`
	// SpillIDs 是被换成占位的工具结果 id。
	SpillIDs []string `json:"spill_ids,omitempty"`
}

// Markdown 按固定小节写出笔记。空的 Next 和空的 spill 列表不写对应小节。
func (n SessionNotes) Markdown() string {
	var b strings.Builder
	b.WriteString("## Objective\n")
	if n.Objective != "" {
		b.WriteString(n.Objective)
		b.WriteByte('\n')
	}
	b.WriteString("\n## Files\n")
	for _, f := range n.Files {
		b.WriteString("- ")
		b.WriteString(f)
		b.WriteByte('\n')
	}
	b.WriteString("\n## Decisions\n")
	for _, d := range n.Decisions {
		b.WriteString("- ")
		b.WriteString(d)
		b.WriteByte('\n')
	}
	b.WriteString("\n## Errors\n")
	for _, e := range n.Errors {
		b.WriteString("- ")
		b.WriteString(e)
		b.WriteByte('\n')
	}
	b.WriteString("\n## Tools\n")
	for _, t := range n.Tools {
		b.WriteString("- ")
		b.WriteString(t)
		b.WriteByte('\n')
	}
	if len(n.SpillIDs) > 0 {
		b.WriteString("\n## Spill ids\n")
		for _, id := range n.SpillIDs {
			b.WriteString("- ")
			b.WriteString(id)
			b.WriteByte('\n')
		}
	}
	if n.Next != "" {
		b.WriteString("\n## Next\n")
		b.WriteString(n.Next)
		b.WriteByte('\n')
	}
	return b.String()
}

// isResumeUser 识别「继续」一类短句。会先去掉首尾空白和句号、叹号、省略号。
func isResumeUser(text string) bool {
	s := strings.TrimSpace(text)
	s = strings.Trim(s, "。.!！…")
	switch strings.ToLower(s) {
	case "继续", "接着", "接着做", "请继续", "往下", "继续吧", "继续干",
		"continue", "go on", "keep going", "resume", "go ahead":
		return true
	default:
		return false
	}
}

// isControlUser 把运行时催促、检查点、占位、提及和 steer 当成控制行。空文本也算。这些行不进入笔记目标。
func isControlUser(text string) bool {
	s := strings.TrimSpace(text)
	if s == "" {
		return true
	}
	if s == toolBudgetNudge || strings.HasPrefix(s, toolBudgetNudge) {
		return true
	}
	if s == planContinueNudge || strings.HasPrefix(s, planContinueNudge) {
		return true
	}
	if s == planContinueNudgeZH || strings.HasPrefix(s, planContinueNudgeZH) {
		return true
	}
	if s == askContinueNudge || strings.HasPrefix(s, askContinueNudge) {
		return true
	}
	if s == askContinueNudgeZH || strings.HasPrefix(s, askContinueNudgeZH) {
		return true
	}
	if strings.HasPrefix(s, rewriteStallPrefixEN) || strings.HasPrefix(s, rewriteStallPrefixZH) {
		return true
	}
	if strings.HasPrefix(s, errorRepeatPrefixEN) || strings.HasPrefix(s, errorRepeatPrefixZH) {
		return true
	}
	if strings.HasPrefix(s, waitLoopPrefixEN) || strings.HasPrefix(s, waitLoopPrefixZH) {
		return true
	}
	if strings.HasPrefix(s, checkFailPrefixEN) || strings.HasPrefix(s, checkFailPrefixZH) {
		return true
	}
	if s == repeatGentleEN || s == repeatGentleZH {
		return true
	}
	if strings.HasPrefix(s, repeatDetailPrefixEN) || strings.HasPrefix(s, repeatDetailPrefixZH) {
		return true
	}
	if strings.HasPrefix(s, promptUpdateLead) {
		return true
	}
	if strings.HasPrefix(s, instrumentPrefixEN) || strings.HasPrefix(s, instrumentPrefixZH) {
		return true
	}
	if strings.Contains(s, "Context checkpoint") {
		return true
	}
	if strings.HasPrefix(s, "[elided ") {
		return true
	}
	if strings.HasPrefix(s, mentionUserPrefix) {
		return true
	}
	if strings.HasPrefix(s, steerUserPrefix) {
		return true
	}
	if strings.HasPrefix(s, hydratePrefix) {
		return true
	}
	return false
}

// ExtractNotes 从轨迹事件收集笔记。steer 来源的用户事件跳过。目标、文件、工具、错误和 spill id 各有条数上限。
func ExtractNotes(evs []trace.Event) SessionNotes {
	n := SessionNotes{}
	files := map[string]bool{}
	tools := map[string]bool{}
	spills := map[string]bool{}
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeUser:
			if ev.Source == "steer" {
				continue
			}
			if t, _ := ev.Payload["text"].(string); t != "" && !isControlUser(t) && !isResumeUser(t) {
				stickObjective(&n, t)
			}
		case trace.TypeToolCall:
			name, _ := ev.Payload["name"].(string)
			if name != "" {
				tools[name] = true
			}
			args, _ := ev.Payload["arguments"].(string)
			for _, p := range extractJSONPaths(args) {
				files[p] = true
			}
		case trace.TypeToolResult:
			content, _ := ev.Payload["content"].(string)
			id, _ := ev.Payload["id"].(string)
			if id != "" && strings.Contains(content, "[elided ") {
				spills[id] = true
			}
			if strings.HasPrefix(content, "ERROR:") {
				n.Errors = appendUnique(n.Errors, capRunes(content, 240))
			}
		}
	}
	n.Files = keys(files)
	n.Tools = keys(tools)
	n.SpillIDs = keys(spills)
	if len(n.Files) > 24 {
		n.Files = n.Files[:24]
	}
	if len(n.Tools) > 24 {
		n.Tools = n.Tools[:24]
	}
	if len(n.Errors) > 12 {
		n.Errors = n.Errors[:12]
	}
	if len(n.SpillIDs) > 32 {
		n.SpillIDs = n.SpillIDs[:32]
	}
	return n
}

// NotesFromMessages 从消息收集笔记。控制行和继续短句不改目标。文件和工具的条数上限在 mergeNotes。
func NotesFromMessages(msgs []Message) SessionNotes {
	n := SessionNotes{}
	files := map[string]bool{}
	tools := map[string]bool{}
	for _, m := range msgs {
		switch m.Role {
		case RoleUser:
			if strings.TrimSpace(m.Content) != "" && !isControlUser(m.Content) && !isResumeUser(m.Content) {
				stickObjective(&n, m.Content)
			}
		case RoleAssistant:
			for _, tc := range m.ToolCalls {
				if tc.Name != "" {
					tools[tc.Name] = true
				}
				for _, p := range extractJSONPaths(tc.Arguments) {
					files[p] = true
				}
			}
		case RoleTool:
			if m.Name != "" {
				tools[m.Name] = true
			}
			if strings.HasPrefix(m.Content, "ERROR:") {
				n.Errors = appendUnique(n.Errors, capRunes(m.Content, 240))
			}
			if alreadyStubbed(m.Content) && m.ToolCallID != "" {
				n.SpillIDs = appendUnique(n.SpillIDs, m.ToolCallID)
			}
		}
	}
	n.Files = keys(files)
	n.Tools = keys(tools)
	return n
}

// stickObjective 把第一句非空用户话记成目标，截到 400 个 rune。之后不同的句子进入决定，只留最近 12 条。提及前缀之后的正文不计入。
func stickObjective(n *SessionNotes, text string) {
	if i := strings.Index(text, mentionUserPrefix); i >= 0 {
		text = text[:i]
	}
	t := capRunes(collapseDoubledText(strings.TrimSpace(text)), 400)
	if t == "" {
		return
	}
	if n.Objective == "" {
		n.Objective = t
		return
	}
	if t != n.Objective {
		n.Decisions = appendUnique(n.Decisions, capRunes(t, 200))
		if len(n.Decisions) > 12 {
			n.Decisions = n.Decisions[len(n.Decisions)-12:]
		}
	}
}

// mergeNotes 保留已有目标。新目标不同且不是继续短句时，收进决定。文件、工具、错误、spill 和决定按给定上限合并。新的 Next 为空时沿用旧的。
func mergeNotes(prev, next SessionNotes) SessionNotes {
	if prev.Objective != "" {
		if next.Objective != "" && next.Objective != prev.Objective && !isResumeUser(next.Objective) {
			next.Decisions = appendUnique(next.Decisions, capRunes(next.Objective, 200))
		}
		next.Objective = prev.Objective
	} else if strings.TrimSpace(next.Objective) == "" || isResumeUser(next.Objective) {
		next.Objective = prev.Objective
	}
	next.Files = unionCap(prev.Files, next.Files, 24)
	next.Tools = unionCap(prev.Tools, next.Tools, 24)
	next.Errors = unionCap(prev.Errors, next.Errors, 12)
	next.SpillIDs = unionCap(prev.SpillIDs, next.SpillIDs, 32)
	if strings.TrimSpace(next.Next) == "" {
		next.Next = prev.Next
	}
	next.Decisions = unionCap(prev.Decisions, next.Decisions, 12)
	return next
}

// ParseNotes 按 Markdown 小节解析笔记。目标是该小节的非空行直接拼接。
func ParseNotes(md string) SessionNotes {
	n := SessionNotes{}
	section := ""
	var obj []string
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "## ") {
			section = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		switch section {
		case "objective":
			if s := strings.TrimSpace(line); s != "" {
				obj = append(obj, s)
			}
		case "files":
			if strings.HasPrefix(strings.TrimSpace(line), "- ") && item != "" {
				n.Files = appendUnique(n.Files, item)
			}
		case "decisions":
			if strings.HasPrefix(strings.TrimSpace(line), "- ") && item != "" {
				n.Decisions = appendUnique(n.Decisions, item)
			}
		case "errors":
			if strings.HasPrefix(strings.TrimSpace(line), "- ") && item != "" {
				n.Errors = appendUnique(n.Errors, item)
			}
		case "tools":
			if strings.HasPrefix(strings.TrimSpace(line), "- ") && item != "" {
				n.Tools = appendUnique(n.Tools, item)
			}
		case "spill ids":
			if strings.HasPrefix(strings.TrimSpace(line), "- ") && item != "" {
				n.SpillIDs = appendUnique(n.SpillIDs, item)
			}
		case "next":
			if s := strings.TrimSpace(line); s != "" {
				if n.Next != "" {
					n.Next += "\n"
				}
				n.Next += s
			}
		}
	}
	n.Objective = strings.Join(obj, "\n")
	return n
}

// WriteNotes 先和 spill 里已有的笔记合并，再以 id notes 写回。spill 为 nil 时不动。
func WriteNotes(spill *Spill, notes SessionNotes) {
	if spill == nil {
		return
	}
	notes = mergeNotes(ParseNotes(ReadNotes(spill)), notes)
	_ = spill.Put("notes", notes.Markdown())
}

// ReadNotes 读取 id 为 notes 的正文。spill 为 nil 或读失败时返回空串。
func ReadNotes(spill *Spill) string {
	if spill == nil {
		return ""
	}
	s, err := spill.Get("notes")
	if err != nil {
		return ""
	}
	return s
}

// extractJSONPaths 从工具参数里取 path、file 和 glob 三个字符串字段里能当笔记路径的值。
func extractJSONPaths(args string) []string {
	if strings.TrimSpace(args) == "" {
		return nil
	}
	m := parseToolArgs(args)
	if len(m) == 0 {
		return nil
	}
	var out []string
	for _, k := range []string{"path", "file", "glob"} {
		if s, ok := m[k].(string); ok && keepNotePath(s) {
			out = append(out, s)
		}
	}
	return out
}

// keepNotePath 拒绝空路径、当前目录、上级目录和带通配的模式。
func keepNotePath(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" || p == "." || p == "./" || p == ".." {
		return false
	}
	if strings.ContainsAny(p, "*?[]") {
		return false
	}
	return true
}

// NormalizeUserText 是落盘和 seed 共用的用户原文：去掉首尾空白，并收起输入法或粘贴造成的整段重复。轨迹和模型看到的是同一份文本。
func NormalizeUserText(s string) string {
	return collapseDoubledText(strings.TrimSpace(s))
}

// collapseDoubledText 在长度至少 24 且前后两半完全相同时只留前一半。更短的文本原样返回，避免把 haha 收成 ha。
func collapseDoubledText(s string) string {
	s = strings.TrimSpace(s)
	rs := []rune(s)
	n := len(rs)
	if n < 24 || n%2 != 0 {
		return s
	}
	mid := n / 2
	if string(rs[:mid]) == string(rs[mid:]) {
		return string(rs[:mid])
	}
	return s
}

// keys 返回映射里的非空键，按字典序排列。
func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// appendUnique 在去掉空白后非空、且尚未出现时追加。
func appendUnique(in []string, v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return in
	}
	for _, x := range in {
		if x == v {
			return in
		}
	}
	return append(in, v)
}

// unionCap 合并两列，去重后排序，n 大于 0 时只留前 n 个。
func unionCap(a, b []string, n int) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range append(append([]string{}, a...), b...) {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	sort.Strings(out)
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}
