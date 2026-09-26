package runtime

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/memory"
)

type ThreadRef struct {
	ID    string
	Title string
}

func (t *WorkspaceTools) readThread(sessionID, query, message string) ToolResult {
	if t == nil {
		return ToolResult{Err: fmt.Errorf("read_thread is not available")}
	}
	sessionID = strings.TrimSpace(sessionID)
	message = strings.TrimSpace(message)
	if sessionID == "" {
		return t.listThreads(query)
	}
	if message != "" {
		return t.sendThread(sessionID, message)
	}
	if t.Sessions == nil {
		return ToolResult{Err: fmt.Errorf("read_thread is not available")}
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

func (t *WorkspaceTools) listThreads(query string) ToolResult {
	if t.ListThreads == nil {
		return ToolResult{Err: fmt.Errorf("session_id required")}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var b strings.Builder
	n := 0
	for _, ref := range t.ListThreads() {
		if ref.ID == "" || ref.ID == t.SessionID {
			continue
		}
		if strings.Contains(ref.ID, "/tasks/") {
			continue
		}
		blob := strings.ToLower(ref.ID + " " + ref.Title)
		if q != "" && !strings.Contains(blob, q) {
			continue
		}
		title := strings.TrimSpace(ref.Title)
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(&b, "- `%s` %s\n", ref.ID, title)
		n++
		if n >= 24 {
			break
		}
	}
	if n == 0 {
		return ToolResult{Content: "no other sessions"}
	}
	return ToolResult{Content: "sessions:\n" + b.String()}
}

func (t *WorkspaceTools) sendThread(id, text string) ToolResult {
	if t.SendThread == nil {
		return ToolResult{Err: fmt.Errorf("send to another session is not available")}
	}
	if id == t.SessionID {
		return ToolResult{Err: fmt.Errorf("read_thread cannot target the current session")}
	}
	if err := t.SendThread(id, text); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "queued or steered into " + id}
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
