package runtime

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/memory"
)

func (t *WorkspaceTools) readThread(sessionID, query string) ToolResult {
	if t == nil || t.Sessions == nil {
		return ToolResult{Err: fmt.Errorf("read_thread is not available")}
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ToolResult{Err: fmt.Errorf("session_id required")}
	}
	if sessionID == t.SessionID {
		return ToolResult{Err: fmt.Errorf("read_thread cannot target the current session")}
	}
	msgs := t.Sessions(sessionID)
	if len(msgs) == 0 {
		return ToolResult{Content: "no messages in that session"}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var b strings.Builder
	hits := 0
	for _, m := range msgs {
		if m.Role != RoleUser && m.Role != RoleAssistant {
			continue
		}
		if isControlUser(m.Content) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(m.Content), q) {
			continue
		}
		fmt.Fprintf(&b, "## %s\n%s\n\n", m.Role, capRunes(m.Content, 400))
		hits++
		if hits >= 8 {
			break
		}
	}
	if hits == 0 {
		return ToolResult{Content: "no matching snippets"}
	}
	return ToolResult{Content: capRunes(b.String(), 2000)}
}

func consolidateMemory(req RunRequest, notes SessionNotes) {
	if !req.SoftHorizon || req.Tools == nil || req.Tools.Memory == nil {
		return
	}
	obj := strings.TrimSpace(notes.Objective)
	if obj == "" {
		return
	}
	var b strings.Builder
	b.WriteString(obj)
	if len(notes.Files) > 0 {
		b.WriteString("\nfiles: ")
		b.WriteString(strings.Join(notes.Files, ", "))
	}
	text := capRunes(b.String(), 800)
	for _, it := range req.Tools.Memory.Search(obj, memory.KindEpisodic, 4) {
		if it.Text == text || strings.HasPrefix(it.Text, obj) {
			return
		}
	}
	req.Tools.Memory.Write(memory.Item{Kind: memory.KindEpisodic, Text: text})
}
