// project.go 把 dsh 日志投影成界面已经认识的轨迹事件。
// 历史是播种进去的，不重复发出。来源为空的用户消息已经由发送方落盘，这里也不发。

package dshdesk

import (
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// seedHistory 把 hist 按原顺序写入 agent 的日志。系统、用户、助手和工具结果各自对应一种事件。
func seedHistory(agent *dsh.Agent, hist []runtime.Message) {
	if agent == nil || agent.Session() == nil {
		return
	}
	for _, msg := range hist {
		switch msg.Role {
		case runtime.RoleSystem:
			if msg.Content == "" {
				continue
			}
			agent.Session().Append(dsh.EventSystem, 0, 0, dsh.SystemCommit{Op: "append", Text: msg.Content})
		case runtime.RoleUser:
			if msg.Content == "" && len(msg.Parts) == 0 {
				continue
			}
			agent.Session().Append(dsh.EventUser, 0, 0, dsh.UserMessage{Text: msg.Content, Source: "history", Parts: toDshParts(msg.Parts)})
		case runtime.RoleAssistant:
			if msg.Content == "" && len(msg.ToolCalls) == 0 {
				continue
			}
			out := dsh.Assistant{Text: msg.Content, Finish: "stop"}
			for _, call := range msg.ToolCalls {
				out.ToolCalls = append(out.ToolCalls, dsh.ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments})
			}
			if len(out.ToolCalls) > 0 {
				out.Finish = "tool-calls"
			}
			agent.Session().Append(dsh.EventAssistant, 0, 0, out)
		case runtime.RoleTool:
			agent.Session().Append(dsh.EventToolResult, 0, 0, dsh.ToolResultEvent{
				ID: msg.ToolCallID, Name: msg.Name, Text: msg.Content,
			})
		}
	}
}

// project 从 cursor 之后把新日志发给 turn。stop 关上时再刷一次。
func project(turn Turn, agent *dsh.Agent, cursor int, stop <-chan struct{}) {
	if agent == nil {
		return
	}
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	flush := func() {
		cursor = emitNew(turn, agent, cursor)
	}
	for {
		select {
		case <-stop:
			flush()
			return
		case <-tick.C:
			flush()
		}
	}
}

// emitNew 发出 events 里下标从 cursor 起的新事件。返回新的下标。
func emitNew(turn Turn, agent *dsh.Agent, cursor int) int {
	evs := agent.Session().Events()
	for i := cursor; i < len(evs); i++ {
		emitDsh(turn, evs[i])
	}
	return len(evs)
}

// emitDsh 把 ev 译成一条轨迹。系统提示、请求头和来源为空的用户消息不发。
func emitDsh(turn Turn, ev dsh.Event) {
	switch ev.Type {
	case dsh.EventTurnStart:
		turn.emit(trace.Event{Type: trace.TypeTurnStart, Source: "dsh", Payload: map[string]any{"turn": ev.Turn}}, true)
	case dsh.EventTurnEnd:
		end, _ := ev.Data.(dsh.TurnEnd)
		turn.emit(trace.Event{Type: trace.TypeTurnEnd, Source: "dsh", Payload: map[string]any{
			"turn": ev.Turn, "kind": end.Kind, "reason": end.Reason,
		}}, true)
	case dsh.EventUser:
		msg, _ := ev.Data.(dsh.UserMessage)
		if msg.Source != "steer" && msg.Source != "job" {
			return
		}
		turn.emit(trace.Event{Type: trace.TypeUser, Source: msg.Source, Payload: map[string]any{
			"text": msg.Text, "name": msg.Source,
		}}, true)
	case dsh.EventAssistant:
		msg, _ := ev.Data.(dsh.Assistant)
		if strings.TrimSpace(msg.Text) == "" {
			return
		}
		turn.emit(trace.Event{Type: trace.TypeAssistant, Source: "model", Payload: map[string]any{"text": msg.Text}}, true)
	case dsh.EventToolCall:
		call, _ := ev.Data.(dsh.ToolCall)
		turn.emit(trace.Event{Type: trace.TypeToolCall, Source: "tool", Payload: map[string]any{
			"id": call.ID, "name": call.Name, "arguments": call.Arguments,
		}}, true)
	case dsh.EventToolResult:
		res, _ := ev.Data.(dsh.ToolResultEvent)
		turn.emit(trace.Event{Type: trace.TypeToolResult, Source: "tool", Payload: map[string]any{
			"id": res.ID, "name": res.Name, "content": res.Text, "ok": !res.IsError,
		}}, true)
	}
}

// emit 补上会话、回合和指纹后发出。persist 为真且 Trace 非空时写入 JSONL。OnEvent 非空时通知界面。
func (t Turn) emit(ev trace.Event, persist bool) {
	ev.SessionID = t.SessionID
	ev.TurnID = t.TurnID
	ev.HarnessSnapshot = t.HarnessHash
	ev.ModelFingerprint = t.ModelFingerprint
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	if ev.Payload == nil {
		ev.Payload = map[string]any{}
	}
	if t.TurnID != "" {
		if _, ok := ev.Payload["turn_id"]; !ok {
			ev.Payload["turn_id"] = t.TurnID
		}
	}
	if persist && t.Trace != nil {
		_ = t.Trace.Append(ev)
	}
	if t.OnEvent != nil {
		t.OnEvent(ev)
	}
}

// wireStream 让 bridge 把流式块发给 turn。第一块之前发开始帧，finish 块发结束帧。不写入 JSONL。
func wireStream(bridge *Bridge, turn Turn) {
	if bridge == nil {
		return
	}
	attempt := turn.TurnID
	if attempt == "" {
		attempt = turn.SessionID
	}
	rev := 1
	started := false
	bridge.OnStream = func(ch runtime.StreamChunk) {
		if !started {
			started = true
			turn.emit(trace.Event{Type: trace.TypeAssistantStream, Source: "model", Payload: map[string]any{
				"kind": "start", "attempt_id": attempt, "revision": rev,
			}}, false)
			rev++
		}
		if ch.Type == runtime.ChunkFinish {
			turn.emit(trace.Event{Type: trace.TypeAssistantStream, Source: "model", Payload: map[string]any{
				"kind": "end", "attempt_id": attempt, "revision": rev,
			}}, false)
			rev++
			return
		}
		turn.emit(trace.Event{Type: trace.TypeAssistantStream, Source: "model", Payload: map[string]any{
			"kind": "chunk", "attempt_id": attempt, "revision": rev, "index": ch.Index, "chunk": chunkPayload(ch),
		}}, false)
		rev++
	}
}

// chunkPayload 留下界面拼字需要的字段。ch 的类型、下标、文本、调用 id、名字和参数增量原样放进 map。
func chunkPayload(ch runtime.StreamChunk) map[string]any {
	out := map[string]any{"type": ch.Type}
	if ch.Index != 0 {
		out["index"] = ch.Index
	}
	if ch.Text != "" {
		out["text"] = ch.Text
	}
	if ch.ID != "" {
		out["id"] = ch.ID
	}
	if ch.Name != "" {
		out["name"] = ch.Name
	}
	if ch.ArgumentsDelta != "" {
		out["argumentsDelta"] = ch.ArgumentsDelta
	}
	return out
}

// toDshParts 把 parts 收成 dsh 片段。图片地址放在 URL。
func toDshParts(parts []runtime.ContentPart) []dsh.Part {
	if len(parts) == 0 {
		return nil
	}
	out := make([]dsh.Part, 0, len(parts))
	for _, part := range parts {
		out = append(out, dsh.Part{Type: part.Type, Text: part.Text, URL: part.ImageURL, MIME: part.MIME})
	}
	return out
}
