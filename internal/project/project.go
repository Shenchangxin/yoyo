package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Root        string    `json:"root"`
	Instructions string   `json:"instructions,omitempty"`
	Links       []string  `json:"links,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Store struct {
	mu   sync.Mutex
	path string
	all  []Project
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "projects.json")}
	_ = s.load()
	return s, nil
}

func (s *Store) Create(p Project) Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = strings.ToLower(strings.ReplaceAll(p.Name, " ", "-"))
		if p.ID == "" {
			p.ID = time.Now().UTC().Format("p-20060102T150405")
		}
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	s.all = append(s.all, p)
	_ = s.flush()
	return p
}

func (s *Store) List() []Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Project(nil), s.all...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) Get(id string) (Project, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.all {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.all)
}

func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.all, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
