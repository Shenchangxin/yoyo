package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/hostopen"
)

const lineageCap = 200

type HarnessDirs struct {
	Home    string `json:"home"`
	CAS     string `json:"cas"`
	Refs    string `json:"refs"`
	Archive string `json:"archive"`
}

type LineageNode struct {
	Hash        string                    `json:"hash"`
	Parent      string                    `json:"parent,omitempty"`
	ID          string                    `json:"id,omitempty"`
	Note        string                    `json:"note,omitempty"`
	Model       string                    `json:"model,omitempty"`
	CreatedAt   time.Time                 `json:"created_at,omitempty"`
	Refs        []string                  `json:"refs,omitempty"`
	Changes     []artifact.MaterialChange `json:"changes,omitempty"`
	Path        string                    `json:"path,omitempty"`
	Seed        bool                      `json:"seed,omitempty"`
	ProposalID  string                    `json:"proposal_id,omitempty"`
	Accepted    bool                      `json:"accepted,omitempty"`
	FromArchive bool                      `json:"from_archive,omitempty"`
}

type HarnessLineage struct {
	Active string        `json:"active"`
	Dirs   HarnessDirs   `json:"dirs"`
	Nodes  []LineageNode `json:"nodes"`
}

func (a *App) HarnessDirs() HarnessDirs {
	if a.Home == nil {
		return HarnessDirs{}
	}
	return HarnessDirs{
		Home:    a.Home.Root,
		CAS:     a.Home.CAS(),
		Refs:    a.Home.Refs(),
		Archive: a.Home.Archive(),
	}
}

func (a *App) HarnessState() (map[string]any, error) {
	refs, err := a.ListHarnesses()
	if err != nil {
		return nil, err
	}
	snap, _ := a.LoadSnapshot(a.ActiveHash())
	lin, _ := a.HarnessLineage()
	return map[string]any{
		"active":   a.ActiveHash(),
		"refs":     refs,
		"snapshot": snap,
		"dirs":     lin.Dirs,
		"lineage":  lin.Nodes,
	}, nil
}

func (a *App) HarnessLineage() (HarnessLineage, error) {
	out := HarnessLineage{Active: a.ActiveHash(), Dirs: a.HarnessDirs()}
	if a.CAS == nil {
		return out, nil
	}
	seeds := map[string][]string{}
	if refs, err := a.ListHarnesses(); err == nil {
		for name, hash := range refs {
			hash = strings.TrimSpace(hash)
			if hash == "" {
				continue
			}
			seeds[hash] = append(seeds[hash], name)
		}
	}
	archiveBySnap := map[string]evolve.Node{}
	if a.Archive != nil {
		for _, n := range a.Archive.List() {
			h := strings.TrimSpace(n.Snapshot)
			if h == "" {
				h = strings.TrimSpace(n.ID)
			}
			if h == "" {
				continue
			}
			if prev, ok := archiveBySnap[h]; !ok || n.CreatedAt.After(prev.CreatedAt) {
				archiveBySnap[h] = n
			}
			if _, ok := seeds[h]; !ok {
				seeds[h] = nil
			}
		}
	}
	seen := map[string]bool{}
	var hashes []string
	var walk func(string)
	walk = func(hash string) {
		hash = strings.TrimSpace(hash)
		if hash == "" || seen[hash] {
			return
		}
		if len(hashes) >= lineageCap {
			return
		}
		seen[hash] = true
		hashes = append(hashes, hash)
		snap, err := a.CAS.GetSnapshot(hash)
		if err != nil {
			return
		}
		walk(snap.Parent)
	}
	for hash := range seeds {
		walk(hash)
	}
	nodes := make([]LineageNode, 0, len(hashes))
	for _, hash := range hashes {
		node := LineageNode{Hash: hash, Path: a.CAS.Path(hash), Refs: sortedCopy(seeds[hash])}
		if env, _, err := a.CAS.GetEnvelope(hash); err == nil {
			node.CreatedAt = env.CreatedAt
			if node.ID == "" {
				node.ID = env.ID
			}
		}
		snap, err := a.CAS.GetSnapshot(hash)
		if err == nil {
			node.Parent = snap.Parent
			node.ID = snap.ID
			node.Note = snap.Note
			node.Model = snap.ModelFingerprint
			node.Seed = snap.Parent == ""
			if snap.Parent != "" {
				if parent, err := a.CAS.GetSnapshot(snap.Parent); err == nil {
					node.Changes = artifact.MaterialDiff(a.CAS, parent, snap)
				}
			}
		}
		if arch, ok := archiveBySnap[hash]; ok {
			node.FromArchive = true
			node.ProposalID = arch.ProposalID
			node.Accepted = arch.Accepted
			if node.Note == "" {
				node.Note = arch.Note
			}
		}
		nodes = append(nodes, node)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		ti, tj := nodes[i].CreatedAt, nodes[j].CreatedAt
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return nodes[i].Hash > nodes[j].Hash
	})
	out.Nodes = nodes
	return out, nil
}

func (a *App) MaterializeSnapshot(hash string) (string, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		hash = a.ActiveHash()
	}
	if hash == "" {
		return "", errNoSnapshot
	}
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(a.Home.Tmp(), "harness", shortRef(hash))
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, "prompts"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, "skills"), 0o755); err != nil {
		return "", err
	}
	if err := writeJSONFile(filepath.Join(dir, "snapshot.json"), snap); err != nil {
		return "", err
	}
	dump := func(hash, name string) {
		if hash == "" {
			return
		}
		env, _, err := a.CAS.GetEnvelope(hash)
		if err != nil {
			return
		}
		_ = writeJSONFile(filepath.Join(dir, name), env.Payload)
	}
	dump(snap.Playbook, "playbook.json")
	dump(snap.LoopPreset, "loop.json")
	dump(snap.PolicyPack, "policy.json")
	dump(snap.EvalSuite, "eval.json")
	for i, h := range snap.PromptFragments {
		v, _, err := artifact.Decode[artifact.PromptFragment](a.CAS, h)
		if err != nil {
			continue
		}
		name := v.ID
		if name == "" {
			name = fmt.Sprintf("%02d", i)
		}
		_ = writeJSONFile(filepath.Join(dir, "prompts", name+".json"), v)
	}
	for _, h := range snap.Skills {
		v, _, err := artifact.Decode[artifact.Skill](a.CAS, h)
		if err != nil {
			continue
		}
		name := v.Name
		if name == "" {
			name = shortRef(h)
		}
		_ = writeJSONFile(filepath.Join(dir, "skills", sanitizeFile(name)+".json"), v)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "hash\t%s\nparent\t%s\nid\t%s\nnote\t%s\nmodel\t%s\n", hash, snap.Parent, snap.ID, snap.Note, snap.ModelFingerprint)
	fmt.Fprintf(&b, "playbook\t%s\nloop\t%s\npolicy\t%s\neval\t%s\n", snap.Playbook, snap.LoopPreset, snap.PolicyPack, snap.EvalSuite)
	fmt.Fprintf(&b, "prompts\t%s\nskills\t%s\ntools\t%s\n", strings.Join(snap.PromptFragments, ","), strings.Join(snap.Skills, ","), strings.Join(snap.Tools, ","))
	if err := os.WriteFile(filepath.Join(dir, "MANIFEST.txt"), []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) RevealHarness(hash string) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		if a.Home == nil {
			return os.ErrInvalid
		}
		return hostopen.Path(a.Home.CAS())
	}
	dir, err := a.MaterializeSnapshot(hash)
	if err != nil {
		if a.CAS != nil {
			p := a.CAS.Path(hash)
			if _, st := os.Stat(p); st == nil {
				return hostopen.Reveal(p)
			}
		}
		return err
	}
	return hostopen.Path(dir)
}

func writeJSONFile(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func shortRef(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) > 12 {
		return hash[:12]
	}
	if hash == "" {
		return "harness"
	}
	return hash
}

func sanitizeFile(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, s)
	if s == "" {
		return "item"
	}
	return s
}

func sortedCopy(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
