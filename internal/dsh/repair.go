// repair.go 为失败或取消的步骤补上缺失的工具结果。
// 文案与 DeepSeek Harness 的 TOOL_NOT_STARTED、TOOL_OUTCOME_UNKNOWN 一致。

package dsh

const (
	// ToolNotStarted 表示助手消息里有调用，但日志还没记下 tool/call。
	ToolNotStarted = "TOOL_NOT_STARTED"
	// ToolOutcomeUnknown 表示 tool/call 已记下，但没有结果。
	ToolOutcomeUnknown = "TOOL_OUTCOME_UNKNOWN"
	// ToolAbortedBeforeDispatch 表示取消时这次调用还没开始执行。
	ToolAbortedBeforeDispatch = "ABORTED_BEFORE_DISPATCH"
)

const (
	textNotStarted     = "The tool call was interrupted before the Harness recorded it as started. Retry it if it is still needed."
	textOutcomeUnknown = "The tool call was interrupted after it was recorded, but no result was durably recorded. Its outcome is unknown. Decide whether to retry from the tool semantics: retry only if the operation is read-only or idempotent; if it may have side effects, first verify external state or ask the user. Do not retry blindly."
	textAborted        = "Error: tool call aborted before dispatch"
)

// RecoveryResult 是一条要补进日志的工具结果。Code 是上面三个常量之一。
type RecoveryResult struct {
	Call UnansweredCall
	Code string
	Text string
}

// RecoveryResults 为缺结果的调用配上文案。c.Started 为真时用 TOOL_OUTCOME_UNKNOWN。
func RecoveryResults(calls []UnansweredCall) []RecoveryResult {
	out := make([]RecoveryResult, 0, len(calls))
	for _, c := range calls {
		if c.Started {
			out = append(out, RecoveryResult{Call: c, Code: ToolOutcomeUnknown, Text: textOutcomeUnknown})
			continue
		}
		out = append(out, RecoveryResult{Call: c, Code: ToolNotStarted, Text: textNotStarted})
	}
	return out
}
