package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Hub fans session events to HTTP SSE clients without a heavy broker.
// Publish never drops: a full buffer spills to disk and a goroutine delivers.
type Hub struct {
	mu       sync.Mutex
	subs     map[string]map[chan trace.Event]struct{}
	Overflow string
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
		select {
		case ch <- ev:
		default:
			spillOverflow(overflow, ev)
			go func(ch chan trace.Event, ev trace.Event) {
				defer func() { _ = recover() }()
				ch <- ev
			}(ch, ev)
		}
	}
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
