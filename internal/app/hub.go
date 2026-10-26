// 把会话事件分给订阅者，并用序号环留住错过的实时事件。

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

// Hub 把会话事件分给订阅者。背压时实时增量先写入 Overflow 再丢掉，终态事件仍会送出。相邻增量按时间窗合并，扇出记入序号环。
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
	// coalesceWindow 是相邻增量合并前等待的时间。
	coalesceWindow = 50 * time.Millisecond
	// liveLogCap 是序号环最多保留的条数，满了就覆盖最老的。
	liveLogCap = 16384
	// liveInlineBytes 是整份事件放进通知的字节上限。
	liveInlineBytes = 768
	// liveSinceCap 是一页最多返回的事件数，多出来的标 More。
	liveSinceCap = 1024
	// noticeArgBytes 是工具参数能放进压缩通知的字节上限。
	noticeArgBytes = 480
	// noticeBodyBytes 是工具结果正文能放进压缩通知的字节上限。
	noticeBodyBytes = 400
	// noticePatchBytes 是补丁能放进压缩通知的字节上限。
	noticePatchBytes = 4000
)

// liveItem 是序号环里的一条已发布事件。
type liveItem struct {
	seq int64
	ev  trace.Event
}

// deltaBuf 暂存等待合并的一笔增量，并记住 index 范围。
type deltaBuf struct {
	ev        trace.Event
	key       string
	text      strings.Builder
	indexFrom int
	index     int
}

// StreamFrame 是实时片段的旁路帧，不写入 JSONL。
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

// LiveNotice 是给界面的通知。小事件内联，大的只留序号，由 LiveSince 去拉。
type LiveNotice struct {
	Seq       int64        `json:"seq"`
	SessionID string       `json:"session_id"`
	Type      string       `json:"type"`
	Event     *trace.Event `json:"event,omitempty"`
	Frame     *StreamFrame `json:"frame,omitempty"`
}

// NewHub 做一个还没有订阅者的 Hub。
func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan trace.Event]struct{})}
}

// Subscribe 注册对 sessionID 的订阅，返回带缓冲的通道和取消函数。
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

// Publish 把能合并的增量先暂存；其他事件先冲掉暂存再扇出。
// ev 传给 coalesceKey；能合并则传给 holdDelta，否则先 FlushCoalesce 再传给 fanout。
func (h *Hub) Publish(ev trace.Event) {
	if key, ok := coalesceKey(ev); ok {
		h.holdDelta(ev, key)
		return
	}
	h.FlushCoalesce()
	h.fanout(ev)
}

// FlushCoalesce 立刻发出暂存的增量。测试用它代替等待。
func (h *Hub) FlushCoalesce() {
	h.cMu.Lock()
	h.flushPendingLocked()
	h.cMu.Unlock()
}

// holdDelta 把同一合并键的文本拼在一起，并保留更大的 index。键变了先冲掉上一笔，再起定时器。
// ev 传给 payloadDeltaText 和 cloneEvent，并用来读 index 和 index_from。
// key 与 pending.key 相同则把文本拼上去，否则先冲掉上一笔，再作为新的合并键。
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

// flushPendingLocked 在已持有合并锁时把暂存文本和 index 范围写回事件并扇出。扇出前会放开这把锁。
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

// fanout 先记入序号环，再发给该会话和通配订阅者。
// ev 传给 record，并按 SessionID 发给该会话和通配订阅者的 deliver。
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

// record 把事件放进序号环并返回新序号。环满了就按序号覆盖最老的位置。
// ev 经 cloneEvent 后写入序号环。
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

// Head 返回最新序号。没有发布过或 Hub 为 nil 时返回 0。
func (h *Hub) Head() int64 {
	if h == nil {
		return 0
	}
	h.logMu.Lock()
	defer h.logMu.Unlock()
	return h.seq
}

// LiveBatch 是一次从序号环拉取的结果。Gap 表示不能顺着补，More 表示这一页后面还有，Head 与事件在同一把锁下读出。
type LiveBatch struct {
	Events []trace.Event `json:"events"`
	Head   int64         `json:"head"`
	Gap    bool          `json:"gap"`
	More   bool          `json:"more,omitempty"`
}

// Batch 返回序号大于 after 的事件，可按会话过滤，并带上 head、Gap 和 More。Hub 为 nil 时返回空页。
// sessionID 非空时只保留该会话或 SessionID 为空的事件。
func (h *Hub) Batch(after int64, sessionID string) LiveBatch {
	if h == nil {
		return LiveBatch{Events: []trace.Event{}}
	}
	h.logMu.Lock()
	defer h.logMu.Unlock()
	b := LiveBatch{Events: make([]trace.Event, 0, 16), Head: h.seq}
	// 环里的序号连续，并以 h.seq 结尾，所以仍保留的最老序号是 h.seq 减去长度再加一。落在 (after, oldest) 里的已经被覆盖。
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
		// 保留游标之后最老的一页，渲染方按序应用再继续拉。Gap 表示空洞，不是这一页被截断。
		b.Events = b.Events[:liveSinceCap]
		b.More = true
	}
	return b
}

// Since 只返回 Batch 里的事件，不带 Gap。要区分空洞和分页时应使用 Batch。
// after 传给 Batch。
// sessionID 传给 Batch。
func (h *Hub) Since(after int64, sessionID string) []trace.Event {
	return h.Batch(after, sessionID).Events
}

// NoticeOf 做成界面通知。增量带上事件和帧；工具调用和结果走压缩字段；其余超过内联上限的只留序号。
// ev 用来填 Seq、SessionID 和 Type，并传给 liveDelta、frameOf、compactToolNotice 和 mayInlineNotice。
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

// compactToolNotice 给工具调用和结果带上名称、路径和短字段。超限的参数、补丁和结果正文留在环上。
// ev 的 Type 不是工具调用或工具结果时返回 nil，否则从 Payload 抄名称、路径和短字段。
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

// liveDelta 判断事件是不是应合并的实时增量，包括进度中的工具调用和助手流的片段。
// ev 传给 liveProgressCall，并用来读 Type 和 Payload 里的 kind 或 delta。
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

// liveProgressCall 判断这是不是带 progress 的工具调用。
// ev 的 Type 不是工具调用或 Payload 为 nil 时返回 false，否则读 progress。
func liveProgressCall(ev trace.Event) bool {
	if ev.Type != trace.TypeToolCall || ev.Payload == nil {
		return false
	}
	p, _ := ev.Payload["progress"].(bool)
	return p
}

// mayInlineNotice 判断事件能否整份放进通知。已落定的助手和推理正文不内联。
// ev 传给 liveProgressCall，并用来读 Type 和 Payload 里的 delta。
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

// deliver 能写入通道就直接送。通道满了先把事件写入 Overflow；终态事件再另起发送，实时增量到此丢掉。
// ch 用来发送 ev，写不进且仍要保留时再另起发送。
// ev 写入 ch，写不进时传给 spillOverflow。
// overflow 传给 spillOverflow。
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

// KeepHubEvent 为真表示背压下这条事件仍要送到慢的界面。
// ev 传给 keepUnderBackpressure。
func KeepHubEvent(ev trace.Event) bool {
	return keepUnderBackpressure(ev)
}

// keepUnderBackpressure 保留回合与步骤的起止、错误、审批、用户、注入、收件箱，以及来源是 update 的 system。带 delta 的增量不保留。
// ev 用来读 Type、Payload 里的 kind 或 delta，以及 Source。
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

// coalesceKey 给可合并的助手流或带 delta 的事件生成键。键相同的相邻增量会拼在一起。
// ev 的 Payload 为 nil 时返回空串和 false；助手流看 chunk 的 type，带 delta 的助手、推理和工具结果用 SessionID、Type、id 和 round 拼键。
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

// frameOf 从事件抽出实时帧，带上 index 范围和块类型。
// ev 用来读 Type、Payload 和 Seq，并传给 payloadDeltaText。
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

// payloadDeltaText 取出增量文本。助手流看 chunk，工具结果看 content，其余看 text。
// ev 的 Payload 为 nil 时返回空串；助手流从 chunk 取 text 或 argumentsDelta，工具结果取 content，其余取 text。
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

// setDeltaText 把合并后的文本写回事件里对应的字段。
// ev 的 Payload 为 nil 时先建成空 map；助手流写 chunk，工具结果写 content，其余写 text。
// text 在助手流的 tool-call-delta 时写入 argumentsDelta，其他助手流写入 chunk 的 text，工具结果写入 content，其余写入 Payload 的 text。
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

// payloadHas 判断载荷里有没有这个键。
// key 用作 p 的查找键。
func payloadHas(p map[string]any, key string) bool {
	if p == nil {
		return false
	}
	_, ok := p[key]
	return ok
}

// payloadMap 在值是 map 时返回它，否则返回 nil。
// v 为 nil 或不是 map[string]any 时返回 nil，否则返回该 map。
func payloadMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// cloneEvent 拷贝事件，并单独拷贝载荷 map。
// ev 整份拷贝；Payload 为 nil 时不另建 map，否则单独拷贝载荷。
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

// spillOverflow 把事件追加进 Overflow 目录下该会话的 jsonl。目录为空则什么都不做。
// dir 为空则返回，否则建目录并在其下追加 jsonl。
// ev 用来读 SessionID，并编码写入该文件。
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
