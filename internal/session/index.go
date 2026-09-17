package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// Index is a rebuildable search sidecar. JSONL remains the source of truth.
type Index struct {
	mu   sync.Mutex
	path string
	docs map[string]string
}

func OpenIndex(dir string) *Index {
	idx := &Index{path: filepath.Join(dir, "_search.json"), docs: map[string]string{}}
	if b, err := os.ReadFile(idx.path); err == nil {
		_ = json.Unmarshal(b, &idx.docs)
	}
	return idx
}

func (x *Index) Put(id, blob string) {
	if x == nil {
		return
	}
	x.mu.Lock()
	x.docs[id] = strings.ToLower(blob)
	x.mu.Unlock()
}

func (x *Index) Delete(id string) {
	if x == nil {
		return
	}
	x.mu.Lock()
	delete(x.docs, id)
	x.mu.Unlock()
}

func (x *Index) Match(q string) []string {
	if x == nil {
		return nil
	}
	q = strings.ToLower(strings.TrimSpace(q))
	x.mu.Lock()
	defer x.mu.Unlock()
	if q == "" {
		out := make([]string, 0, len(x.docs))
		for id := range x.docs {
			out = append(out, id)
		}
		return out
	}
	terms := tokenize(q)
	var out []string
	for id, blob := range x.docs {
		ok := true
		for _, t := range terms {
			if !strings.Contains(blob, t) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, id)
		}
	}
	return out
}

func (x *Index) Flush() error {
	if x == nil {
		return nil
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	b, err := json.Marshal(x.docs)
	if err != nil {
		return err
	}
	return os.WriteFile(x.path, b, 0o644)
}

func tokenize(s string) []string {
	var b strings.Builder
	var out []string
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}
