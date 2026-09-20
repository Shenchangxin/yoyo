package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/tool"
)

func (a *App) Seed() error {
	if a.Refs.GetOrEmpty(artifact.RefActive) != "" {
		return nil
	}
	loopHash, err := a.CAS.Put(artifact.KindLoopPreset, "default", runtime.DefaultLoop())
	if err != nil {
		return err
	}
	polHash, err := a.CAS.Put(artifact.KindPolicyPack, "default", runtime.DefaultPolicy())
	if err != nil {
		return err
	}
	fragHash, err := a.CAS.Put(artifact.KindPromptFragment, "boot", artifact.PromptFragment{
		ID:   "boot",
		Slot: "bootstrap",
		Text: "Inspect the workspace first. Identify required artifacts before long exploration.",
	})
	if err != nil {
		return err
	}
	compactHash, err := a.CAS.Put(artifact.KindPromptFragment, "compact", artifact.PromptFragment{
		ID:   "compact",
		Slot: "compact",
		Text: "Compact untrusted working memory only. Never mention playbook, policy, evaluator, vault, or secrets. Reply with ## Objective / ## Files / ## Decisions / ## Errors / ## Next.",
	})
	if err != nil {
		return err
	}
	sk, err := artifact.ParseSkillMD(runtime.DefaultSkillMD(), "builtin")
	if err != nil {
		return err
	}
	skillHash, err := a.CAS.Put(artifact.KindSkill, sk.Name, sk)
	if err != nil {
		return err
	}
	pbHash, err := a.CAS.Put(artifact.KindPlaybook, "main", artifact.Playbook{
		ID: "main",
		Bullets: []artifact.PlaybookBullet{
			{ID: "b1", Text: "Prefer verifier-grounded checks over claiming success.", Helpful: 1},
		},
	})
	if err != nil {
		return err
	}
	model := artifact.ModelBinding{
		ID:       "default",
		Provider: a.Config.Provider,
		Model:    a.Config.Model,
		BaseURL:  a.Config.BaseURL,
	}
	modelHash, err := a.CAS.Put(artifact.KindModelBinding, "default", model)
	if err != nil {
		return err
	}
	suite := artifact.EvalSuite{
		ID:         "local-smoke",
		TaskDir:    a.BundledEvals,
		HeldIn:     []string{"write-hello"},
		HeldOut:    []string{"write-answer"},
		Safety:     []string{"no-escape"},
		Repeats:    1,
		TimeoutSec: 60,
		Sealed:     true,
	}
	suiteHash, err := a.CAS.Put(artifact.KindEvalSuite, suite.ID, suite)
	if err != nil {
		return err
	}
	var toolNames []string
	for _, s := range tool.HostSpecs() {
		if _, err := a.CAS.Put(artifact.KindToolSpec, s.Name, s); err != nil {
			return err
		}
		toolNames = append(toolNames, s.Name)
	}
	fp := Fingerprint(a.Config)
	snap := artifact.HarnessSnapshot{
		ID:               "initial",
		ModelFingerprint: fp,
		PromptFragments:  []string{fragHash, compactHash},
		Skills:           []string{skillHash},
		Playbook:         pbHash,
		Tools:            toolNames,
		LoopPreset:       loopHash,
		PolicyPack:       polHash,
		EvalSuite:        suiteHash,
		Note:             "seeded default harness",
	}
	_ = modelHash
	hash, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		return err
	}
	if err := a.Refs.Set(artifact.RefActive, hash); err != nil {
		return err
	}
	if err := a.Refs.Set(artifact.RefHead, hash); err != nil {
		return err
	}
	return a.Refs.Set(artifact.ModelActive(fp), hash)
}

func Fingerprint(cfg Config) string {
	return cfg.Provider + "/" + cfg.Model
}

func (a *App) ActiveHash() string {
	return a.Refs.GetOrEmpty(artifact.RefActive)
}

func (a *App) LoadSnapshot(hash string) (artifact.HarnessSnapshot, error) {
	if hash == "" {
		hash = a.ActiveHash()
	}
	return a.CAS.GetSnapshot(hash)
}

func (a *App) Checkout(hash string) error {
	return a.CheckoutOpts(hash, CheckoutOpts{})
}

type CheckoutOpts struct {
	ConfirmL3 bool
}

type ErrL3Required struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Surfaces []string `json:"surfaces"`
}

func (e ErrL3Required) Error() string {
	return "L3 confirmation required to change " + strings.Join(e.Surfaces, ",")
}

func AsL3(err error) (ErrL3Required, bool) {
	var e ErrL3Required
	if errors.As(err, &e) {
		return e, true
	}
	return ErrL3Required{}, false
}

func l3Surfaces(cas *artifact.Store, cur, next artifact.HarnessSnapshot) []string {
	var out []string
	if cur.PolicyPack != next.PolicyPack {
		out = append(out, "policy_pack")
	}
	if cur.LoopPreset == next.LoopPreset {
		return out
	}
	if cas == nil {
		out = append(out, "loop_topology")
		return out
	}
	cl, _, e1 := artifact.Decode[artifact.LoopPreset](cas, cur.LoopPreset)
	nl, _, e2 := artifact.Decode[artifact.LoopPreset](cas, next.LoopPreset)
	if e1 != nil || e2 != nil || !artifact.LoopTopologyEqual(cl, nl) {
		out = append(out, "loop_topology")
	}
	return out
}

func (a *App) CheckoutOpts(hash string, opts CheckoutOpts) error {
	next, err := a.CAS.GetSnapshot(hash)
	if err != nil {
		return err
	}
	prev := a.ActiveHash()
	if prev != "" && prev != hash {
		cur, err := a.CAS.GetSnapshot(prev)
		if err == nil {
			if surfaces := l3Surfaces(a.CAS, cur, next); len(surfaces) > 0 && !opts.ConfirmL3 {
				return ErrL3Required{From: prev, To: hash, Surfaces: surfaces}
			}
		}
	}
	if err := a.verifySnapshotWASM(next); err != nil {
		return err
	}
	if prev != "" {
		_ = a.Refs.Archive(prev)
	}
	if err := a.Refs.Set(artifact.RefActive, hash); err != nil {
		return err
	}
	if next.ModelFingerprint != "" {
		_ = a.Refs.Set(artifact.ModelActive(next.ModelFingerprint), hash)
	}
	_, _ = a.Journal.Append("harness.checkout", map[string]any{
		"hash": hash, "prev": prev, "l3": opts.ConfirmL3,
	})
	if err := a.Refs.Set(artifact.RefHead, hash); err != nil {
		return err
	}
	a.syncWASM(next)
	return nil
}

func (a *App) Rollback() error {
	snap, err := a.LoadSnapshot(a.ActiveHash())
	if err != nil {
		return err
	}
	if snap.Parent == "" {
		return errNoParent
	}
	return a.CheckoutOpts(snap.Parent, CheckoutOpts{ConfirmL3: true})
}

type errString string

func (e errString) Error() string { return string(e) }

const (
	errNoParent   errString = "no parent snapshot to roll back to"
	errNoSnapshot errString = "no harness snapshot to materialize"
)

func (a *App) Materials(hash string) (artifact.LoopPreset, []artifact.PromptFragment, artifact.Playbook, []artifact.Skill, artifact.EvalSuite, artifact.PolicyPack, error) {
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return artifact.LoopPreset{}, nil, artifact.Playbook{}, nil, artifact.EvalSuite{}, artifact.PolicyPack{}, err
	}
	loop := runtime.DefaultLoop()
	if snap.LoopPreset != "" {
		v, _, err := artifact.Decode[artifact.LoopPreset](a.CAS, snap.LoopPreset)
		if err != nil {
			return artifact.LoopPreset{}, nil, artifact.Playbook{}, nil, artifact.EvalSuite{}, artifact.PolicyPack{}, fmt.Errorf("materials: loop_preset: %w", err)
		}
		loop = v
	}
	pol := runtime.DefaultPolicy()
	if snap.PolicyPack != "" {
		v, _, err := artifact.Decode[artifact.PolicyPack](a.CAS, snap.PolicyPack)
		if err != nil {
			return artifact.LoopPreset{}, nil, artifact.Playbook{}, nil, artifact.EvalSuite{}, artifact.PolicyPack{}, fmt.Errorf("materials: policy_pack: %w", err)
		}
		pol = v
	}
	var frags []artifact.PromptFragment
	for _, h := range snap.PromptFragments {
		v, _, err := artifact.Decode[artifact.PromptFragment](a.CAS, h)
		if err != nil {
			return artifact.LoopPreset{}, nil, artifact.Playbook{}, nil, artifact.EvalSuite{}, artifact.PolicyPack{}, fmt.Errorf("materials: prompt_fragment: %w", err)
		}
		frags = append(frags, v)
	}
	var pb artifact.Playbook
	if snap.Playbook != "" {
		v, _, err := artifact.Decode[artifact.Playbook](a.CAS, snap.Playbook)
		if err != nil {
			return artifact.LoopPreset{}, nil, artifact.Playbook{}, nil, artifact.EvalSuite{}, artifact.PolicyPack{}, fmt.Errorf("materials: playbook: %w", err)
		}
		pb = v
	}
	var skills []artifact.Skill
	for _, h := range snap.Skills {
		v, _, err := artifact.Decode[artifact.Skill](a.CAS, h)
		if err != nil {
			return artifact.LoopPreset{}, nil, artifact.Playbook{}, nil, artifact.EvalSuite{}, artifact.PolicyPack{}, fmt.Errorf("materials: skill: %w", err)
		}
		skills = append(skills, v)
	}
	var suite artifact.EvalSuite
	if snap.EvalSuite != "" {
		v, _, err := artifact.Decode[artifact.EvalSuite](a.CAS, snap.EvalSuite)
		if err != nil {
			return artifact.LoopPreset{}, nil, artifact.Playbook{}, nil, artifact.EvalSuite{}, artifact.PolicyPack{}, fmt.Errorf("materials: eval_suite: %w", err)
		}
		suite = v
		if suite.TaskDir == "" {
			suite.TaskDir = a.BundledEvals
		}
	}
	return loop, frags, pb, skills, suite, pol, nil
}
