// 把一份 L1 提议写进内容寻址库，或把多份正交提议合并到同一父快照。不跑 Harbor，也不移动 refs。quality.go 在写入前做准入。

package evolve

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	wasm "github.com/Shenchangxin/yoyo/internal/plugin/wasm"
)

// Candidate 是已经落到快照上的一条提议，以及它挂上的纤维。
type Candidate struct {
	Proposal Proposal
	Snapshot artifact.HarnessSnapshot
	Hash     string
	Fiber    *kernel.Fiber
}

// ApplyProposal 克隆父快照，把 Parent 和 Note 设成父哈希与审计说明，再写入唯一的 L1 载荷。载荷多于一种时 applyOne 返回错误。
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

// MergeProposals 把每条提议的 L1 载荷写到同一份父快照上。同一 surfaceKey 只保留先出现的那条。空列表返回 nothing to merge。Note 写成 merge-accepted。
func MergeProposals(cas *artifact.Store, base artifact.HarnessSnapshot, parent string, props []Proposal) (artifact.HarnessSnapshot, string, error) {
	next := artifact.CloneSnapshot(base)
	next.Parent = parent
	next.Note = "merge-accepted"
	if len(props) == 0 {
		return next, "", fmt.Errorf("nothing to merge")
	}
	seen := map[string]bool{}
	for _, p := range props {
		key := surfaceKey(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		if err := applyOne(cas, &next, p); err != nil {
			return next, "", err
		}
	}
	hash, err := cas.PutSnapshot(next)
	return next, hash, err
}

// applyOne 要求提议恰好一种 L1 载荷，并按片段、剧本、技能、指令槽或中间件写入。中间件的 MiddlewareN 小于等于 0 时用 2。指令槽空时用 verification。
func applyOne(cas *artifact.Store, next *artifact.HarnessSnapshot, p Proposal) error {
	if payloadCount(p) > 1 {
		return fmt.Errorf("multiple L1 surfaces")
	}
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
	case p.InstructionText != "":
		loop := artifact.LoopPreset{ID: "evolved"}
		if next.LoopPreset != "" {
			if v, _, err := artifact.Decode[artifact.LoopPreset](cas, next.LoopPreset); err == nil {
				loop = v
			}
		}
		loop = loop.SetInstructionSlot(or(p.InstructionSlot, "verification"), p.InstructionText)
		h, err := cas.Put(artifact.KindLoopPreset, loop.ID, loop)
		if err != nil {
			return err
		}
		next.LoopPreset = h
	case p.MiddlewareText != "":
		loop := artifact.LoopPreset{ID: "evolved"}
		if next.LoopPreset != "" {
			if v, _, err := artifact.Decode[artifact.LoopPreset](cas, next.LoopPreset); err == nil {
				loop = v
			}
		}
		n := p.MiddlewareN
		if n <= 0 {
			n = 2
		}
		loop.MaxRecentToolErrors = n
		loop.ToolErrorInstruction = p.MiddlewareText
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

// AdmitWASM 把一段 WASM 作为插件纤维装上，卸载时调用 host.Unload。export 传给 Load，燃料参数为 0。
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
