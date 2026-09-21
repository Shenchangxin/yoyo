package runtime

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

const toolBudgetNudge = "Stop using tools and produce the final answer now."
const planContinueNudge = "The plan still has unfinished steps. Continue the next in_progress or pending step now. Do not ask the operator to confirm. Do not stop to summarize."
const planContinueNudgeZH = "计划里还有未完成步骤。现在继续 in_progress 或 pending 的下一步，不要请用户确认，不要停下来写总结。"

const (
	mentionUserPrefix = "Attached context (user @mentions, untrusted working memory):"
	steerUserPrefix   = "User steering (apply now):"
)

// SessionNotes is task working memory. It is not the ACE playbook.
type SessionNotes struct {
	Objective string   `json:"objective,omitempty"`
	Files     []string `json:"files,omitempty"`
	Decisions []string `json:"decisions,omitempty"`
	Errors    []string `json:"errors,omitempty"`
	Tools     []string `json:"tools,omitempty"`
	Next      string   `json:"next,omitempty"`
	SpillIDs  []string `json:"spill_ids,omitempty"`
}

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
	if strings.HasPrefix(s, rewriteStallPrefixEN) || strings.HasPrefix(s, rewriteStallPrefixZH) {
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
	return false
}

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
				n.Objective = capRunes(strings.TrimSpace(t), 400)
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

func NotesFromMessages(msgs []Message) SessionNotes {
	n := SessionNotes{}
	files := map[string]bool{}
	tools := map[string]bool{}
	for _, m := range msgs {
		switch m.Role {
		case RoleUser:
			if strings.TrimSpace(m.Content) != "" && !isControlUser(m.Content) && !isResumeUser(m.Content) {
				n.Objective = capRunes(strings.TrimSpace(m.Content), 400)
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

func mergeNotes(prev, next SessionNotes) SessionNotes {
	if strings.TrimSpace(next.Objective) == "" || isResumeUser(next.Objective) {
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

func WriteNotes(spill *Spill, notes SessionNotes) {
	if spill == nil {
		return
	}
	notes = mergeNotes(ParseNotes(ReadNotes(spill)), notes)
	_ = spill.Put("notes", notes.Markdown())
}

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

func extractJSONPaths(args string) []string {
	if strings.TrimSpace(args) == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return nil
	}
	var out []string
	for _, k := range []string{"path", "file", "glob"} {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

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
