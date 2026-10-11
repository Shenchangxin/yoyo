package runtime

import (
	"encoding/json"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

const (
	uiAssistantBytes = 24_000
	uiResultBytes    = 1_200
	uiUserBytes      = 8_000
	uiPatchBytes     = 8_000
)

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
		path := salvageJSONStringField(args, "path")
		if compact == args {
			if _, ok := unmarshalObject(args); ok {
				return ev
			}
			if path == "" {
				return ev
			}
			stub, _ := json.Marshal(map[string]any{"path": path})
			compact = string(stub)
		}
		p := clonePayload(ev.Payload)
		p["arguments"] = compact
		p["bytes"] = len(args)
		if path != "" {
			p["path"] = path
		}
		ev.Payload = p
		return ev
	case trace.TypeToolResult:
		ev = slimPayloadString(ev, "content", uiResultBytes)
		return slimPayloadString(ev, "patch", uiPatchBytes)
	case trace.TypeAssistant, trace.TypeReasoning:
		if delta, _ := ev.Payload["delta"].(bool); delta {
			return ev
		}
		return slimPayloadString(ev, "text", uiAssistantBytes)
	case trace.TypeUser:
		return slimPayloadString(ev, "text", uiUserBytes)
	case trace.TypeCompact:
		if ev.Payload["tail"] == nil {
			return ev
		}
		p := clonePayload(ev.Payload)
		delete(p, "tail")
		ev.Payload = p
		return ev
	default:
		return ev
	}
}

func slimPayloadString(ev trace.Event, key string, cap int) trace.Event {
	s, _ := ev.Payload[key].(string)
	if cap <= 0 || len(s) <= cap {
		return ev
	}
	p := clonePayload(ev.Payload)
	p[key] = s[:cap] + "\n…"
	p["bytes"] = len(s)
	ev.Payload = p
	return ev
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
// ledger noise. Event identity stays intact so the process rail can fold
// intermediate assistants and tools; payload slimming is what bounds Wails
// and WebView memory. ProjectPage still windows older *user turns* by byte cap.
func UITrajectory(evs []trace.Event) []trace.Event {
	return dropSupersededErrors(dropUINoise(SlimTrajectory(evs)))
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

// dropSupersededErrors keeps at most the last error after the latest
// user/assistant/tool progress. A Stopped chip is a turn status, not a
// letter — resume and the next user message must not keep it on screen.
func dropSupersededErrors(evs []trace.Event) []trace.Event {
	lastProgress := -1
	hasError := false
	for i, ev := range evs {
		switch ev.Type {
		case trace.TypeError:
			hasError = true
		case trace.TypeUser, trace.TypeAssistant, trace.TypeReasoning, trace.TypeToolCall, trace.TypeToolResult:
			lastProgress = i
		}
	}
	if !hasError {
		return evs
	}
	keep := -1
	for i := lastProgress + 1; i < len(evs); i++ {
		if evs[i].Type == trace.TypeError {
			keep = i
		}
	}
	out := make([]trace.Event, 0, len(evs))
	for i, ev := range evs {
		if ev.Type == trace.TypeError && i != keep {
			continue
		}
		out = append(out, ev)
	}
	return out
}
