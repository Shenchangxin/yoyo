package kernel

import (
	"fmt"
	"sync"
)

// Handler receives an event payload and may return a replacement.
type Handler func(payload any) (any, error)

type subscription struct {
	id string
	fn Handler
}

// EventBus implements emit / serial / waterfall dispatch.
type EventBus struct {
	mu       sync.RWMutex
	seq      int
	handlers map[string][]subscription
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]subscription)}
}

// On registers a listener. The returned function unsubscribes it.
func (b *EventBus) On(name string, fn Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	id := fmt.Sprintf("%s#%d", name, b.seq)
	b.handlers[name] = append(b.handlers[name], subscription{id: id, fn: fn})
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.handlers[name]
		out := list[:0]
		for _, s := range list {
			if s.id != id {
				out = append(out, s)
			}
		}
		b.handlers[name] = out
	}
}

// Emit fires listeners and ignores return values (observe-only).
func (b *EventBus) Emit(name string, payload any) {
	b.mu.RLock()
	list := append([]subscription(nil), b.handlers[name]...)
	b.mu.RUnlock()
	for _, s := range list {
		_, _ = s.fn(payload)
	}
}

// Serial runs listeners in order; the first non-nil result wins and stops the rest.
func (b *EventBus) Serial(name string, payload any) (any, error) {
	b.mu.RLock()
	list := append([]subscription(nil), b.handlers[name]...)
	b.mu.RUnlock()
	for _, s := range list {
		out, err := s.fn(payload)
		if err != nil {
			return nil, err
		}
		if out != nil {
			return out, nil
		}
	}
	return nil, nil
}

// Waterfall threads the payload through every listener (around-middleware).
func (b *EventBus) Waterfall(name string, payload any) (any, error) {
	b.mu.RLock()
	list := append([]subscription(nil), b.handlers[name]...)
	b.mu.RUnlock()
	cur := payload
	for _, s := range list {
		out, err := s.fn(cur)
		if err != nil {
			return nil, err
		}
		if out != nil {
			cur = out
		}
	}
	return cur, nil
}
