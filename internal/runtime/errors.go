package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

const (
	ErrKindOverflow  = "overflow"
	ErrKindAuth      = "auth"
	ErrKindRateLimit = "rate_limit"
	ErrKindInvalid   = "invalid"
	ErrKindTimeout   = "timeout"
	ErrKindCanceled  = "canceled"
	ErrKindProvider  = "provider"
	ErrKindBudget    = "budget"
	ErrKindMaxTurns  = "max_turns"
	ErrKindStuck     = "stuck"
	ErrKindModel     = "model"
	ErrKindUnknown   = "unknown"
)

// maxTurnsInfo is the terminal card for a loop that ran out of turns. It is
// retryable: "continue this turn" resumes from history without a new user
// message, which is exactly what an operator wants after a long tool run.
func maxTurnsInfo(limit int) ErrorInfo {
	return ErrorInfo{
		Kind:      ErrKindMaxTurns,
		Title:     "Reached the turn limit",
		Hint:      "The loop stopped before the model finished. Continue this turn to keep going.",
		Detail:    "max turns reached (" + strconv.Itoa(limit) + ")",
		Retryable: true,
	}
}

// stuckLoopInfo is the terminal card for a SoftHorizon loop that kept
// repeating the same failure, or kept the same check FAIL, after a stall
// nudge. Retryable: the operator can steer or continue with a different
// approach.
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

// ErrorInfo is a UI-facing projection of a provider/runtime failure.
type ErrorInfo struct {
	Kind      string
	Title     string
	Hint      string
	Detail    string
	Retryable bool
}

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

// FailoverWorthy is true when another Settings provider or model may succeed.
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

// FailoverSkipProvider is true when the rest of this provider's models will
// fail the same way (bad key, rate limit, dead endpoint).
func FailoverSkipProvider(err error) bool {
	switch ClassifyError(err).Kind {
	case ErrKindAuth, ErrKindRateLimit, ErrKindTimeout, ErrKindProvider:
		return true
	default:
		return false
	}
}

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

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
