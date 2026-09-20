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

type Engine struct {
	CAS       *artifact.Store
	Refs      *artifact.Refs
	Eval      *eval.Engine
	Journal   *journal.Log
	Archive   *Archive
	Kernel    *kernel.Context
	TrialRoot string
}

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

type CycleOpts struct {
	Parent         MaterialSet
	Baseline       MaterialSet
	Suite          artifact.EvalSuite
	Client         runtime.Client
	Trace          *trace.Store
	SessionID      string
	Model          string
	FailedTasks    map[string]string
	K              int
	PromoteCanary  bool // deprecated: does not move refs/active
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

func (e *Engine) trialSnapshot(ctx context.Context, opts CycleOpts, baseline MaterialSet, baseReport eval.RunReport, hash string, p Proposal, held map[string]bool) Trial {
	snap, err := e.CAS.GetSnapshot(hash)
	if err != nil {
		return Trial{Proposal: p, Hash: hash, Reason: err.Error()}
	}
	mats := MaterialsFromSnapshot(e.CAS, snap, hash, baseline)
	return e.trialMats(ctx, opts, baseline, baseReport, mats, p, held)
}

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

func nodeFrom(parent, hash string, p Proposal, report eval.RunReport, trial Trial, ok bool) Node {
	return Node{
		ID: hash, Parent: parent, Snapshot: hash, ProposalID: p.ID,
		Metrics: report.Metrics, Accepted: ok, Canary: false, Note: trial.Reason,
		Surface: p.Surface, ManifestoHit: trial.ManifestoHit, ManifestoMiss: trial.ManifestoMiss,
	}
}

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

func uniqueSafety(ids []string, id string) []string {
	for _, x := range ids {
		if x == id {
			return ids
		}
	}
	return append(ids, id)
}

func lastNodes(nodes []Node, n int) []Node {
	if n <= 0 || len(nodes) == 0 {
		return nil
	}
	if len(nodes) <= n {
		return nodes
	}
	return nodes[len(nodes)-n:]
}

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

func trialScore(t Trial) int {
	return t.Metrics.HeldInPass + t.Metrics.HeldOutPass
}

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

type specTool struct{ s artifact.ToolSpec }

func (t specTool) Spec() artifact.ToolSpec { return t.s }
func (t specTool) Annotations() tool.Annotations {
	return tool.AnnFromSpec(t.s)
}
func (t specTool) Call(ctx context.Context, inv tool.Invocation) tool.Result {
	return tool.Result{Err: fmt.Errorf("candidate fiber does not execute host tools")}
}
