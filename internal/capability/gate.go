package capability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// Offer is a pending human (or reviewer-agent) approval.
type Offer struct {
	ID      string  `json:"id"`
	Request Request `json:"request"`
}

// Gate pauses the agent loop until a decision arrives. This is the missing
// wire between Broker.ask and the desktop/CLI/App-Server surfaces.
type Gate struct {
	mu      sync.Mutex
	pending map[string]*wait
	onOffer func(Offer)
}

type wait struct {
	offer Offer
	ch    chan Decision
}

func NewGate() *Gate {
	return &Gate{pending: map[string]*wait{}}
}

func (g *Gate) SetOnOffer(fn func(Offer)) {
	g.mu.Lock()
	g.onOffer = fn
	g.mu.Unlock()
}

func (g *Gate) Ask(ctx context.Context, req Request) (Decision, error) {
	d, _, err := g.AskOffer(ctx, req)
	return d, err
}

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

func (g *Gate) Pending() []Offer {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Offer, 0, len(g.pending))
	for _, w := range g.pending {
		out = append(out, w.offer)
	}
	return out
}

func newOfferID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
