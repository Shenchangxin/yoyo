package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	coalesceWindow   = 50 * time.Millisecond
	liveLogCap       = 16384
	liveInlineBytes  = 768
	liveSinceCap     = 1024
	noticeArgBytes   = 480
	noticeBodyBytes  = 400
	noticePatchBytes = 4000
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
	Kind         string         `json:"kind,omitempty"`
	AttemptID    string         `json:"attempt_id,omitempty"`
	Revision     int            `json:"revision,omitempty"`
	Index        int            `json:"index"`
	IndexFrom    int            `json:"index_from"`
	Block        string         `json:"block,omitempty"`
	Delta        string         `json:"delta,omitempty"`
	Chunk        map[string]any `json:"chunk,omitempty"`
	Turn         int            `json:"turn,omitempty"`
	Step         int            `json:"step,omitempty"`
	Outcome      string         `json:"outcome,omitempty"`
	CommittedSeq int64          `json:"committed_seq,omitempty"`
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
	from := idx
	if payloadHas(ev.Payload, "index_from") {
		from = payloadInt(ev.Payload, "index_from")
	}
	if h.pending != nil {
		if h.pending.key == key {
			h.pending.text.WriteString(text)
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
	if ev.Payload == nil {
		ev.Payload = map[string]any{}
	}
	ev.Payload["index"] = h.pending.index
	ev.Payload["index_from"] = h.pending.indexFrom
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

// LiveBatch is one pull from the live ring. Gap means the caller cannot page
// forward: the ring rolled past the cursor, or the cursor is ahead of a
// restarted seq space, so the trajectory has to be reloaded. More means this
// page is the oldest liveSinceCap events after the cursor and the ring still
// holds later matches — apply them, then pull again; do not reload. Head is
// read under the same lock as Events.
type LiveBatch struct {
	Events []trace.Event `json:"events"`
	Head   int64         `json:"head"`
	Gap    bool          `json:"gap"`
	More   bool          `json:"more,omitempty"`
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
		// Keep the oldest page after the cursor so the renderer can apply
		// in order and pull the rest. Gap stays a hole, not a page cut.
		b.Events = b.Events[:liveSinceCap]
		b.More = true
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
	if compact := compactToolNotice(ev); compact != nil {
		n.Event = compact
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

// compactToolNotice is the Wails fast path for tool_call / tool_result: a
// name, path, and line-count ride along so the process rail can move without
// waiting for LiveSync. Heavy arguments, patches, and result bodies stay on
// the ring for a later pull when they exceed the notice caps.
func compactToolNotice(ev trace.Event) *trace.Event {
	if ev.Type != trace.TypeToolCall && ev.Type != trace.TypeToolResult {
		return nil
	}
	out := ev
	if ev.Payload == nil {
		return &out
	}
	p := make(map[string]any, 16)
	for _, k := range []string{
		"id", "name", "path", "error", "ok", "created", "progress",
		"bytes", "duration_ms", "elapsed_ms", "call_id", "round", "untrusted",
	} {
		if v, ok := ev.Payload[k]; ok {
			p[k] = v
		}
	}
	for _, k := range []string{"added", "removed", "paths"} {
		if v, ok := ev.Payload[k]; ok {
			p[k] = v
		}
	}
	if args, ok := ev.Payload["arguments"].(string); ok {
		if len(args) <= noticeArgBytes {
			p["arguments"] = args
		} else {
			p["bytes"] = len(args)
		}
	}
	if s, ok := ev.Payload["content"].(string); ok && len(s) <= noticeBodyBytes {
		p["content"] = s
	}
	if s, ok := ev.Payload["patch"].(string); ok && len(s) <= noticePatchBytes {
		p["patch"] = s
	}
	if ev.Type == trace.TypeToolResult {
		p["delta"] = false
	}
	out.Payload = p
	return &out
}

func liveDelta(ev trace.Event) bool {
	if liveProgressCall(ev) {
		return true
	}
	if ev.Type == trace.TypeAssistantStream {
		kind, _ := ev.Payload["kind"].(string)
		return kind == "chunk" || kind == "start" || kind == "end"
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
	case trace.TypeAssistantStream:
		return true
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
	if ev.Type == trace.TypeAssistantStream {
		kind, _ := ev.Payload["kind"].(string)
		return kind == "start" || kind == "end"
	}
	if ev.Payload != nil {
		if delta, _ := ev.Payload["delta"].(bool); delta {
			return false
		}
	}
	switch ev.Type {
	case trace.TypeTurnEnd, trace.TypeTurnStart, trace.TypeStepStart, trace.TypeStepEnd, trace.TypeError, trace.TypeApproval, trace.TypeUser, trace.TypeInject, trace.TypeInbox:
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
	if ev.Type == trace.TypeAssistantStream {
		kind, _ := ev.Payload["kind"].(string)
		if kind != "chunk" {
			return "", false
		}
		ch := payloadMap(ev.Payload["chunk"])
		typ, _ := ch["type"].(string)
		switch typ {
		case "text-delta", "reasoning-delta", "tool-call-delta":
		default:
			return "", false
		}
		attempt, _ := ev.Payload["attempt_id"].(string)
		idx := payloadInt(ch, "index")
		return ev.SessionID + "\x00" + attempt + "\x00" + typ + "\x00" + strconv.Itoa(idx), true
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
	if ev.Type == trace.TypeAssistantStream {
		kind, _ := ev.Payload["kind"].(string)
		idx := payloadInt(ev.Payload, "index")
		from := idx
		if payloadHas(ev.Payload, "index_from") {
			from = payloadInt(ev.Payload, "index_from")
		}
		ch := payloadMap(ev.Payload["chunk"])
		block := ""
		if ch != nil {
			if t, _ := ch["type"].(string); t != "" {
				block = t
			}
		}
		attempt, _ := ev.Payload["attempt_id"].(string)
		outcome, _ := ev.Payload["outcome"].(string)
		return &StreamFrame{
			Kind:         kind,
			AttemptID:    attempt,
			Revision:     payloadInt(ev.Payload, "revision"),
			Index:        idx,
			IndexFrom:    from,
			Block:        block,
			Delta:        payloadDeltaText(ev),
			Chunk:        ch,
			Turn:         payloadInt(ev.Payload, "turn"),
			Step:         payloadInt(ev.Payload, "step"),
			Outcome:      outcome,
			CommittedSeq: ev.Seq,
		}
	}
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
	idx := payloadInt(ev.Payload, "index")
	from := idx
	if payloadHas(ev.Payload, "index_from") {
		from = payloadInt(ev.Payload, "index_from")
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
	if ev.Type == trace.TypeAssistantStream {
		ch := payloadMap(ev.Payload["chunk"])
		if ch == nil {
			return ""
		}
		if t, _ := ch["text"].(string); t != "" {
			return t
		}
		if t, _ := ch["argumentsDelta"].(string); t != "" {
			return t
		}
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
	if ev.Type == trace.TypeAssistantStream {
		ch := payloadMap(ev.Payload["chunk"])
		if ch == nil {
			ch = map[string]any{}
		} else {
			cp := make(map[string]any, len(ch)+1)
			for k, v := range ch {
				cp[k] = v
			}
			ch = cp
		}
		typ, _ := ch["type"].(string)
		if typ == "tool-call-delta" {
			ch["argumentsDelta"] = text
		} else {
			ch["text"] = text
		}
		ev.Payload["chunk"] = ch
		return
	}
	if ev.Type == trace.TypeToolResult {
		ev.Payload["content"] = text
		return
	}
	ev.Payload["text"] = text
}

func payloadHas(p map[string]any, key string) bool {
	if p == nil {
		return false
	}
	_, ok := p[key]
	return ok
}

func payloadMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
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
