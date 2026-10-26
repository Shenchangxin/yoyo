// 工具钩子的内核事件名，以及工具前后的 ToolHook 载荷。
// 纤维用 Serial（先返回错误者否决）或 Waterfall（改写载荷）订阅。TCB 不得注册能关掉 eval、vault 或 journal 的监听器。
// 不读 .yoyo/hooks.json，那在 hooks_file.go。loop.go 在工具前后调用 applyPreTool 和 applyPostTool。

package runtime

import "github.com/Shenchangxin/yoyo/internal/kernel"

const (
	// HookPreTool 是工具执行前的事件名 agent.pre_tool。
	HookPreTool = "agent.pre_tool"
	// HookPostTool 是工具执行后的事件名 agent.post_tool。
	HookPostTool = "agent.post_tool"
	// HookCompact 是压缩事件名 agent.compact。
	HookCompact = "agent.compact"
	// HookPreCompact 是压缩前事件名 agent.pre_compact。
	HookPreCompact = "agent.pre_compact"
	// HookTurnEnd 是回合结束事件名 agent.turn_end。
	HookTurnEnd = "agent.turn_end"
	// HookStop 是停止事件名 agent.stop。
	HookStop = "agent.stop"
	// HookSessionStart 是会话开始事件名 agent.session_start。
	HookSessionStart = "agent.session_start"
)

// ToolHook 是工具前后钩子的载荷。Deny 为真时 loop 不执行该工具。
type ToolHook struct {
	Name      string
	Arguments string
	SessionID string
	Result    string
	Deny      bool
	Reason    string
}

// applyPreTool 用 Serial 跑 HookPreTool。总线为空则原样返回。总线返回错误时把 Deny 设为真，Reason 用该错误。返回值不是 ToolHook 时沿用入参。
func applyPreTool(bus *kernel.EventBus, hook ToolHook) ToolHook {
	if bus == nil {
		return hook
	}
	out, err := bus.Serial(HookPreTool, hook)
	if err != nil {
		hook.Deny = true
		hook.Reason = err.Error()
		return hook
	}
	if h, ok := out.(ToolHook); ok {
		return h
	}
	return hook
}

// applyPostTool 用 Waterfall 跑 HookPostTool。总线为空、出错、结果为空或类型不对时沿用入参。
func applyPostTool(bus *kernel.EventBus, hook ToolHook) ToolHook {
	if bus == nil {
		return hook
	}
	out, err := bus.Waterfall(HookPostTool, hook)
	if err != nil || out == nil {
		return hook
	}
	if h, ok := out.(ToolHook); ok {
		return h
	}
	return hook
}
