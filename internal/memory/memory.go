package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Kind string

const (
	KindProfile  Kind = "profile"
	KindProject  Kind = "project"
	KindEpisodic Kind = "episodic"
	KindSemantic Kind = "semantic"
)

type Item struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Text      string    `json:"text"`
	Project   string    `json:"project,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Staging   bool      `json:"staging,omitempty"`
}

type Store struct {
	mu   sync.Mutex
	path string
	items []Item
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "memory.json")}
	_ = s.load()
	return s, nil
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
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Write(it Item) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it.ID == "" {
		it.ID = time.Now().UTC().Format("20060102T150405.000000000")
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = time.Now().UTC()
	}
	it.Staging = it.Kind != KindProfile
	if it.Kind == KindProfile {
		it.Staging = false
	}
	s.items = append(s.items, it)
	_ = s.flush()
	return it
}

func (s *Store) Forget(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	ok := false
	for _, it := range s.items {
		if it.ID == id {
			ok = true
			continue
		}
		out = append(out, it)
	}
	s.items = out
	_ = s.flush()
	return ok
}

func (s *Store) Search(q string, kind Kind, limit int) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var out []Item
	for _, it := range s.items {
		if kind != "" && it.Kind != kind {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.Text), q) && !strings.Contains(strings.ToLower(it.Project), q) {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *Store) ProfilePin(budget int) string {
	items := s.Search("", KindProfile, 32)
	var b strings.Builder
	n := 0
	for _, it := range items {
		if it.Staging {
			continue
		}
		line := "- " + it.Text + "\n"
		if budget > 0 && n+len(line) > budget {
			break
		}
		b.WriteString(line)
		n += len(line)
	}
	return b.String()
}

func (s *Store) Promote(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Staging = false
			_ = s.flush()
			return true
		}
	}
	return false
}

func (s *Store) All() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Item(nil), s.items...)
	return out
}

func (s *Store) Path() string { return s.path }
