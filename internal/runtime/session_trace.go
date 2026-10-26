// 把会话 JSONL 投影成检查器和命令行用的摘要视图。
// 账本仍只追加。不改写原事件。形状行计入 Projections，只有 kind 为 checkpoint 的压缩计入 Compactions。

package runtime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

const (
	// traceDetailRunes 是单条明细的符文上限。
	traceDetailRunes = 4_000
	// uiTraceEvents 是界面时间线默认保留的条数。
	uiTraceEvents = 180
)

// elidedSpillRe 从被省略的工具结果里抽出 spill id 和字节数。
var elidedSpillRe = regexp.MustCompile(`\[elided tool_result id=([A-Za-z0-9_-]+) name=\S+ bytes=(\d+)`)

// TraceView 是会话 JSONL 的摘要优先投影。账本仍只追加；这是检查器和命令行的读模型。
type TraceView struct {
	StartedAt time.Time    `json:"started_at,omitempty"`
	EndedAt   time.Time    `json:"ended_at,omitempty"`
	Stats     TraceStats   `json:"stats"`
	Events    []TraceEvent `json:"events"`
}

// TraceStats 是整本账本的计数。Projections 统计抽样的提示投影行，它们不是压缩：没有从记录里摘要或删除任何东西。
type TraceStats struct {
	Events      int `json:"events"`
	Users       int `json:"users"`
	Assistants  int `json:"assistants"`
	ToolCalls   int `json:"tool_calls"`
	ToolResults int `json:"tool_results"`
	Errors      int `json:"errors"`
	Compactions int `json:"compactions"`
	// Projections 统计抽样的提示投影账本行。它们不是压缩：没有从记录里摘要或删除任何东西。
	Projections int `json:"projections"`
	Deltas      int `json:"deltas"`
	Tokens      int `json:"tokens,omitempty"`
	DurationMs  int `json:"duration_ms,omitempty"`
	SpillBytes  int `json:"spill_bytes,omitempty"`
}

// TraceEvent 是投影后的一条。Lane 由事件类型决定。Detail 截到 traceDetailRunes。
type TraceEvent struct {
	Index            int    `json:"index"`
	TS               string `json:"ts,omitempty"`
	Type             string `json:"type"`
	Source           string `json:"source,omitempty"`
	Lane             string `json:"lane"`
	Round            string `json:"round,omitempty"`
	Name             string `json:"name,omitempty"`
	ID               string `json:"id,omitempty"`
	Summary          string `json:"summary"`
	Detail           string `json:"detail,omitempty"`
	Bytes            int    `json:"bytes,omitempty"`
	ElapsedMs        int    `json:"elapsed_ms,omitempty"`
	SpillID          string `json:"spill_id,omitempty"`
	Tokens           int    `json:"tokens,omitempty"`
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`
	CachedTokens     int    `json:"cached_tokens,omitempty"`
	TtftMs           int    `json:"ttft_ms,omitempty"`
	CacheReported    bool   `json:"cache_reported,omitempty"`
	Error            bool   `json:"error,omitempty"`
}

// ProjectTrace 把事件收成视图。delta 计入 Deltas，但不进入 Events。只有 kind 为 checkpoint 的压缩计入 Compactions；kind 为 shape 的计入 Projections。
// 只有检查点才压缩：它摘要历史并重新播种提示。抽样的 shape 行只描述一轮提示如何投影，不能算进压缩次数。
// evs 的长度写入 Stats.Events；逐条看时间和类型，delta、kind、tokens 从载荷读取，非 delta 再交给 projectTraceEvent。
func ProjectTrace(evs []trace.Event) TraceView {
	out := TraceView{Events: make([]TraceEvent, 0, len(evs))}
	out.Stats.Events = len(evs)
	var first, last time.Time
	for i, ev := range evs {
		if !ev.TS.IsZero() {
			if first.IsZero() || ev.TS.Before(first) {
				first = ev.TS
			}
			if last.IsZero() || ev.TS.After(last) {
				last = ev.TS
			}
		}
		delta := payloadBool(ev.Payload, "delta")
		if delta {
			out.Stats.Deltas++
		}
		switch ev.Type {
		case trace.TypeUser:
			out.Stats.Users++
		case trace.TypeAssistant:
			if !delta {
				out.Stats.Assistants++
			}
		case trace.TypeToolCall:
			out.Stats.ToolCalls++
		case trace.TypeToolResult:
			out.Stats.ToolResults++
		case trace.TypeError:
			out.Stats.Errors++
		case trace.TypeCompact:
			// 只有检查点才压缩。抽样的 shape 行只描述这一轮提示如何投影，不能算进压缩次数。
			switch payloadStr(ev.Payload, "kind") {
			case "checkpoint":
				out.Stats.Compactions++
			case "shape":
				out.Stats.Projections++
			}
			if n := payloadInt(ev.Payload, "tokens"); n > 0 {
				out.Stats.Tokens = n
			}
		}
		if delta {
			continue
		}
		row := projectTraceEvent(i, ev)
		if row.Error && ev.Type != trace.TypeError {
			out.Stats.Errors++
		}
		out.Events = append(out.Events, row)
	}
	if !first.IsZero() {
		out.StartedAt = first.UTC()
		out.EndedAt = last.UTC()
		out.Stats.DurationMs = int(last.Sub(first) / time.Millisecond)
		if out.Stats.DurationMs < 0 {
			out.Stats.DurationMs = 0
		}
	}
	return out
}

// TrimTraceView 保留整本账本的统计，但只把滤掉噪声后的短时间线送过 Wails。完整 jsonl 仍在盘上。
// keep 小于等于 0 时用 uiTraceEvents。file_change、system、eval、evolve，以及非 checkpoint 的 compaction 被丢掉。超出 keep 时只留末尾。
// view 的 Events 按类型过滤，超出 keep 时只留末尾，其余字段随 view 返回。
func TrimTraceView(view TraceView, keep int) TraceView {
	if keep <= 0 {
		keep = uiTraceEvents
	}
	out := make([]TraceEvent, 0, min(len(view.Events), keep))
	for _, ev := range view.Events {
		switch ev.Type {
		case "file_change", "system", "eval", "evolve":
			continue
		case "compaction":
			if ev.Name != "checkpoint" {
				continue
			}
		}
		out = append(out, ev)
	}
	if len(out) > keep {
		out = out[len(out)-keep:]
	}
	view.Events = out
	return view
}

// projectTraceEvent 把一条轨迹收成摘要。工具结果若正文里有省略标记，会补上 spill id 和字节数。字节超过 defaultIngestBudget 时摘要只写字节数。
// ev 的 Payload、Type、Source 和 TS 用来填这一行；工具结果再从正文字段补 spill id 和字节数。
func projectTraceEvent(i int, ev trace.Event) TraceEvent {
	p := ev.Payload
	id := payloadStr(p, "id")
	name := payloadStr(p, "name")
	round := payloadStr(p, "round")
	if round == "" {
		round = payloadStr(p, "id")
	}
	text := payloadStr(p, "text")
	content := payloadStr(p, "content")
	args := payloadString(payloadAny(p, "arguments"))
	note := payloadStr(p, "note")
	kind := payloadStr(p, "kind")
	row := TraceEvent{
		Index:     i,
		Type:      string(ev.Type),
		Source:    ev.Source,
		Lane:      eventLane(ev.Type),
		Round:     round,
		Name:      name,
		ID:        id,
		Bytes:     payloadInt(p, "bytes"),
		ElapsedMs: payloadInt(p, "elapsed_ms"),
		SpillID:   payloadStr(p, "spill_id"),
		Tokens:    payloadInt(p, "tokens"),
	}
	if !ev.TS.IsZero() {
		row.TS = ev.TS.UTC().Format(time.RFC3339Nano)
	}
	switch ev.Type {
	case trace.TypeUser:
		row.Summary = firstNonEmpty(oneLine(text, 120), "user")
		row.Detail = capTrace(text, traceDetailRunes)
	case trace.TypeAssistant, trace.TypeReasoning:
		row.Summary = firstNonEmpty(oneLine(text, 120), string(ev.Type))
		row.Detail = capTrace(text, traceDetailRunes)
	case trace.TypeSystem:
		row.Summary = "system · " + strconv.Itoa(utf8.RuneCountInString(text)) + " chars"
		row.Detail = capTrace(text, traceDetailRunes)
	case trace.TypeToolCall:
		row.Summary = firstNonEmpty(name, "tool") + " · " + firstNonEmpty(oneLine(args, 80), "call")
		row.Detail = capTrace(args, traceDetailRunes)
		if row.Bytes == 0 {
			row.Bytes = len(args)
		}
	case trace.TypeToolResult:
		if row.SpillID == "" {
			if m := elidedSpillRe.FindStringSubmatch(content); len(m) == 3 {
				row.SpillID = m[1]
				if row.Bytes == 0 {
					row.Bytes, _ = strconv.Atoi(m[2])
				}
			}
		}
		if row.Bytes == 0 {
			row.Bytes = len(content)
		}
		row.Error = strings.HasPrefix(content, "ERROR:")
		label := firstNonEmpty(name, "tool")
		if row.Error {
			row.Summary = label + " · error"
		} else if row.Bytes > defaultIngestBudget {
			row.Summary = label + " · " + strconv.Itoa(row.Bytes) + " bytes"
		} else {
			row.Summary = label + " · result"
		}
		row.Detail = capTrace(content, traceDetailRunes)
	case trace.TypeCompact:
		row.Name = kind
		row.Summary = firstNonEmpty(kind, "compact")
		if note != "" {
			row.Summary += " · " + oneLine(note, 80)
		}
		row.Detail = capTrace(firstNonEmpty(payloadStr(p, "summary"), note), traceDetailRunes)
	case trace.TypeInject:
		src := firstNonEmpty(ev.Source, payloadStr(p, "name"), "inject")
		row.Summary = src + " · " + firstNonEmpty(oneLine(text, 80), "context")
		row.Detail = capTrace(text, traceDetailRunes)
	case trace.TypeError:
		row.Error = true
		row.Summary = firstNonEmpty(oneLine(text, 120), payloadStr(p, "error"), "error")
		row.Detail = capTrace(firstNonEmpty(text, payloadStr(p, "error")), traceDetailRunes)
	case trace.TypeApproval:
		dec := payloadStr(p, "decision")
		act := firstNonEmpty(payloadStr(p, "action"), payloadStr(p, "command"), "approval")
		if dec != "" {
			row.Summary = "approval · " + dec + " · " + act
		} else {
			row.Summary = "approval · " + act
		}
		row.Detail = capTrace(strings.TrimSpace(payloadStr(p, "command")+"\n"+payloadStr(p, "path")), traceDetailRunes)
	case trace.TypeSubagent:
		row.Summary = "subagent · " + firstNonEmpty(oneLine(payloadStr(p, "summary"), 80), oneLine(payloadStr(p, "prompt"), 80), "task")
		row.Detail = capTrace(firstNonEmpty(payloadStr(p, "summary"), payloadStr(p, "prompt"), payloadStr(p, "error")), traceDetailRunes)
		row.Error = payloadStr(p, "error") != ""
	case trace.TypeTurnEnd:
		ok := payloadBool(p, "ok")
		if ok {
			row.Summary = "turn end"
		} else {
			row.Summary = "turn end · failed"
			row.Error = true
		}
		row.Detail = capTrace(text, traceDetailRunes)
	case trace.TypeUsage:
		row.PromptTokens = payloadInt(p, "prompt_tokens", "tokens")
		row.CompletionTokens = payloadInt(p, "completion_tokens")
		row.CachedTokens = payloadInt(p, "cached_tokens")
		row.TtftMs = payloadInt(p, "ttft_ms")
		if row.ElapsedMs == 0 {
			row.ElapsedMs = payloadInt(p, "elapsed_ms")
		}
		row.Tokens = row.PromptTokens
		row.CacheReported = payloadBool(p, "cache_reported") || row.CachedTokens > 0
		row.Name = firstNonEmpty(name, "model")
		summary := "model"
		if row.ElapsedMs > 0 {
			summary += " · " + strconv.Itoa(row.ElapsedMs) + "ms"
		}
		if row.TtftMs > 0 {
			summary += " · ttft " + strconv.Itoa(row.TtftMs) + "ms"
		}
		if row.PromptTokens > 0 && row.CacheReported {
			pct := (row.CachedTokens * 100) / row.PromptTokens
			summary += " · cache " + strconv.Itoa(pct) + "%"
		}
		row.Summary = summary
		row.Detail = capTrace(fmt.Sprintf("prompt=%d completion=%d cached=%d elapsed_ms=%d ttft_ms=%d",
			row.PromptTokens, row.CompletionTokens, row.CachedTokens, row.ElapsedMs, row.TtftMs), traceDetailRunes)
	default:
		row.Summary = firstNonEmpty(oneLine(text, 120), string(ev.Type))
		row.Detail = capTrace(firstNonEmpty(text, content, note), traceDetailRunes)
	}
	return row
}

// eventLane 把工具、批准和子代理放进 exec，压缩和注入放进 memory，错误放进 error，回合结束放进 meta，其余放进 dialog。
func eventLane(t trace.EventType) string {
	switch t {
	case trace.TypeToolCall, trace.TypeToolResult, trace.TypeApproval, trace.TypeSubagent:
		return "exec"
	case trace.TypeCompact, trace.TypeInject, trace.TypeSystem:
		return "memory"
	case trace.TypeError:
		return "error"
	case trace.TypeTurnEnd, trace.TypeEval, trace.TypeEvolve:
		return "meta"
	default:
		return "dialog"
	}
}

// payloadAny 读取载荷字段。载荷为空时返回 nil。
// key 作为字段名从载荷取值。
func payloadAny(p map[string]any, key string) any {
	if p == nil {
		return nil
	}
	return p[key]
}

// payloadStr 把载荷字段收成字符串。
// key 作为字段名取出后交给 payloadString；载荷为 nil 时返回空串。
func payloadStr(p map[string]any, key string) string {
	if p == nil {
		return ""
	}
	return payloadString(p[key])
}

// payloadInt 按给定键的顺序读取第一个能转成整数的值。
// keys 按顺序试，第一个能转成整数的值就返回；载荷为 nil 时返回 0。
func payloadInt(p map[string]any, keys ...string) int {
	if p == nil {
		return 0
	}
	for _, key := range keys {
		switch v := p[key].(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float64:
			return int(v)
		case float32:
			return int(v)
		case string:
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err == nil {
				return n
			}
		}
	}
	return 0
}

// payloadBool 读取布尔值。字符串 true 或 1 也算真。
// key 作为字段名取值；布尔真，或字符串 true、1，才返回真。
func payloadBool(p map[string]any, key string) bool {
	if p == nil {
		return false
	}
	switch v := p[key].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	}
	return false
}

// oneLine 把空白折成空格，再按符文截断。
// s 先把连续空白收成单个空格，再交给 capTrace。
func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	return capTrace(s, n)
}

// capTrace 按符文截断，超出时加上省略号。n 小于等于 0 或字符串为空时原样返回。
// s 在 n 小于等于 0 或自身为空时原样返回；rune 数超过 n 时截到前 n 个并加上省略号。
func capTrace(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// firstNonEmpty 返回第一段非空白字符串。
// ss 从前往后看，返回第一段去掉空白后仍非空的字符串。
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
