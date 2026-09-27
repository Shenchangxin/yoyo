package runtime

import "github.com/Shenchangxin/yoyo/internal/trace"

// slimLiveEvent copies a heavy write_file/str_replace payload for the live
// bus. jsonl keeps the full arguments (DumpSession / trajectory). Shipping
// every file body through Wails + React is what wedged session 477c0d8b0b8ac64e.
func slimLiveEvent(ev trace.Event) trace.Event {
	if ev.Payload == nil {
		return ev
	}
	switch ev.Type {
	case trace.TypeToolCall:
		name, _ := ev.Payload["name"].(string)
		args, _ := ev.Payload["arguments"].(string)
		if !heavyCallName(name) || args == "" {
			return ev
		}
		compact := compactCallArgs(name, args, heavyCallRunes)
		if compact == args {
			return ev
		}
		p := clonePayload(ev.Payload)
		p["arguments"] = compact
		p["bytes"] = len(args)
		ev.Payload = p
		return ev
	case trace.TypeToolResult:
		content, _ := ev.Payload["content"].(string)
		if len(content) <= 4_000 {
			return ev
		}
		p := clonePayload(ev.Payload)
		p["content"] = content[:4_000] + "\n…"
		p["bytes"] = len(content)
		ev.Payload = p
		return ev
	default:
		return ev
	}
}

func clonePayload(in map[string]any) map[string]any {
	p := make(map[string]any, len(in)+1)
	for k, v := range in {
		p[k] = v
	}
	return p
}

// SlimTrajectory is the UI projection of a session jsonl: tool_call bodies
// stay on disk, the renderer only sees path + a short stub.
func SlimTrajectory(evs []trace.Event) []trace.Event {
	if len(evs) == 0 {
		return evs
	}
	out := make([]trace.Event, len(evs))
	for i, ev := range evs {
		out[i] = slimLiveEvent(ev)
	}
	return out
}

// UITrajectory is the renderer projection: slim write/result bodies and drop
// ledger noise. Conversation turns and tool pairs stay intact so the UI can
// virtualize rows and fold steps — it must not drop history.
func UITrajectory(evs []trace.Event) []trace.Event {
	return dropUINoise(SlimTrajectory(evs))
}

func dropUINoise(evs []trace.Event) []trace.Event {
	out := make([]trace.Event, 0, len(evs))
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeFileChange, trace.TypeTurnEnd, trace.TypeSystem, trace.TypeEval, trace.TypeEvolve:
			continue
		case trace.TypeCompact:
			if ev.Payload == nil {
				continue
			}
			kind, _ := ev.Payload["kind"].(string)
			if kind != "checkpoint" {
				continue
			}
		}
		out = append(out, ev)
	}
	return out
}
