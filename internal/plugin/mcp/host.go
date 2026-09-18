package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ProtocolVersion = "2025-11-25"

// Host speaks MCP JSON-RPC over stdio or Streamable HTTP. Yoyo's loop stays
// the only runtime; MCP tools register into the same Tool surface.
type Host struct {
	mu      sync.Mutex
	servers map[string]*Server
	Timeout time.Duration
	Roots   []string
}

type Server struct {
	Name      string
	Command   string
	Args      []string
	Endpoint  string
	Tools     []Tool
	Resources []Resource
	Prompts   []Prompt
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	stderr    *ringBuf
	http      *http.Client
	mu        sync.Mutex
	nextID    int
	timeout   time.Duration
	roots     []string
}

type Tool struct {
	Server          string
	Name            string
	Description     string
	InputSchema     map[string]any
	ReadOnlyHint    bool
	DestructiveHint bool
	OpenWorldHint   bool
}

type Resource struct {
	Server      string `json:"server"`
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type Prompt struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func NewHost() *Host {
	return &Host{servers: map[string]*Server{}, Timeout: 30 * time.Second}
}

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

func (h *Host) List() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.servers))
	for n := range h.servers {
		out = append(out, n)
	}
	return out
}

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

func (h *Host) Tools() []Tool {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []Tool
	for _, s := range h.servers {
		out = append(out, s.Tools...)
	}
	return out
}

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

func (s *Server) Stderr() string {
	if s == nil || s.stderr == nil {
		return ""
	}
	return s.stderr.String()
}

func (s *Server) startStdio() error {
	cmd := exec.Command(s.Command, s.Args...)
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
	go s.drainStderr(stderr)
	return nil
}

func (s *Server) drainStderr(r io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 && s.stderr != nil {
			s.stderr.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func (s *Server) kill() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
	s.cmd = nil
	s.stdin = nil
	s.stdout = nil
}

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

func (s *Server) handshake() error {
	caps := map[string]any{
		"roots":       map[string]any{"listChanged": true},
		"elicitation": map[string]any{},
	}
	_, err := s.call("initialize", map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities":    caps,
		"clientInfo":      map[string]any{"name": "yoyo", "version": "0.3.0"},
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

func (s *Server) handleIncoming(res rpcRes) {
	switch res.Method {
	case "sampling/createMessage", "sampling/create_message":
		_ = s.write(map[string]any{
			"jsonrpc": "2.0",
			"id":      res.ID,
			"error":   map[string]any{"code": -32000, "message": "sampling is denied: model calls stay in the TCB"},
		})
	case "elicitation/create":
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

func (s *Server) notify(method string, params any) error {
	if s.Endpoint != "" {
		_, err := s.callHTTP(method, params)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(rpcReq{JSONRPC: "2.0", Method: method, Params: params})
}

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

func newRing(n int) *ringBuf { return &ringBuf{buf: make([]byte, n), max: n} }

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
