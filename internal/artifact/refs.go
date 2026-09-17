package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	RefActive  = "active"
	RefCanary  = "canary"
	RefHead    = "HEAD"
	RefStaging = "staging"
)

// Refs is a git-like pointer store over CAS hashes.
type Refs struct {
	Root string
	mu   sync.Mutex
}

func NewRefs(root string) *Refs {
	return &Refs{Root: root}
}

func (r *Refs) path(name string) string {
	name = strings.TrimPrefix(name, "/")
	return filepath.Join(r.Root, filepath.FromSlash(name))
}

func (r *Refs) Set(name, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.path(name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.TrimSpace(hash)+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (r *Refs) Get(name string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, err := os.ReadFile(r.path(name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (r *Refs) GetOrEmpty(name string) string {
	h, err := r.Get(name)
	if err != nil {
		return ""
	}
	return h
}

func (r *Refs) List() (map[string]string, error) {
	out := map[string]string{}
	err := filepath.Walk(r.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".tmp") {
			return nil
		}
		rel, err := filepath.Rel(r.Root, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = strings.TrimSpace(string(b))
		return nil
	})
	return out, err
}

func (r *Refs) Archive(hash string) error {
	if hash == "" {
		return fmt.Errorf("refs: empty archive hash")
	}
	short := hash
	if len(short) > 12 {
		short = short[:12]
	}
	return r.Set("archive/"+short, hash)
}

func ModelActive(fingerprint string) string {
	return "models/" + sanitize(fingerprint) + "/active"
}

func ModelCanary(fingerprint string) string {
	return "models/" + sanitize(fingerprint) + "/canary"
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
