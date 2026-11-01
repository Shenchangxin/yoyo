// project.go 把 dsh 日志投影成界面已经认识的轨迹事件。
// 历史是播种进去的，不重复发出。来源为空的用户消息已经由发送方落盘，这里也不发。

package dshdesk

import (
	"fmt"
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

// emitDsh 把 ev 译成一条轨迹。系统提示、请求头和来源为空的用户消息不发。来源为 job 的完成写回工具结果。
func emitDsh(turn Turn, ev dsh.Event) {
	switch ev.Type {
	case dsh.EventTurnStart:
		turn.emit(trace.Event{Type: trace.TypeTurnStart, Source: "dsh", Payload: map[string]any{"turn": ev.Turn}}, true)
	case dsh.EventTurnEnd:
		end, _ := ev.Data.(dsh.TurnEnd)
		turn.emit(trace.Event{Type: trace.TypeTurnEnd, Source: "dsh", Payload: map[string]any{
			"turn": ev.Turn, "kind": end.Kind, "reason": end.Reason,
		}}, true)
	case dsh.EventStepStart:
		turn.emit(trace.Event{Type: trace.TypeStepStart, Source: "dsh", Payload: map[string]any{
			"turn": ev.Turn, "step": ev.Step,
		}}, true)
	case dsh.EventStepEnd:
		turn.emit(trace.Event{Type: trace.TypeStepEnd, Source: "dsh", Payload: map[string]any{
			"turn": ev.Turn, "step": ev.Step,
		}}, true)
	case dsh.EventUser:
		msg, _ := ev.Data.(dsh.UserMessage)
		if msg.Source == "job" {
			id, body, ok := jobCompletion(msg)
			if !ok {
				return
			}
			// 后台命令的输出写回原来的工具结果。模型仍从会话日志里读到这条完成，界面不再把它当成一条新消息。
			turn.emit(trace.Event{Type: trace.TypeToolResult, Source: "tool", Payload: map[string]any{
				"id": id, "content": body, "ok": true,
			}}, true)
			return
		}
		if msg.Source != "steer" {
			return
		}
		turn.emit(trace.Event{Type: trace.TypeUser, Source: msg.Source, Payload: map[string]any{
			"text": msg.Text, "name": msg.Source,
		}}, true)
	case dsh.EventAssistant:
		msg, _ := ev.Data.(dsh.Assistant)
		if strings.TrimSpace(msg.Text) == "" && msg.AttemptID == "" {
			return
		}
		payload := map[string]any{}
		if msg.Text != "" {
			payload["text"] = msg.Text
		}
		if msg.AttemptID != "" {
			payload["id"] = msg.AttemptID
			payload["round"] = msg.AttemptID
			payload["attempt_id"] = msg.AttemptID
			payload["turn"] = ev.Turn
			payload["step"] = ev.Step
		}
		if msg.Interrupted {
			payload["interrupted"] = true
		}
		turn.emit(trace.Event{Type: trace.TypeAssistant, Source: "model", Payload: payload}, true)
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

// wireStream 让 bridge 把每一次模型调用发成独立的流。帧序号按块递增，不复用内容块下标。不写入 JSONL。
// 同一回合里下一次调用会重新发开始帧，界面才能接着画，而不是把后一段丢在已经结束的尝试上。
func wireStream(bridge *Bridge, turn Turn) {
	if bridge == nil {
		return
	}
	n := 0
	bridge.OpenStream = func(modelTurn, step int) (string, func(runtime.StreamChunk), func(string)) {
		n++
		id := streamAttemptID(turn, modelTurn, step, n)
		rev := 0
		started := false
		closed := false
		ensure := func() {
			if started {
				return
			}
			started = true
			rev = 1
			emitStream(turn, map[string]any{
				"kind": runtime.StreamFrameStart, "attempt_id": id, "revision": rev,
				"turn": modelTurn, "step": step, "round": id, "id": id,
			})
		}
		emit := func(ch runtime.StreamChunk) {
			if closed {
				return
			}
			if ch.Type == runtime.ChunkFinish {
				ensure()
				closed = true
				rev++
				emitStream(turn, map[string]any{
					"kind": runtime.StreamFrameEnd, "attempt_id": id, "revision": rev,
					"turn": modelTurn, "step": step, "round": id, "id": id,
					"outcome": streamOutcome(ch),
				})
				return
			}
			ensure()
			rev++
			idx := rev - 2
			if idx < 0 {
				idx = 0
			}
			emitStream(turn, map[string]any{
				"kind": runtime.StreamFrameChunk, "attempt_id": id, "revision": rev,
				"index": idx, "turn": modelTurn, "step": step, "round": id, "id": id,
				"chunk": chunkPayload(ch),
			})
		}
		end := func(outcome string) {
			if !started || closed {
				return
			}
			closed = true
			if outcome == "" {
				outcome = runtime.StreamOutcomeCommitted
			}
			rev++
			emitStream(turn, map[string]any{
				"kind": runtime.StreamFrameEnd, "attempt_id": id, "revision": rev,
				"turn": modelTurn, "step": step, "round": id, "id": id,
				"outcome": outcome,
			})
		}
		return id, emit, end
	}
}

// streamAttemptID 拼出这一次模型调用的流标识。turn 提供回合 id，空则用会话 id。
// modelTurn 和 step 是代理里的回合与步骤。n 是这一轮桌面发送里的第几次调用，从 1 起。
func streamAttemptID(turn Turn, modelTurn, step, n int) string {
	base := turn.TurnID
	if base == "" {
		base = turn.SessionID
	}
	if base == "" {
		base = "turn"
	}
	return fmt.Sprintf("%s:%d:%d:%d", base, modelTurn, step, n)
}

// streamOutcome 从结束块读结果。ch 的原因是 aborted 或 error 时为 abandoned，其余为 committed。
func streamOutcome(ch runtime.StreamChunk) string {
	if ch.Reason != nil {
		switch ch.Reason.Kind {
		case runtime.FinishAborted, runtime.FinishError:
			return runtime.StreamOutcomeAbandoned
		}
	}
	return runtime.StreamOutcomeCommitted
}

// emitStream 发出一帧助手流。payload 原样放进载荷。不写入 JSONL。
func emitStream(turn Turn, payload map[string]any) {
	turn.emit(trace.Event{Type: trace.TypeAssistantStream, Source: "model", Payload: payload}, false)
}

// recordedTurn 读出轨迹里已经记下的最大回合号。store 或 sessionID 为空、读失败时返回 0。
// 新进程的代理用它接着编号，避免下一句又从 1 开始。
func recordedTurn(store *trace.Store, sessionID string) int {
	if store == nil || sessionID == "" {
		return 0
	}
	evs, err := store.Read(sessionID)
	if err != nil {
		return 0
	}
	max := 0
	for _, ev := range evs {
		if ev.Type != trace.TypeTurnStart && ev.Type != trace.TypeTurnEnd {
			continue
		}
		n := numberOf(ev.Payload["turn"])
		if n > max {
			max = n
		}
	}
	return max
}

// numberOf 把轨迹载荷里的数字收成 int。JSON 读回来是 float64。不是数字时返回 0。
func numberOf(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// jobCompletion 从一条后台完成里取出工具调用 id 和输出。msg 的正文要以 job 加空格开头，并含有 finished。对不上时 ok 为假。
func jobCompletion(msg dsh.UserMessage) (id, body string, ok bool) {
	const head = "job "
	text := msg.Text
	if !strings.HasPrefix(text, head) {
		return "", "", false
	}
	rest := text[len(head):]
	const marker = " finished"
	i := strings.Index(rest, marker)
	if i <= 0 {
		return "", "", false
	}
	id = strings.TrimSpace(rest[:i])
	if id == "" || strings.ContainsAny(id, " \t\r\n") {
		return "", "", false
	}
	body = strings.TrimPrefix(rest[i+len(marker):], "\n")
	body = strings.TrimRight(body, "\r\n")
	if strings.TrimSpace(body) == "" {
		body = "finished"
	}
	return id, body, true
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
