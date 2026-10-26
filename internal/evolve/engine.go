// Package evolve 在 Harbor 门上提议并试用 L1 改动。Cycle 先把胜出哈希写到 refs/canary；只有 PromoteActive 为真才写 refs/active。
// Overlay 的预算由 OverlayBudget 给出：k 经 ClampOverlayK 限制在 OverlayK 到 OverlayMaxK，再返回 OverlayR×k 次 rollout 和 OverlayP×k 次改进会话。OverlayK 为 1，OverlayMaxK 为 4，OverlayR 为 4，OverlayP 为 3。
// Overlay 开始时 KeptBase 为真。结束时 assertRefsFrozen 对照进入前记下的 active、canary、staging。Overlay 不写这三枚引用。
package evolve

import (
	"context"
	"fmt"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/journal"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/tool"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Engine 串起内容寻址库、引用、评测、日志、档案和内核。TrialRoot 非空时试用会把非 held 题写到磁盘。
type Engine struct {
	CAS       *artifact.Store
	Refs      *artifact.Refs
	Eval      *eval.Engine
	Journal   *journal.Log
	Archive   *Archive
	Kernel    *kernel.Context
	TrialRoot string
}

// CycleResult 是一轮 Cycle 的证据、提议、试用和胜出哈希。ActiveMoved 只在 publishCanary 真的写了 refs/active 时为真。
type CycleResult struct {
	Evidence       EvidenceBundle `json:"evidence"`
	Reflection     Reflection     `json:"reflection"`
	ParentSelected string         `json:"parent_selected,omitempty"`
	Proposals      []Proposal     `json:"proposals"`
	Tried          []Trial        `json:"tried"`
	Promoted       string         `json:"promoted,omitempty"`
	Merged         bool           `json:"merged,omitempty"`
	ActiveMoved    bool           `json:"active_moved,omitempty"`
	Spend          Spend          `json:"spend"`
	Compare        *LabCompare    `json:"compare,omitempty"`
}

// Trial 是一条提议的 Harbor 结果。Accepted 来自 eval.Promote，再被宣言未命中时改成假。
type Trial struct {
	Proposal      Proposal     `json:"proposal"`
	Hash          string       `json:"hash"`
	Metrics       eval.Metrics `json:"metrics"`
	Accepted      bool         `json:"accepted"`
	Reason        string       `json:"reason"`
	ManifestoHit  int          `json:"manifesto_hit,omitempty"`
	ManifestoMiss int          `json:"manifesto_miss,omitempty"`
	Spend         Spend        `json:"spend,omitempty"`
}

// CycleOpts 是一轮演化的材料和预算。K 小于等于 0 时 Cycle 用 3。PromoteRepeats 小于等于 0 时用 2，并把套件 Repeats 抬到不低于它。
type CycleOpts struct {
	Parent      MaterialSet
	Baseline    MaterialSet
	Suite       artifact.EvalSuite
	Client      runtime.Client
	Trace       *trace.Store
	SessionID   string
	Model       string
	FailedTasks map[string]string
	K           int
	// PromoteCanary 已废弃。Cycle 不读它，也不会因此移动 refs/active。
	PromoteCanary bool
	// PromoteActive 为真时，publishCanary 在写完 canary 之后再写 refs/active。
	PromoteActive  bool
	HeldOut        []string
	StagingHash    string
	IsolateRoot    string
	PromoteRepeats int
	MaxUSD         float64
	MaxWall        time.Duration
	Behavior       bool
	IndexTransfer  bool
	USDPerMTok     float64
}

// Cycle 先评基线，再挖掘失败、反射、策展并提议。held 与安全题不进入失败表。每条提议先 AdmitQuality，再试用。
// 套件有 EvolveIn 时，挖掘用 EvolveIn 当 HeldIn 且清空 HeldOut 与 Transfer；胜出者再在原来的 HeldIn 上复评，复评不接受则清空胜出哈希。
// 预算用尽的提议记 reason 为 budget，不再评测。StagingHash 非空且不同于基线时，先试用该快照，然后把 refs/staging 写成空串。
// 胜出后 publishCanary 写 canary。只有 PromoteActive 为真才移动 active。
// ctx 传给 evalMats、ReflectLLM、Propose、trialSnapshot、trialMats 和 pickWinner。
// opts 读取 Parent、Baseline、K、PromoteRepeats、Suite、Behavior、IndexTransfer、IsolateRoot、Trace、Client、Model、FailedTasks、StagingHash、MaxUSD、MaxWall；K 小于等于 0 时改成 3，PromoteRepeats 小于等于 0 时改成 2，套件 Repeats 低于它时抬到它。
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
	if opts.PromoteRepeats <= 0 {
		opts.PromoteRepeats = 2
	}
	if opts.Suite.Repeats < opts.PromoteRepeats {
		opts.Suite.Repeats = opts.PromoteRepeats
	}
	opts.Suite.Safety = uniqueSafety(opts.Suite.Safety, "no-escape")
	if opts.Behavior {
		eval.ApplyBehavior(&opts.Suite)
	}
	if opts.IndexTransfer {
		eval.ApplyIndexTransfer(&opts.Suite)
	}
	res.ParentSelected = parent.Hash
	if opts.IsolateRoot == "" {
		if root, stop, err := ScratchGit(); err == nil {
			defer stop()
			opts.IsolateRoot = root
		}
	}

	held := heldSet(opts)
	mineOpts, promoteOpts := splitCycleSuites(opts)

	baseReport, err := e.evalMats(ctx, mineOpts, baseline, "-base")
	if err != nil {
		return res, err
	}
	AddSpend(&res.Spend, baseReport)

	var promoteBase eval.RunReport
	if promoteOpts.Suite.ID != mineOpts.Suite.ID || !sameIDs(promoteOpts.Suite.HeldIn, mineOpts.Suite.HeldIn) {
		promoteBase, err = e.evalMats(ctx, promoteOpts, baseline, "-promote-base")
		if err != nil {
			return res, err
		}
		AddSpend(&res.Spend, promoteBase)
	} else {
		promoteBase = baseReport
	}

	events := e.collectEvents(opts.Trace, baseReport)
	passed := map[string]bool{}
	safeFailed := map[string]string{}
	for _, r := range baseReport.Results {
		if r.Pass {
			passed[r.ID] = true
			continue
		}
		if held[r.ID] || r.Kind == "safety" {
			continue
		}
		safeFailed[r.ID] = r.Error
		if safeFailed[r.ID] == "" {
			safeFailed[r.ID] = "verifier_fail"
		}
	}
	for k, v := range opts.FailedTasks {
		if held[k] || passed[k] {
			continue
		}
		if _, ok := safeFailed[k]; ok {
			continue
		}
		safeFailed[k] = v
	}

	res.Evidence = Mine(events, safeFailed)
	outcomes := map[string]repeatOutcome{}
	for _, r := range baseReport.Results {
		if held[r.ID] || r.Kind == "safety" {
			continue
		}
		oc := repeatOutcome{PassCount: r.RepeatsPass, FailCount: r.Repeats - r.RepeatsPass}
		if r.Repeats <= 1 {
			if r.Pass {
				oc = repeatOutcome{PassCount: 1}
			} else {
				oc = repeatOutcome{FailCount: 1}
			}
		}
		outcomes[r.ID] = oc
		if r.Pass {
			res.Evidence.Passing = append(res.Evidence.Passing, PassSummary{TaskID: r.ID, Note: "held-in pass"})
		}
	}
	res.Evidence = AttachPairs(res.Evidence, events, outcomes, held)
	if e.Archive != nil {
		for _, n := range lastNodes(e.Archive.List(), 12) {
			res.Evidence.Prior = append(res.Evidence.Prior, PriorTrial{
				Hash: n.ID, Surface: n.Surface, Reason: n.Note, Accepted: n.Accepted,
			})
		}
	}

	res.Reflection = Reflect(events, safeFailed, parent.Playbook)
	if llm, lerr := ReflectLLM(ctx, opts.Client, opts.Model, events, safeFailed, parent.Playbook); lerr == nil {
		if len(llm.Insights) > 0 || len(llm.BulletTags) > 0 {
			res.Reflection = llm
		}
	}
	curated := Curate(parent.Playbook, res.Reflection)
	props, err := Propose(ctx, opts.Client, opts.Model, res.Evidence, opts.K)
	if err != nil {
		return res, err
	}
	heldInText := loadHeldInInstructions(e.Eval, mineOpts.Suite)
	for i := range props {
		props[i] = sanitizeHeldOut(props[i], held)
		if props[i].Surface == "" {
			props[i].Surface = "prompt_fragment"
		}
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

	intent, err := e.Journal.Append("evolve.intent", map[string]any{
		"base": parent.Hash, "baseline": baseline.Hash, "n": len(props),
	})
	if err != nil {
		return res, err
	}

	cycleStart := time.Now()
	overBudget := func() bool {
		if opts.MaxUSD > 0 && res.Spend.USD >= opts.MaxUSD {
			return true
		}
		if opts.MaxWall > 0 && time.Since(cycleStart) >= opts.MaxWall {
			return true
		}
		return false
	}

	var accepted []Trial
	if opts.StagingHash != "" && opts.StagingHash != baseline.Hash {
		trial := e.trialSnapshot(ctx, mineOpts, baseline, baseReport, opts.StagingHash, Proposal{
			ID:      "online-ace-stage",
			Surface: "playbook",
			Audit:   "online ACE staging (Harbor gate)",
		}, held)
		res.Tried = append(res.Tried, trial)
		if trial.Accepted {
			accepted = append(accepted, trial)
		}
		_ = e.Refs.Set(artifact.RefStaging, "")
	}

	for _, p := range props {
		if overBudget() {
			res.Tried = append(res.Tried, Trial{Proposal: p, Reason: "budget"})
			continue
		}
		if err := AdmitQuality(p, parent.Playbook, parent.Loop.PlaybookTokens, held, heldInText); err != nil {
			res.Tried = append(res.Tried, Trial{Proposal: p, Reason: err.Error()})
			continue
		}
		candSnap, hash, err := ApplyProposal(e.CAS, parent.Snap, parent.Hash, p)
		if err != nil {
			res.Tried = append(res.Tried, Trial{Proposal: p, Reason: err.Error()})
			continue
		}
		mats := matsFrom(parent, candSnap, hash, p)
		trial := e.trialMats(ctx, mineOpts, baseline, baseReport, mats, p, held)
		AddSpend(&res.Spend, eval.RunReport{TokensIn: trial.Spend.TokensIn, TokensOut: trial.Spend.TokensOut, USD: trial.Spend.USD, WallMs: trial.Spend.WallMs})
		res.Tried = append(res.Tried, trial)
		if trial.Accepted {
			accepted = append(accepted, trial)
		}
	}

	winner, merged := e.pickWinner(ctx, mineOpts, parent, baseline, baseReport, accepted, held)
	if winner.Hash != "" && !sameIDs(promoteOpts.Suite.HeldIn, mineOpts.Suite.HeldIn) {
		promoMats, err := e.matsOfHash(winner.Hash, parent)
		if err == nil {
			promo := e.trialMats(ctx, promoteOpts, baseline, promoteBase, promoMats, winner.Proposal, held)
			AddSpend(&res.Spend, eval.RunReport{TokensIn: promo.Spend.TokensIn, TokensOut: promo.Spend.TokensOut, USD: promo.Spend.USD, WallMs: promo.Spend.WallMs})
			winner.Accepted = promo.Accepted
			winner.Reason = promo.Reason
			winner.Metrics = promo.Metrics
			winner.ManifestoHit = promo.ManifestoHit
			winner.ManifestoMiss = promo.ManifestoMiss
			if !promo.Accepted {
				winner.Hash = ""
			}
		}
	}
	if winner.Hash != "" {
		res.Promoted = winner.Hash
		res.Merged = merged
		res.ActiveMoved = e.publishCanary(opts, baseline, winner.Hash)
	}
	_ = e.Journal.Commit(intent.Seq)
	return res, nil
}

// matsFrom 在父材料上叠一条提议。片段追加，剧本条目做增量，技能解析成功才追加。指令槽为空时用 verification。中间件次数小于等于 0 时用 2。
// parent 的 Loop、Fragments、Playbook、Skills 复制进结果。
// snap 写入结果的 Snap。
// hash 写入结果的 Hash。
// p 有 Fragment 就追加，有 PlaybookBullet 就 ApplyDelta，SkillMD 解析成功才追加技能，InstructionText 非空时 SetInstructionSlot，MiddlewareText 非空时写入中间件字段。
func matsFrom(parent MaterialSet, snap artifact.HarnessSnapshot, hash string, p Proposal) MaterialSet {
	mats := MaterialSet{
		Snap:      snap,
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
	if p.InstructionText != "" {
		mats.Loop = mats.Loop.SetInstructionSlot(or(p.InstructionSlot, "verification"), p.InstructionText)
	}
	if p.MiddlewareText != "" {
		n := p.MiddlewareN
		if n <= 0 {
			n = 2
		}
		mats.Loop.MaxRecentToolErrors = n
		mats.Loop.ToolErrorInstruction = p.MiddlewareText
	}
	return mats
}

// pickWinner 在没有接受项时返回空试用。只有一条时直接返回它，合并标志为假。
// 多条时去掉 online-ace-stage，再按表面各留得分最高的一条。至少两条才合并；合并试用被接受则返回它且合并标志为真，否则退回得分最高的单条。
// ctx 在合并评测时传给 trialMats。
// opts 传给 trialMats。
// parent 的 Snap 和 Hash 交给 MergeProposals，解码材料时作为 fallback。
// baseline 传给 trialMats。
// baseReport 传给 trialMats。
// accepted 长度为 0 时返回空试用和 false，长度为 1 时返回那一条和 false；更多时跳过 ID 为 online-ace-stage 的提议。
// held 传给 trialMats。
func (e *Engine) pickWinner(ctx context.Context, opts CycleOpts, parent, baseline MaterialSet, baseReport eval.RunReport, accepted []Trial, held map[string]bool) (Trial, bool) {
	if len(accepted) == 0 {
		return Trial{}, false
	}
	if len(accepted) == 1 {
		return accepted[0], false
	}
	var props []Proposal
	for _, t := range accepted {
		if t.Proposal.ID == "online-ace-stage" {
			continue
		}
		props = append(props, t.Proposal)
	}
	if len(props) < 2 {
		return bestTrial(accepted), false
	}
	props = exclusiveSurfaces(accepted)
	if len(props) < 2 {
		return bestTrial(accepted), false
	}
	snap, hash, err := MergeProposals(e.CAS, parent.Snap, parent.Hash, props)
	if err != nil {
		return bestTrial(accepted), false
	}
	mats := MaterialsFromSnapshot(e.CAS, snap, hash, parent)
	trial := e.trialMats(ctx, opts, baseline, baseReport, mats, Proposal{
		ID:      "merge-accepted",
		Surface: "merge",
		Audit:   "merged all Harbor-accepted L1 deltas",
	}, held)
	if trial.Accepted {
		return trial, true
	}
	return bestTrial(accepted), false
}

// bestTrial 取得分最高的试用。得分是 HeldInPass 加 HeldOutPass。相等时保留先出现的那条。
// ts 以第一条为起点，其后每条用 HeldInPass 加 HeldOutPass，更大才替换。
func bestTrial(ts []Trial) Trial {
	best := ts[0]
	bestScore := best.Metrics.HeldInPass + best.Metrics.HeldOutPass
	for _, t := range ts[1:] {
		s := t.Metrics.HeldInPass + t.Metrics.HeldOutPass
		if s > bestScore {
			best, bestScore = t, s
		}
	}
	return best
}

// publishCanary 把哈希写到 refs/canary，模型指纹非空时再写该指纹的 canary，并归档。PromoteActive 为假时返回 false，不写 active。为真时再写 refs/active 和指纹的 active，并返回 true。
// opts 只看 PromoteActive，为假时不写 active 并返回 false。
// baseline 只读 Snap.ModelFingerprint，非空时用来写该指纹的 canary，PromoteActive 为真时再写 active。
// hash 写到 refs/canary 并交给 Archive；PromoteActive 为真时再写到 refs/active。
func (e *Engine) publishCanary(opts CycleOpts, baseline MaterialSet, hash string) bool {
	_ = e.Refs.Set(artifact.RefCanary, hash)
	if fp := baseline.Snap.ModelFingerprint; fp != "" {
		_ = e.Refs.Set(artifact.ModelCanary(fp), hash)
	}
	_ = e.Refs.Archive(hash)
	if !opts.PromoteActive {
		return false
	}
	_ = e.Refs.Set(artifact.RefActive, hash)
	if fp := baseline.Snap.ModelFingerprint; fp != "" {
		_ = e.Refs.Set(artifact.ModelActive(fp), hash)
	}
	return true
}

// evalMats 用材料调用 eval.Run。AllowLLMCompact 由 eval 的 runTask 在副本上写成 false，这里不改循环字段。
// ctx 传给 Eval.Run。
// opts 的 USDPerMTok 写入 Meter；Suite、Client、Trace、SessionID、Model、IsolateRoot 写入 RunOpts。
// mats 的 Snap、Hash、Loop、Fragments、Playbook、Skills 写入 RunOpts。
// suffix 接在 opts.SessionID 后面，作为 RunOpts 的 SessionID。
func (e *Engine) evalMats(ctx context.Context, opts CycleOpts, mats MaterialSet, suffix string) (eval.RunReport, error) {
	meter := &runtime.Meter{USDPerMTok: opts.USDPerMTok}
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
		Meter:       meter,
	})
}

// trialSnapshot 按哈希取出快照再试用。取快照失败时原因是该错误，不评测。
// ctx 传给 trialMats。
// opts 传给 trialMats。
// baseline 在取到快照后作为 MaterialsFromSnapshot 的 fallback，并传给 trialMats。
// baseReport 传给 trialMats。
// hash 交给 CAS.GetSnapshot；失败时写进返回试用的 Hash 和 Reason。
// held 传给 trialMats。
func (e *Engine) trialSnapshot(ctx context.Context, opts CycleOpts, baseline MaterialSet, baseReport eval.RunReport, hash string, p Proposal, held map[string]bool) Trial {
	snap, err := e.CAS.GetSnapshot(hash)
	if err != nil {
		return Trial{Proposal: p, Hash: hash, Reason: err.Error()}
	}
	mats := MaterialsFromSnapshot(e.CAS, snap, hash, baseline)
	return e.trialMats(ctx, opts, baseline, baseReport, mats, p, held)
}

// trialMats 注册一条不执行宿主工具的候选纤维，再跑评测。通过与否先看 eval.Promote。Promote 通过但 PredictedFixes 非空且命中数为 0 时改成拒绝，原因是 manifesto_miss。
// 未接受或评测出错时卸下纤维。无论是否接受都写入档案。
// ctx 传给 evalMats。
// opts 传给 evalMats；Parent.Hash 交给 nodeFrom。
// baseline 本函数不读取。
// baseReport 的 Metrics 作为 eval.Promote 的第一参。
// mats 的 Hash 用来拼纤维名和试用哈希，Snap.Tools、Skills 放进纤维，并传给 evalMats。
// p 写入试用的 Proposal，ID 接在会话后缀 -cand- 后面，并交给 scoreManifesto、logTrial 和 nodeFrom。
// held 交给 scoreManifesto 和 logTrial。
func (e *Engine) trialMats(ctx context.Context, opts CycleOpts, baseline MaterialSet, baseReport eval.RunReport, mats MaterialSet, p Proposal, held map[string]bool) Trial {
	short := mats.Hash
	if len(short) > 8 {
		short = short[:8]
	}
	fiber, ferr := e.Kernel.Plugin("candidate:"+short, func(c *kernel.Context) error {
		reg := tool.NewRegistry()
		want := map[string]bool{}
		for _, n := range mats.Snap.Tools {
			want[n] = true
		}
		for _, s := range tool.HostSpecs() {
			if len(want) > 0 && !want[s.Name] {
				continue
			}
			spec := s
			reg.Register(specTool{spec})
		}
		if err := c.Provide("candidate:"+short, mats.Hash); err != nil {
			return err
		}
		if err := c.Provide("candidate:"+short+":tools", reg); err != nil {
			return err
		}
		return c.Provide("candidate:"+short+":skills", mats.Skills)
	})
	trial := Trial{Proposal: p, Hash: mats.Hash}
	if ferr != nil {
		trial.Reason = ferr.Error()
		return trial
	}
	report, rerr := e.evalMats(ctx, opts, mats, "-cand-"+p.ID)
	trial.Metrics = report.Metrics
	trial.Spend = Spend{TokensIn: report.TokensIn, TokensOut: report.TokensOut, USD: report.USD, WallMs: report.WallMs}
	hit, miss := scoreManifesto(p, report, held)
	trial.ManifestoHit, trial.ManifestoMiss = hit, miss
	e.logTrial(mats.Hash, report, p, held)
	if rerr != nil {
		trial.Reason = rerr.Error()
		_ = fiber.Dispose()
		_ = e.Archive.Add(nodeFrom(opts.Parent.Hash, mats.Hash, p, report, trial, false))
		return trial
	}
	ok, reason := eval.Promote(baseReport.Metrics, report.Metrics)
	if ok && len(p.PredictedFixes) > 0 && hit == 0 {
		ok = false
		reason = "manifesto_miss"
	}
	trial.Accepted = ok
	trial.Reason = reason
	if !ok {
		_ = fiber.Dispose()
	}
	_ = e.Archive.Add(nodeFrom(opts.Parent.Hash, mats.Hash, p, report, trial, ok))
	return trial
}

// nodeFrom 用试用和报告拼一条档案节点。Canary 固定写成 false。
// parent 写入节点的 Parent。
// hash 同时写入 ID 和 Snapshot。
// p 的 ID 写入 ProposalID，Surface 写入 Surface。
// report 的 Metrics 写入节点。
// trial 的 Reason 写入 Note，ManifestoHit 和 ManifestoMiss 原样写入。
// ok 写入 Accepted。
func nodeFrom(parent, hash string, p Proposal, report eval.RunReport, trial Trial, ok bool) Node {
	return Node{
		ID: hash, Parent: parent, Snapshot: hash, ProposalID: p.ID,
		Metrics: report.Metrics, Accepted: ok, Canary: false, Note: trial.Reason,
		Surface: p.Surface, ManifestoHit: trial.ManifestoHit, ManifestoMiss: trial.ManifestoMiss,
	}
}

// scoreManifesto 数 PredictedFixes。held 里的编号跳过。结果里该题通过则命中加一，否则未命中加一。
// p 的 PredictedFixes 逐个编号，在 held 里的跳过。
// report 的 Results 按 ID 记下 Pass，用来区分命中和未命中。
func scoreManifesto(p Proposal, report eval.RunReport, held map[string]bool) (hit, miss int) {
	byID := map[string]bool{}
	for _, r := range report.Results {
		byID[r.ID] = r.Pass
	}
	for _, id := range p.PredictedFixes {
		if held[id] {
			continue
		}
		if byID[id] {
			hit++
		} else {
			miss++
		}
	}
	return hit, miss
}

// playbookHas 在编号相同，或两边正文都非空且相同时为真。
// pb 的 Bullets 与另一条比较 ID，或在两边 Text 都非空时比较 Text。
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

// uniqueSafety 在编号尚未出现时追加。
// ids 里已有与 id 相同的项时原样返回，否则把 id 追加到末尾。
// id 与 ids 的每一项比较。
func uniqueSafety(ids []string, id string) []string {
	for _, x := range ids {
		if x == id {
			return ids
		}
	}
	return append(ids, id)
}

// lastNodes 返回末尾 n 条。n 小于等于 0 或列表为空时返回 nil。不足 n 条时返回全部。
// nodes 为空时返回 nil；长度不超过 n 时整份返回，否则返回末尾 n 条。
func lastNodes(nodes []Node, n int) []Node {
	if n <= 0 || len(nodes) == 0 {
		return nil
	}
	if len(nodes) <= n {
		return nodes
	}
	return nodes[len(nodes)-n:]
}

// exclusiveSurfaces 跳过 online-ace-stage，每个 surfaceKey 只留 trialScore 更高的一条，顺序按该表面第一次出现。
// accepted 里 ID 为 online-ace-stage 的跳过，其余按 surfaceKey 分组，只留 trialScore 更高的提议。
func exclusiveSurfaces(accepted []Trial) []Proposal {
	best := map[string]Trial{}
	var order []string
	for _, t := range accepted {
		if t.Proposal.ID == "online-ace-stage" {
			continue
		}
		s := surfaceKey(t.Proposal)
		prev, ok := best[s]
		if !ok {
			order = append(order, s)
			best[s] = t
			continue
		}
		if trialScore(t) > trialScore(prev) {
			best[s] = t
		}
	}
	var props []Proposal
	for _, s := range order {
		props = append(props, best[s].Proposal)
	}
	return props
}

// trialScore 是 HeldInPass 与 HeldOutPass 之和。
func trialScore(t Trial) int {
	return t.Metrics.HeldInPass + t.Metrics.HeldOutPass
}

// splitCycleSuites 在 EvolveIn 为空时两份都是原选项。否则挖掘套件的 HeldIn 换成 EvolveIn，HeldOut 与 Transfer 清空；晋升套件保留原来的 HeldIn。
// opts 在 Suite.EvolveIn 为空时原样作为两份返回；否则挖掘份的 HeldIn 换成 EvolveIn 并清空 HeldOut 与 Transfer，晋升份的 HeldIn 复制原来的 HeldIn。
func splitCycleSuites(opts CycleOpts) (mine, promote CycleOpts) {
	mine, promote = opts, opts
	if len(opts.Suite.EvolveIn) == 0 {
		return opts, opts
	}
	ms, ps := opts.Suite, opts.Suite
	ms.HeldIn = append([]string{}, opts.Suite.EvolveIn...)
	ms.HeldOut = nil
	ms.Transfer = nil
	ps.HeldIn = append([]string{}, opts.Suite.HeldIn...)
	mine.Suite = ms
	promote.Suite = ps
	return mine, promote
}

// sameIDs 要求长度相同且逐项相等。
// b 与 a 长度不同则返回 false，否则逐项和 a 比较。
func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// collectEvents 按结果里的尝试会话读轨迹。同一会话只读一次。存储为 nil 时返回 nil。
// st 为 nil 时返回 nil；否则对每个未读过的会话调用 Read。
// report 的 Results 里每个 Attempt，SessionID 非空且没读过时用来 Read。
func (e *Engine) collectEvents(st *trace.Store, report eval.RunReport) []trace.Event {
	if st == nil {
		return nil
	}
	var events []trace.Event
	seen := map[string]bool{}
	for _, r := range report.Results {
		for _, a := range r.Attempts {
			if a.SessionID == "" || seen[a.SessionID] {
				continue
			}
			seen[a.SessionID] = true
			more, _ := st.Read(a.SessionID)
			events = append(events, more...)
		}
	}
	return events
}

// loadHeldInInstructions 读挖掘用的题面。EvolveIn 非空时用它，否则用 HeldIn。引擎为 nil 或读空时跳过。
// eng 为 nil 时返回 nil；否则对每个题号调用 ReadInstruction。
// suite 的 EvolveIn 长度大于 0 时用它作为题号，否则用 HeldIn；ReadInstruction 也收到这份 suite。
func loadHeldInInstructions(eng *eval.Engine, suite artifact.EvalSuite) []string {
	if eng == nil {
		return nil
	}
	ids := suite.HeldIn
	if len(suite.EvolveIn) > 0 {
		ids = suite.EvolveIn
	}
	var out []string
	for _, id := range ids {
		if s := eng.ReadInstruction(suite, id); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// specTool 把工具声明包成注册表项。Call 不执行宿主工具。
type specTool struct{ s artifact.ToolSpec }

// Spec 返回包进去的工具声明。
func (t specTool) Spec() artifact.ToolSpec { return t.s }

// Annotations 由声明推导注解。
func (t specTool) Annotations() tool.Annotations {
	return tool.AnnFromSpec(t.s)
}

// Call 固定返回候选纤维不执行宿主工具的错误。
// ctx 本函数不读取。
// inv 本函数不读取。
func (t specTool) Call(ctx context.Context, inv tool.Invocation) tool.Result {
	return tool.Result{Err: fmt.Errorf("candidate fiber does not execute host tools")}
}
