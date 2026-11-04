// session.go 持有追加型会话日志，以及从日志投影出的模型可见表面。
// 它不跑循环，也不执行工具。Agent 在每一步前后写入它。

package dsh

import (
	"fmt"
	"sync"
)

const (
	// EventTurnStart 是回合开始。
	EventTurnStart = "turn/start"
	// EventTurnEnd 是回合结束。
	EventTurnEnd = "turn/end"
	// EventStepStart 是步骤开始。
	EventStepStart = "step/start"
	// EventStepEnd 是步骤结束。
	EventStepEnd = "step/end"
	// EventUser 是用户或宿主投递的消息。
	EventUser = "user/message"
	// EventAssistant 是已落盘的助手消息。
	EventAssistant = "assistant/message"
	// EventAssistantAttempt 是没有可见正文的流结束。
	EventAssistantAttempt = "assistant/attempt"
	// EventSystem 是系统提示节点。
	EventSystem = "system/message"
	// EventToolCall 是已记录开始的工具调用。
	EventToolCall = "tool/call"
	// EventToolResult 是工具结果。
	EventToolResult = "tool/result"
	// EventRequestHeader 是一次请求信封。
	EventRequestHeader = "request/header"
	// EventInbox 是收件箱拼接。
	EventInbox = "agent/inbox/spliced"
)

// Event 是日志里的一条。Seq 从 1 递增。Data 的具体类型由 Type 决定。
type Event struct {
	Seq  int
	Type string
	Turn int
	Step int
	Data any
}

// ToolCall 是模型发出的一次工具调用。ID 在同一次助手消息里唯一。
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// Message 是发给模型的一条表面消息。Role 取 system、user、assistant、tool。
type Message struct {
	Role       string
	Text       string
	Name       string
	ToolCallID string
	ToolCalls  []ToolCall
	Parts      []Part
}

// Part 是一条多模态片段。URL 非空时按图片发给模型。
type Part struct {
	Type string
	Text string
	URL  string
	MIME string
}

// SystemCommit 是一次系统提示投影。Op 为 append 或 replace。Index 只在 replace 时使用。
type SystemCommit struct {
	Op    string
	Index int
	Text  string
}

type surfaceNode struct {
	kind      string
	text      string
	name      string
	toolID    string
	toolCalls []ToolCall
	isError   bool
	parts     []Part
}

// Session 是一份追加型日志。Derive 读当前表面，不改日志。
type Session struct {
	mu     sync.Mutex
	events []Event
	nodes  []surfaceNode
	seq    int
}

// NewSession 返回一份空日志。
func NewSession() *Session {
	return &Session{}
}

// Restore 用 events 换掉日志，并按同样的顺序重放表面。回合号留给 ResumeTurn。
func (s *Session) Restore(events []Event) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
	s.nodes = nil
	s.seq = 0
	for _, ev := range events {
		s.seq++
		ev.Seq = s.seq
		s.events = append(s.events, ev)
		if ev.Data != nil {
			s.applyLocked(ev.Type, ev.Data)
		}
	}
}

// Reset 清空日志和模型可见表面。检查点替换历史时调用。
func (s *Session) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.events = nil
	s.nodes = nil
	s.seq = 0
	s.mu.Unlock()
}

// Append 追加一条事件并更新表面。turn 和 step 写入事件。data 原样保存。返回序号。
func (s *Session) Append(typ string, turn, step int, data any) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	s.events = append(s.events, Event{Seq: s.seq, Type: typ, Turn: turn, Step: step, Data: data})
	s.applyLocked(typ, data)
	return s.seq
}

// applyLocked 按 typ 把 data 应用到表面节点。调用时要已持有锁。
func (s *Session) applyLocked(typ string, data any) {
	switch typ {
	case EventSystem:
		c := data.(SystemCommit)
		if c.Op == "replace" && c.Index >= 0 && c.Index < len(s.systemIndexesLocked()) {
			idx := s.systemIndexesLocked()[c.Index]
			s.nodes[idx].text = c.Text
			return
		}
		s.nodes = append(s.nodes, surfaceNode{kind: "system", text: c.Text})
	case EventUser:
		m := data.(UserMessage)
		s.nodes = append(s.nodes, surfaceNode{kind: "user", text: m.Text, name: m.Source, parts: append([]Part(nil), m.Parts...)})
	case EventAssistant:
		a := data.(Assistant)
		s.nodes = append(s.nodes, surfaceNode{kind: "assistant", text: a.Text, toolCalls: append([]ToolCall(nil), a.ToolCalls...)})
	case EventToolResult:
		r := data.(ToolResultEvent)
		s.nodes = append(s.nodes, surfaceNode{kind: "tool", text: r.Text, name: r.Name, toolID: r.ID, isError: r.IsError})
	}
}

// systemIndexesLocked 返回系统节点在表面里的下标。调用时要已持有锁。
func (s *Session) systemIndexesLocked() []int {
	var idx []int
	for i, n := range s.nodes {
		if n.kind == "system" {
			idx = append(idx, i)
		}
	}
	return idx
}

// Events 返回日志副本。
func (s *Session) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}

// SystemTexts 按表面顺序返回系统节点正文，包含被清空的节点。
func (s *Session) SystemTexts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, n := range s.nodes {
		if n.kind == "system" {
			out = append(out, n.text)
		}
	}
	return out
}

// Derive 返回模型可见消息。空的系统节点不出现。工具结果保留错误标记在正文里。
func (s *Session) Derive() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Message
	for _, n := range s.nodes {
		switch n.kind {
		case "system":
			if n.text == "" {
				continue
			}
			out = append(out, Message{Role: "system", Text: n.text})
		case "user":
			out = append(out, Message{Role: "user", Text: n.text, Name: n.name, Parts: append([]Part(nil), n.parts...)})
		case "assistant":
			out = append(out, Message{Role: "assistant", Text: n.text, ToolCalls: append([]ToolCall(nil), n.toolCalls...)})
		case "tool":
			out = append(out, Message{Role: "tool", Text: n.text, Name: n.name, ToolCallID: n.toolID})
		}
	}
	return out
}

// ToolResultEvent 是一条已落盘的工具结果。ID 对应工具调用。
type ToolResultEvent struct {
	ID      string
	Name    string
	Text    string
	IsError bool
	Code    string
}

// Assistant 是一次模型完成。Finish 取 stop、tool-calls、max-tokens、error。Interrupted 表示流被取消后留下已说出的前缀。
type Assistant struct {
	Text        string
	ToolCalls   []ToolCall
	Finish      string
	Interrupted bool
	// AttemptID 是这一次模型调用的流标识。界面用它把增量收成同一条助手消息。空表示没有流。
	AttemptID string
}

// UnansweredToolCalls 返回本步助手消息里还没有结果的调用。started 为真表示日志里已有 tool/call。
func (s *Session) UnansweredToolCalls(turn, step int) []UnansweredCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	type call struct {
		id, name, args string
		started        bool
	}
	var calls []call
	answered := map[string]bool{}
	for _, ev := range s.events {
		if ev.Turn != turn || ev.Step != step {
			continue
		}
		switch ev.Type {
		case EventAssistant:
			a := ev.Data.(Assistant)
			for _, c := range a.ToolCalls {
				calls = append(calls, call{id: c.ID, name: c.Name, args: c.Arguments})
			}
		case EventToolCall:
			c := ev.Data.(ToolCall)
			for i := range calls {
				if calls[i].id == c.ID {
					calls[i].started = true
				}
			}
		case EventToolResult:
			r := ev.Data.(ToolResultEvent)
			answered[r.ID] = true
		}
	}
	var out []UnansweredCall
	for _, c := range calls {
		if answered[c.id] {
			continue
		}
		out = append(out, UnansweredCall{ID: c.id, Name: c.name, Arguments: c.args, Started: c.started})
	}
	return out
}

// UnansweredCall 是一步里缺结果的工具调用。Started 为真时恢复码用 TOOL_OUTCOME_UNKNOWN。
type UnansweredCall struct {
	ID        string
	Name      string
	Arguments string
	Started   bool
}

// RepairOpenTools 把助手的 tool_calls 和结果配回相邻的位置。
// 结果若被用户或系统消息隔开，就挪到这条助手消息后面；结果没落盘时按中断补一条。
// 已经配对的日志不动。返回补上或挪动的调用数。
func (s *Session) RepairOpenTools() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) == 0 {
		return 0
	}
	resultAt := map[string]int{}
	started := map[string]bool{}
	for i, ev := range s.events {
		switch ev.Type {
		case EventToolCall:
			call, ok := ev.Data.(ToolCall)
			if ok && call.ID != "" {
				started[call.ID] = true
			}
		case EventToolResult:
			res, ok := ev.Data.(ToolResultEvent)
			if ok && res.ID != "" {
				resultAt[res.ID] = i
			}
		}
	}
	type openCall struct {
		id, name, args string
		turn, step     int
	}
	var pending []openCall
	placed := map[string]bool{}
	out := make([]Event, 0, len(s.events)+4)
	changed := 0
	synth := 0
	flush := func() {
		for _, call := range pending {
			if call.id == "" || placed[call.id] {
				continue
			}
			if idx, ok := resultAt[call.id]; ok {
				out = append(out, s.events[idx])
				placed[call.id] = true
				changed++
				continue
			}
			known := started[call.id]
			rec := RecoveryResults([]UnansweredCall{{
				ID: call.id, Name: call.name, Arguments: call.args, Started: known,
			}})[0]
			if !known {
				out = append(out, Event{
					Type: EventToolCall, Turn: call.turn, Step: call.step,
					Data: ToolCall{ID: call.id, Name: call.name, Arguments: call.args},
				})
			}
			out = append(out, Event{
				Type: EventToolResult, Turn: call.turn, Step: call.step,
				Data: ToolResultEvent{ID: call.id, Name: call.name, Text: rec.Text, IsError: true, Code: rec.Code},
			})
			placed[call.id] = true
			changed++
		}
		pending = nil
	}
	for _, ev := range s.events {
		switch ev.Type {
		case EventAssistant:
			flush()
			msg, ok := ev.Data.(Assistant)
			if !ok || len(msg.ToolCalls) == 0 {
				out = append(out, ev)
				continue
			}
			calls := append([]ToolCall(nil), msg.ToolCalls...)
			for i := range calls {
				if calls[i].ID == "" {
					synth++
					name := calls[i].Name
					if name == "" {
						name = "tool"
					}
					calls[i].ID = fmt.Sprintf("call_repair_%s_%d", name, synth)
					changed++
				}
				pending = append(pending, openCall{
					id: calls[i].ID, name: calls[i].Name, args: calls[i].Arguments,
					turn: ev.Turn, step: ev.Step,
				})
			}
			msg.ToolCalls = calls
			ev.Data = msg
			out = append(out, ev)
		case EventToolResult:
			res, ok := ev.Data.(ToolResultEvent)
			if ok && res.ID != "" && placed[res.ID] {
				continue
			}
			if ok && res.ID != "" {
				placed[res.ID] = true
				next := make([]openCall, 0, len(pending))
				for _, call := range pending {
					if call.id != res.ID {
						next = append(next, call)
					}
				}
				pending = next
			}
			out = append(out, ev)
		case EventUser, EventSystem:
			flush()
			out = append(out, ev)
		default:
			out = append(out, ev)
		}
	}
	flush()
	if changed == 0 {
		return 0
	}
	s.events = nil
	s.nodes = nil
	s.seq = 0
	for _, ev := range out {
		s.seq++
		ev.Seq = s.seq
		s.events = append(s.events, ev)
		if ev.Data != nil {
			s.applyLocked(ev.Type, ev.Data)
		}
	}
	return changed
}
