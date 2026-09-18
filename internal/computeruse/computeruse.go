package computeruse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Event struct {
	TS      time.Time `json:"ts"`
	Op      string    `json:"op"`
	App     string    `json:"app"`
	Detail  string    `json:"detail"`
	Display string    `json:"display"`
}

type Host struct {
	mu        sync.Mutex
	dir       string
	allow     map[string]bool
	events    []Event
	display   string
	hwnd      uintptr
}

func Open(dir string) (*Host, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	h := &Host{dir: dir, allow: map[string]bool{}, display: "virtual"}
	_ = h.loadAllow()
	h.ensureVirtual()
	return h, nil
}

func (h *Host) Allow(app string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allow[strings.ToLower(app)] = true
	_ = h.flushAllow()
}

func (h *Host) Allowed() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	for k := range h.allow {
		out = append(out, k)
	}
	return out
}

func (h *Host) Act(app, op, detail string) (Event, error) {
	h.ensureVirtual()
	app = strings.ToLower(strings.TrimSpace(app))
	h.mu.Lock()
	defer h.mu.Unlock()
	if app != "" && !h.allow[app] && !h.allow["*"] {
		return Event{}, fmt.Errorf("computer_use: app %q not on allowlist (virtual display only; never the operator desktop)", app)
	}
	ev := Event{TS: time.Now().UTC(), Op: op, App: app, Detail: detail, Display: h.display}
	h.events = append(h.events, ev)
	raw, _ := json.MarshalIndent(h.events, "", "  ")
	_ = os.WriteFile(filepath.Join(h.dir, "recording.json"), raw, 0o600)
	return ev, nil
}

func (h *Host) Recording() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Event(nil), h.events...)
}

func (h *Host) loadAllow() error {
	b, err := os.ReadFile(filepath.Join(h.dir, "allow.json"))
	if err != nil {
		return err
	}
	var list []string
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	for _, a := range list {
		h.allow[strings.ToLower(a)] = true
	}
	return nil
}

func (h *Host) flushAllow() error {
	var list []string
	for a := range h.allow {
		list = append(list, a)
	}
	b, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(filepath.Join(h.dir, "allow.json"), b, 0o644)
}
