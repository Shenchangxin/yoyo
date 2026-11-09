// server.go 拥有 yoyo 内置的 MCP 控制面：HTTP 端点、按 token 的工单登记，以及会话号。外部 agent 经它调用自己那张工单的 orchestrator_* 工具。
// 不拥有工具本身（superagent 的执行器建好，app 登记），不拥有外部进程（acpworker），也不拥有 stdio 桥（bridge.go）。
// 安全边界：只监听 127.0.0.1；每张工单一个随机 Bearer token，只能列出和调用这张工单登记的工具；Host 和 Origin 必须指向本机；请求体最多 1 MB。
// 只实现 Streamable HTTP 里用不到 SSE 的部分：每个请求回一个 JSON，GET 回 405。

package mcpserve

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/version"
)

// Versions 是支持的协议版本，新的在前。对方要的版本不在里面时回第一个。
var Versions = []string{"2025-11-25", "2025-06-18", "2025-03-26"}

// MaxBody 是一个请求体的字节上限。超过时回 413。
const MaxBody = 1 << 20

// maxSessions 是一个 token 同时保留的会话数。再开新的时丢掉最早的。
const maxSessions = 8

// ErrClosed 表示控制面已经关闭，不能再登记。
var ErrClosed = errors.New("mcpserve: closed")

// JSON-RPC 错误码。
const (
	codeParse    = -32700
	codeInvalid  = -32600
	codeMethod   = -32601
	codeParams   = -32602
	codeInternal = -32603
)

// Server 是控制面。用 New 建，第一次 Register 时才监听；方法可以并发调用。
type Server struct {
	mu     sync.Mutex
	srv    *http.Server
	url    string
	closed bool
	// regs 按 token 存工单登记。
	regs map[string]*entry
	// sessions 是会话号到 token 的映射。
	sessions map[string]string
}

// entry 是一张工单的登记。names、tools、schemas 登记后不再改；sessions 只在持有 Server.mu 时读写。
type entry struct {
	ticket   string
	names    []string
	tools    map[string]dsh.Tool
	schemas  map[string]any
	sessions []string
}

// New 建一个还没监听的控制面。
func New() *Server {
	return &Server{regs: map[string]*entry{}, sessions: map[string]string{}}
}

// NewToken 返回 32 字节随机数的十六进制串，用作一张工单的 token。
func NewToken() string {
	var b [32]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Register 登记 ticketID 的工具，token 是调用它们的唯一凭据。schemas 按工具名给参数 schema，缺的退回工具自带的 Parameters。
// 第一次登记时开始监听。同一 token 再登记会整组替换工具，已开的会话保留。没有名字或 Run 的工具跳过。
func (s *Server) Register(token, ticketID string, tools []dsh.Tool, schemas map[string]any) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("mcpserve: empty token")
	}
	e := &entry{ticket: ticketID, tools: map[string]dsh.Tool{}, schemas: schemas}
	for _, tool := range tools {
		if tool.Name == "" || tool.Run == nil {
			continue
		}
		if _, dup := e.tools[tool.Name]; !dup {
			e.names = append(e.names, tool.Name)
		}
		e.tools[tool.Name] = tool
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.listenLocked(); err != nil {
		return err
	}
	if old := s.regs[token]; old != nil {
		e.sessions = old.sessions
	}
	s.regs[token] = e
	return nil
}

// Unregister 撤销 token 和它开过的会话。之后带这个 token 的请求回 401。不认识的 token 什么都不做。
func (s *Server) Unregister(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.regs[token]
	if e == nil {
		return
	}
	for _, sid := range e.sessions {
		delete(s.sessions, sid)
	}
	delete(s.regs, token)
}

// URL 返回控制面的地址 http://127.0.0.1:<port>/mcp。还没监听或已关闭时返回空串。
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

// Close 停止监听，清掉全部登记和会话。之后 Register 返回 ErrClosed。可以重复调用。
func (s *Server) Close() error {
	s.mu.Lock()
	srv := s.srv
	s.closed = true
	s.srv = nil
	s.url = ""
	s.regs = map[string]*entry{}
	s.sessions = map[string]string{}
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return srv.Close()
	}
	return nil
}

// listenLocked 在还没监听时开 127.0.0.1 上的随机端口。调用方持有 s.mu。
func (s *Server) listenLocked() error {
	if s.closed {
		return ErrClosed
	}
	if s.srv != nil {
		return nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
	s.srv = srv
	s.url = "http://" + ln.Addr().String() + "/mcp"
	go func() { _ = srv.Serve(ln) }()
	return nil
}

// ServeHTTP 处理 /mcp。依次检查路径、Host、Origin、方法和 token，再按 JSON-RPC 方法分派。
// 带会话号的请求必须是这个 token 开过、还没结束的会话，否则回 404；不带会话号的请求照常处理。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/mcp" {
		http.NotFound(w, r)
		return
	}
	if !loopbackHost(r.Host) || !loopbackOrigin(r.Header.Get("Origin")) {
		diaglog.Warn("mcpserve rejected a non-local request", "host", r.Host, "origin", r.Header.Get("Origin"))
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		w.Header().Set("Allow", "POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := bearer(r.Header.Get("Authorization"))
	s.mu.Lock()
	e := s.regs[token]
	s.mu.Unlock()
	if token == "" || e == nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="yoyo"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sid := r.Header.Get("Mcp-Session-Id")
	if r.Method == http.MethodDelete {
		s.endSession(token, sid)
		w.WriteHeader(http.StatusOK)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBody))
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	body = bytes.TrimSpace(body)
	if len(body) > 0 && body[0] == '[' {
		s.serveBatch(w, r, token, sid, e, body)
		return
	}
	var msg message
	if err := json.Unmarshal(body, &msg); err != nil {
		writeJSON(w, http.StatusBadRequest, errorReply(nil, codeParse, "parse error"))
		return
	}
	if !msg.request() {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if msg.Method == "initialize" {
		if id := s.openSession(token); id != "" {
			w.Header().Set("Mcp-Session-Id", id)
		}
	} else if sid != "" && !s.ownsSession(token, sid) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, e.handle(r.Context(), msg))
}

// serveBatch 处理一组消息。组里只有通知和响应时回 202；initialize 不能放进组里。
func (s *Server) serveBatch(w http.ResponseWriter, r *http.Request, token, sid string, e *entry, body []byte) {
	var msgs []message
	if err := json.Unmarshal(body, &msgs); err != nil || len(msgs) == 0 {
		writeJSON(w, http.StatusBadRequest, errorReply(nil, codeInvalid, "invalid batch"))
		return
	}
	if sid != "" && !s.ownsSession(token, sid) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	var out []reply
	for _, m := range msgs {
		switch {
		case !m.request():
		case m.Method == "initialize":
			out = append(out, errorReply(m.ID, codeInvalid, "initialize cannot be batched"))
		default:
			out = append(out, e.handle(r.Context(), m))
		}
	}
	if len(out) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handle 回复一个请求。
func (e *entry) handle(ctx context.Context, m message) reply {
	switch m.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(m.Params, &p)
		return result(m.ID, map[string]any{
			"protocolVersion": pickVersion(p.ProtocolVersion),
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "yoyo", "version": version.Version},
		})
	case "ping":
		return result(m.ID, map[string]any{})
	case "tools/list":
		return result(m.ID, map[string]any{"tools": e.list()})
	case "tools/call":
		return e.call(ctx, m)
	default:
		return errorReply(m.ID, codeMethod, "method not found: "+m.Method)
	}
}

// list 按登记顺序列出工具。schema 先用登记时给的，再用工具自带的 Parameters，都没有时是空对象。
func (e *entry) list() []map[string]any {
	out := make([]map[string]any, 0, len(e.names))
	for _, name := range e.names {
		tool := e.tools[name]
		schema := e.schemas[name]
		if schema == nil {
			schema = tool.Parameters
		}
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		out = append(out, map[string]any{"name": name, "description": tool.Description, "inputSchema": schema})
	}
	return out
}

// call 跑这张工单的一个工具。工具不在登记里时回参数错误；工具自己的失败写进 isError，不算协议错误。日志只记工单、工具名和是否出错，不记参数。
func (e *entry) call(ctx context.Context, m message) reply {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(m.Params, &p); err != nil {
		return errorReply(m.ID, codeParams, "invalid params")
	}
	tool, ok := e.tools[p.Name]
	if !ok {
		return errorReply(m.ID, codeParams, "unknown tool: "+p.Name)
	}
	args := bytes.TrimSpace(p.Arguments)
	if len(args) == 0 || bytes.Equal(args, []byte("null")) {
		args = []byte("{}")
	}
	res, err := tool.Run(ctx, dsh.ToolCall{ID: "mcp-" + strings.Trim(string(m.ID), `"`), Name: p.Name, Arguments: string(args)})
	text, isErr := res.Text, res.IsError
	if err != nil {
		text, isErr = err.Error(), true
	}
	diaglog.Debug("mcpserve tool call", "ticket", e.ticket, "tool", p.Name, "is_error", isErr)
	return result(m.ID, map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isErr,
	})
}

// openSession 给 token 开一个新会话并返回会话号。token 已撤销时返回空串。
func (s *Server) openSession(token string) string {
	sid := NewToken()[:32]
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.regs[token]
	if e == nil {
		return ""
	}
	if len(e.sessions) >= maxSessions {
		delete(s.sessions, e.sessions[0])
		e.sessions = e.sessions[1:]
	}
	e.sessions = append(e.sessions, sid)
	s.sessions[sid] = token
	return sid
}

// ownsSession 在 sid 是 token 开过、还没结束的会话时为真。
func (s *Server) ownsSession(token, sid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[sid] == token
}

// endSession 结束 token 的会话 sid。别人的或不存在的会话不动。
func (s *Server) endSession(token, sid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sid == "" || s.sessions[sid] != token {
		return
	}
	delete(s.sessions, sid)
	if e := s.regs[token]; e != nil {
		kept := e.sessions[:0]
		for _, id := range e.sessions {
			if id != sid {
				kept = append(kept, id)
			}
		}
		e.sessions = kept
	}
}

// message 是收到的一条 JSON-RPC 消息。
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// request 在 m 需要回复时为真。没有 method 的是对方的响应；没有 id、id 为 null 或方法以 notifications/ 开头的是通知。
func (m message) request() bool {
	if m.Method == "" || strings.HasPrefix(m.Method, "notifications/") {
		return false
	}
	id := bytes.TrimSpace(m.ID)
	return len(id) > 0 && !bytes.Equal(id, []byte("null"))
}

// reply 是发出的一条 JSON-RPC 响应。
type reply struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError 是响应里的错误。
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// result 是给 id 的成功响应。
func result(id json.RawMessage, v any) reply {
	return reply{JSONRPC: "2.0", ID: id, Result: v}
}

// errorReply 是给 id 的错误响应。id 为空时写成 null。
func errorReply(id json.RawMessage, code int, text string) reply {
	return reply{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: text}}
}

// writeJSON 用 status 写出 v。
func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// pickVersion 在 asked 受支持时原样回它，否则回最新的版本。
func pickVersion(asked string) string {
	for _, v := range Versions {
		if v == asked {
			return v
		}
	}
	return Versions[0]
}

// bearer 取出 Authorization 头里的 Bearer token。格式不对时返回空串。
func bearer(header string) string {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// loopbackHost 在 Host 头指向本机回环地址时为真，挡住 DNS 重绑定。
func loopbackHost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return loopbackName(host)
}

// loopbackOrigin 在没有 Origin 头，或 Origin 指向本机时为真，挡住网页里的脚本。Origin 为 null 时为假。
func loopbackOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return loopbackName(u.Hostname())
}

// loopbackName 在 name 是 localhost、127.0.0.1 或 ::1 时为真。
func loopbackName(name string) bool {
	name = strings.ToLower(strings.Trim(name, "[]"))
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}
