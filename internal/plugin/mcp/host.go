package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Host speaks MCP JSON-RPC over stdio. No LangChain: the agent loop remains
// Yoyo's, and MCP tools are just another ExtraTool backend.
type Host struct {
	mu      sync.Mutex
	servers map[string]*Server
}

type Server struct {
	Name    string
	Command string
	Args    []string
	Tools   []Tool
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  int
}

type Tool struct {
	Server      string
	Name        string
	Description string
	InputSchema map[string]any
}

func NewHost() *Host {
	return &Host{servers: map[string]*Server{}}
}

func (h *Host) Start(name, command string, args []string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.servers[name]; ok {
		return fmt.Errorf("mcp: %s already running", name)
	}
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return err
	}
	s := &Server{
		Name:    name,
		Command: command,
		Args:    args,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
	}
	if err := s.handshake(); err != nil {
		_ = cmd.Process.Kill()
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
	if !ok || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	return s.cmd.Process.Kill()
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
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Tools   int      `json:"tools"`
}

func (h *Host) Info() []Info {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Info, 0, len(h.servers))
	for _, s := range h.servers {
		out = append(out, Info{Name: s.Name, Command: s.Command, Args: s.Args, Tools: len(s.Tools)})
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

func (s *Server) handshake() error {
	_, err := s.call("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "yoyo", "version": "0.2.4"},
	})
	if err != nil {
		return err
	}
	_ = s.notify("notifications/initialized", map[string]any{})
	raw, err := s.call("tools/list", map[string]any{})
	if err != nil {
		return err
	}
	var listed struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		return err
	}
	for _, t := range listed.Tools {
		s.Tools = append(s.Tools, Tool{
			Server:      s.Name,
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return nil
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
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Server) call(method string, params any) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := s.nextID
	if err := s.write(rpcReq{JSONRPC: "2.0", ID: id, Method: method, Params: params}); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		res, err := s.read()
		if err != nil {
			return nil, err
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

func (s *Server) notify(method string, params any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(rpcReq{JSONRPC: "2.0", Method: method, Params: params})
}

func (s *Server) write(v any) error {
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
