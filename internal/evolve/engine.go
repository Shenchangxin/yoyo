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
	Evidence  EvidenceBundle `json:"evidence"`
	Proposals []Proposal     `json:"proposals"`
	Tried     []Trial        `json:"tried"`
	Promoted  string         `json:"promoted,omitempty"`
}

type Trial struct {
	Proposal Proposal     `json:"proposal"`
	Hash     string       `json:"hash"`
	Metrics  eval.Metrics `json:"metrics"`
	Accepted bool         `json:"accepted"`
	Reason   string       `json:"reason"`
}

type CycleOpts struct {
	Base          artifact.HarnessSnapshot
	BaseHash      string
	Suite         artifact.EvalSuite
	Loop          artifact.LoopPreset
	Fragments     []artifact.PromptFragment
	Playbook      artifact.Playbook
	Skills        []artifact.Skill
	Client        runtime.Client
	Trace         *trace.Store
	SessionID     string
	Model         string
	FailedTasks   map[string]string
	K             int
	PromoteCanary bool
}

func (e *Engine) Cycle(ctx context.Context, opts CycleOpts) (CycleResult, error) {
	var res CycleResult
	events, _ := opts.Trace.Read(opts.SessionID)
	res.Evidence = Mine(events, opts.FailedTasks)
	props, err := Propose(ctx, opts.Client, opts.Model, res.Evidence, opts.K)
	if err != nil {
		return res, err
	}
	res.Proposals = props

	baseReport, err := e.Eval.Run(ctx, eval.RunOpts{
		Suite:     opts.Suite,
		Snapshot:  opts.Base,
		Hash:      opts.BaseHash,
		Client:    opts.Client,
		Loop:      opts.Loop,
		Fragments: opts.Fragments,
		Playbook:  opts.Playbook,
		Skills:    opts.Skills,
		Trace:     opts.Trace,
		SessionID: opts.SessionID + "-base",
		Model:     opts.Model,
	})
	if err != nil {
		return res, err
	}

	intent, err := e.Journal.Append("evolve.intent", map[string]any{"base": opts.BaseHash, "n": len(props)})
	if err != nil {
		return res, err
	}

	for _, p := range props {
		candSnap, hash, err := ApplyProposal(e.CAS, opts.Base, opts.BaseHash, p)
		if err != nil {
			res.Tried = append(res.Tried, Trial{Proposal: p, Reason: err.Error()})
			continue
		}
		frags := append([]artifact.PromptFragment{}, opts.Fragments...)
		pb := opts.Playbook
		skills := append([]artifact.Skill{}, opts.Skills...)
		if p.Fragment != nil {
			frags = append(frags, *p.Fragment)
		}
		if p.PlaybookBullet != nil {
			pb = pb.ApplyDelta([]artifact.PlaybookBullet{*p.PlaybookBullet}, nil)
		}
		if p.SkillMD != "" {
			if sk, err := artifact.ParseSkillMD(p.SkillMD, "evolve"); err == nil {
				skills = append(skills, sk)
			}
		}
		fiber, ferr := e.Kernel.Plugin("candidate:"+hash[:8], func(c *kernel.Context) error {
			return c.Provide("candidate:"+hash[:8], hash)
		})
		if ferr != nil {
			res.Tried = append(res.Tried, Trial{Proposal: p, Hash: hash, Reason: ferr.Error()})
			continue
		}
		report, rerr := e.Eval.Run(ctx, eval.RunOpts{
			Suite:     opts.Suite,
			Snapshot:  candSnap,
			Hash:      hash,
			Client:    opts.Client,
			Loop:      opts.Loop,
			Fragments: frags,
			Playbook:  pb,
			Skills:    skills,
			Trace:     opts.Trace,
			SessionID: opts.SessionID + "-cand-" + p.ID,
			Model:     opts.Model,
		})
		trial := Trial{Proposal: p, Hash: hash, Metrics: report.Metrics}
		if rerr != nil {
			trial.Reason = rerr.Error()
			_ = fiber.Dispose()
			res.Tried = append(res.Tried, trial)
			_ = e.Archive.Add(Node{ID: hash, Parent: opts.BaseHash, Snapshot: hash, ProposalID: p.ID, Metrics: report.Metrics, Note: trial.Reason})
			continue
		}
		ok, reason := eval.Promote(baseReport.Metrics, report.Metrics)
		trial.Accepted = ok
		trial.Reason = reason
		if !ok {
			_ = fiber.Dispose()
		} else {
			_ = e.Refs.Set(artifact.RefCanary, hash)
			if opts.PromoteCanary {
				_ = e.Refs.Set(artifact.RefActive, hash)
				if fp := opts.Base.ModelFingerprint; fp != "" {
					_ = e.Refs.Set(artifact.ModelActive(fp), hash)
				}
				res.Promoted = hash
			}
			_ = e.Refs.Archive(hash)
		}
		_ = e.Archive.Add(Node{
			ID: hash, Parent: opts.BaseHash, Snapshot: hash, ProposalID: p.ID,
			Metrics: report.Metrics, Accepted: ok, Canary: ok, Note: reason,
		})
		res.Tried = append(res.Tried, trial)
		if !ok {
			continue
		}
	}
	_ = e.Journal.Commit(intent.Seq)
	if res.Promoted == "" && len(res.Tried) == 0 {
		return res, fmt.Errorf("no proposals evaluated")
	}
	return res, nil
}
