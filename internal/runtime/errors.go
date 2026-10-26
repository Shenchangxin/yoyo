// 把模型和循环的失败收成界面卡片。Run 在停止前调用 ClassifyError。
// 不重试请求。是否换供应商由 FailoverWorthy 和 FailoverSkipProvider 回答。

package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

const (
	// ErrKindOverflow 是上下文溢出。卡片可重试。
	ErrKindOverflow = "overflow"
	// ErrKindAuth 是供应商拒绝了密钥。卡片不可重试。
	ErrKindAuth = "auth"
	// ErrKindRateLimit 是限流。卡片可重试。
	ErrKindRateLimit = "rate_limit"
	// ErrKindInvalid 是供应商拒绝了这一轮请求。卡片可重试。
	ErrKindInvalid = "invalid"
	// ErrKindTimeout 是超时。卡片可重试。
	ErrKindTimeout = "timeout"
	// ErrKindCanceled 是这一轮被打断。卡片可重试。
	ErrKindCanceled = "canceled"
	// ErrKindProvider 是供应商连接或端点失败。卡片可重试。
	ErrKindProvider = "provider"
	// ErrKindBudget 是会话预算用尽。卡片不可重试。
	ErrKindBudget = "budget"
	// ErrKindMaxTurns 是轮数用尽。卡片可重试。
	ErrKindMaxTurns = "max_turns"
	// ErrKindStuck 是重复循环被停住。卡片可重试。
	ErrKindStuck = "stuck"
	// ErrKindPlanOpen 是计划还有步骤没做完。卡片可重试。
	ErrKindPlanOpen = "plan_open"
	// ErrKindModel 是当前模型不可用。卡片可重试。
	ErrKindModel = "model"
	// ErrKindUnknown 是没有归进上面种类的失败。nil 错误也用它，卡片可重试。
	ErrKindUnknown = "unknown"
)

// maxTurnsInfo 是轮数用尽的终局卡片，可重试。继续这一轮会从历史接着走，不再另起一条用户消息。
func maxTurnsInfo(limit int) ErrorInfo {
	return ErrorInfo{
		Kind:      ErrKindMaxTurns,
		Title:     "Reached the turn limit",
		Hint:      "The loop stopped before the model finished. Continue this turn to keep going.",
		Detail:    "max turns reached (" + strconv.Itoa(limit) + ")",
		Retryable: true,
	}
}

// stuckLoopInfo 是 SoftHorizon 上同一失败或同一条检查 FAIL 停住后的卡片，可重试。detail 去掉空白后为空时，改成 the same tool failure kept repeating。
func stuckLoopInfo(detail string) ErrorInfo {
	if strings.TrimSpace(detail) == "" {
		detail = "the same tool failure kept repeating"
	}
	return ErrorInfo{
		Kind:      ErrKindStuck,
		Title:     "Stopped a repeating loop",
		Hint:      "The observation did not change. Edit the product the check cites, or change approach, then continue this turn.",
		Detail:    detail,
		Retryable: true,
	}
}

// planOpenInfo 是聊天回合停住时计划仍有步骤未完成的卡片，可重试。催促次数用尽，或催促之后模型改用散文回答，都会走到这里。没有这张卡片时，这一轮看起来像任务已经做完。
func planOpenInfo(open, total int) ErrorInfo {
	return ErrorInfo{
		Kind:      ErrKindPlanOpen,
		Title:     "Plan not finished",
		Hint:      "The agent stopped before its plan was done. Continue this turn to resume the open steps.",
		Detail:    strconv.Itoa(open) + " of " + strconv.Itoa(total) + " plan steps still open",
		Retryable: true,
	}
}

// ErrorInfo 是给界面的失败投影。Retryable 为零值 false 时，卡片不可重试。
type ErrorInfo struct {
	Kind      string
	Title     string
	Hint      string
	Detail    string
	Retryable bool
}

// Payload 把卡片收成 error、detail、kind、title、hint、retryable。error 和 title 都是 Title。
func (e ErrorInfo) Payload() map[string]any {
	return map[string]any{
		"error":     e.Title,
		"detail":    e.Detail,
		"kind":      e.Kind,
		"title":     e.Title,
		"hint":      e.Hint,
		"retryable": e.Retryable,
	}
}

// ClassifyError 把失败收成界面卡片。err 为 nil 时种类是 unknown，标题是 Something went wrong，可重试。
// 先认 StopStuck。然后区分上下文超时和取消。ErrBudget 不可重试。其余按错误文本的小写匹配，先命中的分支生效。
// 上下文溢出可重试。鉴权失败不可重试。被切断的流是供应商连接中断，可重试，不会落成 unknown。文本里的 budget exceeded 同样不可重试。
func ClassifyError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{Kind: ErrKindUnknown, Title: "Something went wrong", Retryable: true}
	}
	var stuck *StopError
	if errors.As(err, &stuck) && stuck.Reason == StopStuck {
		return stuckLoopInfo(stuck.Error())
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrorInfo{
				Kind: ErrKindTimeout, Title: "Request timed out",
				Hint: "The model did not respond in time.", Detail: err.Error(), Retryable: true,
			}
		}
		return ErrorInfo{
			Kind: ErrKindCanceled, Title: "Stopped",
			Hint: "This turn was interrupted.", Detail: err.Error(), Retryable: true,
		}
	}
	var budget ErrBudget
	if errors.As(err, &budget) {
		return ErrorInfo{
			Kind: ErrKindBudget, Title: "Budget reached",
			Hint:   "Raise the session budget in Settings, or start a new chat.",
			Detail: err.Error(), Retryable: false,
		}
	}
	raw := err.Error()
	detail := extractProviderMsg(raw)
	low := strings.ToLower(raw + " " + detail)
	info := ErrorInfo{Kind: ErrKindUnknown, Title: "Something went wrong", Hint: "Retry the last message.", Detail: truncate(detail, 1200), Retryable: true}

	switch {
	case IsContextOverflow(err):
		info.Kind, info.Title, info.Hint = ErrKindOverflow, "Context too large", "Yoyo will compact on the next send. If this persists, run /compact or start a new chat."
		info.Retryable = true
	case containsAny(low, "invalid_api_key", "incorrect api key", "unauthorized", "401", "authentication", "invalid token", "expired token", "invalid or expired"):
		info.Kind, info.Title, info.Hint = ErrKindAuth, "Provider rejected the key", "Check the API key and base URL in Settings."
		info.Retryable = false
	case containsAny(low, "rate limit", "rate_limit", "too many requests", "429"):
		info.Kind, info.Title, info.Hint = ErrKindRateLimit, "Rate limited", "Wait a moment, then retry."
		info.Retryable = true
	case containsAny(low, "model_not_found", "model not found", "unknown model", "invalid model", "no such model", "model does not exist"):
		info.Kind, info.Title, info.Hint = ErrKindModel, "This model is not available", "Pick another model in the composer, or add it on the provider in Settings."
		info.Retryable = true
	case containsAny(low, "tool_calls", "invalid_parameter", "invalid_request", "unrecognized request argument"):
		info.Kind, info.Title, info.Hint = ErrKindInvalid, "The provider rejected this turn", "Usually a malformed tool history. Retry; start a new chat if it repeats."
		info.Retryable = true
	case containsAny(low, "timeout", "deadline exceeded", "i/o timeout"):
		info.Kind, info.Title, info.Hint = ErrKindTimeout, "Request timed out", "The model did not respond in time."
		info.Retryable = true
	case containsAny(low, "unexpected eof", "connection reset by peer", "broken pipe", "http2: response body closed", "stream error"):
		// 流被切断是传输失败。不单列这一支时，卡片只会写成 Something went wrong，看不出原因。
		info.Kind, info.Title, info.Hint = ErrKindProvider, "The model connection dropped", "The model stopped mid-answer. Retry the turn."
		info.Retryable = true
	case containsAny(low, "connection refused", "no such host", "network is unreachable", "dial tcp"):
		info.Kind, info.Title, info.Hint = ErrKindProvider, "The model endpoint failed", "The provider returned an error. Retry in a moment."
		info.Retryable = true
	case containsAny(low, "all_channel_models_failed", "internal server error", "502", "503", "504", "overloaded"):
		info.Kind, info.Title, info.Hint = ErrKindProvider, "The model endpoint failed", "The provider returned an error. Retry in a moment."
		info.Retryable = true
	case containsAny(low, "budget exceeded"):
		info.Kind, info.Title, info.Hint = ErrKindBudget, "Budget reached", "Raise the session budget in Settings, or start a new chat."
		info.Retryable = false
	}
	return info
}

// FailoverWorthy 判断换一个设置里的供应商或模型是否还可能成功。err 为 nil，或是上下文取消时，返回 false。
// 种类是溢出、预算、取消、非法请求、轮数用尽或卡住时返回 false。超时不在这张表里，返回 true。
func FailoverWorthy(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	switch ClassifyError(err).Kind {
	case ErrKindOverflow, ErrKindBudget, ErrKindCanceled, ErrKindInvalid, ErrKindMaxTurns, ErrKindStuck:
		return false
	default:
		return true
	}
}

// FailoverSkipProvider 在这个供应商剩下的模型会以同样方式失败时返回 true：鉴权、限流、超时、供应商错误。其他种类返回 false。err 为 nil 时种类是 unknown，返回 false。
func FailoverSkipProvider(err error) bool {
	switch ClassifyError(err).Kind {
	case ErrKindAuth, ErrKindRateLimit, ErrKindTimeout, ErrKindProvider:
		return true
	default:
		return false
	}
}

// extractProviderMsg 从第一个 { 到最后一个 } 里取供应商文案。按 msg、message、error 的顺序，字符串取非空白的值，对象再取里面的 message。解析失败时返回原文。
func extractProviderMsg(s string) string {
	i := strings.Index(s, "{")
	if i < 0 {
		return s
	}
	raw := s[i:]
	if j := strings.LastIndex(raw, "}"); j >= 0 {
		raw = raw[:j+1]
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return s
	}
	for _, k := range []string{"msg", "message", "error"} {
		switch v := m[k].(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return v
			}
		case map[string]any:
			if msg, _ := v["message"].(string); strings.TrimSpace(msg) != "" {
				return msg
			}
		}
	}
	return s
}

// containsAny 在 s 含有任一 needle 时返回 true。
func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
