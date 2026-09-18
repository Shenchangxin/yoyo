package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

func (a *App) RenameSession(id, title string) error {
	m, err := a.GetSession(id)
	if err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("empty title")
	}
	m.Title = title
	return a.writeSession(m)
}

func (a *App) ForkSession(id string) (SessionMeta, error) {
	src, err := a.GetSession(id)
	if err != nil {
		return SessionMeta{}, err
	}
	dst, err := a.NewSession(src.Workspace)
	if err != nil {
		return SessionMeta{}, err
	}
	dst.Title = "fork of " + titleFrom(src.Title)
	dst.Harness = session.ResolveHarness(session.NormalizePolicy(src.HarnessPolicy), src.Harness, a.ActiveHash())
	dst.HarnessPolicy = session.Pin
	dst.ModelFingerprint = src.ModelFingerprint
	dst.Model = src.Model
	dst.LoadedSkills = append([]string(nil), src.LoadedSkills...)
	dst.PlanText = src.PlanText
	dst.AuthMode = capability.ParseAuthMode(src.AuthMode)
	if err := a.writeSession(dst); err != nil {
		return dst, err
	}
	a.applySessionAuth(dst.ID, dst.AuthMode)
	srcLog := filepath.Join(a.Home.Sessions(), src.ID+".jsonl")
	dstLog := filepath.Join(a.Home.Sessions(), dst.ID+".jsonl")
	if b, err := os.ReadFile(srcLog); err == nil {
		if err := os.WriteFile(dstLog, b, 0o644); err != nil {
			return dst, err
		}
	}
	_ = runtime.CopyTree(a.Home.SessionSpill(src.ID), a.Home.SessionSpill(dst.ID))
	if src.Workspace != "" {
		_ = runtime.CopyTree(
			filepath.Join(src.Workspace, ".yoyo", "context", src.ID),
			filepath.Join(src.Workspace, ".yoyo", "context", dst.ID),
		)
	}
	return dst, nil
}

func (a *App) ListSessions() ([]SessionMeta, error) {
	seen := map[string]bool{}
	var ids []string
	if listed, err := a.Traces.ListSessions(); err == nil {
		for _, id := range listed {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	entries, err := os.ReadDir(a.Home.Sessions())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".meta.json") {
			id := strings.TrimSuffix(name, ".meta.json")
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	var out []SessionMeta
	for _, id := range ids {
		m, err := a.GetSession(id)
		if err != nil {
			// jsonl without meta is a leftover, not a chat
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (a *App) Playbook() (artifact.Playbook, error) {
	_, _, pb, _, _, _, err := a.Materials(a.ActiveHash())
	if st := a.Refs.GetOrEmpty(artifact.RefStaging); st != "" {
		if _, _, staged, _, _, _, e2 := a.Materials(st); e2 == nil && len(staged.Bullets) > 0 {
			pb = staged
		}
	}
	return pb, err
}

func (a *App) RatePlaybook(bulletID string, helpful bool) (artifact.Playbook, error) {
	bulletID = strings.TrimSpace(bulletID)
	if bulletID == "" {
		return artifact.Playbook{}, fmt.Errorf("empty bullet id")
	}
	pb, err := a.Playbook()
	if err != nil {
		return artifact.Playbook{}, err
	}
	found := false
	out := artifact.Playbook{ID: pb.ID, Bullets: make([]artifact.PlaybookBullet, len(pb.Bullets))}
	copy(out.Bullets, pb.Bullets)
	for i := range out.Bullets {
		if out.Bullets[i].ID != bulletID {
			continue
		}
		found = true
		if helpful {
			out.Bullets[i].Helpful++
		} else {
			out.Bullets[i].Harmful++
		}
		out.Bullets[i].Source = "user-thumb"
	}
	if !found {
		return artifact.Playbook{}, fmt.Errorf("unknown bullet %s", bulletID)
	}
	next := out
	active := a.ActiveHash()
	base, err := a.LoadSnapshot(active)
	if err != nil {
		return next, err
	}
	pbHash, err := a.CAS.Put(artifact.KindPlaybook, next.ID, next)
	if err != nil {
		return next, err
	}
	snap := artifact.CloneSnapshot(base)
	snap.Parent = active
	snap.Playbook = pbHash
	snap.Note = "user-thumb-stage"
	hash, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		return next, err
	}
	if err := a.Refs.Set(artifact.RefStaging, hash); err != nil {
		return next, err
	}
	_ = a.Archive.Add(evolve.Node{
		ID: hash, Parent: active, Snapshot: hash, ProposalID: "user-thumb",
		Note: "user-thumb-stage",
	})
	_, _ = a.Journal.Append("ace.thumb", map[string]any{"hash": hash, "bullet": bulletID, "helpful": helpful})
	return next, nil
}

func (a *App) WorkspaceDiff(workspace string) (string, error) {
	if workspace == "" {
		workspace = a.Workspace()
	}
	cmd := exec.Command("git", "-C", workspace, "diff", "--no-color")
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", err
	}
	text := string(out)
	if len(text) > 48_000 {
		text = text[:48_000] + "\n…[diff truncated]"
	}
	if strings.TrimSpace(text) == "" {
		return "(no unstaged git diff)", nil
	}
	return text, nil
}

func (a *App) WorkspaceHunks(workspace string) (map[string]any, error) {
	diff, err := a.WorkspaceDiff(workspace)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"diff":  diff,
		"hunks": runtime.ParseDiffHunks(diff),
	}, nil
}

func (a *App) ApplyWorkspaceHunks(workspace string, ids []string) error {
	if workspace == "" {
		workspace = a.Workspace()
	}
	diff, err := a.WorkspaceDiff(workspace)
	if err != nil {
		return err
	}
	return runtime.ApplyHunks(workspace, diff, ids)
}

func (a *App) ReverseWorkspaceHunks(workspace string, ids []string, snapshot string) error {
	if workspace == "" {
		workspace = a.Workspace()
	}
	if strings.TrimSpace(snapshot) == "" {
		return fmt.Errorf("missing snapshot for undo")
	}
	return runtime.ReverseApplyHunks(workspace, snapshot, ids)
}
