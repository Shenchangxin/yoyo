package app

import (
	"sync"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Hub fans session events to HTTP SSE clients without a heavy broker.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan trace.Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan trace.Event]struct{}{}}
}

func (h *Hub) Subscribe(sessionID string) (<-chan trace.Event, func()) {
	ch := make(chan trace.Event, 64)
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
	defer h.mu.Unlock()
	for _, id := range []string{ev.SessionID, "*"} {
		for ch := range h.subs[id] {
			select {
			case ch <- ev:
			default:
			}
		}
	}
}
