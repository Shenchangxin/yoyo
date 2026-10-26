// 给实时总线和界面准备的轨迹投影。JSONL 仍由 emit 写入完整参数，这里只改副本。
// UITrajectory 被界面读取。重写入参的压缩规则在 context_shape.go 的 compactCallArgs。

package runtime

import (
	"encoding/json"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// 实时副本的字节上限。助手和推理的已结算文本是 24000，工具结果正文是 1200，用户文本是 8000，补丁是 8000。delta 文本不按助手上限裁。
const (
	uiAssistantBytes = 24_000
	uiResultBytes    = 1_200
	uiUserBytes      = 8_000
	uiPatchBytes     = 8_000
)

// slimLiveEvent 给实时总线复制一份变瘦的事件，不改调用方手里的 payload。heavyCallName 为真且参数非空时，把参数换成 compactCallArgs 的结果；压缩结果和原文一样、又不是合法 JSON 对象、但能抽出 path 时，参数改成只含 path 的对象。
// 工具结果分别裁 content 和 patch。助手和推理先丢掉 stream。delta 为真时文本保持原样。已结算文本按 uiAssistantBytes 裁。用户文本按 uiUserBytes 裁。检查点事件若带 tail 就删掉 tail。
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
		ev = dropStreamTape(ev)
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

// dropStreamTape 从副本里删除 stream。没有 payload 或没有这个键时原样返回。
func dropStreamTape(ev trace.Event) trace.Event {
	if ev.Payload == nil {
		return ev
	}
	if _, ok := ev.Payload["stream"]; !ok {
		return ev
	}
	p := clonePayload(ev.Payload)
	delete(p, "stream")
	ev.Payload = p
	return ev
}

// slimPayloadString 按字节裁切一个字符串字段。cap 小于等于 0 或长度不超过 cap 时原样返回。裁切后在末尾加上换行和省略号，并把 bytes 设成原长度。
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

// clonePayload 浅拷贝 payload，避免改到 JSONL 里的那一份。
func clonePayload(in map[string]any) map[string]any {
	p := make(map[string]any, len(in)+1)
	for k, v := range in {
		p[k] = v
	}
	return p
}

// SlimTrajectory 是会话 JSONL 的界面投影。工具调用的完整正文仍留在原切片里，返回值里的重写入调用只剩路径和短桩。空切片原样返回。
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

// UITrajectory 是渲染器看到的轨迹：先瘦身写入和结果，再丢掉账本噪声，最后只留最后一条进展之后的最后一张错误卡片。事件条数和身份保留，进程轨仍能折叠中间的助手和工具；瘦的是 payload，用来限住 Wails 和 WebView 的内存。更早的用户回合仍由 ProjectPage 按字节开窗。
func UITrajectory(evs []trace.Event) []trace.Event {
	return dropSupersededErrors(dropUINoise(SlimTrajectory(evs)))
}

// dropUINoise 丢掉 file_change、turn_end、system、eval、evolve、usage。compact 只留 kind 为 checkpoint 的；payload 为 nil 的 compact 也丢掉。
func dropUINoise(evs []trace.Event) []trace.Event {
	out := make([]trace.Event, 0, len(evs))
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeFileChange, trace.TypeTurnEnd, trace.TypeSystem, trace.TypeEval, trace.TypeEvolve, trace.TypeUsage:
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

// dropSupersededErrors 最多留下最后一条进展之后的最后一张错误。进展包括用户、助手、推理、工具调用、工具结果、turn_start、step_start、step_end。没有错误时原样返回。进展出现在错误之后时，这些错误全部去掉，恢复和后一条用户消息不会把停止卡片留在界面上。
func dropSupersededErrors(evs []trace.Event) []trace.Event {
	lastProgress := -1
	hasError := false
	for i, ev := range evs {
		switch ev.Type {
		case trace.TypeError:
			hasError = true
		case trace.TypeUser, trace.TypeAssistant, trace.TypeReasoning, trace.TypeToolCall, trace.TypeToolResult, trace.TypeTurnStart, trace.TypeStepStart, trace.TypeStepEnd:
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
