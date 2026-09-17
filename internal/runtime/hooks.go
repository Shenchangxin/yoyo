package runtime

import "github.com/Shenchangxin/yoyo/internal/kernel"

// Kernel event names. Fibers subscribe with Serial (first veto wins) or
// Waterfall (rewrite). The TCB must never register listeners that can
// disable eval, vault, or the journal.
const (
	HookPreTool  = "agent.pre_tool"
	HookPostTool = "agent.post_tool"
	HookCompact  = "agent.compact"
	HookTurnEnd  = "agent.turn_end"
	HookStop     = "agent.stop"
)

// ToolHook is the payload for pre/post tool hooks.
type ToolHook struct {
	Name      string
	Arguments string
	SessionID string
	Result    string
	Deny      bool
	Reason    string
}

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
