package desktop

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/skillpack"
)

// companionPulseKind 把实时条目映射到桌面宠物通道。user 且 source 不是 steer 时为 loading，steer 返回空。tool_call 为 loading，turn_end 为 done，approval 和 ask_user 为 approval，error 为 error，eval 和 evolve 用同名。system 只在 source 为 update 时返回 update。其他类型返回空。loading 不会变成系统通知。
// typ 为 user 且 source 不是 steer 时返回 loading，source 为 steer 时返回空。tool_call 返回 loading，turn_end 返回 done，approval 和 ask_user 返回 approval，error 返回 error，eval 和 evolve 返回同名。system 只在 source 为 update 时返回 update。其他返回空。
func companionPulseKind(typ, source string) string {
	switch typ {
	case "user":
		if source == "steer" {
			return ""
		}
		return "loading"
	case "tool_call":
		return "loading"
	case "turn_end":
		return "done"
	case "approval", "ask_user":
		return "approval"
	case "error":
		return "error"
	case "eval":
		return "eval"
	case "evolve":
		return "evolve"
	case "system":
		if source == "update" {
			return "update"
		}
		return ""
	default:
		return ""
	}
}

// companionPulseTitle 只在 kind 为 loading 时改标题。tool_call 优先用技能标签，其次工具名，再次 fallback，并截到 42 个字。其他 loading 用正文，同样截断。其余 kind 返回 fallback。
// typ 为 tool_call 且 kind 为 loading 时，和 tool、args 一起传给 skillpack.CallLabel。
// text 在 kind 为 loading 且 typ 不是 tool_call 时去掉空白，非空则截到 42 个字后返回。
// args 在 typ 为 tool_call 且 kind 为 loading 时，和 tool 一起传给 skillpack.CallLabel。
func companionPulseTitle(kind, typ, fallback, text, tool, args string) string {
	switch kind {
	case "loading":
		if typ == "tool_call" {
			if s := skillpack.CallLabel(tool, args); s != "" {
				return clipRunes(s, 42)
			}
			if s := strings.TrimSpace(tool); s != "" {
				return s
			}
			return fallback
		}
		if s := strings.TrimSpace(text); s != "" {
			return clipRunes(s, 42)
		}
		return fallback
	default:
		return fallback
	}
}

// clipRunes 按 rune 截断。超过 n 时留下前 n 个字并加上省略号。
func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// answeredOrAsk 只在类型为 approval 时可能为真。payload 已有 decision，或 action 为 ask_user 时返回真，避免把已经处理过的审批或重复的提问再通知一次。
// typ 不是 approval 时返回假。
func answeredOrAsk(typ string, payload map[string]any) bool {
	if typ != "approval" {
		return false
	}
	return payloadStr(payload, "decision") != "" || payloadStr(payload, "action") == "ask_user"
}

// quietOutcome 为真时宠物仍收到脉冲，但不再发系统通知。error 的 kind 为 canceled，或 turn_end 的 stop 为 plan_open 时返回真。
// typ 为 error 时从 payload 取 kind，等于 runtime.ErrKindCanceled 则返回真。为 turn_end 时取 stop，等于 runtime.StopPlanOpen 的字符串则返回真。
// payload 传给 payloadStr，读取 kind 或 stop。
func quietOutcome(typ string, payload map[string]any) bool {
	switch typ {
	case "error":
		return payloadStr(payload, "kind") == runtime.ErrKindCanceled
	case "turn_end":
		return payloadStr(payload, "stop") == string(runtime.StopPlanOpen)
	}
	return false
}

// oneLine 把空白收成单个空格，再按 rune 截到 n 个字。
// s 先把空白收成单个空格，再传给 clipRunes。
func oneLine(s string, n int) string {
	return clipRunes(strings.Join(strings.Fields(s), " "), n)
}

// lineJoin 丢掉空白片段，用中点 · 把剩下的连成一行。
// parts 去掉空白后为空的丢掉，剩下的用中点 · 连接。
func lineJoin(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

// approvalLine 写出正在等的动作，以及它碰到的 command 或 path。两边都有时用冒号接上，detail 最多 120 个字。
func approvalLine(p map[string]any) string {
	action := strings.TrimSpace(payloadStr(p, "action"))
	detail := oneLine(firstPayloadStr(p, "command", "path"), 120)
	switch {
	case action == "":
		return detail
	case detail == "":
		return action
	default:
		return action + ": " + detail
	}
}
