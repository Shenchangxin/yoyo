// client.go 拥有 ACP 客户端：启动 agent 进程、initialize 握手、会话的新建/加载/续接/关闭、一轮提示、取消、登录登出和关闭。
// agent 发来的更新、权限请求和文件读写交给 Handler；这里只负责协议规则，比如取消时把挂起的权限请求答成 cancelled。
// buddy 的探测、登录和 acpworker 调用它。

package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/procgroup"
)

var (
	// ErrProtocol 表示对方的协议版本不是 1。
	ErrProtocol = errors.New("acp: unsupported protocol version")
	// ErrAuthRequired 表示对方要先登录。RPCError 的 -32000 或消息里带 auth 都算。
	ErrAuthRequired = errors.New("acp: authentication required")
	// ErrUnsupported 表示对方没有声明这项能力。
	ErrUnsupported = errors.New("acp: not supported by this agent")
)

const (
	// defaultInitTimeout 是握手的默认上限。
	defaultInitTimeout = 20 * time.Second
	// cancelGrace 是取消之后还等 agent 回 stopReason 的时间。
	cancelGrace = 10 * time.Second
	// closeGrace 是关掉 stdin 后等进程自己退出的时间，过了就杀整棵树。
	closeGrace = 2 * time.Second
	// stderrSize 是 stderr 环形缓冲的大小。
	stderrSize = 64 << 10
)

// ExitError 是 agent 进程退出后连接断开的原因。Stderr 是打过码的尾部。
type ExitError struct {
	Code   int
	Stderr string
}

// Error 写成 agent exited: <退出码> <stderr 尾部>。
func (e *ExitError) Error() string {
	s := fmt.Sprintf("agent exited: %d", e.Code)
	if e.Stderr != "" {
		s += " " + e.Stderr
	}
	return s
}

// Is 让 errors.Is(err, ErrClosed) 成立。
func (e *ExitError) Is(target error) bool { return target == ErrClosed }

// Handler 收 agent 发来的东西。为 nil 的函数：更新丢掉，请求回 -32601。
type Handler struct {
	// Update 收到一条会话更新。replay 为真表示这是 session/load 回放的历史。在读循环里调用，要快。
	Update func(n SessionNotification, replay bool)
	// Permission 决定一次权限请求。ctx 在这一轮被取消时结束，此时客户端已经替它答了 cancelled。
	Permission func(ctx context.Context, req PermissionRequest) (PermissionOutcome, error)
	// ReadFile 返回文件内容。
	ReadFile func(ctx context.Context, req ReadTextFileRequest) (string, error)
	// WriteFile 写文件。
	WriteFile func(ctx context.Context, req WriteTextFileRequest) error
}

// Options 是启动并连上一个 agent。
type Options struct {
	// Command 和 Args 是 agent 的启动命令。
	Command string
	Args    []string
	// Env 是完整环境变量。nil 时继承当前进程的环境。
	Env []string
	// Dir 是工作目录。
	Dir string
	// CmdLine 只在 Windows 上生效：非空时原样作为进程命令行，经 cmd.exe 启动垫片时用它避开参数转义。
	CmdLine string
	// Version 写进 clientInfo。
	Version string
	// Handler 处理 agent 发来的更新和请求。
	Handler Handler
	// Log 记连接诊断。可以为 nil。
	Log func(string)
	// InitTimeout 小于等于 0 时用 20 秒。
	InitTimeout time.Duration
}

// Client 是一条已经握手的 ACP 连接。
type Client struct {
	conn    *Conn
	proc    *procgroup.Proc
	stdin   io.Closer
	reader  io.Closer
	stderr  *Ring
	init    InitializeResponse
	handler Handler

	mu       sync.Mutex
	replay   map[string]bool
	perms    map[string]map[int]context.CancelFunc
	permSeq  int
	sessions map[string]bool
	closed   bool
}

// Dial 用 procgroup 隐藏窗口启动 agent，再握手。握手失败或版本不对时关掉进程返回错误。
func Dial(ctx context.Context, opts Options) (*Client, error) {
	if strings.TrimSpace(opts.Command) == "" {
		return nil, errors.New("acp: no command")
	}
	cmd := exec.Command(opts.Command, opts.Args...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	SetCmdLine(cmd, opts.CmdLine)
	inR, inW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		_ = inR.Close()
		_ = inW.Close()
		return nil, err
	}
	ring := NewRing(stderrSize)
	cmd.Stdin = inR
	cmd.Stdout = outW
	cmd.Stderr = ring
	proc, err := procgroup.Start(cmd, procgroup.Options{Hide: true})
	_ = inR.Close()
	_ = outW.Close()
	if err != nil {
		_ = inW.Close()
		_ = outR.Close()
		return nil, err
	}
	c := newClient(outR, inW, opts)
	c.proc = proc
	c.stdin = inW
	c.stderr = ring
	go func() {
		<-proc.Done()
		// 孙进程可能还握着 stdout，读不到 EOF；主进程退出就算断开。
		time.Sleep(100 * time.Millisecond)
		c.conn.Shutdown(c.exitError())
		_ = outR.Close()
	}()
	if err := c.handshake(ctx, opts); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

// Connect 在已有的读写两端上握手，不启动进程。acptest 的同进程假 agent 用它。
func Connect(ctx context.Context, r io.Reader, w io.Writer, opts Options) (*Client, error) {
	c := newClient(r, w, opts)
	if closer, ok := w.(io.Closer); ok {
		c.stdin = closer
	}
	if closer, ok := r.(io.Closer); ok {
		c.reader = closer
	}
	c.stderr = NewRing(stderrSize)
	if err := c.handshake(ctx, opts); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

// newClient 建客户端和底下的连接。
func newClient(r io.Reader, w io.Writer, opts Options) *Client {
	c := &Client{
		handler:  opts.Handler,
		replay:   map[string]bool{},
		perms:    map[string]map[int]context.CancelFunc{},
		sessions: map[string]bool{},
	}
	c.conn = NewConn(r, w, ConnOptions{OnRequest: c.onRequest, OnNotify: c.onNotify, Log: opts.Log})
	return c
}

// handshake 发 initialize。对方的版本不是 1 时返回包着 ErrProtocol 的错误。
func (c *Client) handshake(ctx context.Context, opts Options) error {
	timeout := opts.InitTimeout
	if timeout <= 0 {
		timeout = defaultInitTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req := InitializeRequest{
		ProtocolVersion: ProtocolVersion,
		ClientCapabilities: ClientCapabilities{
			FS:       FileSystemCapability{ReadTextFile: c.handler.ReadFile != nil, WriteTextFile: c.handler.WriteFile != nil},
			Terminal: false,
			Auth:     &ClientAuthCapability{Terminal: true},
		},
		ClientInfo: &Implementation{Name: "yoyo", Title: "Yoyo", Version: opts.Version},
	}
	var resp InitializeResponse
	if err := c.call(ctx, "initialize", req, &resp); err != nil {
		return err
	}
	if v := resp.Version(); v != ProtocolVersion {
		return fmt.Errorf("%w: agent speaks %s", ErrProtocol, strings.TrimSpace(string(resp.ProtocolVersion)))
	}
	c.init = resp
	return nil
}

// call 发请求；连接断开时换成带退出码和 stderr 尾部的错误。
func (c *Client) call(ctx context.Context, method string, params, result any) error {
	return c.wrap(c.conn.Call(ctx, method, params, result))
}

// wrap 把连接断开的错误换成 exitError。
func (c *Client) wrap(err error) error {
	if err == nil || !errors.Is(err, ErrClosed) {
		return err
	}
	var exit *ExitError
	if errors.As(err, &exit) {
		return err
	}
	if c.proc != nil {
		select {
		case <-c.proc.Done():
			return c.exitError()
		case <-time.After(closeGrace):
		}
	}
	return err
}

// exitError 用退出码和 stderr 尾部描述进程退出。
func (c *Client) exitError() error {
	code := -1
	if c.proc != nil {
		code = c.proc.ExitCode()
	}
	return &ExitError{Code: code, Stderr: c.stderr.Tail(200)}
}

// Init 返回握手结果。
func (c *Client) Init() InitializeResponse { return c.init }

// Caps 返回对方声明的能力。
func (c *Client) Caps() AgentCapabilities { return c.init.AgentCapabilities }

// Done 在连接断开后关闭。
func (c *Client) Done() <-chan struct{} { return c.conn.Done() }

// Err 返回连接断开的原因。
func (c *Client) Err() error { return c.wrap(c.conn.Err()) }

// Stderr 返回 agent 的 stderr 缓冲。Connect 建的客户端是空的。
func (c *Client) Stderr() *Ring { return c.stderr }

// Pid 返回 agent 进程号。Connect 建的客户端为 0。
func (c *Client) Pid() int { return c.proc.Pid() }

// NewSession 新建会话。对方没声明时去掉 additionalDirectories 和 systemPrompt。
func (c *Client) NewSession(ctx context.Context, req NewSessionRequest) (NewSessionResponse, error) {
	caps := c.init.AgentCapabilities.SessionCapabilities
	if !caps.AdditionalDirectories {
		req.AdditionalDirectories = nil
	}
	if !caps.SystemPrompt {
		req.SystemPrompt = nil
	}
	if req.McpServers == nil {
		req.McpServers = []McpServer{}
	}
	var resp NewSessionResponse
	if err := c.call(ctx, "session/new", req, &resp); err != nil {
		return NewSessionResponse{}, err
	}
	if resp.SessionID == "" {
		return NewSessionResponse{}, errors.New("acp: session/new returned no sessionId")
	}
	c.track(resp.SessionID)
	return resp, nil
}

// LoadSession 加载旧会话。加载期间收到的更新标成回放。对方没声明 loadSession 时返回 ErrUnsupported。
func (c *Client) LoadSession(ctx context.Context, req LoadSessionRequest) error {
	if !c.init.AgentCapabilities.LoadSession {
		return ErrUnsupported
	}
	if !c.init.AgentCapabilities.SessionCapabilities.AdditionalDirectories {
		req.AdditionalDirectories = nil
	}
	if req.McpServers == nil {
		req.McpServers = []McpServer{}
	}
	c.mu.Lock()
	c.replay[req.SessionID] = true
	c.mu.Unlock()
	err := c.call(ctx, "session/load", req, nil)
	c.mu.Lock()
	delete(c.replay, req.SessionID)
	c.mu.Unlock()
	if err == nil {
		c.track(req.SessionID)
	}
	return err
}

// ResumeSession 续接旧会话，不回放历史。对方没声明 resume 时返回 ErrUnsupported。
func (c *Client) ResumeSession(ctx context.Context, req LoadSessionRequest) error {
	if !c.init.AgentCapabilities.SessionCapabilities.Resume {
		return ErrUnsupported
	}
	if !c.init.AgentCapabilities.SessionCapabilities.AdditionalDirectories {
		req.AdditionalDirectories = nil
	}
	if req.McpServers == nil {
		req.McpServers = []McpServer{}
	}
	if err := c.call(ctx, "session/resume", req, nil); err != nil {
		return err
	}
	c.track(req.SessionID)
	return nil
}

// CloseSession 关掉会话。对方没声明 close 时返回 ErrUnsupported。
func (c *Client) CloseSession(ctx context.Context, sessionID string) error {
	if !c.init.AgentCapabilities.SessionCapabilities.Close {
		return ErrUnsupported
	}
	err := c.call(ctx, "session/close", SessionIDParams{SessionID: sessionID}, nil)
	c.mu.Lock()
	delete(c.sessions, sessionID)
	c.mu.Unlock()
	return err
}

// Reopen 续上旧会话：先 resume（对方声明了才试），再 load（同样），都不行就按 fresh 新建。
// 返回会话号和用的方式：resume、load 或 new。old.SessionID 为空时直接新建。连接断开或要求登录时不再往下试。
func (c *Client) Reopen(ctx context.Context, old LoadSessionRequest, fresh NewSessionRequest) (string, string, error) {
	if old.SessionID != "" {
		if c.init.AgentCapabilities.SessionCapabilities.Resume {
			err := c.ResumeSession(ctx, old)
			if err == nil {
				return old.SessionID, "resume", nil
			}
			if stopTrying(err) {
				return "", "", err
			}
		}
		if c.init.AgentCapabilities.LoadSession {
			err := c.LoadSession(ctx, old)
			if err == nil {
				return old.SessionID, "load", nil
			}
			if stopTrying(err) {
				return "", "", err
			}
		}
	}
	resp, err := c.NewSession(ctx, fresh)
	if err != nil {
		return "", "", err
	}
	return resp.SessionID, "new", nil
}

// stopTrying 在连接已断、要求登录或调用方已取消时为真，这些情况换一种续接方式也没用。
func stopTrying(err error) bool {
	return errors.Is(err, ErrClosed) || errors.Is(err, ErrAuthRequired) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// track 记下一个开着的会话，Close 时去关它。
func (c *Client) track(sessionID string) {
	c.mu.Lock()
	c.sessions[sessionID] = true
	c.mu.Unlock()
}

// Prompt 发一轮提示，阻塞到拿到 stopReason。ctx 取消时先发 session/cancel，再最多等十秒让对方回 cancelled。
func (c *Client) Prompt(ctx context.Context, sessionID string, prompt []ContentBlock) (StopReason, error) {
	type result struct {
		stop StopReason
		err  error
	}
	done := make(chan result, 1)
	go func() {
		var resp PromptResponse
		err := c.call(context.Background(), "session/prompt", PromptRequest{SessionID: sessionID, Prompt: prompt}, &resp)
		done <- result{resp.StopReason, err}
	}()
	select {
	case r := <-done:
		return r.stop, r.err
	case <-ctx.Done():
	}
	_ = c.Cancel(sessionID)
	select {
	case r := <-done:
		if r.err == nil && r.stop == "" {
			r.stop = StopCancelled
		}
		return r.stop, r.err
	case <-time.After(cancelGrace):
		return StopCancelled, ctx.Err()
	}
}

// Cancel 取消 sessionID 这一轮：挂起的权限请求立刻答成 cancelled，再发 session/cancel。
func (c *Client) Cancel(sessionID string) error {
	c.mu.Lock()
	pending := c.perms[sessionID]
	delete(c.perms, sessionID)
	c.mu.Unlock()
	for _, cancel := range pending {
		cancel()
	}
	return c.conn.Notify("session/cancel", SessionIDParams{SessionID: sessionID})
}

// Authenticate 用 methodID 登录。terminal 型由调用方另开终端，不走这里。
func (c *Client) Authenticate(ctx context.Context, methodID string) error {
	return c.call(ctx, "authenticate", AuthenticateRequest{MethodID: methodID}, nil)
}

// Logout 退出登录。对方没声明 auth.logout 时返回 ErrUnsupported。
func (c *Client) Logout(ctx context.Context) error {
	if c.init.AgentCapabilities.Auth == nil || !c.init.AgentCapabilities.Auth.Logout {
		return ErrUnsupported
	}
	err := c.call(ctx, "logout", struct{}{}, nil)
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) && rpcErr.Code == CodeMethodNotFound {
		err = c.call(ctx, "auth/logout", struct{}{}, nil)
	}
	return err
}

// Close 能关会话就先发 session/close，再关 stdin，两秒后杀整棵进程树。重复调用只做一次。
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	var open []string
	for id := range c.sessions {
		open = append(open, id)
	}
	c.mu.Unlock()
	if c.init.AgentCapabilities.SessionCapabilities.Close {
		for _, id := range open {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = c.conn.Call(ctx, "session/close", SessionIDParams{SessionID: id}, nil)
			cancel()
		}
	}
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.proc != nil {
		select {
		case <-c.proc.Done():
		case <-time.After(closeGrace):
		}
		_ = c.proc.Kill()
	}
	c.conn.Shutdown(ErrClosed)
	if c.reader != nil {
		_ = c.reader.Close()
	}
	return nil
}

// onNotify 处理 agent 的通知。只认 session/update。
func (c *Client) onNotify(method string, params json.RawMessage) {
	if method != "session/update" || c.handler.Update == nil {
		return
	}
	var n SessionNotification
	if err := json.Unmarshal(params, &n); err != nil {
		return
	}
	c.mu.Lock()
	replay := c.replay[n.SessionID]
	c.mu.Unlock()
	c.handler.Update(n, replay)
}

// methodNotFound 返回 -32601。
func methodNotFound(method string) error {
	return &RPCError{Code: CodeMethodNotFound, Message: "method not found: " + method}
}

// onRequest 处理 agent 的请求：权限、读文件、写文件。其余回 -32601。
func (c *Client) onRequest(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "session/request_permission":
		if c.handler.Permission == nil {
			return nil, methodNotFound(method)
		}
		var req PermissionRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
		}
		return c.permission(ctx, req), nil
	case "fs/read_text_file":
		if c.handler.ReadFile == nil {
			return nil, methodNotFound(method)
		}
		var req ReadTextFileRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
		}
		text, err := c.handler.ReadFile(ctx, req)
		if err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return ReadTextFileResponse{Content: text}, nil
	case "fs/write_text_file":
		if c.handler.WriteFile == nil {
			return nil, methodNotFound(method)
		}
		var req WriteTextFileRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
		}
		if err := c.handler.WriteFile(ctx, req); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return struct{}{}, nil
	default:
		return nil, methodNotFound(method)
	}
}

// permission 交给 Handler 决定。这一轮被取消或请求被撤回时，不等 Handler，直接答 cancelled。
func (c *Client) permission(ctx context.Context, req PermissionRequest) PermissionResponse {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.mu.Lock()
	c.permSeq++
	seq := c.permSeq
	if c.perms[req.SessionID] == nil {
		c.perms[req.SessionID] = map[int]context.CancelFunc{}
	}
	c.perms[req.SessionID][seq] = cancel
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.perms[req.SessionID], seq)
		c.mu.Unlock()
	}()
	type result struct {
		out PermissionOutcome
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := c.handler.Permission(ctx, req)
		ch <- result{out, err}
	}()
	select {
	case r := <-ch:
		if ctx.Err() != nil {
			return PermissionResponse{Outcome: Cancelled()}
		}
		if r.err != nil || r.out.Outcome == "" {
			return PermissionResponse{Outcome: Reject(req.Options)}
		}
		return PermissionResponse{Outcome: r.out}
	case <-ctx.Done():
		return PermissionResponse{Outcome: Cancelled()}
	}
}

// Reject 选 options 里的 reject_once，没有时选 reject_always；都没有时答 cancelled。
func Reject(options []PermissionOption) PermissionOutcome {
	for _, kind := range []string{PermRejectOnce, PermRejectAlways} {
		for _, opt := range options {
			if opt.Kind == kind {
				return Selected(opt.OptionID)
			}
		}
	}
	return Cancelled()
}

// Allow 选 options 里的 allow_once，没有时选 allow_always；都没有时返回空 Outcome。
func Allow(options []PermissionOption) PermissionOutcome {
	for _, kind := range []string{PermAllowOnce, PermAllowAlways} {
		for _, opt := range options {
			if opt.Kind == kind {
				return Selected(opt.OptionID)
			}
		}
	}
	return PermissionOutcome{}
}
