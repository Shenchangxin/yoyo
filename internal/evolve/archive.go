package evolve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

type Node struct {
	ID            string       `json:"id"`
	Parent        string       `json:"parent"`
	Snapshot      string       `json:"snapshot"`
	ProposalID    string       `json:"proposal_id,omitempty"`
	Metrics       eval.Metrics `json:"metrics"`
	Accepted      bool         `json:"accepted"`
	Canary        bool         `json:"canary"`
	Note          string       `json:"note,omitempty"`
	Surface       string       `json:"surface,omitempty"`
	ManifestoHit  int          `json:"manifesto_hit,omitempty"`
	ManifestoMiss int          `json:"manifesto_miss,omitempty"`
	TransferFail  bool         `json:"transfer_fail,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

type Archive struct {
	Path  string
	mu    sync.Mutex
	nodes []Node
}

func OpenArchive(dir string) (*Archive, error) {
	p := filepath.Join(dir, "nodes.json")
	a := &Archive{Path: p}
	b, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(b, &a.nodes)
	}
	return a, nil
}

func (a *Archive) Add(n Node) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	a.nodes = append(a.nodes, n)
	return a.flush()
}

func (a *Archive) FlagTransfer(id string, fail bool) error {
	if a == nil || id == "" {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.nodes {
		if a.nodes[i].ID == id {
			a.nodes[i].TransferFail = fail
		}
	}
	return a.flush()
}

func (a *Archive) List() []Node {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Node, len(a.nodes))
	copy(out, a.nodes)
	return out
}

func (a *Archive) flush() error {
	if err := os.MkdirAll(filepath.Dir(a.Path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(a.nodes, "", "  ")
	if err != nil {
		return err
	}
	tmp := a.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, a.Path)
}
