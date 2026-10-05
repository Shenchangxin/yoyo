package filestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var ErrNotFound = errors.New("filestore: not found")

// Store is a process-local JSON document tree. Writes are temp+rename.
// Yoyo is a single GUI process; the mutex serializes job workers and RPC.
type Store struct {
	Root string
	mu   sync.Mutex
}

func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{Root: root}, nil
}

func (s *Store) Abs(rel string) string {
	rel = filepath.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	return filepath.Join(s.Root, rel)
}

func (s *Store) Put(rel string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putLocked(rel, v)
}

func (s *Store) putLocked(rel string, v any) error {
	path := s.Abs(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *Store) Get(rel string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(rel, v)
}

func (s *Store) getLocked(rel string, v any) error {
	b, err := os.ReadFile(s.Abs(rel))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	return json.Unmarshal(b, v)
}

func (s *Store) Delete(rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.Abs(rel))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Store) Exists(rel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := os.Stat(s.Abs(rel))
	return err == nil
}

func (s *Store) IDs(dir string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ents, err := os.ReadDir(s.Abs(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".json"))
	}
	return ids, nil
}

func LoadAll[T any](s *Store, dir string) ([]T, error) {
	ids, err := s.IDs(dir)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(ids))
	for _, id := range ids {
		var v T
		if err := s.Get(filepath.Join(dir, id+".json"), &v); err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("%s/%s: %w", dir, id, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func Rel(col, id string) string {
	id = strings.TrimSpace(id)
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "\\", "_")
	id = strings.ReplaceAll(id, "..", "_")
	return filepath.Join(col, id+".json")
}
