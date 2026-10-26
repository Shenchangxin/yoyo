// Package mcp 通过 stdio 或 Streamable HTTP 与 MCP 服务器交换 JSON-RPC。协议版本是 2025-11-25。sampling/createMessage 与 sampling/create_message 被拒绝，错误码 -32000，文案是 "sampling is denied: model calls stay in the TCB"。elicitation/create 同样被拒绝，错误码 -32000，文案是 "elicitation must go through the host Gate"。
package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// ProtocolVersion 是 initialize 使用的 MCP 协议版本，值为 2025-11-25。
const ProtocolVersion = "2025-11-25"

// Host 持有已连接的 MCP 服务器，经 stdio 或 Streamable HTTP 收发 JSON-RPC。Yoyo 自己的循环仍是唯一运行时。
type Host struct {
	mu        sync.Mutex
	servers   map[string]*Server
	Timeout   time.Duration
	Roots     []string
	StderrDir string
}

// Server 是一台已握手的 MCP 服务器，以及它列出的工具、资源和提示。
type Server struct {
	Name       string
	Command    string
	Args       []string
	Endpoint   string
	Tools      []Tool
	Resources  []Resource
	Prompts    []Prompt
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	stderr     *ringBuf
	stderrFile string
	http       *http.Client
	mu         sync.Mutex
	nextID     int
	timeout    time.Duration
	roots      []string
	stopping   atomic.Bool
}

// Tool 是某台服务器列出的工具，带只读、破坏性和开放世界三个注解。
type Tool struct {
	Server          string
	Name            string
	Description     string
	InputSchema     map[string]any
	ReadOnlyHint    bool
	DestructiveHint bool
	OpenWorldHint   bool
}

// Resource 是某台服务器列出的资源。
type Resource struct {
	Server      string `json:"server"`
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Prompt 是某台服务器列出的提示。
type Prompt struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NewHost 返回服务器表为空、Timeout 为 30 秒的 Host。
func NewHost() *Host {
	return &Host{servers: map[string]*Server{}, Timeout: 30 * time.Second}
}

// SetRoots 换上根目录副本，并给每台已连接服务器发送 notifications/roots/list_changed。
func (h *Host) SetRoots(roots []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Roots = append([]string(nil), roots...)
	for _, s := range h.servers {
		if s == nil {
			continue
		}
		s.mu.Lock()
		s.roots = append([]string(nil), h.Roots...)
		s.mu.Unlock()
		_ = s.notify("notifications/roots/list_changed", map[string]any{})
	}
}

// Start 用 command 和 args 拉起 stdio 进程并握手。同名已在跑时返回 "mcp: <name> already running"。Timeout 小于等于 0 时该服务器用 30 秒。标准错误环形缓冲是 16 KiB。StderrDir 非空时，日志追加写到该目录下按名字净化后的 .stderr.log。
func (h *Host) Start(name, command string, args []string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.servers[name]; ok {
		return fmt.Errorf("mcp: %s already running", name)
	}
	s := &Server{
		Name:    name,
		Command: command,
		Args:    args,
		timeout: h.Timeout,
		roots:   append([]string(nil), h.Roots...),
		stderr:  newRing(16 << 10),
	}
	if h.StderrDir != "" {
		_ = os.MkdirAll(h.StderrDir, 0o755)
		s.stderrFile = filepath.Join(h.StderrDir, sanitizeMCPName(name)+".stderr.log")
	}
	if s.timeout <= 0 {
		s.timeout = 30 * time.Second
	}
	if err := s.startStdio(); err != nil {
		return err
	}
	if err := s.handshake(); err != nil {
		s.kill()
		return err
	}
	h.servers[name] = s
	return nil
}

// StartHTTP 连接 endpoint 并握手。同名已在跑时返回 "mcp: <name> already running"。Timeout 小于等于 0 时，服务器超时和 HTTP 客户端都改为 30 秒。
func (h *Host) StartHTTP(name, endpoint string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.servers[name]; ok {
		return fmt.Errorf("mcp: %s already running", name)
	}
	s := &Server{
		Name:     name,
		Endpoint: endpoint,
		timeout:  h.Timeout,
		roots:    append([]string(nil), h.Roots...),
		http:     &http.Client{Timeout: h.Timeout},
		stderr:   newRing(16 << 10),
	}
	if s.timeout <= 0 {
		s.timeout = 30 * time.Second
		s.http.Timeout = s.timeout
	}
	if err := s.handshake(); err != nil {
		return err
	}
	h.servers[name] = s
	return nil
}

// Stop 停掉并摘掉该名字的服务器。名字不存在时返回 nil。
// name 作为 servers 的键；没有该键时返回 nil，有则摘掉并 kill。
func (h *Host) Stop(name string) error {
	h.mu.Lock()
	s, ok := h.servers[name]
	delete(h.servers, name)
	h.mu.Unlock()
	if !ok {
		return nil
	}
	s.kill()
	return nil
}

// List 返回当前服务器名字，顺序不定。
func (h *Host) List() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.servers))
	for n := range h.servers {
		out = append(out, n)
	}
	return out
}

// Info 是一台服务器的摘要：命令或端点，工具、资源、提示的数量，以及环形缓冲里的标准错误。
type Info struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	Endpoint  string   `json:"endpoint,omitempty"`
	Tools     int      `json:"tools"`
	Resources int      `json:"resources"`
	Prompts   int      `json:"prompts"`
	Stderr    string   `json:"stderr,omitempty"`
}

// Info 返回每台服务器的 Info。
func (h *Host) Info() []Info {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Info, 0, len(h.servers))
	for _, s := range h.servers {
		out = append(out, Info{
			Name: s.Name, Command: s.Command, Args: s.Args, Endpoint: s.Endpoint,
			Tools: len(s.Tools), Resources: len(s.Resources), Prompts: len(s.Prompts),
			Stderr: s.Stderr(),
		})
	}
	return out
}

// Tools 返回全部服务器当前列出的工具。
func (h *Host) Tools() []Tool {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []Tool
	for _, s := range h.servers {
		out = append(out, s.Tools...)
	}
	return out
}

// Call 对指定服务器调用 tools/call。argsJSON 非空时按 JSON 解成 arguments，解失败则 arguments 为 nil。未知服务器返回 "mcp: unknown server"。
func (h *Host) Call(server, tool, argsJSON string) (string, error) {
	h.mu.Lock()
	s := h.servers[server]
	h.mu.Unlock()
	if s == nil {
		return "", fmt.Errorf("mcp: unknown server %s", server)
	}
	if err := s.ensure(); err != nil {
		return "", err
	}
	var args any
	if argsJSON != "" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}
	raw, err := s.call("tools/call", map[string]any{
		"name":      tool,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadResource 调用 resources/read，把结果 JSON 当字符串返回。未知服务器返回 "mcp: unknown server"。
// uri 放进 resources/read 参数的 uri 字段。
func (h *Host) ReadResource(server, uri string) (string, error) {
	h.mu.Lock()
	s := h.servers[server]
	h.mu.Unlock()
	if s == nil {
		return "", fmt.Errorf("mcp: unknown server %s", server)
	}
	raw, err := s.call("resources/read", map[string]any{"uri": uri})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// GetPrompt 调用 prompts/get。未知服务器返回 "mcp: unknown server"。
// name 放进 prompts/get 参数的 name 字段。
// args 放进参数的 arguments 字段。
func (h *Host) GetPrompt(server, name string, args map[string]any) (string, error) {
	h.mu.Lock()
	s := h.servers[server]
	h.mu.Unlock()
	if s == nil {
		return "", fmt.Errorf("mcp: unknown server %s", server)
	}
	raw, err := s.call("prompts/get", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Close 停掉全部服务器，并返回 nil。
func (h *Host) Close() error {
	h.mu.Lock()
	names := make([]string, 0, len(h.servers))
	for n := range h.servers {
		names = append(names, n)
	}
	h.mu.Unlock()
	for _, n := range names {
		_ = h.Stop(n)
	}
	return nil
}

// Stderr 返回环形缓冲里的标准错误。服务器或缓冲为 nil 时返回空串。
func (s *Server) Stderr() string {
	if s == nil || s.stderr == nil {
		return ""
	}
	return s.stderr.String()
}

// startStdio 启动命令并接上 stdin、stdout、stderr，后台排空标准错误并等待进程退出。
func (s *Server) startStdio() error {
	cmd := exec.Command(s.Command, s.Args...)
	osutil.HideConsole(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	s.cmd = cmd
	s.stdin = stdin
	s.stdout = bufio.NewReader(stdout)
	s.stopping.Store(false)
	go s.drainStderr(stderr)
	go s.watchExit(cmd)
	return nil
}

// drainStderr 把标准错误写入环形缓冲。stderrFile 非空时再以 0o600 追加到该文件。
func (s *Server) drainStderr(r io.Reader) {
	var file *os.File
	if s.stderrFile != "" {
		file, _ = os.OpenFile(s.stderrFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if file != nil {
			defer file.Close()
		}
	}
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if s.stderr != nil {
				s.stderr.Write(buf[:n])
			}
			if file != nil {
				_, _ = file.Write(buf[:n])
			}
		}
		if err != nil {
			return
		}
	}
}

// sanitizeMCPName 只保留字母、数字、下划线和连字符，其余换成下划线。空白或结果为空时返回 unknown。
// name 先去掉两端空白，空则返回 unknown；其余只保留字母、数字、下划线和连字符。
func sanitizeMCPName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

// watchExit 等待进程结束。stopping 已置位时记一条退出信息，否则记 mcp crashed。
// cmd 为 nil 时直接返回，否则 Wait。
func (s *Server) watchExit(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	err := cmd.Wait()
	if s.stopping.Load() {
		diaglog.Info("mcp process exited", "component", "mcp", "name", s.Name)
		return
	}
	diaglog.Error("mcp crashed", "component", "mcp", "name", s.Name, "err", err)
}

// kill 置上 stopping，杀掉进程，并清空 cmd、stdin、stdout。
func (s *Server) kill() {
	s.stopping.Store(true)
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.cmd = nil
	s.stdin = nil
	s.stdout = nil
}

// ensure 在 HTTP 端点上直接返回 nil。stdio 进程仍在则返回 nil，否则先 kill，再 startStdio 并 handshake。
func (s *Server) ensure() error {
	if s.Endpoint != "" {
		return nil
	}
	if s.cmd != nil && s.cmd.ProcessState == nil {
		return nil
	}
	s.kill()
	if err := s.startStdio(); err != nil {
		return err
	}
	return s.handshake()
}

// handshake 发送 initialize。capabilities 里 roots.listChanged 为 true，并带上空的 elicitation。clientInfo 的 name 是 yoyo，version 是 0.6.0。成功后发送 notifications/initialized；有根目录时再发 notifications/roots/list_changed。然后刷新工具、资源和提示。
func (s *Server) handshake() error {
	caps := map[string]any{
		"roots":       map[string]any{"listChanged": true},
		"elicitation": map[string]any{},
	}
	_, err := s.call("initialize", map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities":    caps,
		"clientInfo":      map[string]any{"name": "yoyo", "version": "0.6.0"},
	})
	if err != nil {
		return err
	}
	_ = s.notify("notifications/initialized", map[string]any{})
	if len(s.roots) > 0 {
		_ = s.notify("notifications/roots/list_changed", map[string]any{})
	}
	if err := s.refreshTools(); err != nil {
		return err
	}
	s.refreshResources()
	s.refreshPrompts()
	return nil
}

// refreshTools 调用 tools/list，并用 annotations 里的 readOnlyHint、destructiveHint、openWorldHint 填 Tool。
func (s *Server) refreshTools() error {
	raw, err := s.call("tools/list", map[string]any{})
	if err != nil {
		return err
	}
	var listed struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
			Annotations map[string]any `json:"annotations"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		return err
	}
	s.Tools = s.Tools[:0]
	for _, t := range listed.Tools {
		ann := t.Annotations
		s.Tools = append(s.Tools, Tool{
			Server:          s.Name,
			Name:            t.Name,
			Description:     t.Description,
			InputSchema:     t.InputSchema,
			ReadOnlyHint:    boolOf(ann, "readOnlyHint"),
			DestructiveHint: boolOf(ann, "destructiveHint"),
			OpenWorldHint:   boolOf(ann, "openWorldHint"),
		})
	}
	return nil
}

// refreshResources 调用 resources/list。调用失败或 JSON 无效时保留原列表。
func (s *Server) refreshResources() {
	raw, err := s.call("resources/list", map[string]any{})
	if err != nil {
		return
	}
	var listed struct {
		Resources []struct {
			URI         string `json:"uri"`
			Name        string `json:"name"`
			Description string `json:"description"`
			MimeType    string `json:"mimeType"`
		} `json:"resources"`
	}
	if json.Unmarshal(raw, &listed) != nil {
		return
	}
	s.Resources = s.Resources[:0]
	for _, r := range listed.Resources {
		s.Resources = append(s.Resources, Resource{
			Server: s.Name, URI: r.URI, Name: r.Name, Description: r.Description, MimeType: r.MimeType,
		})
	}
}

// refreshPrompts 调用 prompts/list。调用失败或 JSON 无效时保留原列表。
func (s *Server) refreshPrompts() {
	raw, err := s.call("prompts/list", map[string]any{})
	if err != nil {
		return
	}
	var listed struct {
		Prompts []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"prompts"`
	}
	if json.Unmarshal(raw, &listed) != nil {
		return
	}
	s.Prompts = s.Prompts[:0]
	for _, p := range listed.Prompts {
		s.Prompts = append(s.Prompts, Prompt{Server: s.Name, Name: p.Name, Description: p.Description})
	}
}

// boolOf 取出 map 里的布尔值。键不存在或类型不是 bool 时返回 false。
// k 作为键；map 为 nil、键不存在或值不是 bool 时返回 false。
func boolOf(m map[string]any, k string) bool {
	if m == nil {
		return false
	}
	v, ok := m[k]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

type rpcReq struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcRes struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// call 在有 Endpoint 时走 callHTTP。stdio 写出带 id 的请求，并在超时内读响应。timeout 小于等于 0 时限时 30 秒。读到带 method 的入站消息就交给 handleIncoming 并继续等。id 不匹配则跳过。对端错误返回 "mcp: " 加对方 message。超时返回 "mcp: timeout calling " 加方法名。
// method 在有 Endpoint 时和 params 一起交给 callHTTP；否则写入请求的 Method，超时错误里也会带上它。
// params 写入请求的 Params。
func (s *Server) call(method string, params any) (json.RawMessage, error) {
	if s.Endpoint != "" {
		return s.callHTTP(method, params)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := s.nextID
	if err := s.write(rpcReq{JSONRPC: "2.0", ID: id, Method: method, Params: params}); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(s.timeout)
	if s.timeout <= 0 {
		deadline = time.Now().Add(30 * time.Second)
	}
	for time.Now().Before(deadline) {
		res, err := s.read()
		if err != nil {
			return nil, err
		}
		if res.Method != "" {
			s.handleIncoming(res)
			continue
		}
		if res.ID != id {
			continue
		}
		if res.Error != nil {
			return nil, fmt.Errorf("mcp: %s", res.Error.Message)
		}
		return res.Result, nil
	}
	return nil, fmt.Errorf("mcp: timeout calling %s", method)
}

// callHTTP 向 Endpoint POST 一条 JSON-RPC。Content-Type 是 application/json，Accept 是 application/json, text/event-stream。对端错误返回 "mcp: " 加对方 message。
// method 写入请求的 Method。
// params 写入请求的 Params。
func (s *Server) callHTTP(method string, params any) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := s.nextID
	body, err := json.Marshal(rpcReq{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, s.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	cli := s.http
	if cli == nil {
		cli = http.DefaultClient
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res rpcRes
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, fmt.Errorf("mcp: %s", res.Error.Message)
	}
	return res.Result, nil
}

// handleIncoming 应答服务器主动发来的请求。sampling/createMessage 和 sampling/create_message 都写回 JSON-RPC 错误，code 为 -32000，message 为 "sampling is denied: model calls stay in the TCB"。elicitation/create 同样写回 code -32000，message 为 "elicitation must go through the host Gate"。notifications/tools/list_changed、notifications/resources/list_changed、notifications/prompts/list_changed 会异步刷新对应列表。roots/list 按当前根目录回答，每项 URI 为 file:// 加路径。
func (s *Server) handleIncoming(res rpcRes) {
	switch res.Method {
	case "sampling/createMessage", "sampling/create_message":
		// 模型调用留在宿主循环。服务器不能借 sampling 再开一次。错误码 -32000。
		_ = s.write(map[string]any{
			"jsonrpc": "2.0",
			"id":      res.ID,
			"error":   map[string]any{"code": -32000, "message": "sampling is denied: model calls stay in the TCB"},
		})
	case "elicitation/create":
		// 向用户要输入必须走宿主 Gate。同样回 -32000，不在 MCP 连接里直接弹问。
		_ = s.write(map[string]any{
			"jsonrpc": "2.0",
			"id":      res.ID,
			"error":   map[string]any{"code": -32000, "message": "elicitation must go through the host Gate"},
		})
	case "notifications/tools/list_changed":
		go func() { _ = s.refreshTools() }()
	case "notifications/resources/list_changed":
		go s.refreshResources()
	case "notifications/prompts/list_changed":
		go s.refreshPrompts()
	case "roots/list":
		var roots []map[string]string
		for _, r := range s.roots {
			roots = append(roots, map[string]string{"uri": "file://" + r, "name": r})
		}
		_ = s.write(map[string]any{"jsonrpc": "2.0", "id": res.ID, "result": map[string]any{"roots": roots}})
	}
}

// notify 发送一条不带 id 的 JSON-RPC。Endpoint 非空时改走 callHTTP，因此 HTTP 上的通知仍会带上 id。
// method 在有 Endpoint 时和 params 一起交给 callHTTP；否则写入不带 id 的请求 Method。
// params 写入请求的 Params。
func (s *Server) notify(method string, params any) error {
	if s.Endpoint != "" {
		_, err := s.callHTTP(method, params)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(rpcReq{JSONRPC: "2.0", Method: method, Params: params})
}

// write 以 Content-Length 头加 JSON 正文写入 stdin。stdin 为 nil 时返回 "mcp: not connected"。
// v 先 json.Marshal，再带上 Content-Length 头写入 stdin。
func (s *Server) write(v any) error {
	if s.stdin == nil {
		return fmt.Errorf("mcp: not connected")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	hdr := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(b))
	if _, err := io.WriteString(s.stdin, hdr); err != nil {
		return err
	}
	_, err = s.stdin.Write(b)
	return err
}

// read 读取一条响应。下一字节是 '{' 时按一行 JSON 解析。否则读头直到空行，按 Content-Length 读定长正文。没有长度时返回 "mcp: missing content-length"。
func (s *Server) read() (rpcRes, error) {
	var res rpcRes
	b, err := s.stdout.Peek(1)
	if err != nil {
		return res, err
	}
	if b[0] == '{' {
		line, err := s.stdout.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			return res, err
		}
		if err := json.Unmarshal(bytes.TrimSpace(line), &res); err != nil {
			return res, err
		}
		return res, nil
	}
	n := 0
	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return res, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			n, _ = strconv.Atoi(strings.TrimSpace(line[len("Content-Length:"):]))
		}
	}
	if n <= 0 {
		return res, fmt.Errorf("mcp: missing content-length")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(s.stdout, buf); err != nil {
		return res, err
	}
	if err := json.Unmarshal(buf, &res); err != nil {
		return res, err
	}
	return res, nil
}

type ringBuf struct {
	mu   sync.Mutex
	buf  []byte
	max  int
	off  int
	full bool
}

// newRing 分配容量为 n 的环形缓冲。
func newRing(n int) *ringBuf { return &ringBuf{buf: make([]byte, n), max: n} }

// Write 按字节写入环形缓冲，写满后覆盖最旧的字节。
// p 的每个字节写入缓冲，返回 len(p) 和 nil。
func (r *ringBuf) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range p {
		r.buf[r.off] = c
		r.off = (r.off + 1) % r.max
		if r.off == 0 {
			r.full = true
		}
	}
	return len(p), nil
}

// String 按写入顺序返回缓冲里现有的字节。
func (r *ringBuf) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		return string(r.buf[:r.off])
	}
	out := make([]byte, r.max)
	copy(out, r.buf[r.off:])
	copy(out[r.max-r.off:], r.buf[:r.off])
	return string(out)
}
