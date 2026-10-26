// 实验室对照：同一基线上的 Best-of-N、独立随机补丁和顺序累积补丁。不移动 refs。Cycle 的用量累加也在这里。

package evolve

import (
	"context"
	"fmt"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
)

// Spend 累加一次或一轮的令牌、美元和墙钟毫秒。
type Spend struct {
	TokensIn  int     `json:"tokens_in,omitempty"`
	TokensOut int     `json:"tokens_out,omitempty"`
	USD       float64 `json:"usd,omitempty"`
	WallMs    int64   `json:"wall_ms,omitempty"`
}

// AddSpend 把一份运行报告的用量加进 Spend。s 为 nil 时直接返回。
// r 的 TokensIn、TokensOut、USD、WallMs 加进 s。
func AddSpend(s *Spend, r eval.RunReport) {
	if s == nil {
		return
	}
	s.TokensIn += r.TokensIn
	s.TokensOut += r.TokensOut
	s.USD += r.USD
	s.WallMs += r.WallMs
}

// LabCompare 是 SearchBaselines 的三组对照和总用量。
type LabCompare struct {
	BestOfN          []eval.RunReport `json:"best_of_n,omitempty"`
	IID              []Trial          `json:"iid,omitempty"`
	SCS              []Trial          `json:"scs,omitempty"`
	Spend            Spend            `json:"spend"`
	Snapshot         string           `json:"snapshot,omitempty"`
	ModelFingerprint string           `json:"model_fingerprint,omitempty"`
}

// iidTexts 是随机 L1 片段轮换的五句文案。
var iidTexts = []string{
	"Write the required output file before exploring further.",
	"If the same command fails twice, change strategy.",
	"Verify workspace artifacts exist before stopping.",
	"Do not retry identical tool invocations.",
	"Name the output path first, then implement.",
}

// IIDProposal 按序号从 iidTexts 取一句，做成 prompt_fragment。序号对表长取模。
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

// SearchBaselines 先把基线重复跑 n 次作为 Best-of-N，再对父材料做 n 次独立 IID，最后从基线出发做 n 次顺序 SCS。
// n 小于等于 0 时用 3。父哈希为空时父材料改成基线。MaxUSD 或 MaxWall 达到后停止后续轮次。SCS 只在试用被接受且哈希非空时把累积材料换成该哈希。
// ctx 的 Err 非空时返回已有结果和该错误。
// opts 的 Baseline 写入报告并作为 Best-of-N 的材料；Parent.Hash 为空时把 Parent 改成 Baseline；MaxUSD 大于 0 且 Spend.USD 已达到它，或 MaxWall 大于 0 且已用时间达到它，就不再开新一轮；整份 opts 传给 evalMats 和 patchTrial。
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

// patchTrial 把提议应用到父快照上，再按 trialMats 对基线报告做 Harbor 门。
// ctx 传给 trialMats。
// opts 的 Baseline 和整份 opts 传给 trialMats。
// parent 的 Snap 和 Hash 交给 ApplyProposal。
// base 传给 trialMats。
// held 传给 trialMats。
func (e *Engine) patchTrial(ctx context.Context, opts CycleOpts, parent MaterialSet, base eval.RunReport, p Proposal, held map[string]bool) Trial {
	candSnap, hash, err := ApplyProposal(e.CAS, parent.Snap, parent.Hash, p)
	if err != nil {
		return Trial{Proposal: p, Reason: err.Error()}
	}
	mats := matsFrom(parent, candSnap, hash, p)
	return e.trialMats(ctx, opts, opts.Baseline, base, mats, p, held)
}

// matsOfHash 从库里取出快照并解码成材料，解码失败的字段沿用 parent。
// hash 交给 CAS.GetSnapshot，成功后再交给 MaterialsFromSnapshot。
func (e *Engine) matsOfHash(hash string, parent MaterialSet) (MaterialSet, error) {
	snap, err := e.CAS.GetSnapshot(hash)
	if err != nil {
		return MaterialSet{}, err
	}
	return MaterialsFromSnapshot(e.CAS, snap, hash, parent), nil
}

// heldSet 收集调用方 HeldOut、套件 HeldOut、套件 Transfer，以及 IndexTransfer 的全部编号。
// opts 的 HeldOut、Suite.HeldOut、Suite.Transfer 放进集合，再加上 IndexTransfer 每个规格的 ID。
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
