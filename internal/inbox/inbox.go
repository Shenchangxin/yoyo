package inbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Kind string

const (
	KindApproval  Kind = "approval"
	KindSchedule  Kind = "schedule"
	KindAsk       Kind = "ask_user"
	KindHeartbeat Kind = "heartbeat"
	KindArtifact  Kind = "artifact"
)

type Item struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	SessionID string    `json:"session_id,omitempty"`
	Unread    bool      `json:"unread"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu    sync.Mutex
	path  string
	items []Item
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "inbox.json")}
	_ = s.load()
	return s, nil
}

func (s *Store) Push(it Item) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it.ID == "" {
		it.ID = time.Now().UTC().Format("in-20060102T150405.000000000")
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = time.Now().UTC()
	}
	it.Unread = true
	s.items = append(s.items, it)
	if len(s.items) > 500 {
		s.items = s.items[len(s.items)-500:]
	}
	_ = s.flush()
	return it
}

func (s *Store) List() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Item(nil), s.items...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) Unread() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, it := range s.items {
		if it.Unread {
			n++
		}
	}
	return n
}

func (s *Store) MarkRead(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Unread = false
		}
	}
	_ = s.flush()
}

func (s *Store) Dismiss(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	found := false
	for _, it := range s.items {
		if it.ID == id {
			found = true
			continue
		}
		out = append(out, it)
	}
	if !found {
		return false
	}
	s.items = out
	_ = s.flush()
	return true
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.items)
}

func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
