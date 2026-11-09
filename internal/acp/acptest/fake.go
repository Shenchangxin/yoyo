// Package acptest 是可脚本化的假 ACP agent，给 acp、buddy、acpworker 和 app 的测试用。
// 同进程版用两条管道连到客户端；子进程版由测试二进制自己扮演，按名字选内置脚本。
// 它能声明任意能力、回放更新、主动发权限请求和文件读写，也能模拟崩溃和要求登录。
package acptest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// Script 描述假 agent 的行为。零值是一个声明 loadSession、回显提示的 agent。
type Script struct {
	// Version 是握手时报的协议版本。0 时报 1。
	Version int
	// Caps 是 agentCapabilities 的原文。nil 时报 {"loadSession":true}。
	Caps map[string]any
	// AuthMethods 原样报给客户端。
	AuthMethods []acp.AuthMethod
	// NeedAuth 为真时，authenticate 之前 session/new 回 -32000。
	NeedAuth bool
	// Models 非空时 session/new 带上 models。
	Models []acp.ModelInfo
	// Replay 是 session/load 时先回放的更新原文。
	Replay []map[string]any
	// LoadFails 和 ResumeFails 让对应调用回错误。
	LoadFails, ResumeFails bool
	// OnPrompt 处理一轮提示并返回 stopReason。nil 时回显提示文字。
	OnPrompt func(t *Turn) acp.StopReason
}

// Agent 是一个跑起来的假 agent。
type Agent struct {
	script Script
	conn   *acp.Conn
	crash  func(code int, stderr string)

	mu       sync.Mutex
	authed   bool
	n        int
	sessions map[string]*Session
	turns    map[string]context.CancelFunc
	calls    []string
}

// Session 是假 agent 记下的一个会话。
type Session struct {
	ID         string
	Cwd        string
	McpServers []acp.McpServer
	Dirs       []string
	System     *acp.SystemPrompt
}

// Turn 是正在处理的一轮提示，脚本通过它发更新和请求。
type Turn struct {
	// Ctx 在客户端发 session/cancel 时结束。
	Ctx       context.Context
	SessionID string
	Session   Session
	Prompt    []acp.ContentBlock
	agent     *Agent
}

// newAgent 在 r、w 上起假 agent。crash 决定崩溃时怎么断开。
func newAgent(script Script, r io.Reader, w io.Writer, crash func(code int, stderr string)) *Agent {
	a := &Agent{script: script, crash: crash, sessions: map[string]*Session{}, turns: map[string]context.CancelFunc{}}
	a.conn = acp.NewConn(r, w, acp.ConnOptions{OnRequest: a.onRequest, OnNotify: a.onNotify})
	return a
}

// Calls 返回收到的请求方法名，按到达顺序。
func (a *Agent) Calls() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.calls...)
}

// Sessions 返回已经建过或加载过的会话。
func (a *Agent) Sessions() []Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Session, 0, len(a.sessions))
	for _, s := range a.sessions {
		out = append(out, *s)
	}
	return out
}

// Done 在连接断开后关闭。
func (a *Agent) Done() <-chan struct{} { return a.conn.Done() }

// Pipe 在同一进程里起假 agent，返回给客户端用的读端和写端。崩溃时两条管道都断开。
func Pipe(script Script) (io.Reader, io.WriteCloser, *Agent) {
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	var a *Agent
	a = newAgent(script, toAgentR, toClientW, func(int, string) {
		_ = toClientW.Close()
		_ = toAgentR.Close()
		a.conn.Shutdown(acp.ErrClosed)
	})
	return toClientR, toAgentW, a
}

// onRequest 回应客户端的请求。
func (a *Agent) onRequest(ctx context.Context, method string, params json.RawMessage) (any, error) {
	a.mu.Lock()
	a.calls = append(a.calls, method)
	a.mu.Unlock()
	switch method {
	case "initialize":
		version := a.script.Version
		if version == 0 {
			version = acp.ProtocolVersion
		}
		caps := a.script.Caps
		if caps == nil {
			caps = map[string]any{"loadSession": true}
		}
		return map[string]any{
			"protocolVersion":   version,
			"agentCapabilities": caps,
			"authMethods":       nonNil(a.script.AuthMethods),
			"agentInfo":         map[string]any{"name": "fake", "version": "0.0.1"},
		}, nil
	case "authenticate":
		a.mu.Lock()
		a.authed = true
		a.mu.Unlock()
		return map[string]any{}, nil
	case "logout":
		a.mu.Lock()
		a.authed = false
		a.mu.Unlock()
		return map[string]any{}, nil
	case "session/new":
		if err := a.needAuth(); err != nil {
			return nil, err
		}
		var req acp.NewSessionRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &acp.RPCError{Code: acp.CodeInvalidParams, Message: err.Error()}
		}
		a.mu.Lock()
		a.n++
		s := &Session{ID: fmt.Sprintf("sess-%d", a.n), Cwd: req.Cwd, McpServers: req.McpServers, Dirs: req.AdditionalDirectories, System: req.SystemPrompt}
		a.sessions[s.ID] = s
		a.mu.Unlock()
		out := map[string]any{"sessionId": s.ID}
		if len(a.script.Models) > 0 {
			out["models"] = map[string]any{"currentModelId": a.script.Models[0].ModelID, "availableModels": a.script.Models}
		}
		return out, nil
	case "session/load", "session/resume":
		if err := a.needAuth(); err != nil {
			return nil, err
		}
		if (method == "session/load" && a.script.LoadFails) || (method == "session/resume" && a.script.ResumeFails) {
			return nil, &acp.RPCError{Code: acp.CodeInternalError, Message: "session not found"}
		}
		var req acp.LoadSessionRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &acp.RPCError{Code: acp.CodeInvalidParams, Message: err.Error()}
		}
		a.mu.Lock()
		a.sessions[req.SessionID] = &Session{ID: req.SessionID, Cwd: req.Cwd, McpServers: req.McpServers, Dirs: req.AdditionalDirectories}
		a.mu.Unlock()
		if method == "session/load" {
			for _, u := range a.script.Replay {
				_ = a.conn.Notify("session/update", map[string]any{"sessionId": req.SessionID, "update": u})
			}
		}
		return map[string]any{}, nil
	case "session/close":
		var req acp.SessionIDParams
		_ = json.Unmarshal(params, &req)
		a.mu.Lock()
		delete(a.sessions, req.SessionID)
		a.mu.Unlock()
		return map[string]any{}, nil
	case "session/prompt":
		var req acp.PromptRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &acp.RPCError{Code: acp.CodeInvalidParams, Message: err.Error()}
		}
		return map[string]any{"stopReason": a.prompt(req)}, nil
	default:
		return nil, &acp.RPCError{Code: acp.CodeMethodNotFound, Message: "method not found: " + method}
	}
}

// needAuth 在要求登录且还没登录时返回 -32000。
func (a *Agent) needAuth() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.script.NeedAuth && !a.authed {
		return &acp.RPCError{Code: acp.CodeAuthRequired, Message: "Authentication required"}
	}
	return nil
}

// prompt 跑一轮脚本。客户端的 session/cancel 会结束 Turn.Ctx。
func (a *Agent) prompt(req acp.PromptRequest) acp.StopReason {
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.turns[req.SessionID] = cancel
	var sess Session
	if s := a.sessions[req.SessionID]; s != nil {
		sess = *s
	}
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.turns, req.SessionID)
		a.mu.Unlock()
		cancel()
	}()
	t := &Turn{Ctx: ctx, SessionID: req.SessionID, Session: sess, Prompt: req.Prompt, agent: a}
	if a.script.OnPrompt == nil {
		t.Say("echo: " + t.Text())
		return acp.StopEndTurn
	}
	return a.script.OnPrompt(t)
}

// onNotify 处理 session/cancel。
func (a *Agent) onNotify(method string, params json.RawMessage) {
	if method != "session/cancel" {
		return
	}
	var req acp.SessionIDParams
	if json.Unmarshal(params, &req) != nil {
		return
	}
	a.mu.Lock()
	cancel := a.turns[req.SessionID]
	a.calls = append(a.calls, method)
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// nonNil 把 nil 切片换成空切片，JSON 里写成 []。
func nonNil(methods []acp.AuthMethod) []acp.AuthMethod {
	if methods == nil {
		return []acp.AuthMethod{}
	}
	return methods
}

// Text 返回提示里全部文字。
func (t *Turn) Text() string {
	var b strings.Builder
	for _, block := range t.Prompt {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}

// Update 发一条会话更新。u 里要有 sessionUpdate。
func (t *Turn) Update(u map[string]any) {
	_ = t.agent.conn.Notify("session/update", map[string]any{"sessionId": t.SessionID, "update": u})
}

// Say 发一段助手正文。
func (t *Turn) Say(text string) {
	t.Update(map[string]any{"sessionUpdate": acp.UpdateAgentChunk, "content": map[string]any{"type": "text", "text": text}})
}

// Think 发一段推理。
func (t *Turn) Think(text string) {
	t.Update(map[string]any{"sessionUpdate": acp.UpdateThoughtChunk, "content": map[string]any{"type": "text", "text": text}})
}

// Tool 发一次工具调用。call 里不用写 sessionUpdate。
func (t *Turn) Tool(call map[string]any) {
	call["sessionUpdate"] = acp.UpdateToolCall
	t.Update(call)
}

// ToolUpdate 发一次工具调用的更新。
func (t *Turn) ToolUpdate(call map[string]any) {
	call["sessionUpdate"] = acp.UpdateToolUpdate
	t.Update(call)
}

// Permission 向客户端要一次权限，返回客户端的选择。
func (t *Turn) Permission(toolCall map[string]any, options []acp.PermissionOption) (acp.PermissionOutcome, error) {
	var resp acp.PermissionResponse
	err := t.agent.conn.Call(context.Background(), "session/request_permission", map[string]any{
		"sessionId": t.SessionID,
		"toolCall":  toolCall,
		"options":   options,
	}, &resp)
	return resp.Outcome, err
}

// ReadFile 请客户端读 path。
func (t *Turn) ReadFile(path string) (string, error) {
	var resp acp.ReadTextFileResponse
	err := t.agent.conn.Call(context.Background(), "fs/read_text_file", acp.ReadTextFileRequest{SessionID: t.SessionID, Path: path}, &resp)
	return resp.Content, err
}

// WriteFile 请客户端写 path。
func (t *Turn) WriteFile(path, content string) error {
	return t.agent.conn.Call(context.Background(), "fs/write_text_file", acp.WriteTextFileRequest{SessionID: t.SessionID, Path: path, Content: content}, nil)
}

// Call 向客户端发任意请求，测试用它检查未实现的方法。
func (t *Turn) Call(method string, params, result any) error {
	return t.agent.conn.Call(context.Background(), method, params, result)
}

// Crash 模拟 agent 崩溃：stderr 写进 stderr，然后以 code 退出（同进程版只断开管道）。
func (t *Turn) Crash(code int, stderr string) {
	t.agent.crash(code, stderr)
}

// WaitCancel 阻塞到这一轮被取消或 d 用完。取消时返回真。
func (t *Turn) WaitCancel(d time.Duration) bool {
	select {
	case <-t.Ctx.Done():
		return true
	case <-time.After(d):
		return false
	}
}

// DefaultPermOptions 是常见的四个权限选项。
func DefaultPermOptions() []acp.PermissionOption {
	return []acp.PermissionOption{
		{OptionID: "allow", Name: "Allow", Kind: acp.PermAllowOnce},
		{OptionID: "always", Name: "Always allow", Kind: acp.PermAllowAlways},
		{OptionID: "reject", Name: "Reject", Kind: acp.PermRejectOnce},
		{OptionID: "never", Name: "Never", Kind: acp.PermRejectAlways},
	}
}

// exitProcess 是子进程版的崩溃：写 stderr 再退出。测试可以换掉它。
var exitProcess = func(code int, stderr string) {
	if stderr != "" {
		_, _ = os.Stderr.WriteString(stderr + "\n")
	}
	os.Exit(code)
}
