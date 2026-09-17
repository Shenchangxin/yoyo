package runtime

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

const traceDetailRunes = 4_000

var elidedSpillRe = regexp.MustCompile(`\[elided tool_result id=([A-Za-z0-9_-]+) name=\S+ bytes=(\d+)`)

// TraceView is a summary-first projection of a session JSONL. The ledger
// stays append-only; this is a read model for inspectors and the CLI.
type TraceView struct {
	StartedAt time.Time    `json:"started_at,omitempty"`
	EndedAt   time.Time    `json:"ended_at,omitempty"`
	Stats     TraceStats   `json:"stats"`
	Events    []TraceEvent `json:"events"`
}

type TraceStats struct {
	Events      int `json:"events"`
	Users       int `json:"users"`
	Assistants  int `json:"assistants"`
	ToolCalls   int `json:"tool_calls"`
	ToolResults int `json:"tool_results"`
	Errors      int `json:"errors"`
	Compactions int `json:"compactions"`
	Deltas      int `json:"deltas"`
	Tokens      int `json:"tokens,omitempty"`
	DurationMs  int `json:"duration_ms,omitempty"`
	SpillBytes  int `json:"spill_bytes,omitempty"`
}

type TraceEvent struct {
	Index     int    `json:"index"`
	TS        string `json:"ts,omitempty"`
	Type      string `json:"type"`
	Source    string `json:"source,omitempty"`
	Lane      string `json:"lane"`
	Round     string `json:"round,omitempty"`
	Name      string `json:"name,omitempty"`
	ID        string `json:"id,omitempty"`
	Summary   string `json:"summary"`
	Detail    string `json:"detail,omitempty"`
	Bytes     int    `json:"bytes,omitempty"`
	ElapsedMs int    `json:"elapsed_ms,omitempty"`
	SpillID   string `json:"spill_id,omitempty"`
	Tokens    int    `json:"tokens,omitempty"`
	Error     bool   `json:"error,omitempty"`
}

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
			out.Stats.Compactions++
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
		} else if row.Bytes > ingestPreviewRunes {
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
	default:
		row.Summary = firstNonEmpty(oneLine(text, 120), string(ev.Type))
		row.Detail = capTrace(firstNonEmpty(text, content, note), traceDetailRunes)
	}
	return row
}

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

func payloadAny(p map[string]any, key string) any {
	if p == nil {
		return nil
	}
	return p[key]
}

func payloadStr(p map[string]any, key string) string {
	if p == nil {
		return ""
	}
	return payloadString(p[key])
}

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

func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	return capTrace(s, n)
}

func capTrace(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
