package runtime

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeebo/blake3"
)

// Spill is the Claude-style tool-result store: compaction is a view,
// full bytes live on disk and can be pulled back with recall_context.
type Spill struct {
	Dir string
}

func NewSpill(dir string) *Spill {
	if dir == "" {
		return nil
	}
	return &Spill{Dir: dir}
}

func (s *Spill) Put(id, content string) string {
	if s == nil || s.Dir == "" || content == "" {
		return id
	}
	if id == "" {
		sum := blake3.Sum256([]byte(content))
		id = hex.EncodeToString(sum[:8])
	}
	id = sanitizeID(id)
	_ = os.MkdirAll(s.Dir, 0o755)
	_ = os.WriteFile(filepath.Join(s.Dir, id+".txt"), []byte(content), 0o644)
	return id
}

func (s *Spill) Get(id string) (string, error) {
	if s == nil || s.Dir == "" {
		return "", os.ErrNotExist
	}
	b, err := os.ReadFile(filepath.Join(s.Dir, sanitizeID(id)+".txt"))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	var b strings.Builder
	for _, r := range id {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "x"
	}
	if len(out) > 64 {
		return out[:64]
	}
	return out
}
