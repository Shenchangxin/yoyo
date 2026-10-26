// 把已记录的轨迹折成脚本化模型，供回放。不执行工具，也不写轨迹。

package eval

import (
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// ScriptFromTrace 按事件顺序生成 ScriptedClient。助手文本各成一条消息。连续的工具调用先攒起来，遇到下一条助手文本或结束时合成一条带 ToolCalls 的助手消息。其他事件类型忽略。
// events 按顺序看 Type：TypeAssistant 先收起尚未合成的工具调用，再从 Payload 的 text 做一条助手消息；TypeToolCall 从 Payload 取 id、name、arguments 追加到待合成列表；其他类型忽略。
func ScriptFromTrace(events []trace.Event) *runtime.ScriptedClient {
	var steps []runtime.Message
	var pending []runtime.ToolCall
	flush := func() {
		if len(pending) == 0 {
			return
		}
		steps = append(steps, runtime.Message{Role: runtime.RoleAssistant, ToolCalls: pending})
		pending = nil
	}
	for _, ev := range events {
		switch ev.Type {
		case trace.TypeAssistant:
			flush()
			text, _ := ev.Payload["text"].(string)
			steps = append(steps, runtime.Message{Role: runtime.RoleAssistant, Content: text})
		case trace.TypeToolCall:
			id, _ := ev.Payload["id"].(string)
			name, _ := ev.Payload["name"].(string)
			args, _ := ev.Payload["arguments"].(string)
			pending = append(pending, runtime.ToolCall{ID: id, Name: name, Arguments: args})
		}
	}
	flush()
	return &runtime.ScriptedClient{Steps: steps}
}
