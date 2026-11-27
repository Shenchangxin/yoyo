// 输出 token 被截断时的回复修剪。Run 在结算前调用 trimTruncatedReply，丢掉参数不完整的工具调用。
// 不重试模型。被截断的调用不会执行。

package runtime

import "encoding/json"

// writeFilePartMax 是单次写入正文的硬上限，32KB。12KB 会拒绝大约 13KB、仍能一次写完的完整草稿。32KB 仍会拆开 35KB 以上被截断的转储。
const writeFilePartMax = 32 * 1024

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
