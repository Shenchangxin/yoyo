// 把一次审批挂起，直到 Resolve 写入决定或上下文取消。不解释能力级别，也不做路径判断。
// app 在启动时调用 SetOnOffer，会话路径调用 AskOffer。

package capability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// Offer 是一条待决的人工或审核代理审批。
type Offer struct {
	ID      string  `json:"id"`
	Request Request `json:"request"`
}

// Gate 把调用方停住，直到决定到达。它把 Broker 的审批函数接到 app。
type Gate struct {
	mu      sync.Mutex
	pending map[string]*wait
	onOffer func(Offer)
}

// wait 是一条待决审批。ch 的缓冲是 1，Resolve 可以在 AskOffer 开始接收之前把决定放进去。
type wait struct {
	offer Offer
	ch    chan Decision
}

// NewGate 返回待决表为空的 Gate。
func NewGate() *Gate {
	return &Gate{pending: map[string]*wait{}}
}

// SetOnOffer 替换新 Offer 出现时的回调。fn 可以为 nil。
func (g *Gate) SetOnOffer(fn func(Offer)) {
	g.mu.Lock()
	g.onOffer = fn
	g.mu.Unlock()
}

// Ask 调用 AskOffer，丢掉 Offer，只返回决定和错误。
// ctx 交给 AskOffer。
// req 交给 AskOffer。
func (g *Gate) Ask(ctx context.Context, req Request) (Decision, error) {
	d, _, err := g.AskOffer(ctx, req)
	return d, err
}

// AskOffer 登记一条待决审批并阻塞。ctx 为 nil 时用 context.Background。id 由 newOfferID 生成。
// 登记后先放开锁，再调用 onOffer。Resolve 送来决定则返回该决定；ctx 取消时从待决表删除，并返回 Deny、这份 Offer 和 ctx.Err()。
// req 放进 Offer 的 Request。
func (g *Gate) AskOffer(ctx context.Context, req Request) (Decision, Offer, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	id := newOfferID()
	w := &wait{offer: Offer{ID: id, Request: req}, ch: make(chan Decision, 1)}
	g.mu.Lock()
	g.pending[id] = w
	on := g.onOffer
	g.mu.Unlock()
	if on != nil {
		on(w.offer)
	}
	select {
	case d := <-w.ch:
		return d, w.offer, nil
	case <-ctx.Done():
		g.mu.Lock()
		delete(g.pending, id)
		g.mu.Unlock()
		return Deny, w.offer, ctx.Err()
	}
}

// Resolve 按 id 取出待决项并移出表，再非阻塞地把决定送入通道。
// id 不在表里时返回 unknown approval。通道里已经有尚未取走的决定时返回 already resolved。
func (g *Gate) Resolve(id string, d Decision) error {
	g.mu.Lock()
	w, ok := g.pending[id]
	if ok {
		delete(g.pending, id)
	}
	g.mu.Unlock()
	if !ok {
		return fmt.Errorf("capability: unknown approval %s", id)
	}
	select {
	case w.ch <- d:
		return nil
	default:
		return fmt.Errorf("capability: approval %s already resolved", id)
	}
}

// Pending 返回当前待决 Offer 的副本。遍历 map，顺序不稳定。
func (g *Gate) Pending() []Offer {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Offer, 0, len(g.pending))
	for _, w := range g.pending {
		out = append(out, w.offer)
	}
	return out
}

// newOfferID 读取 8 字节随机数并编成十六进制。rand.Read 的错误被忽略，失败时仍编码这 8 字节。
func newOfferID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
