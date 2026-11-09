package mcpserve

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// echo 造一个名为 name 的工具，回 "name 参数"。
func echo(name string) dsh.Tool {
	return dsh.Tool{
		Name:        name,
		Description: "echo " + name,
		Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			return dsh.Result{Text: name + " " + call.Arguments}, nil
		},
	}
}

// serveEcho 建一个控制面，给工单 t1 登记 orchestrator_report 和 orchestrator_follow 两个 echo 工具；只有前者带 schema。
func serveEcho(t *testing.T) (*Server, string) {
	t.Helper()
	s := New()
	t.Cleanup(func() { _ = s.Close() })
	token := NewToken()
	schemas := map[string]any{"orchestrator_report": map[string]any{
		"type":       "object",
		"properties": map[string]any{"state": map[string]any{"type": "string"}},
		"required":   []string{"state"},
	}}
	if err := s.Register(token, "t1", []dsh.Tool{echo("orchestrator_report"), echo("orchestrator_follow")}, schemas); err != nil {
		t.Fatal(err)
	}
	return s, token
}

// req 是一次 HTTP 请求的可选项。auth 原样写进 Authorization 头；host 非空时替换 Host 头。
type req struct {
	auth, session, origin, host string
}

// as 返回用 token 作 Bearer 的请求项。
func as(token string) req { return req{auth: "Bearer " + token} }

// do 向控制面发一个请求，返回响应和读完的正文。
func do(t *testing.T, s *Server, o req, method, body string) (*http.Response, []byte) {
	t.Helper()
	r, err := http.NewRequest(method, s.URL(), strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	if o.auth != "" {
		r.Header.Set("Authorization", o.auth)
	}
	if o.session != "" {
		r.Header.Set("Mcp-Session-Id", o.session)
	}
	if o.origin != "" {
		r.Header.Set("Origin", o.origin)
	}
	if o.host != "" {
		r.Host = o.host
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, data
}

// rpcBody 拼一条 JSON-RPC 请求。
func rpcBody(id int, method string, params any) string {
	data, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	return string(data)
}

// rpcOut 是测试里解开的 JSON-RPC 响应。
type rpcOut struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// parse 把 data 解成一条新的 JSON-RPC 响应。
func parse(t *testing.T, data []byte) rpcOut {
	t.Helper()
	var out rpcOut
	decode(t, data, &out)
	return out
}

// decode 把 data 解成 v，失败时测试失败。
func decode(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
}

// callText 解开 tools/call 的结果，返回第一段文字和 isError。
func callText(t *testing.T, raw json.RawMessage) (string, bool) {
	t.Helper()
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	decode(t, raw, &res)
	if len(res.Content) != 1 || res.Content[0].Type != "text" {
		t.Fatalf("content = %s", raw)
	}
	return res.Content[0].Text, res.IsError
}

// TestHandshakeListsOnlyRegisteredTools 锁住登记前不监听；握手回协商的版本、tools 能力、serverInfo 和会话号；
// 通知回 202；工具按登记顺序列出，schema 先用登记时给的，没有时是空对象；ping 回空对象，未知方法和 GET 被拒。
func TestHandshakeListsOnlyRegisteredTools(t *testing.T) {
	if New().URL() != "" {
		t.Fatal("listening before the first Register")
	}
	s, token := serveEcho(t)
	if u := s.URL(); !strings.HasPrefix(u, "http://127.0.0.1:") || !strings.HasSuffix(u, "/mcp") {
		t.Fatalf("url = %q", u)
	}
	resp, data := do(t, s, as(token), http.MethodPost, rpcBody(1, "initialize", map[string]any{"protocolVersion": "2025-06-18"}))
	sid := resp.Header.Get("Mcp-Session-Id")
	if resp.StatusCode != http.StatusOK || sid == "" {
		t.Fatalf("initialize = %d session %q: %s", resp.StatusCode, sid, data)
	}
	var out rpcOut
	out = parse(t, data)
	var init struct {
		ProtocolVersion string         `json:"protocolVersion"`
		Capabilities    map[string]any `json:"capabilities"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	decode(t, out.Result, &init)
	if init.ProtocolVersion != "2025-06-18" || init.Capabilities["tools"] == nil || init.ServerInfo.Name != "yoyo" || init.ServerInfo.Version == "" {
		t.Fatalf("initialize result = %s", out.Result)
	}
	_, data = do(t, s, as(token), http.MethodPost, rpcBody(2, "initialize", map[string]any{"protocolVersion": "1999-01-01"}))
	out = parse(t, data)
	decode(t, out.Result, &init)
	if init.ProtocolVersion != Versions[0] {
		t.Fatalf("unknown version answered with %q", init.ProtocolVersion)
	}

	withSession := as(token)
	withSession.session = sid
	for _, note := range []string{
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":9,"method":"notifications/initialized","params":{}}`,
		`{"jsonrpc":"2.0","id":5,"result":{}}`,
	} {
		if resp, data := do(t, s, withSession, http.MethodPost, note); resp.StatusCode != http.StatusAccepted || len(data) != 0 {
			t.Fatalf("%s = %d %s", note, resp.StatusCode, data)
		}
	}

	_, data = do(t, s, withSession, http.MethodPost, rpcBody(3, "tools/list", map[string]any{}))
	out = parse(t, data)
	var list struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	decode(t, out.Result, &list)
	if len(list.Tools) != 2 || list.Tools[0].Name != "orchestrator_report" || list.Tools[1].Name != "orchestrator_follow" {
		t.Fatalf("tools = %s", out.Result)
	}
	if list.Tools[0].InputSchema["required"] == nil || list.Tools[0].Description != "echo orchestrator_report" {
		t.Fatalf("registered schema lost: %s", out.Result)
	}
	if list.Tools[1].InputSchema["type"] != "object" || len(list.Tools[1].InputSchema) != 1 {
		t.Fatalf("default schema = %v", list.Tools[1].InputSchema)
	}

	_, data = do(t, s, withSession, http.MethodPost, rpcBody(4, "ping", nil))
	out = parse(t, data)
	if string(out.Result) != "{}" || out.Error != nil {
		t.Fatalf("ping = %s", data)
	}
	_, data = do(t, s, withSession, http.MethodPost, rpcBody(5, "resources/list", nil))
	out = parse(t, data)
	if out.Error == nil || out.Error.Code != codeMethod || string(out.ID) != "5" {
		t.Fatalf("unknown method = %s", data)
	}
	if resp, _ := do(t, s, withSession, http.MethodGet, ""); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET = %d", resp.StatusCode)
	}
	if resp, _ := do(t, s, as(token), http.MethodPost, `{"jsonrpc":`); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("parse error = %d", resp.StatusCode)
	}
}

// TestTokenOnlyReachesItsTicket 锁住一个 token 只能列出和调用自己工单的工具：调别的工单才有的工具回参数错误，那边的工具不会被跑。
func TestTokenOnlyReachesItsTicket(t *testing.T) {
	s := New()
	t.Cleanup(func() { _ = s.Close() })
	var mu sync.Mutex
	var ran []string
	record := func(ticket, name string) dsh.Tool {
		return dsh.Tool{Name: name, Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			mu.Lock()
			ran = append(ran, ticket+" "+name+" "+call.Arguments)
			mu.Unlock()
			return dsh.Result{Text: "ok " + ticket}, nil
		}}
	}
	a, b := NewToken(), NewToken()
	if a == b || len(a) != 64 {
		t.Fatalf("tokens %q %q", a, b)
	}
	if err := s.Register(a, "ta", []dsh.Tool{record("ta", "orchestrator_report")}, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(b, "tb", []dsh.Tool{record("tb", "orchestrator_report"), record("tb", "only_b")}, nil); err != nil {
		t.Fatal(err)
	}

	_, data := do(t, s, as(a), http.MethodPost, rpcBody(1, "tools/call", map[string]any{"name": "orchestrator_report", "arguments": map[string]any{"state": "progress"}}))
	var out rpcOut
	out = parse(t, data)
	if text, isErr := callText(t, out.Result); text != "ok ta" || isErr {
		t.Fatalf("a's report = %q %v", text, isErr)
	}
	_, data = do(t, s, as(a), http.MethodPost, rpcBody(2, "tools/call", map[string]any{"name": "only_b", "arguments": map[string]any{}}))
	out = parse(t, data)
	if out.Error == nil || out.Error.Code != codeParams || out.Result != nil {
		t.Fatalf("a calling b's tool = %s", data)
	}
	_, data = do(t, s, as(a), http.MethodPost, rpcBody(3, "tools/list", nil))
	if strings.Contains(string(data), "only_b") {
		t.Fatalf("a sees b's tool: %s", data)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(ran) != 1 || ran[0] != `ta orchestrator_report {"state":"progress"}` {
		t.Fatalf("ran = %q", ran)
	}
}

// TestRequestsNeedAValidBearerToken 锁住没带 token、不是 Bearer、token 为空或不认识时都回 401 并带 WWW-Authenticate，工具不会被跑。
func TestRequestsNeedAValidBearerToken(t *testing.T) {
	s := New()
	t.Cleanup(func() { _ = s.Close() })
	var runs atomic.Int32
	tool := dsh.Tool{Name: "orchestrator_report", Run: func(context.Context, dsh.ToolCall) (dsh.Result, error) {
		runs.Add(1)
		return dsh.Result{Text: "ran"}, nil
	}}
	token := NewToken()
	if err := s.Register(token, "t1", []dsh.Tool{tool}, nil); err != nil {
		t.Fatal(err)
	}
	body := rpcBody(1, "tools/call", map[string]any{"name": "orchestrator_report"})
	for _, auth := range []string{"", "Basic " + token, "Bearer", "Bearer   ", "Bearer " + NewToken(), token} {
		resp, _ := do(t, s, req{auth: auth}, http.MethodPost, body)
		if resp.StatusCode != http.StatusUnauthorized || !strings.HasPrefix(resp.Header.Get("WWW-Authenticate"), "Bearer") {
			t.Fatalf("auth %q = %d", auth, resp.StatusCode)
		}
	}
	if resp, _ := do(t, s, req{}, http.MethodDelete, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("DELETE without token = %d", resp.StatusCode)
	}
	if runs.Load() != 0 {
		t.Fatalf("tool ran %d times without a valid token", runs.Load())
	}
	if resp, _ := do(t, s, req{auth: "bearer " + token}, http.MethodPost, body); resp.StatusCode != http.StatusOK || runs.Load() != 1 {
		t.Fatalf("lower-case scheme = %d runs %d", resp.StatusCode, runs.Load())
	}
	if err := s.Register("  ", "t2", []dsh.Tool{tool}, nil); err == nil {
		t.Fatal("blank token registered")
	}
}

// TestForeignOriginAndHostAreForbidden 锁住 Origin 不是本机、Origin 为 null、Host 不是本机时回 403，带着正确 token 也一样；本机 Origin 照常通过。
func TestForeignOriginAndHostAreForbidden(t *testing.T) {
	s := New()
	t.Cleanup(func() { _ = s.Close() })
	var runs atomic.Int32
	tool := dsh.Tool{Name: "orchestrator_report", Run: func(context.Context, dsh.ToolCall) (dsh.Result, error) {
		runs.Add(1)
		return dsh.Result{Text: "ran"}, nil
	}}
	token := NewToken()
	if err := s.Register(token, "t1", []dsh.Tool{tool}, nil); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(s.URL())
	if err != nil {
		t.Fatal(err)
	}
	body := rpcBody(1, "tools/call", map[string]any{"name": "orchestrator_report"})
	for _, o := range []req{
		{origin: "https://evil.example"},
		{origin: "http://localhost.evil.example"},
		{origin: "null"},
		{host: "evil.example:" + u.Port()},
		{host: "127.0.0.1.nip.io:" + u.Port()},
	} {
		o.auth = "Bearer " + token
		if resp, _ := do(t, s, o, http.MethodPost, body); resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%+v = %d", o, resp.StatusCode)
		}
	}
	if runs.Load() != 0 {
		t.Fatalf("tool ran %d times from a foreign page", runs.Load())
	}
	for _, origin := range []string{"http://localhost:5173", "http://127.0.0.1:9", "http://[::1]:9"} {
		o := as(token)
		o.origin = origin
		if resp, data := do(t, s, o, http.MethodPost, body); resp.StatusCode != http.StatusOK {
			t.Fatalf("origin %s = %d %s", origin, resp.StatusCode, data)
		}
	}
	o := as(token)
	o.host = "localhost:" + u.Port()
	if resp, _ := do(t, s, o, http.MethodPost, body); resp.StatusCode != http.StatusOK || runs.Load() != 4 {
		t.Fatalf("localhost host = %d runs %d", resp.StatusCode, runs.Load())
	}
}

// TestUnregisterRevokesTokenAndSessions 锁住撤销后原 token 回 401；重新登记的新 token 带旧会话号回 404，不带会话号照常处理。
func TestUnregisterRevokesTokenAndSessions(t *testing.T) {
	s, token := serveEcho(t)
	resp, _ := do(t, s, as(token), http.MethodPost, rpcBody(1, "initialize", nil))
	sid := resp.Header.Get("Mcp-Session-Id")
	s.Unregister(token)
	s.Unregister("never-registered")
	call := rpcBody(2, "tools/call", map[string]any{"name": "orchestrator_report"})
	if resp, _ := do(t, s, as(token), http.MethodPost, call); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked token = %d", resp.StatusCode)
	}
	fresh := NewToken()
	if err := s.Register(fresh, "t1", []dsh.Tool{echo("orchestrator_report")}, nil); err != nil {
		t.Fatal(err)
	}
	stale := as(fresh)
	stale.session = sid
	if resp, _ := do(t, s, stale, http.MethodPost, call); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("old session on a new token = %d", resp.StatusCode)
	}
	if resp, data := do(t, s, as(fresh), http.MethodPost, call); resp.StatusCode != http.StatusOK {
		t.Fatalf("new token without a session = %d %s", resp.StatusCode, data)
	}
}

// TestDeleteEndsOnlyYourOwnSession 锁住 DELETE 只结束这个 token 开的会话：别的 token 删不掉，自己删掉后再带它回 404。
func TestDeleteEndsOnlyYourOwnSession(t *testing.T) {
	s, token := serveEcho(t)
	other := NewToken()
	if err := s.Register(other, "t2", []dsh.Tool{echo("orchestrator_report")}, nil); err != nil {
		t.Fatal(err)
	}
	resp, _ := do(t, s, as(token), http.MethodPost, rpcBody(1, "initialize", nil))
	mine := as(token)
	mine.session = resp.Header.Get("Mcp-Session-Id")
	theirs := as(other)
	theirs.session = mine.session
	if resp, _ := do(t, s, theirs, http.MethodDelete, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("foreign DELETE = %d", resp.StatusCode)
	}
	if resp, _ := do(t, s, theirs, http.MethodPost, rpcBody(2, "ping", nil)); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("other token using my session = %d", resp.StatusCode)
	}
	if resp, _ := do(t, s, mine, http.MethodPost, rpcBody(3, "ping", nil)); resp.StatusCode != http.StatusOK {
		t.Fatalf("session ended by another token: %d", resp.StatusCode)
	}
	if resp, _ := do(t, s, mine, http.MethodDelete, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE = %d", resp.StatusCode)
	}
	if resp, _ := do(t, s, mine, http.MethodPost, rpcBody(4, "ping", nil)); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ended session = %d", resp.StatusCode)
	}
}

// TestSessionsPerTokenAreCapped 锁住一个 token 最多留 maxSessions 个会话，再开新的时最早的那个失效。
func TestSessionsPerTokenAreCapped(t *testing.T) {
	s, token := serveEcho(t)
	var sids []string
	for i := 0; i <= maxSessions; i++ {
		resp, _ := do(t, s, as(token), http.MethodPost, rpcBody(i, "initialize", nil))
		sids = append(sids, resp.Header.Get("Mcp-Session-Id"))
	}
	first, last := as(token), as(token)
	first.session, last.session = sids[0], sids[maxSessions]
	if resp, _ := do(t, s, first, http.MethodPost, rpcBody(1, "ping", nil)); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("oldest session = %d", resp.StatusCode)
	}
	if resp, _ := do(t, s, last, http.MethodPost, rpcBody(1, "ping", nil)); resp.StatusCode != http.StatusOK {
		t.Fatalf("newest session = %d", resp.StatusCode)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sessions) != maxSessions {
		t.Fatalf("sessions = %d", len(s.sessions))
	}
}

// TestOversizedBodyIsRejected 锁住超过 1 MB 的请求体回 413，工具不会被跑。
func TestOversizedBodyIsRejected(t *testing.T) {
	s := New()
	t.Cleanup(func() { _ = s.Close() })
	var runs atomic.Int32
	tool := dsh.Tool{Name: "orchestrator_report", Run: func(context.Context, dsh.ToolCall) (dsh.Result, error) {
		runs.Add(1)
		return dsh.Result{}, nil
	}}
	token := NewToken()
	if err := s.Register(token, "t1", []dsh.Tool{tool}, nil); err != nil {
		t.Fatal(err)
	}
	body := rpcBody(1, "tools/call", map[string]any{"name": "orchestrator_report", "arguments": map[string]any{"summary": strings.Repeat("a", MaxBody)}})
	if resp, _ := do(t, s, as(token), http.MethodPost, body); resp.StatusCode != http.StatusRequestEntityTooLarge || runs.Load() != 0 {
		t.Fatalf("oversized = %d runs %d", resp.StatusCode, runs.Load())
	}
}

// TestToolFailuresComeBackAsResults 锁住工具回错误结果或返回 error 时，响应是 isError 为真的结果；空参数按 {} 交给工具；坏的 params 是协议错误。
func TestToolFailuresComeBackAsResults(t *testing.T) {
	s := New()
	t.Cleanup(func() { _ = s.Close() })
	seen := make(chan string, 4)
	tools := []dsh.Tool{
		{Name: "soft", Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			seen <- call.Arguments
			return dsh.Result{Text: "bad_report", IsError: true}, nil
		}},
		{Name: "hard", Run: func(context.Context, dsh.ToolCall) (dsh.Result, error) {
			return dsh.Result{}, errors.New("engine gone")
		}},
		{Name: "no-run"},
	}
	token := NewToken()
	if err := s.Register(token, "t1", tools, nil); err != nil {
		t.Fatal(err)
	}
	var out rpcOut
	_, data := do(t, s, as(token), http.MethodPost, rpcBody(1, "tools/call", map[string]any{"name": "soft"}))
	out = parse(t, data)
	if text, isErr := callText(t, out.Result); text != "bad_report" || !isErr {
		t.Fatalf("soft = %q %v", text, isErr)
	}
	_, data = do(t, s, as(token), http.MethodPost, rpcBody(2, "tools/call", map[string]any{"name": "hard", "arguments": map[string]any{"x": 1}}))
	out = parse(t, data)
	if text, isErr := callText(t, out.Result); text != "engine gone" || !isErr {
		t.Fatalf("hard = %q %v", text, isErr)
	}
	if got := <-seen; got != "{}" || len(seen) != 0 {
		t.Fatalf("arguments seen = %q, %d more", got, len(seen))
	}
	for i, body := range []string{
		rpcBody(3, "tools/call", map[string]any{"name": "no-run"}),
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":"soft"}`,
	} {
		_, data = do(t, s, as(token), http.MethodPost, body)
		out = parse(t, data)
		if out.Error == nil || out.Error.Code != codeParams {
			t.Fatalf("case %d = %s", i, data)
		}
	}
}

// TestBatchRepliesToRequestsOnly 锁住一组消息只回其中的请求；组里只有通知时回 202；initialize 不能放进组里。
func TestBatchRepliesToRequestsOnly(t *testing.T) {
	s, token := serveEcho(t)
	batch := "[" + rpcBody(1, "ping", nil) + `,{"jsonrpc":"2.0","method":"notifications/initialized"},` + rpcBody(2, "initialize", nil) + "]"
	resp, data := do(t, s, as(token), http.MethodPost, batch)
	var outs []rpcOut
	decode(t, data, &outs)
	if resp.StatusCode != http.StatusOK || len(outs) != 2 || string(outs[0].ID) != "1" || outs[1].Error == nil || outs[1].Error.Code != codeInvalid {
		t.Fatalf("batch = %d %s", resp.StatusCode, data)
	}
	if resp.Header.Get("Mcp-Session-Id") != "" {
		t.Fatal("batched initialize opened a session")
	}
	if resp, _ := do(t, s, as(token), http.MethodPost, `[{"jsonrpc":"2.0","method":"notifications/initialized"}]`); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("notification batch = %d", resp.StatusCode)
	}
	if resp, _ := do(t, s, as(token), http.MethodPost, `[]`); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty batch = %d", resp.StatusCode)
	}
}

// TestCloseStopsListening 锁住关闭后地址清空、再登记返回 ErrClosed、旧地址连不上，重复关闭没有问题。
func TestCloseStopsListening(t *testing.T) {
	s, _ := serveEcho(t)
	old := s.URL()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s.URL() != "" {
		t.Fatalf("url after close = %q", s.URL())
	}
	if err := s.Register(NewToken(), "t1", nil, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("register after close = %v", err)
	}
	if resp, err := http.Post(old, "application/json", strings.NewReader(rpcBody(1, "ping", nil))); err == nil {
		resp.Body.Close()
		t.Fatalf("old address still answers: %d", resp.StatusCode)
	}
}
