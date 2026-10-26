// 跨会话读取、列出和发送。@session 注入与 read_thread 用同一套片段上限，默认预算 1600 字节。不把整段轨迹倒进提示。

package runtime

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/memory"
)

// ThreadRef 是另一场会话的 ID 和标题。
type ThreadRef struct {
	ID    string
	Title string
}

// readThread 在 sessionID 为空时列出会话。message 非空时发送。不能读当前会话。片段最多 8 条、每条 400 个 rune、合计 2000 个 rune。
// query 在 sessionID 去掉空白后为空时交给 listThreads；否则作为 formatThreadSnippets 的过滤词。
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

// formatThreadSnippets 只保留 user 和 assistant，并跳过 isControlUser。query 非空时按小写子串过滤。达到 maxHits 停止，再用 capRunes 截到 total。
// msgs 只保留 user 和 assistant，并跳过 isControlUser。
// perMsg 作为 capRunes 的上限截每条正文。
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

// FormatSessionMentions 为 @session 注入有界摘录，策略与 read_thread 相同，不倒出整段轨迹。budget 小于等于 0 时用 1600 字节。剩余不足 64 时停止。已写入后再超出则写下 session mentions truncated。
// refs 只处理 Kind 为 session 且 Ref 非空的项。
// currentID 交给 sessionMentionChunk。
// tools 交给 sessionMentionChunk。
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

// sessionMentionChunk 能从 ListThreads 取标题。ID 等于当前会话时只说明这是当前对话，不附消息。没有命中时提示改用 read_thread。有命中时用 formatThreadSnippets，每条 400 个 rune，条数 8，总长是 remain 减 80。
// id 去掉空白后用来对 ListThreads 的 ID，并和 currentID 比较。
// currentID 与去掉空白后的 id 相等时只说明这是当前对话。
// tools 非 nil 时用 ListThreads 取标题，用 Sessions 取消息。
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

// listThreads 在 ListThreads 为空时要求 session_id。跳过空 ID、当前会话和子会话。最多 24 条。没有其他会话时返回 no other sessions。
// query 去掉空白并转成小写；非空时用来过滤会话 ID 和标题。
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

// sendThread 在 SendThread 为空时返回错误。不能发给当前会话。成功则说明已排队或并入该会话。
// id 等于当前 SessionID 时返回错误，否则作为第一参交给 SendThread。
// text 作为第二参交给 SendThread。
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

// consolidateMemory 只在 SoftHorizon、工具和 Memory 都存在且 Objective 非空时写入。正文截到 800 个 rune。若已有相同文本或以前缀开头的情景记忆则不写。
// req 的 SoftHorizon 为假，或 Tools、Memory 为空时直接返回；否则用 Memory 按目标搜索并写入。
// notes 的 Objective 去掉空白后为空则返回；否则和 Files 拼成正文。
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
