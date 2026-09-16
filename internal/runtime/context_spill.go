package runtime

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zeebo/blake3"
)

// Spill is the lossless tool-result store: compaction is a view, full bytes
// live on disk and can be pulled back with recall_context.
//
// Dir is canonical (YOYO_HOME). Mirror, when set, dual-writes into the
// workspace jail so grep/read_file can see the same ids (DCD).
type Spill struct {
	Dir    string
	Mirror string
	mu     sync.Mutex
}

func NewSpill(dir string) *Spill {
	if dir == "" {
		return nil
	}
	return &Spill{Dir: dir}
}

func SessionSpillDir(homeRoot, sessionID string) string {
	if homeRoot == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(homeRoot, "sessions", sessionID, "spill")
}

func WorkspaceSpillDir(workspace, sessionID string) string {
	if workspace == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "context", sessionID, "spill")
}

// BindSpill points Dir at YOYO_HOME and mirrors into the workspace jail.
func BindSpill(homeRoot, workspace, sessionID string) *Spill {
	if sessionID == "" {
		return nil
	}
	dir := SessionSpillDir(homeRoot, sessionID)
	ws := WorkspaceSpillDir(workspace, sessionID)
	if dir == "" {
		dir = ws
		ws = ""
	}
	if dir == "" {
		return nil
	}
	if ws != "" && samePath(dir, ws) {
		ws = ""
	}
	return &Spill{Dir: dir, Mirror: ws}
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, err := filepath.Abs(a)
	if err != nil {
		return a == b
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		return a == b
	}
	return aa == bb
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
	s.mu.Lock()
	defer s.mu.Unlock()
	writeSpillFile(s.Dir, id, content)
	if s.Mirror != "" {
		writeSpillFile(s.Mirror, id, content)
	}
	return id
}

func writeSpillFile(dir, id, content string) {
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, id+".txt"), []byte(content), 0o644)
}

func (s *Spill) Get(id string) (string, error) {
	if s == nil || s.Dir == "" {
		return "", os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name := sanitizeID(id) + ".txt"
	b, err := os.ReadFile(filepath.Join(s.Dir, name))
	if err == nil {
		return string(b), nil
	}
	if s.Mirror != "" {
		if b, err2 := os.ReadFile(filepath.Join(s.Mirror, name)); err2 == nil {
			return string(b), nil
		}
	}
	return "", err
}

func (s *Spill) ListIDs() []string {
	if s == nil || s.Dir == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]bool{}
	var ids []string
	collect := func(dir string) {
		if dir == "" {
			return
		}
		ents, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range ents {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".txt") {
				continue
			}
			id := strings.TrimSuffix(name, ".txt")
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	collect(s.Dir)
	collect(s.Mirror)
	return ids
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
