// 一次模型调用的现场身份，以及回合、步骤、流式帧和结算的轨迹事件。
// 不组装正文，那在 assembler.go。loop.go 在回合和步骤边界调用这里。

package runtime

import (
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// streamAttempt 是一次模型调用的现场身份和已组装的结算材料。
// Revision 从开始帧的 1 起递增。
type streamAttempt struct {
	ID          string
	Turn        int
	Step        int
	RoundID     string
	Blocks      []ContentBlock
	Stream      []AssistantStreamRecord
	Interrupted bool
	Revision    int
}

// emitTurnStart 发出回合开始。TurnID 非空时写入载荷。
// req 传给 emit，TurnID 非空时写入 turn_id。
// turn 写入载荷的 turn。
func emitTurnStart(req RunRequest, turn int) {
	payload := map[string]any{"turn": turn}
	if req.TurnID != "" {
		payload["turn_id"] = req.TurnID
	}
	emit(req, trace.TypeTurnStart, "runtime", payload)
}

// emitStepStart 发出步骤开始，round 与 id 都写成 roundID。
// req 传给 emit。
// turn 写入载荷的 turn。
// step 写入载荷的 step。
// attemptID 写入载荷的 attempt_id。
func emitStepStart(req RunRequest, turn, step int, roundID, attemptID string) {
	emit(req, trace.TypeStepStart, "runtime", map[string]any{
		"turn": turn, "step": step, "round": roundID, "id": roundID, "attempt_id": attemptID,
	})
}

// emitStepEnd 发出步骤结束，round 与 id 都写成 roundID。
// req 传给 emit。
// turn 写入载荷的 turn。
// step 写入载荷的 step。
func emitStepEnd(req RunRequest, turn, step int, roundID string) {
	emit(req, trace.TypeStepEnd, "runtime", map[string]any{
		"turn": turn, "step": step, "round": roundID, "id": roundID,
	})
}

// emitAttemptStart 发出流开始帧，并把 Revision 设为 1。att 为空则返回。
// req 传给 emit。
func emitAttemptStart(req RunRequest, att *streamAttempt) {
	if att == nil {
		return
	}
	emit(req, trace.TypeAssistantStream, "model", map[string]any{
		"kind":       StreamFrameStart,
		"attempt_id": att.ID,
		"revision":   1,
		"turn":       att.Turn,
		"step":       att.Step,
		"round":      att.RoundID,
		"id":         att.RoundID,
	})
	att.Revision = 1
}

// emitStreamChunk 把一块流式增量记成助手流事件。序号在开始帧的 revision 1 之后从 0 连续编号。
// req 传给 emit。
// att 为 nil 时返回，否则递增 Revision，并把 ID、Turn、Step、RoundID 写入载荷。
// ch 交给 chunkMap，结果写入载荷的 chunk。
func emitStreamChunk(req RunRequest, att *streamAttempt, ch StreamChunk) {
	if att == nil {
		return
	}
	att.Revision++
	idx := att.Revision - 2 // 开始帧占了 revision 1，第一块的下标是 0。
	if idx < 0 {
		idx = 0
	}
	emit(req, trace.TypeAssistantStream, "model", map[string]any{
		"kind":       StreamFrameChunk,
		"attempt_id": att.ID,
		"revision":   att.Revision,
		"index":      idx,
		"turn":       att.Turn,
		"step":       att.Step,
		"round":      att.RoundID,
		"id":         att.RoundID,
		"chunk":      chunkMap(ch),
	})
}

// emitAttemptEnd 发出流结束帧，outcome 原样写入载荷。
// req 传给 emit。
// att 为 nil 时返回，否则递增 Revision，并把 ID、Turn、Step、RoundID 写入载荷。
func emitAttemptEnd(req RunRequest, att *streamAttempt, outcome string) {
	if att == nil {
		return
	}
	att.Revision++
	emit(req, trace.TypeAssistantStream, "model", map[string]any{
		"kind":       StreamFrameEnd,
		"attempt_id": att.ID,
		"revision":   att.Revision,
		"turn":       att.Turn,
		"step":       att.Step,
		"round":      att.RoundID,
		"id":         att.RoundID,
		"outcome":    outcome,
	})
}

// emitSettlement 发出助手结算。没有正文、块和流记录时不发。正文为空时用块里的文本。
// req 传给 emit。
// att 为 nil 时不写回合字段；与 msg 都没有可写的正文、块和流记录时不发。
// msg 的 Content 作为正文，为空且 att 非 nil 时改用块里的文本；ToolCalls 只参与 att 为 nil 时是否提前返回。
func emitSettlement(req RunRequest, att *streamAttempt, msg Message) {
	if att == nil && msg.Content == "" && len(msg.ToolCalls) == 0 {
		return
	}
	payload := map[string]any{}
	roundID := ""
	if att != nil {
		roundID = att.RoundID
		payload["id"] = att.RoundID
		payload["round"] = att.RoundID
		payload["attempt_id"] = att.ID
		payload["turn"] = att.Turn
		payload["step"] = att.Step
		if len(att.Blocks) > 0 {
			payload["blocks"] = att.Blocks
		}
		if recs := recordsAsAny(att.Stream); len(recs) > 0 {
			payload["stream"] = recs
		}
		if att.Interrupted {
			payload["interrupted"] = true
		}
	}
	text := msg.Content
	if text == "" && att != nil {
		text = textFromBlocks(att.Blocks)
	}
	if text != "" {
		payload["text"] = text
	}
	if roundID != "" {
		if _, ok := payload["id"]; !ok {
			payload["id"] = roundID
		}
	}
	if payload["text"] == nil && payload["blocks"] == nil && payload["stream"] == nil {
		return
	}
	emit(req, trace.TypeAssistant, "model", payload)
}

// nowMillis 返回当前 Unix 毫秒。
func nowMillis() int64 {
	return time.Now().UnixMilli()
}
