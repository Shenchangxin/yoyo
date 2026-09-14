package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type Server struct {
	Name    string
	Command string
	Args    []string
	cmd     *exec.Cmd
}

type Host struct {
	mu      sync.Mutex
	servers map[string]*Server
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	h.servers[name] = &Server{Name: name, Command: command, Args: args, cmd: cmd}
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
