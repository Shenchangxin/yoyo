// 检查点之后的工作集补水、今日一行，以及把过期的 read_file 结果标成占位。
// 补水由 forceCheckpoint 调用。今日一行进入动态尾，不进稳定前缀。终端全文由循环写到工作区 terminals。

package runtime

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

// hydratePrefix 是检查点后补进用户消息的工作集标记。isControlUser 认得这个前缀。
const hydratePrefix = "Hydrated working set (untrusted; re-read from disk after checkpoint):"

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
