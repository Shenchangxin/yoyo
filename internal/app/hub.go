package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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
//
// Every fanout is recorded in a seq ring so a slow renderer can pull what
// it missed instead of depending on the in-flight channel.
type Hub struct {
	mu       sync.Mutex
	subs     map[string]map[chan trace.Event]struct{}
	Overflow string

	cMu     sync.Mutex
	pending *deltaBuf
	timer   *time.Timer

	logMu sync.Mutex
	seq   int64
	log   []liveItem
}

const (
	coalesceWindow  = 50 * time.Millisecond
	liveLogCap      = 4096
	liveInlineBytes = 768
	liveSinceCap    = 512
)

type liveItem struct {
	seq int64
	ev  trace.Event
}

type deltaBuf struct {
	ev   trace.Event
	key  string
	text strings.Builder
}

// LiveNotice is the Wails payload. Small events travel inline; larger ones
// are a seq the renderer pulls via LiveSince.
type LiveNotice struct {
	Seq       int64        `json:"seq"`
	SessionID string       `json:"session_id"`
	Type      string       `json:"type"`
	Event     *trace.Event `json:"event,omitempty"`
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan trace.Event]struct{})}
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
	ev.Seq = h.record(ev)
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

func (h *Hub) record(ev trace.Event) int64 {
	h.logMu.Lock()
	defer h.logMu.Unlock()
	h.seq++
	item := liveItem{seq: h.seq, ev: cloneEvent(ev)}
	item.ev.Seq = h.seq
	if len(h.log) < liveLogCap {
		h.log = append(h.log, item)
	} else {
		h.log[int(h.seq-1)%liveLogCap] = item
	}
	return h.seq
}

// Head is the latest live seq, or 0 if nothing has been published.
func (h *Hub) Head() int64 {
	if h == nil {
		return 0
	}
	h.logMu.Lock()
	defer h.logMu.Unlock()
	return h.seq
}

// Since returns recorded events with seq > after, optionally filtered by session.
func (h *Hub) Since(after int64, sessionID string) []trace.Event {
	if h == nil {
		return nil
	}
	h.logMu.Lock()
	defer h.logMu.Unlock()
	if len(h.log) == 0 {
		return nil
	}
	out := make([]trace.Event, 0, 16)
	for _, it := range h.log {
		if it.seq <= after {
			continue
		}
		if sessionID != "" && it.ev.SessionID != sessionID && it.ev.SessionID != "" {
			continue
		}
		out = append(out, it.ev)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	if len(out) > liveSinceCap {
		out = out[len(out)-liveSinceCap:]
	}
	return out
}

func NoticeOf(ev trace.Event) LiveNotice {
	n := LiveNotice{Seq: ev.Seq, SessionID: ev.SessionID, Type: string(ev.Type)}
	if liveDelta(ev) {
		e := ev
		n.Event = &e
		return n
	}
	if !mayInlineNotice(ev) {
		return n
	}
	b, err := json.Marshal(ev)
	if err == nil && len(b) <= liveInlineBytes {
		e := ev
		n.Event = &e
	}
	return n
}

func liveDelta(ev trace.Event) bool {
	if ev.Payload == nil {
		return false
	}
	delta, _ := ev.Payload["delta"].(bool)
	if !delta {
		return false
	}
	switch ev.Type {
	case trace.TypeAssistant, trace.TypeReasoning, trace.TypeToolResult:
		return true
	default:
		return false
	}
}

// mayInlineNotice is the Wails fast path. Tool bodies and settled assistant
// letters are pull-only so Event.Emit never materializes a 20KB JSON blob on
// the WebView UI thread. Tiny token slices still ride along.
func mayInlineNotice(ev trace.Event) bool {
	switch ev.Type {
	case trace.TypeToolCall, trace.TypeToolResult:
		return false
	case trace.TypeAssistant, trace.TypeReasoning:
		if ev.Payload != nil {
			if delta, _ := ev.Payload["delta"].(bool); delta {
				return true
			}
		}
		return false
	default:
		return true
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
