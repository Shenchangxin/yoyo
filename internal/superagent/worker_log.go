// worker_log.go 把工人会话日志收成可落盘的 JSON，并在重启后续回代理。
// 不跑回合。回合仍由 dsh.Agent 负责。

package superagent

import (
	"encoding/json"
	"strings"

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
	case dsh.EventTurnStart, dsh.EventStepStart, dsh.EventStepEnd:
		return nil, true
	default:
		return nil, false
	}
}

// TranscriptLine 是工单会话里给人看的一行。Role 取 user、assistant、tool_call、tool_result。
type TranscriptLine struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Text string `json:"text"`
	Name string `json:"name,omitempty"`
}

// TranscriptLines 把工人日志收成会话行。派工包装、重试和后台完成不进对话。
func TranscriptLines(events []dsh.Event) []TranscriptLine {
	var out []TranscriptLine
	for i, ev := range events {
		switch ev.Type {
		case dsh.EventUser:
			msg, ok := ev.Data.(dsh.UserMessage)
			if !ok {
				continue
			}
			if msg.Source == "assignment" || msg.Source == "retry" || msg.Source == "job" {
				continue
			}
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}
			out = append(out, TranscriptLine{ID: lineID(i, "user"), Role: "user", Text: clipRunes(text, 4000)})
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
				out = append(out, TranscriptLine{
					ID: lineID(i, "call") + "-" + itoa(j), Role: "tool_call", Name: name, Text: clipRunes(call.Arguments, 500),
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
			out = append(out, TranscriptLine{ID: lineID(i, "result"), Role: "tool_result", Name: msg.Name, Text: clipRunes(text, 1200)})
		}
	}
	return out
}

func lineID(i int, role string) string {
	return role + "-" + itoa(i+1)
}

// restoreAgent 把 ticketID 已存的日志灌进 agent，并把回合号接到上次的后面。
func (e *Executor) restoreAgent(agent *dsh.Agent, ticketID string) {
	if e == nil || agent == nil || e.LogLoad == nil {
		return
	}
	body := e.LogLoad(ticketID)
	events, turn := decodeLog(body)
	if len(events) == 0 {
		return
	}
	agent.Session().Restore(events)
	if turn > 0 {
		agent.ResumeTurn(turn)
	}
}

// storeAgent 把 agent 当前日志交给 LogSave。没有日志时不写。
func (e *Executor) storeAgent(agent *dsh.Agent, ticketID string) {
	if e == nil || agent == nil || e.LogSave == nil || agent.Session() == nil {
		return
	}
	body, _ := encodeLog(agent.Session().Events())
	if len(body) == 0 {
		return
	}
	e.LogSave(ticketID, body)
}

// passingError 在失败是暂时的网络或服务端限流时为真。准备命令失败不算。
func passingError(err string) bool {
	text := strings.ToLower(err)
	if text == "" {
		return false
	}
	for _, part := range []string{"408", "429", "500", "502", "503", "504", "timeout", "timed out", "connection refused", "connection reset", "temporarily"} {
		if strings.Contains(text, part) {
			return true
		}
	}
	return false
}

// lastTurnEnd 返回日志里最后一次回合结束。没有时两个都是空串。
func lastTurnEnd(agent *dsh.Agent) (string, string) {
	if agent == nil || agent.Session() == nil {
		return "", ""
	}
	events := agent.Session().Events()
	for i := len(events) - 1; i >= 0; i-- {
		end, ok := events[i].Data.(dsh.TurnEnd)
		if events[i].Type == dsh.EventTurnEnd && ok {
			return end.Kind, end.Reason
		}
	}
	return "", ""
}
