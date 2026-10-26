package api

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// LineClient 是换行分隔的 JSON-RPC 2.0 客户端。没有 id 的通知送进 Notify，带 id 的应答按 id 配给 Call。
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

// NewLineClient 用 w 写请求、r 读应答，并关掉 HTML 转义。Notify 缓冲 64。返回前已经启动 readLoop。
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

// readLoop 不断解码 JSON 对象。带 method 且没有 id 的帧送进 Notify，缓冲满则丢掉。其余帧按 RPCResponse 的 id 交给正在等的 Call。读失败时 failAll 并关掉 closed 和 Notify。
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

// failAll 给每个还在等的 id 送一条 code 为 -32000 的错误应答，然后清空等待表。
func (c *LineClient) failAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.wait {
		ch <- RPCResponse{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: -32000, Message: err.Error()}}
		delete(c.wait, id)
	}
}

// Call 分配递增 id，把 jsonrpc、id、method 和可选 params 写出去，再阻塞等待同一 id 的应答。连接关闭时返回 rpc: connection closed。应答带 Error 时返回它的 Message。
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

// intID 把 JSON 数字收成 int64。接受 float64、int64、int 和 json.Number。其他类型返回假。
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

// mustJoin 把解码出来的字段重新编成一段 JSON，供再解析成 RPCRequest 或 RPCResponse。
func mustJoin(raw map[string]json.RawMessage) []byte {
	b, _ := json.Marshal(raw)
	return b
}
