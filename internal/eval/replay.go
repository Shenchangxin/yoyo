package eval

import (
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// ReplayTools is a scripted model built from a recorded trajectory.
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
