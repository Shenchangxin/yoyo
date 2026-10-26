// 循环停止原因和 StopError。Run 用 setStop 写回调用方传入的指针，用 stopErr 作为返回值。
// 错误卡片的分类在 errors.go。

package runtime

import "fmt"

// StopReason 是循环写进轨迹和 req.Stop 的停止原因。
type StopReason string

const (
	// StopEndTurn 的值是 end_turn，模型没有再调用工具并结束这一轮。
	StopEndTurn StopReason = "end_turn"
	// StopMaxTurns 的值是 max_turns。硬循环用尽 MaxTurns，或 softGuard 判定停止时，Run 都写这个原因。
	StopMaxTurns StopReason = "max_turns"
	// StopMaxTokens 的值是 max_tokens。本文件只定义它，Run 的停止路径没有把它写进 req.Stop。
	StopMaxTokens StopReason = "max_tokens"
	// StopBudget 的值是 budget。Meter.Check 失败时 Run 写这个原因。
	StopBudget StopReason = "budget"
	// StopCancelled 的值是 cancelled。上下文取消时 Run 写这个原因。
	StopCancelled StopReason = "cancelled"
	// StopOverflow 的值是 overflow。上下文溢出的当前步重试仍然溢出时，Run 写这个原因。
	StopOverflow StopReason = "overflow"
	// StopHook 的值是 hook_stop。本文件只定义它。applyStopHook 阻止结束时是把文案追加进消息并继续循环，不返回这个原因。
	StopHook StopReason = "hook_stop"
	// StopModelError 的值是 model_error。模型调用失败且不是溢出或取消时，Run 写这个原因。ReasonOf 对非 StopError 也返回它。
	StopModelError StopReason = "model_error"
	// StopStuck 的值是 stuck。SoftHorizon 上观察没有变化、或同类失败达到停止次数时，Run 写这个原因。
	StopStuck StopReason = "stuck"
	// StopPlanOpen 的值是 plan_open。聊天回合在催促用尽后仍有未完成计划步骤时，Run 写这个原因，并在 turn_end 之后另发一张卡片。
	StopPlanOpen StopReason = "plan_open"
)

// StopError 是带原因的循环错误。Msg 非空时 Error 只返回 Msg。
type StopError struct {
	Reason StopReason
	Msg    string // 非空时 Error 只返回它，不再看 Err
	Err    error
}

// Error 在接收者为 nil 时返回空串。Msg 非空时返回 Msg，否则返回 Err 的文本；Err 也是 nil 时返回 Reason 的字符串。
func (e *StopError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Reason)
}

// Unwrap 返回内部错误。接收者为 nil 时返回 nil。
func (e *StopError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// stopErr 包一层 StopError，Msg 和 Err 都按参数原样放进去。
// reason 放进 StopError 的 Reason。
// msg 放进 StopError 的 Msg。
// err 放进 StopError 的 Err。
func stopErr(reason StopReason, msg string, err error) error {
	return &StopError{Reason: reason, Msg: msg, Err: err}
}

// ReasonOf 在 err 为 nil 时返回 StopEndTurn。err 的动态类型正好是 *StopError 时返回它的 Reason。被再包一层的 StopError 认不出来，返回 StopModelError。
func ReasonOf(err error) StopReason {
	if err == nil {
		return StopEndTurn
	}
	if e, ok := err.(*StopError); ok {
		return e.Reason
	}
	return StopModelError
}

// maxTurnsErr 返回 StopMaxTurns，Msg 和内部错误文本都是 max turns reached。
func maxTurnsErr() error {
	return stopErr(StopMaxTurns, "max turns reached", fmt.Errorf("max turns reached"))
}

// stuckLoopErr 返回 StopStuck。detail 为空时改成 the same tool failure kept repeating。
func stuckLoopErr(detail string) error {
	if detail == "" {
		detail = "the same tool failure kept repeating"
	}
	return stopErr(StopStuck, detail, fmt.Errorf("%s", detail))
}
