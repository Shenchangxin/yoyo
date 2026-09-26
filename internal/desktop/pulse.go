package desktop

import "strings"

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
	default:
		return ""
	}
}

func companionPulseTitle(kind, typ, fallback, text, tool string) string {
	switch kind {
	case "loading":
		if typ == "tool_call" {
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
