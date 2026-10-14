package desktop

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/skillpack"
)

// companionPulseKind is the single mapping from a live item onto the
// desktop-pet channel. "loading" must never become an OS notification.
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

func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// answeredOrAsk reports approval events that must not notify again. A decision
// event answers an approval the user already handled, and an approval whose
// action is ask_user repeats the question the ask_user item already announced.
func answeredOrAsk(typ string, payload map[string]any) bool {
	if typ != "approval" {
		return false
	}
	return payloadStr(payload, "decision") != "" || payloadStr(payload, "action") == "ask_user"
}

// quietOutcome is a run the user should see in the app but not as an OS toast:
// a deliberate Stop, and a turn that ended with plan steps still open. The pet
// still hears both, so its loading state clears.
func quietOutcome(typ string, payload map[string]any) bool {
	switch typ {
	case "error":
		return payloadStr(payload, "kind") == runtime.ErrKindCanceled
	case "turn_end":
		return payloadStr(payload, "stop") == string(runtime.StopPlanOpen)
	}
	return false
}

// oneLine collapses whitespace and clips to n runes, for a single toast line.
func oneLine(s string, n int) string {
	return clipRunes(strings.Join(strings.Fields(s), " "), n)
}

// lineJoin joins the non-empty parts with a middle dot.
func lineJoin(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

// approvalLine names what is waiting: the action and the command or path it touches.
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
