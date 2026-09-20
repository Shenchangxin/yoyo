package artifact

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeebo/blake3"
)

// Store is a git-like content-addressed object store.
type Store struct {
	Root string
}

func NewStore(root string) *Store {
	return &Store{Root: root}
}

func HashBytes(b []byte) string {
	sum := blake3.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Store) pathFor(hash string) string {
	if len(hash) < 4 {
		return filepath.Join(s.Root, hash)
	}
	return filepath.Join(s.Root, hash[:2], hash)
}

// Path is the on-disk location of a CAS object. The file may not exist yet.
func (s *Store) Path(hash string) string {
	return s.pathFor(strings.TrimSpace(hash))
}

func (s *Store) Put(kind Kind, id string, payload any) (string, error) {
	env := Envelope{
		Kind:      kind,
		ID:        id,
		CreatedAt: time.Now().UTC(),
		Payload:   payload,
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	h := HashBytes(raw)
	p := s.pathFor(h)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		return "", err
	}
	return h, nil
}

func (s *Store) PutRaw(b []byte) (string, error) {
	h := HashBytes(b)
	p := s.pathFor(h)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err == nil {
		return h, nil
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return "", err
	}
	return h, nil
}

func (s *Store) GetRaw(hash string) ([]byte, error) {
	hash = strings.TrimSpace(hash)
	p := s.pathFor(hash)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("cas: missing %s: %w", hash, err)
	}
	return b, nil
}

func (s *Store) GetEnvelope(hash string) (Envelope, []byte, error) {
	b, err := s.GetRaw(hash)
	if err != nil {
		return Envelope{}, nil, err
	}
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return Envelope{}, b, err
	}
	return env, b, nil
}

func Decode[T any](s *Store, hash string) (T, Envelope, error) {
	var zero T
	env, raw, err := s.GetEnvelope(hash)
	if err != nil {
		return zero, env, err
	}
	var wrap struct {
		Payload T `json:"payload"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return zero, env, err
	}
	return wrap.Payload, env, nil
}

func (s *Store) Exists(hash string) bool {
	_, err := os.Stat(s.pathFor(strings.TrimSpace(hash)))
	return err == nil
}
