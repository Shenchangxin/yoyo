// 输出 token 被截断时的回复修剪。Run 在结算前调用 trimTruncatedReply，丢掉参数不完整的工具调用。
// 不重试模型。被截断的调用不会执行。

package runtime

import "encoding/json"

// outputCutNudge 在回复停在输出 token 上限时发给模型。它点名应使用的写文件协议，而不是把同一大段再试一次。
const outputCutNudge = "Your last reply hit the output limit before it finished, and any tool call cut off at that point was not run. Continue from where it stopped. For a file write, split the body: write_file with mode=create (or overwrite) for the first part and mode=append for each following part, at most 32 KB per call (hard limit)."

// writeFilePartMax 是单次写入正文的硬上限，32KB。12KB 会拒绝大约 13KB、仍能一次写完的完整草稿。32KB 仍会拆开 35KB 以上被截断的转储。
const writeFilePartMax = 32 * 1024

// trimTruncatedReply 只留下参数完整的工具调用。回复未截断时原样返回，催促文案为空。被输出上限切断的调用不执行，半截文本也不当文件内容留下。
// msg 的 Truncated 为假时原样返回；为真时丢掉 toolArgsTruncated 判定为不完整的工具调用，并带上 outputCutNudge。
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

// isOutputLimitStop 判断供应商的结束原因是不是输出 token 上限。只有 length 和 max_tokens 返回 true。
// reason 等于 length 或 max_tokens 时返回 true。
func isOutputLimitStop(reason string) bool {
	switch reason {
	case "length", "max_tokens":
		return true
	default:
		return false
	}
}

// streamFinishReason 从一条 SSE data 里取出 choices[0].finish_reason。JSON 解析失败或 choices 为空时返回空串。
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
