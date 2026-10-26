// 本文件拥有事件总线：按名字登记监听者，并用 Emit、Serial、Waterfall 分发。
// 不拥有 Fiber，也不会在 Dispose 时自动退订。退订函数要由调用方放进 Effect。

package kernel

import (
	"fmt"
	"sync"
)

// Handler 处理一条事件。返回的非 nil 值会被 Serial 或 Waterfall 采纳。
type Handler func(payload any) (any, error)

// subscription 记住一次 On 的退订 id 和回调。
type subscription struct {
	id string
	fn Handler
}

// EventBus 按事件名保存监听者。分发顺序与 On 的登记顺序一致。
type EventBus struct {
	mu sync.RWMutex
	// 每次 On 加一，从 1 起，只用来拼退订 id。
	seq      int
	handlers map[string][]subscription
}

// NewEventBus 返回没有监听者的总线。Context.New 会用它，测试也可以直接构造。
func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]subscription)}
}

// On 按登记顺序挂上 fn，并返回只摘掉这一次登记的退订函数。
// 退订可以重复调用，第二次找不到这个 id 就什么都不做。
// 序号在整条总线上递增，id 由事件名、# 和序号拼成。这里不检查 fn 是否为 nil，分发到 nil 会 panic。
// name 作为 handlers 的键，并拼进退订 id。
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

// Emit 按登记顺序调用监听者，丢掉返回值和错误。
// 调用前会复制监听列表，这一轮不受期间的 On 或退订影响。
// name 用来复制该名字下的监听列表。
// payload 交给每个监听者，返回值和错误丢掉。
func (b *EventBus) Emit(name string, payload any) {
	b.mu.RLock()
	list := append([]subscription(nil), b.handlers[name]...)
	b.mu.RUnlock()
	for _, s := range list {
		_, _ = s.fn(payload)
	}
}

// Serial 按登记顺序调用监听者。第一个非 nil 的返回值会停住后面的调用，并把该值交给调用方。
// 监听者返回错误时也停，调用方得到 nil 和这个错误，附带的返回值被丢掉。
// 全都返回 nil 且没有错误时，结果是 nil, nil。
// name 用来复制监听列表。
// payload 按登记顺序交给监听者。
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

// Waterfall 按登记顺序把载荷交给每个监听者。返回的非 nil 值成为下一位的输入。
// 返回 nil 时，下一位仍看到上一位留下的载荷。
// 任一监听者返回错误就停，调用方得到 nil 和该错误，已经改过的载荷不返回。
// 没有监听者时原样返回一开始的 payload。
// name 用来复制监听列表。
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
		// 返回 nil 不替换当前载荷。
		if out != nil {
			cur = out
		}
	}
	return cur, nil
}
