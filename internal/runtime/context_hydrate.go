// 检查点之后的工作集补水、今日一行，以及把过期的 read_file 结果标成占位。
// 补水由 forceCheckpoint 调用。今日一行进入动态尾，不进稳定前缀。终端全文由循环写到工作区 terminals。

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

// hydratePrefix 是检查点后补进用户消息的工作集标记。isControlUser 认得这个前缀。
const hydratePrefix = "Hydrated working set (untrusted; re-read from disk after checkpoint):"

// hydrateFileTokenCap 是单个文件内联进补水消息的 token 上限。超过则只写路径。
const hydrateFileTokenCap = 2_000

// hydrateAfterCheckpoint 把最近写过的文件补进一条用户消息。优先取最近 5 个写路径，否则用笔记里的文件，最多计 5 个。
// 缺失或超过 hydrateFileTokenCap 的只记一行并继续。成功内联且计数达到 5 时停止。一条都没有且目标为空时返回空消息。
// workspace 为空时跳过该路径，否则与相对路径拼接后读取。
// msgs 传给 hotWritePaths。
// notes 在写路径为空时用 Files，最多留 5 个；Objective 去掉空白后为空且一条都没记时返回空消息。
// tools 传给 planTextOf。
func hydrateAfterCheckpoint(workspace string, msgs []Message, notes SessionNotes, tools *WorkspaceTools) (extra Message, n int) {
	var b strings.Builder
	b.WriteString(hydratePrefix)
	b.WriteByte('\n')
	if plan := planTextOf(tools); strings.TrimSpace(plan) != "" {
		b.WriteString("Plan is in working memory. Do not invent a new plan.\n")
	}
	paths := hotWritePaths(msgs, 5)
	if len(paths) == 0 {
		paths = notes.Files
		if len(paths) > 5 {
			paths = paths[:5]
		}
	}
	for _, rel := range paths {
		rel = strings.TrimSpace(rel)
		if rel == "" || workspace == "" {
			continue
		}
		p := rel
		if !filepath.IsAbs(p) {
			p = filepath.Join(workspace, rel)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(&b, "Referenced file path=%s (missing on disk)\n", rel)
			n++
			continue
		}
		if estimateTokens(string(raw)) > hydrateFileTokenCap {
			fmt.Fprintf(&b, "Referenced file path=%s bytes=%d — read_file that path; do not rewrite from memory\n", rel, len(raw))
			n++
			continue
		}
		fmt.Fprintf(&b, "Referenced file path=%s\n%s\n", rel, string(raw))
		n++
		if n >= 5 {
			break
		}
	}
	if n == 0 && strings.TrimSpace(notes.Objective) == "" {
		return Message{}, 0
	}
	return Message{Role: RoleUser, Content: b.String()}, n
}

// hotWritePaths 从新到旧收集 write_file、str_replace、apply_patch 的路径，最多 limit 个。limit 小于等于 0 时按 5。
// msgs 从末尾往前找助手消息里的 write_file、str_replace 和 apply_patch。
func hotWritePaths(msgs []Message, limit int) []string {
	if limit <= 0 {
		limit = 5
	}
	seen := map[string]bool{}
	var out []string
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role != RoleAssistant {
			continue
		}
		for j := len(m.ToolCalls) - 1; j >= 0; j-- {
			tc := m.ToolCalls[j]
			if tc.Name != "write_file" && tc.Name != "str_replace" && tc.Name != "apply_patch" {
				continue
			}
			for _, p := range extractJSONPaths(tc.Arguments) {
				if seen[p] {
					continue
				}
				seen[p] = true
				out = append(out, p)
				if len(out) >= limit {
					return out
				}
			}
		}
	}
	return out
}

// AssembleToday 拼一行今日状态：未读收件、下一个启用的日程、列表里的第一个项目。结果截到 400 个 rune。没有内容时返回空串。
// t 为 nil 时返回空串，否则读取 Inbox、Schedule 和 Projects。
func AssembleToday(t *WorkspaceTools) string {
	if t == nil {
		return ""
	}
	var b strings.Builder
	if t.Inbox != nil {
		n := unreadCount(t.Inbox)
		fmt.Fprintf(&b, "inbox_unread=%d (inbox tool for bodies)\n", n)
	}
	if t.Schedule != nil {
		if next := nextSchedule(t.Schedule); next != "" {
			fmt.Fprintf(&b, "next_schedule=%s\n", next)
		}
	}
	if t.Projects != nil {
		if name := firstProject(t.Projects); name != "" {
			fmt.Fprintf(&b, "active_project=%s\n", name)
		}
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		return ""
	}
	return capRunes(s, 400)
}

// unreadCount 返回收件箱未读数。存储为 nil 时返回 0。
// s 为 nil 时返回 0，否则调用 Unread。
func unreadCount(s *inbox.Store) int {
	if s == nil {
		return 0
	}
	return s.Unread()
}

// nextSchedule 返回第一个启用日程的标签，截到 80 个 rune。没有启用项时返回空串。
// s 为 nil 时返回空串，否则从 List 里取第一个 Enabled 的日程。
func nextSchedule(s *schedule.Service) string {
	if s == nil {
		return ""
	}
	jobs := s.List()
	for _, j := range jobs {
		if !j.Enabled {
			continue
		}
		label := j.ID
		if j.Kind != "" {
			label = string(j.Kind) + ":" + j.ID
		}
		return capRunes(label, 80)
	}
	return ""
}

// firstProject 返回项目列表第一项的名字，空名字时用 id，截到 80 个 rune。
func firstProject(s *project.Store) string {
	if s == nil {
		return ""
	}
	all := s.List()
	if len(all) == 0 {
		return ""
	}
	name := all[0].Name
	if name == "" {
		name = all[0].ID
	}
	return capRunes(name, 80)
}

// writeTerminalFile 把终端输出写到工作区 .yoyo/context/<sessionID>/terminals/<callID>.txt。任一关键参数为空则不动。
// workspace 为空时返回，否则传给 WorkspaceContextDir。
// content 为空时返回，否则作为文件正文。
func writeTerminalFile(workspace, sessionID, callID, content string) {
	if workspace == "" || sessionID == "" || content == "" {
		return
	}
	dir := filepath.Join(WorkspaceContextDir(workspace, sessionID), "terminals")
	_ = os.MkdirAll(dir, 0o755)
	id := sanitizeID(callID)
	if id == "" {
		id = "term"
	}
	_ = os.WriteFile(filepath.Join(dir, id+".txt"), []byte(content), 0o644)
}

// markStaleReads 把同一路径上已被后一次 read_file 或后一次写覆盖的旧 read_file 结果换成占位。就地改传入切片，检查点只对副本调用。
// msgs 先扫出每个路径最后一次读和写的下标，再就地改掉过期的 read_file 正文。
func markStaleReads(msgs []Message) int {
	lastRead := map[string]int{}
	lastWrite := map[string]int{}
	for i, m := range msgs {
		if m.Role == RoleAssistant {
			for _, tc := range m.ToolCalls {
				if !heavyCallName(tc.Name) && tc.Name != "write_file" {
					continue
				}
				for _, p := range extractJSONPaths(tc.Arguments) {
					lastWrite[p] = i
				}
			}
		}
		if m.Role == RoleTool && m.Name == "read_file" {
			if p := toolResultPath(msgs, i); p != "" {
				lastRead[p] = i
			}
		}
	}
	n := 0
	for i := range msgs {
		if msgs[i].Role != RoleTool || msgs[i].Name != "read_file" || alreadyStubbed(msgs[i].Content) {
			continue
		}
		p := toolResultPath(msgs, i)
		if p == "" {
			continue
		}
		stale := lastRead[p] != i
		if w, ok := lastWrite[p]; ok && w > i {
			stale = true
		}
		if !stale {
			continue
		}
		id := msgs[i].ToolCallID
		msgs[i].Content = stubTool(id, msgs[i].Name, len(msgs[i].Content)) + "\n[stale path=" + p + " — later read or write is canonical]"
		n++
	}
	return n
}
