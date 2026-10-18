package runtime

import (
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// streamAttempt is one model call's live identity and assembled settlement.
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

func emitTurnStart(req RunRequest, turn int) {
	payload := map[string]any{"turn": turn}
	if req.TurnID != "" {
		payload["turn_id"] = req.TurnID
	}
	emit(req, trace.TypeTurnStart, "runtime", payload)
}

func emitStepStart(req RunRequest, turn, step int, roundID, attemptID string) {
	emit(req, trace.TypeStepStart, "runtime", map[string]any{
		"turn": turn, "step": step, "round": roundID, "id": roundID, "attempt_id": attemptID,
	})
}

func emitStepEnd(req RunRequest, turn, step int, roundID string) {
	emit(req, trace.TypeStepEnd, "runtime", map[string]any{
		"turn": turn, "step": step, "round": roundID, "id": roundID,
	})
}

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

func emitStreamChunk(req RunRequest, att *streamAttempt, ch StreamChunk) {
	if att == nil {
		return
	}
	att.Revision++
	idx := att.Revision - 2 // 0-based dense index after start's revision 1
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

func nowMillis() int64 {
	return time.Now().UnixMilli()
}
