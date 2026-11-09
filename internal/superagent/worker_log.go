// worker_log.go 把工人会话日志收成可落盘的 JSON，并在重启后续回工人。
// 不跑回合。回合由 Worker 负责：dsh 代理或外部 agent 的归一化日志。

package superagent

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

type storedEvent struct {
	Type string          `json:"type"`
	Turn int             `json:"turn"`
	Step int             `json:"step"`
	Data json.RawMessage `json:"data,omitempty"`
}

// encodeLog 把事件收成 JSON，并返回其中最大的回合号。
func encodeLog(events []dsh.Event) ([]byte, int) {
	turn := 0
	out := make([]storedEvent, 0, len(events))
	for _, ev := range events {
		if ev.Turn > turn {
			turn = ev.Turn
		}
		raw, err := json.Marshal(ev.Data)
		if err != nil || string(raw) == "null" {
			raw = nil
		}
		out = append(out, storedEvent{Type: ev.Type, Turn: ev.Turn, Step: ev.Step, Data: raw})
	}
	body, err := json.Marshal(out)
	if err != nil {
		return nil, turn
	}
	return body, turn
}

// decodeLog 把 JSON 还原成事件。认不出的类型丢掉，避免重放时类型对不上。
func decodeLog(body []byte) ([]dsh.Event, int) {
	if len(body) == 0 {
		return nil, 0
	}
	var stored []storedEvent
	if json.Unmarshal(body, &stored) != nil {
		return nil, 0
	}
	turn := 0
	var out []dsh.Event
	for _, ev := range stored {
		if ev.Turn > turn {
			turn = ev.Turn
		}
		data, ok := decodeEventData(ev.Type, ev.Data)
		if !ok && len(ev.Data) > 0 {
			continue
		}
		out = append(out, dsh.Event{Type: ev.Type, Turn: ev.Turn, Step: ev.Step, Data: data})
	}
	return out, turn
}

// decodeEventData 按类型还原载荷。ok 为假表示这个类型不重放。
func decodeEventData(typ string, raw json.RawMessage) (any, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, true
	}
	switch typ {
	case dsh.EventUser:
		var msg dsh.UserMessage
		if json.Unmarshal(raw, &msg) != nil {
			return nil, false
		}
		return msg, true
	case dsh.EventAssistant:
		var msg dsh.Assistant
		if json.Unmarshal(raw, &msg) != nil {
			return nil, false
		}
		return msg, true
	case dsh.EventToolResult:
		var msg dsh.ToolResultEvent
		if json.Unmarshal(raw, &msg) != nil {
			return nil, false
		}
		return msg, true
	case dsh.EventSystem:
		var msg dsh.SystemCommit
		if json.Unmarshal(raw, &msg) != nil {
			return nil, false
		}
		return msg, true
	case dsh.EventTurnEnd:
		var msg dsh.TurnEnd
		if json.Unmarshal(raw, &msg) != nil {
			return nil, false
		}
		return msg, true
	case dsh.EventInbox:
		var msg dsh.UserMessage
		if json.Unmarshal(raw, &msg) != nil {
			return nil, false
		}
		return msg, true
	case dsh.EventTurnStart, dsh.EventStepStart, dsh.EventStepEnd:
		return nil, true
	default:
		return nil, false
	}
}

type reportArgs struct {
	State    string `json:"state"`
	Summary  string `json:"summary"`
	Question string `json:"question"`
}

// rejectedAsk 取出日志里最后一次被拒绝的「做完同时提问」。没有这种调用时 ok 为假。
func rejectedAsk(body []byte) (summary, question string, ok bool) {
	var stored []storedEvent
	if json.Unmarshal(body, &stored) != nil {
		return "", "", false
	}
	calls := map[string]reportArgs{}
	var last reportArgs
	found := false
	for _, ev := range stored {
		switch ev.Type {
		case dsh.EventAssistant:
			var msg dsh.Assistant
			if json.Unmarshal(ev.Data, &msg) != nil {
				continue
			}
			for _, call := range msg.ToolCalls {
				rememberReport(calls, call)
			}
		case dsh.EventToolCall:
			var call dsh.ToolCall
			if json.Unmarshal(ev.Data, &call) != nil {
				continue
			}
			rememberReport(calls, call)
		case dsh.EventToolResult:
			var msg dsh.ToolResultEvent
			if json.Unmarshal(ev.Data, &msg) != nil || msg.Name != "orchestrator_report" || msg.Text != "questionWithResult" {
				continue
			}
			args, known := calls[msg.ID]
			if !known {
				continue
			}
			last = args
			found = true
		}
	}
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(last.Summary), strings.TrimSpace(last.Question), true
}

// rememberReport 记下一次带问题的 orchestrator_report。没有问题的调用不记。
func rememberReport(calls map[string]reportArgs, call dsh.ToolCall) {
	if call.Name != "orchestrator_report" || call.ID == "" {
		return
	}
	var args reportArgs
	if json.Unmarshal([]byte(call.Arguments), &args) != nil || strings.TrimSpace(args.Question) == "" {
		return
	}
	calls[call.ID] = args
}

// askedQuestion 在 ticketID 已经记过 question 时为真。
func askedQuestion(e *Engine, ticketID, question string) bool {
	if e == nil {
		return false
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return true
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return false
	}
	for _, asg := range t.Assignments {
		for _, rep := range asg.Reports {
			if strings.TrimSpace(rep.Question) == question {
				return true
			}
		}
	}
	return false
}

// applyRejectedAsk 把日志里被拒绝的提问按 blocked 补进 ticketID。已经有同一问题时不再补。
func applyRejectedAsk(e *Engine, ticketID string, body []byte) bool {
	summary, question, ok := rejectedAsk(body)
	if !ok || askedQuestion(e, ticketID, question) {
		return false
	}
	return e.Report(ticketID, Report{State: "blocked", Summary: summary, Question: question}) == nil
}

// TranscriptLine 是工单会话里给人看的一行。Role 取 user、assistant、tool_call、tool_result。
// Source 是这条用户消息从哪来的。交接行的 Name 是接手的那次派工。Error 为真表示工具结果失败。
type TranscriptLine struct {
	ID     string `json:"id"`
	Role   string `json:"role"`
	Text   string `json:"text"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source,omitempty"`
	Error  bool   `json:"error,omitempty"`
}

// stuckTurnError 返回还没被后续发言盖过去的那次错误回合。后面又有人说话或工人开口时返回 -1。
func stuckTurnError(events []dsh.Event) int {
	last := -1
	for i, ev := range events {
		end, ok := ev.Data.(dsh.TurnEnd)
		if ev.Type == dsh.EventTurnEnd && ok && end.Kind == "error" && strings.TrimSpace(end.Reason) != "" {
			last = i
		}
	}
	if last < 0 {
		return -1
	}
	for _, ev := range events[last+1:] {
		switch ev.Type {
		case dsh.EventAssistant, dsh.EventUser, dsh.EventToolResult, dsh.EventToolCall:
			return -1
		}
	}
	return last
}

// TranscriptLines 把工人日志收成会话行。派工包装、重试和后台完成不进对话。
// 还没被下一步领走的新消息会先出现。只有停在错误上、后面没有再开口时，才把中断原因留在末尾。
func TranscriptLines(events []dsh.Event) []TranscriptLine {
	claimed := map[string]bool{}
	for _, ev := range events {
		msg, ok := ev.Data.(dsh.UserMessage)
		if ev.Type == dsh.EventUser && ok {
			if text := strings.TrimSpace(msg.Text); text != "" {
				claimed[strings.Join(strings.Fields(text), " ")] = true
			}
		}
	}
	stuck := stuckTurnError(events)
	var out []TranscriptLine
	for i, ev := range events {
		switch ev.Type {
		case dsh.EventInbox:
			msg, ok := ev.Data.(dsh.UserMessage)
			if !ok || msg.Source == "retry" || msg.Source == "job" {
				continue
			}
			text := strings.TrimSpace(msg.Text)
			if text == "" || text == hostRestartNote || claimed[strings.Join(strings.Fields(text), " ")] {
				continue
			}
			out = append(out, TranscriptLine{ID: lineID(i, "inbox"), Role: "user", Source: msg.Source, Text: clipRunes(text, 4000)})
		case dsh.EventTurnEnd:
			if i != stuck {
				continue
			}
			end, _ := ev.Data.(dsh.TurnEnd)
			out = append(out, TranscriptLine{ID: lineID(i, "error"), Role: "assistant", Text: "这一回合中断了：" + clipRunes(strings.TrimSpace(end.Reason), 300)})
		case dsh.EventUser:
			msg, ok := ev.Data.(dsh.UserMessage)
			if !ok {
				continue
			}
			if msg.Source == "retry" || msg.Source == "job" {
				continue
			}
			text := strings.TrimSpace(msg.Text)
			if text == hostRestartNote {
				continue
			}
			if msg.Source == "assignment" {
				text = assignmentBody(text)
			}
			if text == "" || text == hostRestartNote {
				continue
			}
			line := TranscriptLine{ID: lineID(i, "user"), Role: "user", Source: msg.Source, Text: clipRunes(text, 4000)}
			if msg.Source == "handover" {
				line.Name = strings.TrimPrefix(msg.ID, handoverPrefix)
			}
			out = append(out, line)
		case dsh.EventAssistant:
			msg, ok := ev.Data.(dsh.Assistant)
			if !ok {
				continue
			}
			if text := strings.TrimSpace(msg.Text); text != "" {
				out = append(out, TranscriptLine{ID: lineID(i, "assistant"), Role: "assistant", Text: clipRunes(text, 4000)})
			}
			for j, call := range msg.ToolCalls {
				name := strings.TrimSpace(call.Name)
				if name == "" {
					continue
				}
				id := strings.TrimSpace(call.ID)
				if id == "" {
					id = lineID(i, "call") + "-" + itoa(j)
				}
				out = append(out, TranscriptLine{
					ID: id, Role: "tool_call", Name: name, Text: clipRunes(call.Arguments, transcriptClip(name, 500)),
				})
			}
		case dsh.EventToolResult:
			msg, ok := ev.Data.(dsh.ToolResultEvent)
			if !ok {
				continue
			}
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}
			id := strings.TrimSpace(msg.ID)
			if id == "" {
				id = lineID(i, "result")
			}
			out = append(out, TranscriptLine{ID: id, Role: "tool_result", Name: msg.Name, Text: clipRunes(text, transcriptClip(msg.Name, 1200)), Error: msg.IsError})
		}
	}
	return out
}

// transcriptClip 给计划工具留更长的参数和结果，任务页才能把步骤展开。其余工具仍用 fallback。
func transcriptClip(name string, fallback int) int {
	if name == "update_plan" {
		return 8000
	}
	return fallback
}

func lineID(i int, role string) string {
	return role + "-" + itoa(i+1)
}

// scrubRestartNotes 去掉重启说明。它不是新任务，留在日志里会让工人把已经做过的事当成不能再做。
func scrubRestartNotes(events []dsh.Event) []dsh.Event {
	if len(events) == 0 {
		return events
	}
	out := make([]dsh.Event, 0, len(events))
	for _, ev := range events {
		if ev.Type == dsh.EventUser {
			msg, ok := ev.Data.(dsh.UserMessage)
			if ok {
				msg.Text = dropRestartLines(msg.Text)
				if strings.TrimSpace(msg.Text) == "" {
					continue
				}
				ev.Data = msg
			}
		}
		out = append(out, ev)
	}
	return out
}

// dropRestartLines 删掉和宿主重启说明完全相同的行。
func dropRestartLines(text string) string {
	var keep []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == hostRestartNote {
			continue
		}
		keep = append(keep, line)
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}

// assignmentBody 去掉派工包装，只留人写过的那些话。
func assignmentBody(text string) string {
	var keep []string
	for _, line := range strings.Split(text, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || s == hostRestartNote || strings.HasPrefix(s, "工单 ") || strings.HasPrefix(s, "工作目录 ") || strings.HasPrefix(s, mandatePrefix) || s == "完成后调用 orchestrator_report。" {
			continue
		}
		keep = append(keep, s)
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}

// watchLog 在这一轮还没结束时把 w 的会话写下来，任务页才能跟着长。stop 关上后退出。
func (e *Executor) watchLog(w Worker, ticketID string, stop <-chan struct{}) {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			e.storeAgent(w, ticketID)
		}
	}
}

// restoreAgent 把 ticketID 已存的日志灌进 w，并把回合号接到上次的后面。灌进去时返回真。
// 日志里有交接时只灌最后一次交接之后的那段，更早的留作前缀，新工人读不到别的 agent 留下的工具调用。
// marker 是这次派工的交接消息。日志里还没有它时，已有的全部留作前缀，w 从这条消息开始，返回假。
func (e *Executor) restoreAgent(w Worker, ticketID string, marker *dsh.UserMessage) bool {
	if e == nil || w == nil {
		return false
	}
	var events []dsh.Event
	turn := 0
	if e.LogLoad != nil {
		events, turn = decodeLog(e.LogLoad(ticketID))
		events = scrubRestartNotes(events)
	}
	fresh := marker != nil && !hasMessage(events, marker.ID)
	cut := lastHandover(events)
	if fresh {
		cut = len(events)
	}
	var prefix []dsh.Event
	if cut > 0 {
		prefix, events = events[:cut], events[cut:]
	}
	e.setPrefix(ticketID, w.Session(), prefix)
	if fresh {
		w.Restore(nil, turn)
		if s := w.Session(); s != nil {
			s.Append(dsh.EventUser, turn, 0, *marker)
		}
		return false
	}
	if len(events) == 0 {
		return false
	}
	w.Restore(events, turn)
	return true
}

// storeAgent 把交接前缀和 w 当前日志一起交给 LogSave。没有日志时不写。
// 交接之后旧工人再写不算数，它的日志已经是新工人的前缀。
func (e *Executor) storeAgent(w Worker, ticketID string) {
	if e == nil || w == nil || e.LogSave == nil || w.Session() == nil {
		return
	}
	prefix, ok := e.prefixFor(ticketID, w.Session())
	if !ok {
		return
	}
	body, _ := encodeLog(append(prefix, w.Session().Events()...))
	if len(body) == 0 {
		return
	}
	e.LogSave(ticketID, body)
}

// setPrefix 记下 ticketID 交接之前的日志，以及之后写日志的会话 owner。
func (e *Executor) setPrefix(ticketID string, owner *dsh.Session, events []dsh.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.prefix == nil {
		e.prefix = map[string]logPrefix{}
	}
	e.prefix[ticketID] = logPrefix{owner: owner, events: events}
}

// prefixFor 返回 owner 写盘时要拼在前面的日志副本。前缀属于别的会话时 ok 为假，这次不写。
func (e *Executor) prefixFor(ticketID string, owner *dsh.Session) ([]dsh.Event, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, has := e.prefix[ticketID]
	if !has {
		return nil, true
	}
	if p.owner != owner {
		return nil, false
	}
	return append([]dsh.Event(nil), p.events...), true
}

// lastHandover 返回 events 里最后一条交接消息的下标。没有时为 -1。
func lastHandover(events []dsh.Event) int {
	for i := len(events) - 1; i >= 0; i-- {
		if msg, ok := events[i].Data.(dsh.UserMessage); ok && events[i].Type == dsh.EventUser && msg.Source == "handover" {
			return i
		}
	}
	return -1
}

// hasMessage 在 events 里有编号为 id 的用户消息时为真。
func hasMessage(events []dsh.Event, id string) bool {
	for _, ev := range events {
		if msg, ok := ev.Data.(dsh.UserMessage); ok && ev.Type == dsh.EventUser && msg.ID == id {
			return true
		}
	}
	return false
}

// passingError 在失败是暂时的网络或服务端限流时为真。外部 agent 进程退出也算，重试时工人会重连并续接。准备命令失败不算。
func passingError(err string) bool {
	text := strings.ToLower(err)
	if text == "" {
		return false
	}
	for _, part := range []string{"408", "429", "500", "502", "503", "504", "timeout", "timed out", "connection refused", "connection reset", "connection closed", "broken pipe", "eof", "temporarily", "agent exited", "process exited"} {
		if strings.Contains(text, part) {
			return true
		}
	}
	return false
}

// lastAssistantText 返回 w 最后一个回合里最后一段助手正文。那一回合没有正文时为空串。
func lastAssistantText(w Worker) string {
	if w == nil || w.Session() == nil {
		return ""
	}
	events := w.Session().Events()
	turn := -1
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if turn < 0 {
			turn = ev.Turn
		}
		if ev.Turn != turn {
			break
		}
		if msg, ok := ev.Data.(dsh.Assistant); ok && ev.Type == dsh.EventAssistant {
			if text := strings.TrimSpace(msg.Text); text != "" {
				return text
			}
		}
	}
	return ""
}

// lastTurnEnd 返回 w 日志里最后一次回合结束。没有时两个都是空串。
func lastTurnEnd(w Worker) (string, string) {
	if w == nil || w.Session() == nil {
		return "", ""
	}
	events := w.Session().Events()
	for i := len(events) - 1; i >= 0; i-- {
		end, ok := events[i].Data.(dsh.TurnEnd)
		if events[i].Type == dsh.EventTurnEnd && ok {
			return end.Kind, end.Reason
		}
	}
	return "", ""
}
