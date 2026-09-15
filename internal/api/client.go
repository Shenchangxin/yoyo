package api

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// LineClient is a JSON-RPC 2.0 client over newline-delimited JSON.
// Notifications (no id) are delivered on Notify; requests are matched by id.
type LineClient struct {
	enc    *json.Encoder
	dec    *json.Decoder
	wmu    sync.Mutex
	mu     sync.Mutex
	next   atomic.Int64
	wait   map[int64]chan RPCResponse
	closed chan struct{}
	Notify chan RPCRequest
}

func NewLineClient(r io.Reader, w io.Writer) *LineClient {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	c := &LineClient{
		enc:    enc,
		dec:    json.NewDecoder(r),
		wait:   map[int64]chan RPCResponse{},
		closed: make(chan struct{}),
		Notify: make(chan RPCRequest, 64),
	}
	go c.readLoop()
	return c
}

func (c *LineClient) readLoop() {
	defer close(c.closed)
	defer close(c.Notify)
	for {
		var raw map[string]json.RawMessage
		if err := c.dec.Decode(&raw); err != nil {
			c.failAll(err)
			return
		}
		if _, ok := raw["method"]; ok {
			if _, hasID := raw["id"]; !hasID {
				var n RPCRequest
				_ = json.Unmarshal(mustJoin(raw), &n)
				select {
				case c.Notify <- n:
				default:
				}
				continue
			}
		}
		var res RPCResponse
		b := mustJoin(raw)
		if err := json.Unmarshal(b, &res); err != nil {
			continue
		}
		id, ok := intID(res.ID)
		if !ok {
			continue
		}
		c.mu.Lock()
		ch := c.wait[id]
		delete(c.wait, id)
		c.mu.Unlock()
		if ch != nil {
			ch <- res
		}
	}
}

func (c *LineClient) failAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.wait {
		ch <- RPCResponse{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: -32000, Message: err.Error()}}
		delete(c.wait, id)
	}
}

func (c *LineClient) Call(method string, params any) (any, error) {
	id := c.next.Add(1)
	ch := make(chan RPCResponse, 1)
	c.mu.Lock()
	c.wait[id] = ch
	c.mu.Unlock()
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	c.wmu.Lock()
	err := c.enc.Encode(req)
	c.wmu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.wait, id)
		c.mu.Unlock()
		return nil, err
	}
	select {
	case <-c.closed:
		return nil, fmt.Errorf("rpc: connection closed")
	case res := <-ch:
		if res.Error != nil {
			return nil, fmt.Errorf("%s", res.Error.Message)
		}
		return res.Result, nil
	}
}

func intID(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}

func mustJoin(raw map[string]json.RawMessage) []byte {
	b, _ := json.Marshal(raw)
	return b
}
