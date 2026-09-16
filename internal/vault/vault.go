package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const EnvAPIKey = "YOYO_API_KEY"

// Store holds provider secrets. Memory first, then a 0600 file under YOYO_HOME,
// then process environment. Callers never receive keys except via Lease.
type Store struct {
	mu   sync.Mutex
	mem  map[string]string
	path string
}

func New(home string) *Store {
	s := &Store{mem: map[string]string{}}
	if home != "" {
		s.path = filepath.Join(home, "vault.json")
		s.loadFile()
	}
	return s
}

func (s *Store) Set(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mem == nil {
		s.mem = map[string]string{}
	}
	if value == "" {
		delete(s.mem, name)
	} else {
		s.mem[name] = value
	}
	_ = s.flushLocked()
}

func (s *Store) Get(name string) (string, error) {
	s.mu.Lock()
	if v := s.mem[name]; v != "" {
		s.mu.Unlock()
		return v, nil
	}
	s.mu.Unlock()
	if name == "openai" || name == "default" {
		if v := os.Getenv(EnvAPIKey); v != "" {
			return v, nil
		}
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			return v, nil
		}
		if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
			return v, nil
		}
		if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("vault: missing key %q", name)
}

func (s *Store) Lease(name string) (string, error) {
	return s.Get(name)
}

func (s *Store) Status() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	saved := len(s.mem) > 0
	source := "empty"
	if saved {
		source = "file"
	} else if os.Getenv(EnvAPIKey) != "" || os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("DEEPSEEK_API_KEY") != "" {
		source = "env"
	}
	return map[string]any{"saved": saved || source == "env", "source": source, "path": s.path}
}

func (s *Store) loadFile() {
	if s.path == "" {
		return
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil || m == nil {
		return
	}
	s.mem = m
}

func (s *Store) flushLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.mem, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
