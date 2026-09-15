package evolve

import (
	"context"
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/journal"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type Engine struct {
	CAS     *artifact.Store
	Refs    *artifact.Refs
	Eval    *eval.Engine
	Journal *journal.Log
	Archive *Archive
	Kernel  *kernel.Context
}

type CycleResult struct {
	Evidence       EvidenceBundle `json:"evidence"`
	Reflection     Reflection     `json:"reflection"`
	ParentSelected string         `json:"parent_selected,omitempty"`
	Proposals      []Proposal     `json:"proposals"`
	Tried          []Trial        `json:"tried"`
	Promoted       string         `json:"promoted,omitempty"`
}

type Trial struct {
	Proposal Proposal     `json:"proposal"`
	Hash     string       `json:"hash"`
	Metrics  eval.Metrics `json:"metrics"`
	Accepted bool         `json:"accepted"`
	Reason   string       `json:"reason"`
}

type CycleOpts struct {
	Parent        MaterialSet
	Baseline      MaterialSet
	Suite         artifact.EvalSuite
	Client        runtime.Client
	Trace         *trace.Store
	SessionID     string
	Model         string
	FailedTasks   map[string]string
	K             int
	PromoteCanary bool
	HeldOut       []string
	StagingHash   string
	IsolateRoot   string
}

func (e *Engine) Cycle(ctx context.Context, opts CycleOpts) (CycleResult, error) {
	var res CycleResult
	parent := opts.Parent
	baseline := opts.Baseline
	if baseline.Hash == "" {
		baseline = parent
	}
	if opts.K <= 0 {
		opts.K = 3
	}
	res.ParentSelected = parent.Hash
	if opts.IsolateRoot == "" {
		if root, stop, err := ScratchGit(); err == nil {
			defer stop()
			opts.IsolateRoot = root
		}
	}

	var events []trace.Event
	if opts.Trace != nil {
		events, _ = opts.Trace.Read(opts.SessionID)
	}
	held := map[string]bool{}
	for _, id := range opts.HeldOut {
		held[id] = true
	}
	if len(opts.HeldOut) == 0 {
		for _, id := range opts.Suite.HeldOut {
			held[id] = true
		}
	}
	safeFailed := map[string]string{}
	for k, v := range opts.FailedTasks {
		if held[k] {
			continue
		}
		safeFailed[k] = v
	}
	res.Evidence = Mine(events, safeFailed)
	res.Reflection = Reflect(events, safeFailed, parent.Playbook)
	curated := Curate(parent.Playbook, res.Reflection)
	props, err := Propose(ctx, opts.Client, opts.Model, res.Evidence, opts.K)
	if err != nil {
		return res, err
	}
	for _, b := range curated.Bullets {
		if playbookHas(parent.Playbook, b) {
			continue
		}
		bb := b
		props = append([]Proposal{{
			ID:             "ace-" + b.ID,
			Surface:        "playbook",
			Expected:       "ace curator delta",
			Risk:           "may overfit held-in tasks",
			Audit:          "ace curator incremental bullet",
			PlaybookBullet: &bb,
		}}, props...)
		break
	}
	res.Proposals = props

	baseReport, err := e.evalMats(ctx, opts, baseline, "-base")
	if err != nil {
		return res, err
	}

	intent, err := e.Journal.Append("evolve.intent", map[string]any{
		"base": parent.Hash, "baseline": baseline.Hash, "n": len(props),
	})
	if err != nil {
		return res, err
	}

	if opts.StagingHash != "" && opts.StagingHash != baseline.Hash {
		trial := e.trialSnapshot(ctx, opts, baseline, baseReport, opts.StagingHash, Proposal{
			ID:      "online-ace-stage",
			Surface: "playbook",
			Audit:   "online ACE staging (Harbor gate)",
		})
		res.Tried = append(res.Tried, trial)
		if trial.Accepted {
			res.Promoted = trial.Hash
			_ = e.Refs.Set(artifact.RefStaging, "")
		} else {
			_ = e.Refs.Set(artifact.RefStaging, "")
		}
	}

	for _, p := range props {
		candSnap, hash, err := ApplyProposal(e.CAS, parent.Snap, parent.Hash, p)
		if err != nil {
			res.Tried = append(res.Tried, Trial{Proposal: p, Reason: err.Error()})
			continue
		}
		mats := MaterialSet{
			Snap:      candSnap,
			Hash:      hash,
			Loop:      parent.Loop,
			Fragments: append([]artifact.PromptFragment{}, parent.Fragments...),
			Playbook:  parent.Playbook,
			Skills:    append([]artifact.Skill{}, parent.Skills...),
		}
		if p.Fragment != nil {
			mats.Fragments = append(mats.Fragments, *p.Fragment)
		}
		if p.PlaybookBullet != nil {
			mats.Playbook = mats.Playbook.ApplyDelta([]artifact.PlaybookBullet{*p.PlaybookBullet}, nil)
		}
		if p.SkillMD != "" {
			if sk, err := artifact.ParseSkillMD(p.SkillMD, "evolve"); err == nil {
				mats.Skills = append(mats.Skills, sk)
			}
		}
		trial := e.trialMats(ctx, opts, baseline, baseReport, mats, p)
		res.Tried = append(res.Tried, trial)
		if trial.Accepted {
			res.Promoted = hash
		}
	}
	_ = e.Journal.Commit(intent.Seq)
	if res.Promoted == "" && len(res.Tried) == 0 {
		return res, fmt.Errorf("no proposals evaluated")
	}
	return res, nil
}

func (e *Engine) evalMats(ctx context.Context, opts CycleOpts, mats MaterialSet, suffix string) (eval.RunReport, error) {
	return e.Eval.Run(ctx, eval.RunOpts{
		Suite:       opts.Suite,
		Snapshot:    mats.Snap,
		Hash:        mats.Hash,
		Client:      opts.Client,
		Loop:        mats.Loop,
		Fragments:   mats.Fragments,
		Playbook:    mats.Playbook,
		Skills:      mats.Skills,
		Trace:       opts.Trace,
		SessionID:   opts.SessionID + suffix,
		Model:       opts.Model,
		IsolateRoot: opts.IsolateRoot,
	})
}

func (e *Engine) trialSnapshot(ctx context.Context, opts CycleOpts, baseline MaterialSet, baseReport eval.RunReport, hash string, p Proposal) Trial {
	snap, err := e.CAS.GetSnapshot(hash)
	if err != nil {
		return Trial{Proposal: p, Hash: hash, Reason: err.Error()}
	}
	mats := MaterialsFromSnapshot(e.CAS, snap, hash, baseline)
	return e.trialMats(ctx, opts, baseline, baseReport, mats, p)
}

func (e *Engine) trialMats(ctx context.Context, opts CycleOpts, baseline MaterialSet, baseReport eval.RunReport, mats MaterialSet, p Proposal) Trial {
	fiber, ferr := e.Kernel.Plugin("candidate:"+mats.Hash[:min(8, len(mats.Hash))], func(c *kernel.Context) error {
		return c.Provide("candidate:"+mats.Hash[:min(8, len(mats.Hash))], mats.Hash)
	})
	trial := Trial{Proposal: p, Hash: mats.Hash}
	if ferr != nil {
		trial.Reason = ferr.Error()
		return trial
	}
	report, rerr := e.evalMats(ctx, opts, mats, "-cand-"+p.ID)
	trial.Metrics = report.Metrics
	if rerr != nil {
		trial.Reason = rerr.Error()
		_ = fiber.Dispose()
		_ = e.Archive.Add(Node{ID: mats.Hash, Parent: opts.Parent.Hash, Snapshot: mats.Hash, ProposalID: p.ID, Metrics: report.Metrics, Note: trial.Reason})
		return trial
	}
	ok, reason := eval.Promote(baseReport.Metrics, report.Metrics)
	trial.Accepted = ok
	trial.Reason = reason
	if !ok {
		_ = fiber.Dispose()
	} else {
		_ = e.Refs.Set(artifact.RefCanary, mats.Hash)
		if opts.PromoteCanary {
			_ = e.Refs.Set(artifact.RefActive, mats.Hash)
			if fp := baseline.Snap.ModelFingerprint; fp != "" {
				_ = e.Refs.Set(artifact.ModelActive(fp), mats.Hash)
			}
		}
		_ = e.Refs.Archive(mats.Hash)
	}
	_ = e.Archive.Add(Node{
		ID: mats.Hash, Parent: opts.Parent.Hash, Snapshot: mats.Hash, ProposalID: p.ID,
		Metrics: report.Metrics, Accepted: ok, Canary: ok, Note: reason,
	})
	return trial
}

func playbookHas(pb artifact.Playbook, b artifact.PlaybookBullet) bool {
	for _, x := range pb.Bullets {
		if x.ID == b.ID {
			return true
		}
		if x.Text != "" && b.Text != "" && x.Text == b.Text {
			return true
		}
	}
	return false
}
