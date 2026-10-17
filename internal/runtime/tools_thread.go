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
	body, hits := formatThreadSnippets(msgs, query, 8, 400, 2000)
	if hits == 0 {
		if len(msgs) == 0 {
			return ToolResult{Content: "no messages in that session"}
		}
		return ToolResult{Content: "no matching snippets"}
	}
	return ToolResult{Content: body}
}

func formatThreadSnippets(msgs []Message, query string, maxHits, perMsg, total int) (string, int) {
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
		fmt.Fprintf(&b, "## %s\n%s\n\n", m.Role, capRunes(m.Content, perMsg))
		hits++
		if hits >= maxHits {
			break
		}
	}
	return capRunes(b.String(), total), hits
}

// FormatSessionMentions injects bounded excerpts for @session pins using the
// same snippet policy as read_thread. It does not dump a full trajectory.
func FormatSessionMentions(refs []Mention, currentID string, tools *WorkspaceTools, budget int) string {
	if budget <= 0 {
		budget = 1600
	}
	var b strings.Builder
	used := 0
	for _, ref := range refs {
		if ref.Kind != "session" || ref.Ref == "" {
			continue
		}
		remain := budget - used
		if remain < 64 {
			break
		}
		chunk := sessionMentionChunk(ref.Ref, currentID, tools, remain)
		if chunk == "" {
			continue
		}
		if used+len(chunk) > budget && used > 0 {
			b.WriteString("…[session mentions truncated]\n")
			break
		}
		b.WriteString(chunk)
		used += len(chunk)
	}
	return strings.TrimSpace(b.String())
}

func sessionMentionChunk(id, currentID string, tools *WorkspaceTools, remain int) string {
	id = strings.TrimSpace(id)
	title := id
	if tools != nil && tools.ListThreads != nil {
		for _, t := range tools.ListThreads() {
			if t.ID == id {
				if s := strings.TrimSpace(t.Title); s != "" {
					title = s
				}
				break
			}
		}
	}
	if id == currentID {
		return fmt.Sprintf("## @session:%s\nThis is the current chat (%s).\n", id, title)
	}
	var msgs []Message
	if tools != nil && tools.Sessions != nil {
		msgs = tools.Sessions(id)
	}
	body, hits := formatThreadSnippets(msgs, "", 8, 400, remain-80)
	var b strings.Builder
	fmt.Fprintf(&b, "## @session:%s\n%s\n", id, title)
	if hits == 0 {
		b.WriteString("No messages loaded. Call read_thread with this session_id for snippets.\n")
		return b.String()
	}
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("Use read_thread with this session_id for more.\n")
	return b.String()
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
		if IsChildSessionID(ref.ID) {
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
