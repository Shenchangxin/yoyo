package runtime

import "encoding/json"

// outputCutNudge is sent when a reply stopped at the output-token limit. It
// names the protocol the model should use instead of retrying the same blob.
const outputCutNudge = "Your last reply hit the output limit before it finished, and any tool call cut off at that point was not run. Continue from where it stopped. For a file write, split the body: write_file with mode=create (or overwrite) for the first part and mode=append for each following part, at most 32 KB per call (hard limit)."

// writeFilePartMax is the hard per-call body cap. 12 KB refused complete
// SOURCE_BIBLE / GAME_DESIGN drafts (~13 KB) that this model's 16k output
// cap can emit in one shot. 32 KB still splits a truncated 35 KB+ dump.
const writeFilePartMax = 32 * 1024

// trimTruncatedReply keeps only the tool calls whose arguments are complete.
// A call cut off by the output limit is never executed, and its partial text is
// dropped so it cannot be mistaken for file content. The returned nudge is
// empty when the reply finished normally.
func trimTruncatedReply(msg Message) (Message, string) {
	if !msg.Truncated {
		return msg, ""
	}
	kept := make([]ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		if toolArgsTruncated(tc.Arguments) {
			continue
		}
		kept = append(kept, tc)
	}
	msg.ToolCalls = kept
	return msg, outputCutNudge
}

// isOutputLimitStop reports whether a provider finish reason means the reply
// was cut at the output-token cap rather than ended by the model.
func isOutputLimitStop(reason string) bool {
	switch reason {
	case "length", "max_tokens":
		return true
	default:
		return false
	}
}

// streamFinishReason extracts choices[0].finish_reason from one SSE data
// payload. Callers only parse chunks that mention finish_reason.
func streamFinishReason(data string) string {
	var chunk struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil || len(chunk.Choices) == 0 {
		return ""
	}
	return chunk.Choices[0].FinishReason
}
