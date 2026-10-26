// 演化节点的本地 JSON 档案，路径是目录下的 nodes.json。不决定能否晋升，只追加和列出。Cycle 在试用后调用 Add。

package evolve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

// Node 是档案里的一次试用。TransferFail 会在 ParentWeight 里把权重乘 0.3。
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

// Archive 把节点列表存在 Path 指向的 nodes.json。
type Archive struct {
	Path  string
	mu    sync.Mutex
	nodes []Node
}

// OpenArchive 打开 dir/nodes.json。文件不存在时返回空档案且错误为 nil。解析失败时节点保持空，仍然返回 nil 错误。
func OpenArchive(dir string) (*Archive, error) {
	p := filepath.Join(dir, "nodes.json")
	a := &Archive{Path: p}
	b, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(b, &a.nodes)
	}
	return a, nil
}

// Add 追加一个节点。CreatedAt 为零时写成当前 UTC，然后整表落盘。
func (a *Archive) Add(n Node) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	a.nodes = append(a.nodes, n)
	return a.flush()
}

// FlagTransfer 给编号相同的节点写上 TransferFail。档案为 nil 或编号为空时直接返回 nil。同一编号可命中多条。
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

// List 返回节点的副本。
func (a *Archive) List() []Node {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Node, len(a.nodes))
	copy(out, a.nodes)
	return out
}

// flush 先写 Path.tmp，再改名为 Path。
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
