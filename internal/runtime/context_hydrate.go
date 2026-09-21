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

const hydratePrefix = "Hydrated working set (untrusted; re-read from disk after checkpoint):"

const hydrateFileTokenCap = 2_000

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
	return capRunes(s, 1600)
}

func unreadCount(s *inbox.Store) int {
	if s == nil {
		return 0
	}
	return s.Unread()
}

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
