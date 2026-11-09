// worker_schema.go 拥有五个 orchestrator_* 工具的参数 JSON schema，只给外部工人的 MCP 控制面用。
// 不拥有工具本身（executor.go、worker_tools.go），dsh 工人的工具定义不变。app 把它和 WorkerSpec.Tools 一起交给 mcpserve。

package superagent

// ReportStates 是 orchestrator_report 的 state 可取的值，和 Engine.Report 的校验一致。
var ReportStates = []string{"progress", "waiting", "ready", "blocked", "failed", "dropped"}

// ReplyPurposes 是 orchestrator_reply 的 purpose 可取的值，和 Engine.Reply 的校验一致。
var ReplyPurposes = []string{"acknowledgement", "answer", "result", "attention"}

// WorkerToolSchemas 返回每个 orchestrator_* 工具的参数 schema，键是工具名。字段与工具里的参数解析一一对应，不许多余字段。
func WorkerToolSchemas() map[string]any {
	return map[string]any{
		"orchestrator_report": objectSchema(map[string]any{
			"state":    enumField("这次报告的状态。progress 报完继续做；ready 只在完成标准达到时用；有 question 时不要用 ready。", ReportStates),
			"summary":  stringField("给人看的一句话，也会留在这条会话末尾。"),
			"question": stringField("只在被硬性挡住时填写：没授权的破坏、说明和工单目录以外的必改路径、缺少密钥或登录。"),
		}, "state", "summary"),
		"orchestrator_reply": objectSchema(map[string]any{
			"purpose": enumField("回复的用途。", ReplyPurposes),
			"trigger": stringField("相关的触发编号。acknowledgement 和 answer 必须填。"),
			"text":    stringField("回复正文。"),
		}, "purpose", "text"),
		"orchestrator_handback": objectSchema(map[string]any{
			"trigger": stringField("要交还的触发编号。"),
			"reason":  stringField("为什么超出本工单范围。"),
		}, "trigger"),
		"orchestrator_follow": objectSchema(map[string]any{
			"thread": stringField("只为上下文跟随的线程编号。"),
		}, "thread"),
		"orchestrator_unrelated": objectSchema(map[string]any{
			"thread": stringField("与本工单无关的线程编号。"),
			"reason": stringField("为什么无关。"),
		}, "thread"),
	}
}

// objectSchema 拼一个对象 schema。required 里的字段必须给，其余字段可选，不许多余字段。
func objectSchema(props map[string]any, required ...string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// stringField 是一个带说明的字符串字段。
func stringField(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

// enumField 是一个只能取 values 之一的字符串字段。
func enumField(desc string, values []string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": values}
}
