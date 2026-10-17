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
	ev        trace.Event
	key       string
	text      strings.Builder
	indexFrom int
	index     int
}

// StreamFrame is the live-only dual-write of a token/tool chunk. It does not
// enter JSONL; EventType names stay the durable ledger.
type StreamFrame struct {
	Kind         string `json:"kind,omitempty"`
	Index        int    `json:"index,omitempty"`
	IndexFrom    int    `json:"index_from,omitempty"`
	Block        string `json:"block,omitempty"`
	Delta        string `json:"delta,omitempty"`
	CommittedSeq int64  `json:"committed_seq,omitempty"`
}

// LiveNotice is the Wails payload. Small events travel inline; larger ones
// are a seq the renderer pulls via LiveSince.
type LiveNotice struct {
	Seq       int64        `json:"seq"`
	SessionID string       `json:"session_id"`
	Type      string       `json:"type"`
	Event     *trace.Event `json:"event,omitempty"`
	Frame     *StreamFrame `json:"frame,omitempty"`
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
	idx := payloadInt(ev.Payload, "index")
	from := payloadInt(ev.Payload, "index_from")
	if from == 0 {
		from = idx
	}
	if h.pending != nil {
		if h.pending.key == key {
			h.pending.text.WriteString(text)
			if h.pending.indexFrom == 0 {
				h.pending.indexFrom = from
			}
			if idx > h.pending.index {
				h.pending.index = idx
			}
			return
		}
		h.flushPendingLocked()
	}
	buf := &deltaBuf{ev: cloneEvent(ev), key: key, indexFrom: from, index: idx}
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
	if h.pending.index > 0 {
		if ev.Payload == nil {
			ev.Payload = map[string]any{}
		}
		ev.Payload["index"] = h.pending.index
		if h.pending.indexFrom > 0 {
			ev.Payload["index_from"] = h.pending.indexFrom
		}
	}
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

// LiveBatch is one pull from the live ring. Gap is the part a caller must
// not ignore: when set, Events is not the complete tail after its cursor
// (the ring rolled past the cursor, the batch was cut at liveSinceCap, or
// the cursor is ahead of a restarted seq space), so the caller has to reload
// the trajectory instead of trusting the batch. Head is read under the same
// lock as Events, so every seq <= Head is either in Events or flagged by Gap.
type LiveBatch struct {
	Events []trace.Event `json:"events"`
	Head   int64         `json:"head"`
	Gap    bool          `json:"gap"`
}

// Batch returns the recorded events with seq > after, optionally filtered by
// session, together with the ring head and the gap flag.
func (h *Hub) Batch(after int64, sessionID string) LiveBatch {
	if h == nil {
		return LiveBatch{Events: []trace.Event{}}
	}
	h.logMu.Lock()
	defer h.logMu.Unlock()
	b := LiveBatch{Events: make([]trace.Event, 0, 16), Head: h.seq}
	// Seqs in the ring are contiguous and end at h.seq, so the oldest seq
	// still held is h.seq-len+1. Anything in (after, oldest) was overwritten.
	oldest := h.seq - int64(len(h.log)) + 1
	b.Gap = after+1 < oldest || after > h.seq
	for _, it := range h.log {
		if it.seq <= after {
			continue
		}
		if sessionID != "" && it.ev.SessionID != sessionID && it.ev.SessionID != "" {
			continue
		}
		b.Events = append(b.Events, it.ev)
	}
	sort.Slice(b.Events, func(i, j int) bool { return b.Events[i].Seq < b.Events[j].Seq })
	if len(b.Events) > liveSinceCap {
		// Silent truncation was the old hole: the oldest matches vanished
		// with no signal. Keep the newest and say so.
		b.Events = b.Events[len(b.Events)-liveSinceCap:]
		b.Gap = true
	}
	return b
}

// Since returns recorded events with seq > after, optionally filtered by
// session. It drops the gap signal; renderers should use Batch.
func (h *Hub) Since(after int64, sessionID string) []trace.Event {
	return h.Batch(after, sessionID).Events
}

func NoticeOf(ev trace.Event) LiveNotice {
	n := LiveNotice{Seq: ev.Seq, SessionID: ev.SessionID, Type: string(ev.Type)}
	if liveDelta(ev) {
		e := ev
		n.Event = &e
		n.Frame = frameOf(ev)
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
	if liveProgressCall(ev) {
		return true
	}
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

func liveProgressCall(ev trace.Event) bool {
	if ev.Type != trace.TypeToolCall || ev.Payload == nil {
		return false
	}
	p, _ := ev.Payload["progress"].(bool)
	return p
}

// mayInlineNotice is the Wails fast path. Tool bodies and settled assistant
// letters are pull-only so Event.Emit never materializes a 20KB JSON blob on
// the WebView UI thread. Tiny token slices still ride along.
func mayInlineNotice(ev trace.Event) bool {
	if liveProgressCall(ev) {
		return true
	}
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
	case trace.TypeSystem:
		return ev.Source == "update"
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

func frameOf(ev trace.Event) *StreamFrame {
	block := "text"
	switch ev.Type {
	case trace.TypeReasoning:
		block = "reasoning"
	case trace.TypeToolResult:
		block = "tool"
	}
	if ev.Payload != nil {
		if s, _ := ev.Payload["block"].(string); s != "" {
			block = s
		}
	}
	from := payloadInt(ev.Payload, "index_from")
	idx := payloadInt(ev.Payload, "index")
	if from == 0 {
		from = idx
	}
	return &StreamFrame{
		Kind:         "chunk",
		Index:        idx,
		IndexFrom:    from,
		Block:        block,
		Delta:        payloadDeltaText(ev),
		CommittedSeq: ev.Seq,
	}
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
