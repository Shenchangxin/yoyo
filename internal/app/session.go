package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type SessionMeta struct {
	ID               string    `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	Workspace        string    `json:"workspace"`
	Harness          string    `json:"harness"`
	ModelFingerprint string    `json:"model_fingerprint"`
	Title            string    `json:"title,omitempty"`
}

func (a *App) NewSession(workspace string) (SessionMeta, error) {
	if workspace == "" {
		workspace = a.Workspace()
	}
	id := newID()
	meta := SessionMeta{
		ID:               id,
		CreatedAt:        time.Now().UTC(),
		Workspace:        workspace,
		Harness:          a.ActiveHash(),
		ModelFingerprint: Fingerprint(a.Config),
		Title:            "session",
	}
	b, _ := json.MarshalIndent(meta, "", "  ")
	path := filepath.Join(a.Home.Sessions(), id+".meta.json")
	return meta, os.WriteFile(path, b, 0o644)
}

func (a *App) ListSessions() ([]SessionMeta, error) {
	ids, err := a.Traces.ListSessions()
	if err != nil {
		return nil, err
	}
	var out []SessionMeta
	for _, id := range ids {
		if m, err := a.GetSession(id); err == nil {
			out = append(out, m)
		} else {
			out = append(out, SessionMeta{ID: id})
		}
	}
	return out, nil
}

func (a *App) GetSession(id string) (SessionMeta, error) {
	b, err := os.ReadFile(filepath.Join(a.Home.Sessions(), id+".meta.json"))
	if err != nil {
		return SessionMeta{ID: id}, err
	}
	var m SessionMeta
	return m, json.Unmarshal(b, &m)
}

func (a *App) Trajectory(id string) ([]trace.Event, error) {
	return a.Traces.Read(id)
}

func (a *App) Client() (runtime.Client, error) {
	key, _ := a.Vault.Lease("default")
	return runtime.NewOpenAIClient(a.Config.BaseURL, key), nil
}

func (a *App) Send(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event)) (string, error) {
	meta, err := a.GetSession(sessionID)
	if err != nil {
		meta, err = a.NewSession("")
		if err != nil {
			return "", err
		}
		sessionID = meta.ID
	}
	hash := a.ActiveHash()
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return "", err
	}
	loop, frags, pb, skills, _, err := a.Materials(hash)
	if err != nil {
		return "", err
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return "", err
		}
	}
	skillBodies := map[string]string{}
	for _, s := range skills {
		skillBodies[s.Name] = s.Body
	}
	tools := &runtime.WorkspaceTools{
		Workspace: meta.Workspace,
		SessionID: sessionID,
		Caps:      a.Caps,
		Skills:    skillBodies,
	}
	return runtime.Run(ctx, runtime.RunRequest{
		SessionID:        sessionID,
		User:             message,
		Workspace:        meta.Workspace,
		Harness:          snap,
		HarnessHash:      hash,
		ModelFingerprint: Fingerprint(a.Config),
		Model:            a.Config.Model,
		Loop:             loop,
		Fragments:        frags,
		Playbook:         pb,
		Skills:           skills,
		Tools:            tools,
		Client:           client,
		Trace:            a.Traces,
		OnEvent:          onEvent,
	})
}

func (a *App) RunEval(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	hash := a.ActiveHash()
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return eval.RunReport{}, err
	}
	loop, frags, pb, skills, suite, err := a.Materials(hash)
	if err != nil {
		return eval.RunReport{}, err
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return eval.RunReport{}, err
		}
	}
	return a.Eval.Run(ctx, eval.RunOpts{
		Suite:     suite,
		Snapshot:  snap,
		Hash:      hash,
		Client:    client,
		Loop:      loop,
		Fragments: frags,
		Playbook:  pb,
		Skills:    skills,
		Trace:     a.Traces,
		SessionID: "eval-" + newID()[:8],
		Model:     a.Config.Model,
	})
}

func (a *App) EvolveOnce(ctx context.Context, client runtime.Client, failed map[string]string) (evolve.CycleResult, error) {
	hash := a.ActiveHash()
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	loop, frags, pb, skills, suite, err := a.Materials(hash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return evolve.CycleResult{}, err
		}
	}
	if failed == nil {
		failed = map[string]string{}
		report, err := a.RunEval(ctx, client)
		if err == nil {
			for _, r := range report.Results {
				if !r.Pass {
					failed[r.ID] = r.Error
					if failed[r.ID] == "" {
						failed[r.ID] = "verifier_fail"
					}
				}
			}
		}
	}
	sid := "evolve-" + newID()[:8]
	return a.Evolve.Cycle(ctx, evolve.CycleOpts{
		Base:          snap,
		BaseHash:      hash,
		Suite:         suite,
		Loop:          loop,
		Fragments:     frags,
		Playbook:      pb,
		Skills:        skills,
		Client:        client,
		Trace:         a.Traces,
		SessionID:     sid,
		Model:         a.Config.Model,
		FailedTasks:   failed,
		K:             3,
		PromoteCanary: true,
	})
}

func (a *App) ListHarnesses() (map[string]string, error) {
	return a.Refs.List()
}

func (a *App) Diff(aHash, bHash string) (string, error) {
	left, err := a.LoadSnapshot(aHash)
	if err != nil {
		return "", err
	}
	right, err := a.LoadSnapshot(bHash)
	if err != nil {
		return "", err
	}
	return artifact.SnapshotDiff(left, right), nil
}

func (a *App) LoadWASM(id string, bin []byte, export string) error {
	if err := a.Caps.Check(capability.Request{Level: capability.HighRisk, Action: "load_wasm", SessionID: "ui"}); err != nil {
		if !a.Config.AutoAllow {
			return err
		}
	}
	_, err := evolve.AdmitWASM(a.Kernel, a.WASM, id, bin, export)
	return err
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
