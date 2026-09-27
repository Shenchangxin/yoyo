package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Hub fans session events to HTTP SSE clients without a heavy broker.
// Live events (token deltas, tool_call, tool_result) drop under backpressure
// after spilling to Overflow: a frozen WebView must not spawn unbounded
// send goroutines. Terminal events (turn_end, error, approval, user) still
// wait on a single overflow goroutine. Adjacent token deltas (same session,
// type, and round) coalesce on a 50ms window so Wails and SSE subscribers
// both see ~20fps instead of per-token wakeups.
type Hub struct {
	mu       sync.Mutex
	subs     map[string]map[chan trace.Event]struct{}
	Overflow string

	cMu     sync.Mutex
	pending *deltaBuf
	timer   *time.Timer
}

const coalesceWindow = 50 * time.Millisecond

type deltaBuf struct {
	ev   trace.Event
	key  string
	text strings.Builder
}

func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan trace.Event]struct{}{}}
}

func (h *Hub) Subscribe(sessionID string) (<-chan trace.Event, func()) {
	ch := make(chan trace.Event, 8192)
	h.mu.Lock()
	if h.subs[sessionID] == nil {
		h.subs[sessionID] = map[chan trace.Event]struct{}{}
	}
	h.subs[sessionID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if m := h.subs[sessionID]; m != nil {
			delete(m, ch)
		}
		h.mu.Unlock()
		close(ch)
	}
}

func (h *Hub) Publish(ev trace.Event) {
	if key, ok := coalesceKey(ev); ok {
		h.holdDelta(ev, key)
		return
	}
	h.FlushCoalesce()
	h.fanout(ev)
}

// FlushCoalesce emits any buffered token delta. Tests call it instead of sleeping.
func (h *Hub) FlushCoalesce() {
	h.cMu.Lock()
	h.flushPendingLocked()
	h.cMu.Unlock()
}

func (h *Hub) holdDelta(ev trace.Event, key string) {
	h.cMu.Lock()
	defer h.cMu.Unlock()
	text := payloadDeltaText(ev)
	if h.pending != nil {
		if h.pending.key == key {
			h.pending.text.WriteString(text)
			return
		}
		h.flushPendingLocked()
	}
	buf := &deltaBuf{ev: cloneEvent(ev), key: key}
	buf.text.WriteString(text)
	h.pending = buf
	if h.timer != nil {
		h.timer.Stop()
	}
	h.timer = time.AfterFunc(coalesceWindow, func() { h.FlushCoalesce() })
}

func (h *Hub) flushPendingLocked() {
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
	if h.pending == nil {
		return
	}
	ev := h.pending.ev
	setDeltaText(&ev, h.pending.text.String())
	h.pending = nil
	h.cMu.Unlock()
	h.fanout(ev)
	h.cMu.Lock()
}

func (h *Hub) fanout(ev trace.Event) {
	h.mu.Lock()
	var chans []chan trace.Event
	for _, id := range []string{ev.SessionID, "*"} {
		for ch := range h.subs[id] {
			chans = append(chans, ch)
		}
	}
	overflow := h.Overflow
	h.mu.Unlock()
	for _, ch := range chans {
		deliver(ch, ev, overflow)
	}
}

func deliver(ch chan trace.Event, ev trace.Event, overflow string) {
	defer func() { _ = recover() }()
	select {
	case ch <- ev:
		return
	default:
	}
	spillOverflow(overflow, ev)
	if !keepUnderBackpressure(ev) {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		ch <- ev
	}()
}

// KeepHubEvent is true for terminal events that must still reach a slow UI.
func KeepHubEvent(ev trace.Event) bool {
	return keepUnderBackpressure(ev)
}

func keepUnderBackpressure(ev trace.Event) bool {
	if ev.Payload != nil {
		if delta, _ := ev.Payload["delta"].(bool); delta {
			return false
		}
	}
	switch ev.Type {
	case trace.TypeTurnEnd, trace.TypeError, trace.TypeApproval, trace.TypeUser, trace.TypeInject:
		return true
	default:
		return false
	}
}

func coalesceKey(ev trace.Event) (string, bool) {
	if ev.Payload == nil {
		return "", false
	}
	delta, _ := ev.Payload["delta"].(bool)
	if !delta {
		return "", false
	}
	switch ev.Type {
	case trace.TypeAssistant, trace.TypeReasoning, trace.TypeToolResult:
	default:
		return "", false
	}
	id, _ := ev.Payload["id"].(string)
	round, _ := ev.Payload["round"].(string)
	return ev.SessionID + "\x00" + string(ev.Type) + "\x00" + id + "\x00" + round, true
}

func payloadDeltaText(ev trace.Event) string {
	if ev.Payload == nil {
		return ""
	}
	if ev.Type == trace.TypeToolResult {
		s, _ := ev.Payload["content"].(string)
		return s
	}
	s, _ := ev.Payload["text"].(string)
	return s
}

func setDeltaText(ev *trace.Event, text string) {
	if ev.Payload == nil {
		ev.Payload = map[string]any{}
	}
	if ev.Type == trace.TypeToolResult {
		ev.Payload["content"] = text
		return
	}
	ev.Payload["text"] = text
}

func cloneEvent(ev trace.Event) trace.Event {
	out := ev
	if ev.Payload == nil {
		return out
	}
	p := make(map[string]any, len(ev.Payload)+1)
	for k, v := range ev.Payload {
		p[k] = v
	}
	out.Payload = p
	return out
}

func spillOverflow(dir string, ev trace.Event) {
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	id := ev.SessionID
	if id == "" {
		id = "unknown"
	}
	f, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	_ = enc.Encode(ev)
}
