// journal.go 拥有把 ACP 会话更新写成 dsh 日志的规则：正文、工具调用和结果、计划、fs 写入补记、用量。
// TranscriptLines 只从 EventAssistant.ToolCalls 读工具调用，所以每次调用都挂在一条 EventAssistant 上。
// 不拥有连接、回合开始和结束（worker.go），也不判断权限（perm.go）。

package acpworker

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/dsh"
)

const (
	// maxArgs 是一次调用参数落进日志的字节上限。超过时只留路径、命令这类短字段。
	maxArgs = 8000
	// maxResult 是一条工具结果的字数上限，outputTail 是命令输出留下的末尾字数。
	maxResult  = 8000
	outputTail = 2000
)

// toolCall 是一次 ACP 工具调用合并到现在的样子。id 是日志里的编号，带回合号，免得换会话后撞号。
// acpName 是对方给的名字，name 是记进日志的名字，写进日志后不再改，结果和调用才对得上。
type toolCall struct {
	id        string
	acpName   string
	name      string
	title     string
	kind      string
	status    string
	content   []acp.ToolCallContent
	locations []acp.Location
	rawInput  json.RawMessage
	rawOutput json.RawMessage
	logged    bool
	done      bool
	// denied 表示 yoyo 没批准这次调用，它带的 diff 没有写下去。
	denied bool
}

// journal 把一个工人的 ACP 更新写进它的 dsh 日志。只在 begin 和 finish 之间写；回放和回合外的更新只取用量。
type journal struct {
	mu      sync.Mutex
	sess    *dsh.Session
	dir     string
	onUsage func(acp.UsageUpdate)

	active  bool
	turn    int
	step    int
	seq     int
	text    strings.Builder
	calls   map[string]*toolCall
	order   []*toolCall
	held    []*toolCall
	written []string
}

// newJournal 建一个写进 sess 的 journal。dir 用来把相对路径补成绝对路径。
func newJournal(sess *dsh.Session, dir string, onUsage func(acp.UsageUpdate)) *journal {
	return &journal{sess: sess, dir: dir, onUsage: onUsage, calls: map[string]*toolCall{}}
}

// begin 开始记 turn 这一轮。
func (j *journal) begin(turn int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.active = true
	j.turn = turn
	j.step = 0
	j.seq = 0
	j.text.Reset()
	j.calls = map[string]*toolCall{}
	j.order = nil
	j.held = nil
	j.written = nil
}

// finish 收尾：写出还压着的调用，给结果里不会写成 wrote 的 fs 写入补一对 write_file，按出现顺序给没有结果的调用补结果，
// 最后才写还没落下的正文，它是这一轮最后说的话。normal 为假表示这一轮被取消或出错，没结果的调用记成中断。
func (j *journal) finish(normal bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.active {
		return
	}
	j.flushHeldLocked()
	tail := j.text.String()
	j.text.Reset()
	shown := j.shownLocked(normal)
	for _, path := range j.written {
		if shown[pathKey(path)] {
			continue
		}
		id := j.nextIDLocked("fs")
		args, _ := json.Marshal(map[string]string{"path": path})
		j.appendCallLocked(id, "write_file", string(args))
		j.appendResultLocked(id, "write_file", "wrote "+path, false)
	}
	for _, c := range j.order {
		if !c.logged || c.done {
			continue
		}
		c.done = true
		if normal {
			j.appendResultLocked(c.id, c.name, resultText(c, j.dir), false)
		} else {
			j.appendResultLocked(c.id, c.name, "interrupted", true)
		}
	}
	j.text.WriteString(tail)
	j.flushTextLocked()
	j.active = false
}

// noteWrite 记下这一轮经 fs/write_text_file 写过的文件。
func (j *journal) noteWrite(path string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.active {
		j.written = append(j.written, path)
	}
}

// update 处理一条会话更新。replay 为真的是 session/load 回放的历史，丢掉。
func (j *journal) update(n acp.SessionNotification, replay bool) {
	if replay {
		return
	}
	u := n.Update
	if u.Kind == acp.UpdateUsage {
		if u.Usage != nil && j.onUsage != nil {
			j.onUsage(*u.Usage)
		}
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.active {
		return
	}
	switch u.Kind {
	case acp.UpdateAgentChunk:
		if u.Chunk != nil && u.Chunk.Type == "text" && u.Chunk.Text != "" {
			j.flushHeldLocked()
			j.text.WriteString(u.Chunk.Text)
		}
	case acp.UpdateToolCall, acp.UpdateToolUpdate:
		if u.ToolCall != nil {
			j.toolLocked(*u.ToolCall)
		}
	case acp.UpdatePlan:
		j.planLocked(u.Plan)
	}
}

// merge 把权限请求里带来的调用字段并进已知的调用，返回合并后的副本。还不认识的调用先记下，等通知来了再写。
func (j *journal) merge(tc acp.ToolCall) toolCall {
	j.mu.Lock()
	defer j.mu.Unlock()
	c := j.calls[tc.ToolCallID]
	if c == nil {
		c = &toolCall{}
		if j.active && tc.ToolCallID != "" {
			c.id = j.callIDLocked(tc.ToolCallID)
			j.calls[tc.ToolCallID] = c
			j.order = append(j.order, c)
			j.held = append(j.held, c)
		}
	}
	mergeCall(c, tc)
	return *c
}

// deny 把没被批准的调用记成失败的结果。之后对方再报这次调用的结果时不再补记。
func (j *journal) deny(acpID string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	c := j.calls[acpID]
	if !j.active || c == nil || c.done {
		return
	}
	j.flushHeldLocked()
	if !c.logged {
		j.logCallLocked(c)
	}
	c.done = true
	c.denied = true
	j.appendResultLocked(c.id, c.name, "denied: yoyo did not approve this call", true)
}

// toolLocked 处理 tool_call 和 tool_call_update。新调用先压着，等第一条更新或下一件事再写，参数才齐。
func (j *journal) toolLocked(tc acp.ToolCall) {
	c := j.calls[tc.ToolCallID]
	if c == nil {
		c = &toolCall{id: j.callIDLocked(tc.ToolCallID)}
		j.calls[tc.ToolCallID] = c
		j.order = append(j.order, c)
		mergeCall(c, tc)
		j.held = append(j.held, c)
		if !finished(c.status) {
			return
		}
	} else {
		mergeCall(c, tc)
	}
	j.flushHeldLocked()
	if !c.logged {
		j.logCallLocked(c)
	}
	if finished(c.status) && !c.done {
		c.done = true
		j.appendResultLocked(c.id, c.name, resultText(c, j.dir), c.status == "failed")
	}
}

// planLocked 把计划写成一对合成的 update_plan 调用和结果，形状和 yoyo 自己的 update_plan 一样。
func (j *journal) planLocked(entries []acp.PlanEntry) {
	type step struct {
		Step   string `json:"step"`
		Status string `json:"status"`
	}
	var steps []step
	var b strings.Builder
	for _, e := range entries {
		text := strings.TrimSpace(e.Content)
		if text == "" {
			continue
		}
		status := planStatus(e.Status)
		steps = append(steps, step{Step: text, Status: status})
		fmt.Fprintf(&b, "%d. [%s] %s\n", len(steps), status, text)
	}
	if len(steps) == 0 {
		return
	}
	j.flushHeldLocked()
	args, _ := json.Marshal(map[string]any{"plan": steps})
	id := j.nextIDLocked("plan")
	j.appendCallLocked(id, "update_plan", string(args))
	j.appendResultLocked(id, "update_plan", b.String(), false)
}

// flushHeldLocked 写出压着的调用。调用时要已持有锁。
func (j *journal) flushHeldLocked() {
	held := j.held
	j.held = nil
	for _, c := range held {
		if !c.logged {
			j.logCallLocked(c)
		}
	}
}

// flushTextLocked 把攒着的正文写成一条 EventAssistant。调用时要已持有锁。
func (j *journal) flushTextLocked() {
	text := j.text.String()
	j.text.Reset()
	if strings.TrimSpace(text) == "" {
		return
	}
	j.step++
	j.sess.Append(dsh.EventAssistant, j.turn, j.step, dsh.Assistant{Text: text, Finish: "stop"})
}

// logCallLocked 把 c 写成一条带调用的 EventAssistant，攒着的正文跟它一起写。
func (j *journal) logCallLocked(c *toolCall) {
	c.logged = true
	j.appendCallLocked(c.id, c.name, toolArgs(c, j.dir))
}

// appendCallLocked 写一条带一个调用的 EventAssistant，攒着的正文放进它的 Text。
func (j *journal) appendCallLocked(id, name, args string) {
	text := j.text.String()
	j.text.Reset()
	j.step++
	j.sess.Append(dsh.EventAssistant, j.turn, j.step, dsh.Assistant{
		Text:      text,
		ToolCalls: []dsh.ToolCall{{ID: id, Name: name, Arguments: args}},
		Finish:    "tool_calls",
	})
}

// appendResultLocked 写一条工具结果。
func (j *journal) appendResultLocked(id, name, text string, isError bool) {
	j.sess.Append(dsh.EventToolResult, j.turn, j.step, dsh.ToolResultEvent{ID: id, Name: name, Text: text, IsError: isError})
}

// shownLocked 是这一轮的结果里会写成 wrote 的路径：没被拒、没失败的调用带的 diff。
// normal 为假时，还没结果的调用会记成中断，它们的 diff 不算。只列了位置的调用也不算，任务页认不出它改了哪个文件。
func (j *journal) shownLocked(normal bool) map[string]bool {
	out := map[string]bool{}
	for _, c := range j.order {
		if c.denied || c.status == "failed" || (!normal && !c.done) {
			continue
		}
		for _, p := range diffPaths(c, j.dir) {
			out[pathKey(p)] = true
		}
	}
	return out
}

// diffPaths 是 c 带的 diff 指向的绝对路径。
func diffPaths(c *toolCall, dir string) []string {
	var out []string
	for _, item := range c.content {
		if item.Type != "diff" {
			continue
		}
		if p := absPath(item.Path, dir); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// callIDLocked 是 ACP 调用在日志里的编号。
func (j *journal) callIDLocked(acpID string) string {
	if strings.TrimSpace(acpID) == "" {
		return j.nextIDLocked("call")
	}
	return fmt.Sprintf("t%d-%s", j.turn, acpID)
}

// nextIDLocked 是合成调用的编号。
func (j *journal) nextIDLocked(kind string) string {
	j.seq++
	return fmt.Sprintf("t%d-%s-%d", j.turn, kind, j.seq)
}

// mergeCall 把 tc 里写了的字段并进 c。ACP 的更新只带变了的字段，写了的整项替换。还没写进日志时顺带重算名字。
func mergeCall(c *toolCall, tc acp.ToolCall) {
	if tc.Title != "" {
		c.title = tc.Title
	}
	if tc.Name != "" {
		c.acpName = tc.Name
	}
	if tc.Kind != "" {
		c.kind = tc.Kind
	}
	if tc.Status != "" {
		c.status = tc.Status
	}
	if tc.Content != nil {
		c.content = tc.Content
	}
	if tc.Locations != nil {
		c.locations = tc.Locations
	}
	if len(tc.RawInput) > 0 {
		c.rawInput = tc.RawInput
	}
	if len(tc.RawOutput) > 0 {
		c.rawOutput = tc.RawOutput
	}
	if !c.logged {
		c.name = toolName(c)
	}
}

// finished 在调用已经有结果时为真。
func finished(status string) bool {
	return status == "completed" || status == "failed"
}

// toolName 按 kind 把调用记成 yoyo 的工具名，任务页才认得出改了哪些文件。其余用 ACP 的名字或标题，去掉 yoyo 控制面的前缀。
func toolName(c *toolCall) string {
	switch c.kind {
	case "edit":
		return "edit_file"
	case "read":
		return "read_file"
	case "execute":
		return "shell"
	case "search":
		return "grep"
	case "fetch":
		return "web_fetch"
	case "delete":
		return "delete_file"
	case "move":
		return "move_file"
	}
	name := strings.TrimSpace(c.acpName)
	if name == "" {
		name = strings.TrimSpace(c.title)
	}
	if name == "" {
		return "tool"
	}
	return stripPrefix(name)
}

// stripPrefix 去掉 MCP 工具名上的 yoyo 前缀，orchestrator_report 这类名字才认得出。
func stripPrefix(name string) string {
	for _, prefix := range []string{"mcp__" + mcpName + "__", mcpName + "__", mcpName + "/", mcpName + "."} {
		if strings.HasPrefix(name, prefix) {
			return name[len(prefix):]
		}
	}
	return name
}

// toolArgs 是调用记进日志的参数：rawInput 原样展开，改文件类补上 path，执行类补上 command。太长时只留短字段。
func toolArgs(c *toolCall, dir string) string {
	args := map[string]any{}
	if len(c.rawInput) > 0 {
		if json.Unmarshal(c.rawInput, &args) != nil {
			args = map[string]any{"input": string(c.rawInput)}
		}
	}
	switch c.kind {
	case "edit", "read", "delete", "move", "search":
		if _, ok := args["path"]; !ok {
			if paths := callPaths(c, dir); len(paths) > 0 {
				args["path"] = paths[0]
			}
		}
	case "execute":
		if _, ok := args["command"]; !ok {
			args["command"] = commandOf(c)
		}
	}
	if len(args) == 0 && c.title != "" {
		args["title"] = c.title
	}
	body, err := json.Marshal(args)
	if err != nil {
		return "{}"
	}
	if len(body) <= maxArgs {
		return string(body)
	}
	short := map[string]any{}
	for _, key := range []string{"path", "file_path", "command", "pattern", "url", "title"} {
		if v, ok := args[key].(string); ok {
			short[key] = clipRunes(v, 1000)
		}
	}
	short["truncated"] = true
	body, _ = json.Marshal(short)
	return string(body)
}

// commandOf 取执行类调用的命令：rawInput.command 是字符串或参数数组，都没有时用标题。
func commandOf(c *toolCall) string {
	var in struct {
		Command json.RawMessage `json:"command"`
		Cmd     json.RawMessage `json:"cmd"`
	}
	if len(c.rawInput) > 0 && json.Unmarshal(c.rawInput, &in) == nil {
		for _, raw := range []json.RawMessage{in.Command, in.Cmd} {
			if cmd := commandText(raw); cmd != "" {
				return cmd
			}
		}
	}
	return strings.TrimSpace(c.title)
}

// commandText 把字符串或字符串数组形式的命令收成一行。
func commandText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var parts []string
	if json.Unmarshal(raw, &parts) == nil {
		return strings.TrimSpace(strings.Join(parts, " "))
	}
	return ""
}

// callPaths 返回调用碰到的绝对路径：locations、diff 的路径，以及 rawInput 里常见的路径字段。
func callPaths(c *toolCall, dir string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		p = absPath(p, dir)
		if p == "" || seen[pathKey(p)] {
			return
		}
		seen[pathKey(p)] = true
		out = append(out, p)
	}
	for _, loc := range c.locations {
		add(loc.Path)
	}
	for _, item := range c.content {
		if item.Type == "diff" {
			add(item.Path)
		}
	}
	if len(c.rawInput) > 0 {
		var in map[string]any
		if json.Unmarshal(c.rawInput, &in) == nil {
			for _, key := range []string{"path", "file_path", "filePath", "abs_path", "notebook_path", "source", "destination", "from", "to"} {
				if v, ok := in[key].(string); ok {
					add(v)
				}
			}
		}
	}
	return out
}

// absPath 把 p 补成绝对路径并清理。相对路径按 dir 补；dir 为空时丢掉。
func absPath(p, dir string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		if dir == "" {
			return ""
		}
		p = filepath.Join(dir, p)
	}
	return filepath.Clean(p)
}

// pathKey 是比较路径用的键。Windows 上不分大小写。
func pathKey(p string) string {
	p = filepath.Clean(p)
	if goruntime.GOOS == "windows" {
		return strings.ToLower(p)
	}
	return p
}

// resultText 拼工具结果：文本类 content、每个 diff 先一行 wrote <绝对路径> 再一行摘要、执行输出的末尾。都没有时写 done 或 failed。
// 任务页按行认 wrote，卡片只看第一行，所以 wrote 放在摘要前面。调用失败时 diff 只写一行 not written <路径>。
func resultText(c *toolCall, dir string) string {
	var parts []string
	for _, item := range c.content {
		switch item.Type {
		case "content":
			if item.Content != nil && item.Content.Type == "text" && strings.TrimSpace(item.Content.Text) != "" {
				parts = append(parts, strings.TrimSpace(item.Content.Text))
			}
		case "diff":
			p := absPath(item.Path, dir)
			switch {
			case p == "":
			case c.status == "failed":
				parts = append(parts, "not written "+p)
			default:
				parts = append(parts, "wrote "+p, diffSummary(item, p))
			}
		}
	}
	if out := outputText(c.rawOutput); out != "" && (c.kind == "execute" || len(parts) == 0) {
		parts = append(parts, tailRunes(out, outputTail))
	}
	text := strings.Join(parts, "\n")
	if strings.TrimSpace(text) == "" {
		if c.status == "failed" {
			return "failed"
		}
		return "done"
	}
	return clipRunes(text, maxResult)
}

// diffSummary 是一个 diff 的一行摘要：新建了几行，或从几行改成几行。
func diffSummary(item acp.ToolCallContent, path string) string {
	newLines := countLines(item.NewText)
	if item.OldText == nil {
		return fmt.Sprintf("new file %s: %d lines", path, newLines)
	}
	return fmt.Sprintf("diff %s: %d -> %d lines", path, countLines(*item.OldText), newLines)
}

// countLines 数 text 有几行，空串是 0 行。
func countLines(text string) int {
	if text == "" {
		return 0
	}
	n := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		n++
	}
	return n
}

// outputText 从 rawOutput 里取命令输出：字符串原样用；对象按常见字段取输出、错误输出和退出码；其余用原文。
func outputText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return strings.TrimSpace(string(raw))
	}
	var parts []string
	for _, key := range []string{"formatted_output", "aggregated_output", "output", "stdout", "content"} {
		if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
			parts = append(parts, strings.TrimSpace(v))
			break
		}
	}
	if v, ok := m["stderr"].(string); ok && strings.TrimSpace(v) != "" {
		parts = append(parts, strings.TrimSpace(v))
	}
	for _, key := range []string{"exit_code", "exitCode"} {
		if v, ok := m[key].(float64); ok {
			parts = append(parts, fmt.Sprintf("exit code %d", int(v)))
			break
		}
	}
	if len(parts) == 0 {
		return strings.TrimSpace(string(raw))
	}
	return strings.Join(parts, "\n")
}

// planStatus 把 ACP 的计划状态换成 yoyo 的：completed 记成 complete。
func planStatus(status string) string {
	switch status {
	case "completed":
		return "complete"
	case "in_progress":
		return "in_progress"
	default:
		return "pending"
	}
}
