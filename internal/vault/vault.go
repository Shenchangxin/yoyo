package vault

import (
	"fmt"
	"os"
	"sync"
)

const EnvAPIKey = "YOYO_API_KEY"

// Store holds provider secrets. Memory first; env is the default source.
// OS keychain can be wired later without changing callers.
type Store struct {
	mu  sync.Mutex
	mem map[string]string
}

func New() *Store {
	return &Store{mem: map[string]string{}}
}

func (s *Store) Set(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mem[name] = value
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
