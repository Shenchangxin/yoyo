package evolve

import (
	"context"
	"fmt"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
)

type Spend struct {
	TokensIn  int     `json:"tokens_in,omitempty"`
	TokensOut int     `json:"tokens_out,omitempty"`
	USD       float64 `json:"usd,omitempty"`
	WallMs    int64   `json:"wall_ms,omitempty"`
}

func AddSpend(s *Spend, r eval.RunReport) {
	if s == nil {
		return
	}
	s.TokensIn += r.TokensIn
	s.TokensOut += r.TokensOut
	s.USD += r.USD
	s.WallMs += r.WallMs
}

type LabCompare struct {
	BestOfN          []eval.RunReport `json:"best_of_n,omitempty"`
	IID              []Trial          `json:"iid,omitempty"`
	SCS              []Trial          `json:"scs,omitempty"`
	Spend            Spend            `json:"spend"`
	Snapshot         string           `json:"snapshot,omitempty"`
	ModelFingerprint string           `json:"model_fingerprint,omitempty"`
}

var iidTexts = []string{
	"Write the required output file before exploring further.",
	"If the same command fails twice, change strategy.",
	"Verify workspace artifacts exist before stopping.",
	"Do not retry identical tool invocations.",
	"Name the output path first, then implement.",
}

func IIDProposal(i int) Proposal {
	text := iidTexts[i%len(iidTexts)]
	id := fmt.Sprintf("iid-%d", i+1)
	return Proposal{
		ID:       id,
		Surface:  "prompt_fragment",
		Audit:    "iid random L1",
		Expected: "matched-budget random patch",
		Fragment: &artifact.PromptFragment{
			ID: id, Slot: "verification", Text: text, Surface: "prompt_fragment",
		},
	}
}

func (e *Engine) SearchBaselines(ctx context.Context, opts CycleOpts, n int) (LabCompare, error) {
	out := LabCompare{Snapshot: opts.Baseline.Hash, ModelFingerprint: opts.Baseline.Snap.ModelFingerprint}
	if n <= 0 {
		n = 3
	}
	if opts.Parent.Hash == "" {
		opts.Parent = opts.Baseline
	}
	start := time.Now()
	budgetHit := func() bool {
		if opts.MaxUSD > 0 && out.Spend.USD >= opts.MaxUSD {
			return true
		}
		if opts.MaxWall > 0 && time.Since(start) >= opts.MaxWall {
			return true
		}
		return false
	}
	var base eval.RunReport
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if budgetHit() {
			break
		}
		rep, err := e.evalMats(ctx, opts, opts.Baseline, fmt.Sprintf("-bon-%d", i))
		if err != nil {
			return out, err
		}
		if i == 0 {
			base = rep
		}
		out.BestOfN = append(out.BestOfN, rep)
		AddSpend(&out.Spend, rep)
	}
	held := heldSet(opts)
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if budgetHit() {
			break
		}
		p := IIDProposal(i)
		trial := e.patchTrial(ctx, opts, opts.Parent, base, p, held)
		out.IID = append(out.IID, trial)
		out.Spend.USD += trial.Spend.USD
		out.Spend.TokensIn += trial.Spend.TokensIn
		out.Spend.TokensOut += trial.Spend.TokensOut
		out.Spend.WallMs += trial.Spend.WallMs
	}
	acc := opts.Baseline
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if budgetHit() {
			break
		}
		p := IIDProposal(i + 17)
		p.ID = fmt.Sprintf("scs-%d", i+1)
		p.Audit = "scs sequential L1"
		if p.Fragment != nil {
			p.Fragment.ID = p.ID
		}
		trial := e.patchTrial(ctx, opts, acc, base, p, held)
		out.SCS = append(out.SCS, trial)
		out.Spend.USD += trial.Spend.USD
		out.Spend.TokensIn += trial.Spend.TokensIn
		out.Spend.TokensOut += trial.Spend.TokensOut
		out.Spend.WallMs += trial.Spend.WallMs
		if trial.Accepted && trial.Hash != "" {
			if mats, err := e.matsOfHash(trial.Hash, acc); err == nil {
				acc = mats
			}
		}
	}
	return out, nil
}

func (e *Engine) patchTrial(ctx context.Context, opts CycleOpts, parent MaterialSet, base eval.RunReport, p Proposal, held map[string]bool) Trial {
	candSnap, hash, err := ApplyProposal(e.CAS, parent.Snap, parent.Hash, p)
	if err != nil {
		return Trial{Proposal: p, Reason: err.Error()}
	}
	mats := matsFrom(parent, candSnap, hash, p)
	return e.trialMats(ctx, opts, opts.Baseline, base, mats, p, held)
}

func (e *Engine) matsOfHash(hash string, parent MaterialSet) (MaterialSet, error) {
	snap, err := e.CAS.GetSnapshot(hash)
	if err != nil {
		return MaterialSet{}, err
	}
	return MaterialsFromSnapshot(e.CAS, snap, hash, parent), nil
}

func heldSet(opts CycleOpts) map[string]bool {
	held := map[string]bool{}
	for _, id := range opts.HeldOut {
		held[id] = true
	}
	for _, id := range opts.Suite.HeldOut {
		held[id] = true
	}
	for _, id := range opts.Suite.Transfer {
		held[id] = true
	}
	for _, s := range eval.IndexTransfer {
		held[s.ID] = true
	}
	return held
}
