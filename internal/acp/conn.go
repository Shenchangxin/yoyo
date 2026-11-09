// Package acp 是 Agent Client Protocol v1 的客户端：按换行分隔的 JSON-RPC 2.0 连接、协议类型、启动 agent 进程并握手。
// 它不决定审批结果，也不写工人日志；这些由 Handler 的实现（acpworker、buddy 探测）负责。
// buddy 的探测和登录、acpworker 的工人、acptest 的假 agent 调用它。
package acp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// maxLine 是一行 JSON-RPC 消息的上限。超过的行整行丢掉。
const maxLine = 16 << 20

// JSON-RPC 错误码。
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeAuthRequired   = -32000
	CodeCancelled      = -32800
)

// ErrClosed 表示连接已经断开。具体原因包在里面。
var ErrClosed = errors.New("acp: connection closed")

// RPCError 是对方回的 JSON-RPC 错误。
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error 返回错误码和消息。
func (e *RPCError) Error() string {
	return fmt.Sprintf("acp: %s (%d)", e.Message, e.Code)
}

// Is 让 errors.Is(err, ErrAuthRequired) 对 -32000 或消息里带 auth 的错误成立。
func (e *RPCError) Is(target error) bool {
	if target != ErrAuthRequired {
		return false
	}
	return e.Code == CodeAuthRequired || strings.Contains(strings.ToLower(e.Message), "auth")
}

// message 是线上的一条 JSON-RPC 消息。请求、通知、响应共用。
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RequestFunc 处理对方发来的请求。返回的 error 是 *RPCError 时原样回给对方，其他错误回 -32603。
type RequestFunc func(ctx context.Context, method string, params json.RawMessage) (any, error)

// NotifyFunc 处理对方发来的通知。它在读循环里按顺序调用，不能阻塞太久。
type NotifyFunc func(method string, params json.RawMessage)

// ConnOptions 是连接两端共用的回调。
type ConnOptions struct {
	// OnRequest 为 nil 时所有请求回 -32601。
	OnRequest RequestFunc
	// OnNotify 为 nil 时通知被丢掉。
	OnNotify NotifyFunc
	// Log 记下坏行等诊断信息。可以为 nil。
	Log func(string)
	// Frame 拿到收发的每一行原文，dir 是 send 或 recv。它在读循环和写锁里调用，不能阻塞；line 只在调用期间有效。可以为 nil。
	Frame func(dir string, line []byte)
}

// Conn 是一条按换行分隔的 JSON-RPC 2.0 连接。两端都能发请求和通知。
type Conn struct {
	w    io.Writer
	wmu  sync.Mutex
	r    *bufio.Reader
	opts ConnOptions
	next atomic.Int64

	mu       sync.Mutex
	pending  map[string]chan *message
	incoming map[string]context.CancelFunc
	closed   chan struct{}
	err      error
	once     sync.Once
	base     context.Context
	stop     context.CancelFunc
}

// NewConn 在 r 和 w 上建连接，并开始读。读到 EOF 或出错时连接关闭，Done 返回的通道随之关闭。
func NewConn(r io.Reader, w io.Writer, opts ConnOptions) *Conn {
	base, stop := context.WithCancel(context.Background())
	c := &Conn{
		w:        w,
		r:        bufio.NewReaderSize(r, 64<<10),
		opts:     opts,
		pending:  map[string]chan *message{},
		incoming: map[string]context.CancelFunc{},
		closed:   make(chan struct{}),
		base:     base,
		stop:     stop,
	}
	go c.readLoop()
	return c
}

// Done 在连接关闭后关闭。
func (c *Conn) Done() <-chan struct{} { return c.closed }

// Err 返回连接关闭的原因。还开着时为 nil。
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Shutdown 用 err 关掉连接。挂着的调用都收到这个错误。重复调用只记第一次的原因。
func (c *Conn) Shutdown(err error) {
	c.once.Do(func() {
		if err == nil {
			err = ErrClosed
		}
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
		c.stop()
		close(c.closed)
	})
}

// logf 写一条诊断。
func (c *Conn) logf(format string, args ...any) {
	if c.opts.Log != nil {
		c.opts.Log(fmt.Sprintf(format, args...))
	}
}

// readLoop 逐行读消息并分派：响应交给等待的调用，请求另起协程处理，通知按顺序处理。
func (c *Conn) readLoop() {
	for {
		line, err := readLine(c.r)
		if err == errLineTooLong {
			c.logf("acp: dropped a line over %d bytes", maxLine)
			continue
		}
		if len(bytes.TrimSpace(line)) > 0 {
			if c.opts.Frame != nil {
				c.opts.Frame("recv", line)
			}
			c.dispatch(line)
		}
		if err != nil {
			if err == io.EOF {
				c.Shutdown(ErrClosed)
			} else {
				c.Shutdown(fmt.Errorf("%w: %v", ErrClosed, err))
			}
			return
		}
	}
}

// errLineTooLong 表示这一行超过 maxLine，已经读到行尾丢掉。
var errLineTooLong = errors.New("acp: line too long")

// readLine 读一整行，不含换行。超过 maxLine 时读到行尾并返回 errLineTooLong。
func readLine(r *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		chunk, err := r.ReadSlice('\n')
		if len(buf)+len(chunk) > maxLine {
			for err == bufio.ErrBufferFull {
				_, err = r.ReadSlice('\n')
			}
			if err != nil && err != io.EOF {
				return nil, err
			}
			return nil, errLineTooLong
		}
		buf = append(buf, chunk...)
		if err == bufio.ErrBufferFull {
			continue
		}
		return bytes.TrimRight(buf, "\r\n"), err
	}
}

// dispatch 处理一行。坏行记下后跳过。
func (c *Conn) dispatch(line []byte) {
	var msg message
	if err := json.Unmarshal(line, &msg); err != nil {
		c.logf("acp: skipped a bad line: %v", err)
		return
	}
	hasID := len(msg.ID) > 0 && string(msg.ID) != "null"
	switch {
	case msg.Method != "" && hasID:
		go c.serve(msg)
	case msg.Method != "":
		c.notify(msg)
	case hasID:
		c.mu.Lock()
		ch := c.pending[idKey(msg.ID)]
		delete(c.pending, idKey(msg.ID))
		c.mu.Unlock()
		if ch != nil {
			ch <- &msg
		}
	}
}

// notify 处理一条通知。$/cancel_request 取消对应的进行中请求，其余交给 OnNotify。
func (c *Conn) notify(msg message) {
	if msg.Method == "$/cancel_request" {
		var p struct {
			ID json.RawMessage `json:"id"`
		}
		if json.Unmarshal(msg.Params, &p) == nil {
			c.mu.Lock()
			cancel := c.incoming[idKey(p.ID)]
			c.mu.Unlock()
			if cancel != nil {
				cancel()
			}
		}
		return
	}
	if c.opts.OnNotify != nil {
		c.opts.OnNotify(msg.Method, msg.Params)
	}
}

// serve 处理对方的一条请求，并回结果或错误。对方用 $/cancel_request 取消时回 -32800。
func (c *Conn) serve(msg message) {
	ctx, cancel := context.WithCancel(c.base)
	key := idKey(msg.ID)
	c.mu.Lock()
	c.incoming[key] = cancel
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.incoming, key)
		c.mu.Unlock()
		cancel()
	}()
	if c.opts.OnRequest == nil {
		c.reply(msg.ID, nil, &RPCError{Code: CodeMethodNotFound, Message: "method not found: " + msg.Method})
		return
	}
	result, err := c.opts.OnRequest(ctx, msg.Method, msg.Params)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) {
			c.reply(msg.ID, nil, rpcErr)
			return
		}
		if ctx.Err() != nil {
			c.reply(msg.ID, nil, &RPCError{Code: CodeCancelled, Message: "request cancelled"})
			return
		}
		c.reply(msg.ID, nil, &RPCError{Code: CodeInternalError, Message: err.Error()})
		return
	}
	c.reply(msg.ID, result, nil)
}

// reply 写一条响应。result 为 nil 时写 null。
func (c *Conn) reply(id json.RawMessage, result any, rpcErr *RPCError) {
	out := message{JSONRPC: "2.0", ID: id}
	if rpcErr != nil {
		out.Error = rpcErr
	} else {
		raw, err := json.Marshal(result)
		if err != nil {
			out.Error = &RPCError{Code: CodeInternalError, Message: err.Error()}
		} else {
			out.Result = raw
		}
	}
	_ = c.write(out)
}

// write 把 msg 写成一行。json 会把字符串里的换行转义，所以一条消息总在一行里。
func (c *Conn) write(msg message) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	select {
	case <-c.closed:
		return c.Err()
	default:
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if c.opts.Frame != nil {
		c.opts.Frame("send", raw[:len(raw)-1])
	}
	_, err = c.w.Write(raw)
	return err
}

// Call 发请求并等结果，结果解到 result（可以为 nil）。ctx 取消时发 $/cancel_request 并返回 ctx 的错误。
func (c *Conn) Call(ctx context.Context, method string, params any, result any) error {
	id := c.next.Add(1)
	raw := json.RawMessage(strconv.FormatInt(id, 10))
	ch := make(chan *message, 1)
	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return err
	}
	c.pending[idKey(raw)] = ch
	c.mu.Unlock()
	body, err := marshalParams(params)
	if err != nil {
		c.forget(raw)
		return err
	}
	if err := c.write(message{JSONRPC: "2.0", ID: raw, Method: method, Params: body}); err != nil {
		c.forget(raw)
		return err
	}
	select {
	case msg := <-ch:
		if msg.Error != nil {
			return msg.Error
		}
		if result != nil && len(msg.Result) > 0 && string(msg.Result) != "null" {
			if err := json.Unmarshal(msg.Result, result); err != nil {
				return fmt.Errorf("acp: decode %s result: %w", method, err)
			}
		}
		return nil
	case <-ctx.Done():
		c.forget(raw)
		_ = c.Notify("$/cancel_request", map[string]any{"id": id})
		return ctx.Err()
	case <-c.closed:
		c.forget(raw)
		return c.Err()
	}
}

// forget 拿掉一个不再等的调用。
func (c *Conn) forget(id json.RawMessage) {
	c.mu.Lock()
	delete(c.pending, idKey(id))
	c.mu.Unlock()
}

// Notify 发一条通知，不等回复。
func (c *Conn) Notify(method string, params any) error {
	body, err := marshalParams(params)
	if err != nil {
		return err
	}
	return c.write(message{JSONRPC: "2.0", Method: method, Params: body})
}

// marshalParams 把 params 编成 JSON。nil 时省掉 params。
func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// idKey 把编号规范成可比较的字符串：数字原样，字符串去掉引号。
func idKey(id json.RawMessage) string {
	s := strings.TrimSpace(string(id))
	if unq, err := strconv.Unquote(s); err == nil {
		return "s:" + unq
	}
	return "n:" + s
}
