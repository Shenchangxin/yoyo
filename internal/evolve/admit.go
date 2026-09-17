package evolve

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	wasm "github.com/Shenchangxin/yoyo/internal/plugin/wasm"
)

type Candidate struct {
	Proposal Proposal
	Snapshot artifact.HarnessSnapshot
	Hash     string
	Fiber    *kernel.Fiber
}

func ApplyProposal(cas *artifact.Store, base artifact.HarnessSnapshot, parent string, p Proposal) (artifact.HarnessSnapshot, string, error) {
	next := artifact.CloneSnapshot(base)
	next.Parent = parent
	next.Note = p.Audit
	if err := applyOne(cas, &next, p); err != nil {
		return next, "", err
	}
	hash, err := cas.PutSnapshot(next)
	return next, hash, err
}

// MergeProposals applies every L1 payload onto the same parent snapshot.
func MergeProposals(cas *artifact.Store, base artifact.HarnessSnapshot, parent string, props []Proposal) (artifact.HarnessSnapshot, string, error) {
	next := artifact.CloneSnapshot(base)
	next.Parent = parent
	next.Note = "merge-accepted"
	if len(props) == 0 {
		return next, "", fmt.Errorf("nothing to merge")
	}
	for _, p := range props {
		if err := applyOne(cas, &next, p); err != nil {
			return next, "", err
		}
	}
	hash, err := cas.PutSnapshot(next)
	return next, hash, err
}

func applyOne(cas *artifact.Store, next *artifact.HarnessSnapshot, p Proposal) error {
	switch {
	case p.Fragment != nil:
		h, err := cas.Put(artifact.KindPromptFragment, p.Fragment.ID, *p.Fragment)
		if err != nil {
			return err
		}
		next.PromptFragments = append(next.PromptFragments, h)
	case p.PlaybookBullet != nil:
		pb := artifact.Playbook{ID: "main"}
		if next.Playbook != "" {
			existing, _, err := artifact.Decode[artifact.Playbook](cas, next.Playbook)
			if err != nil {
				return err
			}
			pb = existing
		}
		pb = pb.ApplyDelta([]artifact.PlaybookBullet{*p.PlaybookBullet}, nil)
		h, err := cas.Put(artifact.KindPlaybook, pb.ID, pb)
		if err != nil {
			return err
		}
		next.Playbook = h
	case p.SkillMD != "":
		sk, err := artifact.ParseSkillMD(p.SkillMD, "evolve")
		if err != nil {
			return err
		}
		h, err := cas.Put(artifact.KindSkill, sk.Name, sk)
		if err != nil {
			return err
		}
		next.Skills = append(next.Skills, h)
	case p.InstructionText != "" || p.MiddlewareText != "":
		loop := artifact.LoopPreset{ID: "evolved"}
		if next.LoopPreset != "" {
			if v, _, err := artifact.Decode[artifact.LoopPreset](cas, next.LoopPreset); err == nil {
				loop = v
			}
		}
		if p.InstructionText != "" {
			loop = loop.SetInstructionSlot(or(p.InstructionSlot, "verification"), p.InstructionText)
		}
		if p.MiddlewareText != "" {
			n := p.MiddlewareN
			if n <= 0 {
				n = 2
			}
			loop.MaxRecentToolErrors = n
			loop.ToolErrorInstruction = p.MiddlewareText
		}
		h, err := cas.Put(artifact.KindLoopPreset, loop.ID, loop)
		if err != nil {
			return err
		}
		next.LoopPreset = h
	default:
		return fmt.Errorf("proposal has no L1 payload")
	}
	return nil
}

func AdmitWASM(ctx *kernel.Context, host *wasm.Host, id string, bin []byte, export string) (*kernel.Fiber, error) {
	return ctx.Plugin("wasm:"+id, func(c *kernel.Context) error {
		return c.Effect(func() (func() error, error) {
			if _, err := host.Load(id, bin, export, 0); err != nil {
				return nil, err
			}
			return func() error { return host.Unload(id) }, nil
		})
	})
}
