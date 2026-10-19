package evolve

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

const (
	OverlayK             = 1
	OverlayMaxK          = 4
	OverlayR             = 4
	OverlayP             = 3
	OverlayObsFloor      = 10_000
	overlayStageAttempts = 2
)

// OverlayBudget is the Harness² identity: R=4k solver rollouts, P=3k
// improver sessions. k=1 is four runs and three edits.
func OverlayBudget(k int) (r, p int) {
	k = ClampOverlayK(k)
	return 4 * k, 3 * k
}

func ClampOverlayK(k int) int {
	if k < 1 {
		return 1
	}
	if k > OverlayMaxK {
		return OverlayMaxK
	}
	return k
}

// OverlayRollout is one solver execution the overlay method is allowed to see.
// It never carries a Harbor score.
type OverlayRollout struct {
	Label       string  `json:"label"`
	Delivered   bool    `json:"delivered"`
	Trajectory  string  `json:"trajectory,omitempty"`
	Deliverable string  `json:"deliverable,omitempty"`
	Cached      bool    `json:"cached,omitempty"`
	USD         float64 `json:"usd,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// OverlayRoller runs one isolated probe. The engine never writes global refs.
type OverlayRoller func(ctx context.Context, label string, mats MaterialSet) (OverlayRollout, error)

type overlayLedgerEntry struct {
	Kind     string
	Label    string
	Cached   bool
	USD      float64
	Attempts int
}

// OverlayLedger is the matched budget. Identity is R=4k, P=3k.
type OverlayLedger struct {
	K       int
	entries []overlayLedgerEntry
}

func (l *OverlayLedger) rollout(label string, usd float64, cached bool) {
	l.entries = append(l.entries, overlayLedgerEntry{Kind: "rollout", Label: label, Cached: cached, USD: usd})
}

func (l *OverlayLedger) session(label string, usd float64, attempts int) {
	if attempts < 1 {
		attempts = 1
	}
	l.entries = append(l.entries, overlayLedgerEntry{Kind: "session", Label: label, USD: usd, Attempts: attempts})
}

func (l OverlayLedger) SessionAttempts() int {
	n := 0
	for _, e := range l.entries {
		if e.Kind != "session" {
			continue
		}
		if e.Attempts < 1 {
			n++
			continue
		}
		n += e.Attempts
	}
	return n
}

func (l OverlayLedger) R() int {
	n := 0
	for _, e := range l.entries {
		if e.Kind == "rollout" {
			n++
		}
	}
	return n
}

func (l OverlayLedger) P() int {
	n := 0
	for _, e := range l.entries {
		if e.Kind == "session" {
			n++
		}
	}
	return n
}

func (l OverlayLedger) labels(kind string) []string {
	var out []string
	for _, e := range l.entries {
		if e.Kind == kind {
			out = append(out, e.Label)
		}
	}
	return out
}

func (l OverlayLedger) View() OverlayLedgerView {
	return OverlayLedgerView{
		K:               ClampOverlayK(l.K),
		R:               l.R(),
		P:               l.P(),
		SessionAttempts: l.SessionAttempts(),
		Rollouts:        l.labels("rollout"),
		Sessions:        l.labels("session"),
	}
}

func (l OverlayLedger) IdentityOK() error {
	k := ClampOverlayK(l.K)
	wantR, wantP := OverlayBudget(k)
	rl, sl := l.labels("rollout"), l.labels("session")
	var dupes []string
	seen := map[string]int{}
	for _, label := range append(append([]string{}, rl...), sl...) {
		seen[label]++
	}
	// Duplicates are checked within kind, because inv1 can be both a session
	// and a rollout in Harness². Yoyo k=1 uses distinct session labels.
	count := map[string]int{}
	for _, e := range l.entries {
		key := e.Kind + ":" + e.Label
		count[key]++
		if count[key] == 2 {
			dupes = append(dupes, e.Label)
		}
	}
	if len(dupes) == 0 && l.R() == wantR && l.P() == wantP {
		return nil
	}
	return fmt.Errorf("overlay k=%d R=%d want %d P=%d want %d rollouts=%v sessions=%v dupes=%v",
		k, l.R(), wantR, l.P(), wantP, rl, sl, dupes)
}

// OverlayLedgerView is the operator-visible budget row (JSON).
type OverlayLedgerView struct {
	K               int      `json:"k"`
	R               int      `json:"r"`
	P               int      `json:"p"`
	SessionAttempts int      `json:"session_attempts,omitempty"`
	Rollouts        []string `json:"rollouts,omitempty"`
	Sessions        []string `json:"sessions,omitempty"`
}

// OverlayPair is composer diversity: pairwise hash distance, no extra model call.
type OverlayPair struct {
	A      string `json:"a"`
	B      string `json:"b"`
	Equal  bool   `json:"equal"`
	Prefix int    `json:"prefix"`
}

type OverlayOpts struct {
	Parent          MaterialSet
	Task            string
	Client          runtime.Client
	Model           string
	SessionID       string
	TaskID          string
	Domain          string
	K               int
	Mode            string
	ReuseBase       *OverlayRollout
	Roller          OverlayRoller
	HeldOut         []string
	HeldInText      []string
	MaxUSD          float64
	MaxWall         time.Duration
	Library         *Library
	LibraryText     string
	Scratch         string
	KeepEvidence    bool
	ParallelPropose bool
	Tree            *OverlayTree
}

type OverlayResult struct {
	Adopted         bool              `json:"adopted"`
	KeptBase        bool              `json:"kept_base"`
	Hash            string            `json:"hash,omitempty"`
	Parent          string            `json:"parent,omitempty"`
	Reason          string            `json:"reason,omitempty"`
	INV             Proposal          `json:"inv,omitempty"`
	DOM             Proposal          `json:"dom,omitempty"`
	Compose         string            `json:"compose,omitempty"`
	Rollouts        []OverlayRollout  `json:"rollouts,omitempty"`
	Ledger          OverlayLedger     `json:"-"`
	Budget          OverlayLedgerView `json:"ledger,omitempty"`
	R               int               `json:"r,omitempty"`
	P               int               `json:"p,omitempty"`
	K               int               `json:"k,omitempty"`
	Spend           Spend             `json:"spend,omitempty"`
	Evidence        []string          `json:"evidence,omitempty"`
	FellBack        []string          `json:"fell_back,omitempty"`
	Domain          string            `json:"domain,omitempty"`
	LibraryStatus   string            `json:"library,omitempty"`
	Experience      *Experience       `json:"experience,omitempty"`
	Mode            string            `json:"mode,omitempty"`
	Selection       OverlaySelection  `json:"selection,omitempty"`
	SessionAttempts int               `json:"session_attempts,omitempty"`
	Diversity       []OverlayPair     `json:"diversity,omitempty"`
}

func (e *Engine) Overlay(ctx context.Context, opts OverlayOpts) (OverlayResult, error) {
	var res OverlayResult
	if e == nil || e.CAS == nil {
		return res, fmt.Errorf("overlay: cas required")
	}
	if opts.Roller == nil {
		return res, fmt.Errorf("overlay: roller required")
	}
	if opts.Parent.Hash == "" {
		return res, fmt.Errorf("overlay: parent hash required")
	}
	k := ClampOverlayK(opts.K)
	mode := ClampOverlayMode(opts.Mode)
	parent := opts.Parent
	res.Parent = parent.Hash
	res.Hash = parent.Hash
	res.KeptBase = true
	res.K = k
	res.Mode = mode
	res.Domain = DomainSlug(opts.Domain)
	res.LibraryStatus = LibraryNoEntry
	res.Ledger.K = k
	held := map[string]bool{}
	for _, id := range opts.HeldOut {
		if id != "" {
			held[id] = true
		}
	}
	before := snapshotGlobalRefs(e.Refs)
	var intentSeq uint64
	if e.Journal != nil {
		rec, err := e.Journal.Append("ppc.overlay.intent", map[string]any{
			"session": opts.SessionID, "parent": parent.Hash, "reuse_turn": opts.ReuseBase != nil,
			"k": k, "mode": mode, "domain": res.Domain,
		})
		if err != nil {
			return res, err
		}
		intentSeq = rec.Seq
	}

	start := time.Now()
	overBudget := func() bool {
		if opts.MaxUSD > 0 && res.Spend.USD >= opts.MaxUSD {
			return true
		}
		return opts.MaxWall > 0 && time.Since(start) >= opts.MaxWall
	}

	scratch := strings.TrimSpace(opts.Scratch)
	if scratch == "" {
		dir, err := os.MkdirTemp("", "yoyo-overlay-")
		if err != nil {
			return overlayAbort(e, res, before, intentSeq, err)
		}
		scratch = dir
		if !opts.KeepEvidence {
			defer os.RemoveAll(scratch)
		}
	}
	tree := newOverlayTree(scratch, held)
	opts.Tree = tree
	tree.writeTask(opts.Task)
	tree.writeHarness(parent)
	tree.writeLibrary(opts.Library, e.CAS)

	bases, err := overlayBases(ctx, opts, parent, k, &res)
	if err != nil {
		return overlayAbort(e, res, before, intentSeq, err)
	}
	joined := joinOverlayBases(bases)

	var invs, doms []overlayCand
	var finals []overlayFinal
	if mode == OverlayModeSeq {
		invs, doms, finals, err = e.overlaySeq(ctx, opts, parent, k, joined, held, overBudget, &res)
	} else {
		invs, doms, finals, err = e.overlayAgg(ctx, opts, parent, k, joined, held, overBudget, &res)
	}
	if err != nil {
		return overlayAbort(e, res, before, intentSeq, err)
	}
	if len(invs) > 0 {
		res.INV = invs[0].prop
	}
	if len(doms) > 0 {
		res.DOM = doms[0].prop
	}
	cRolls := overlayPickRolls(res.Rollouts, k, "c")
	sel := overlayPickFinal(ctx, opts, parent, joined, finals, cRolls, held)
	res.Selection = sel
	res.Diversity = overlayComposeDiversity(finals)
	if sel.Why != "" && opts.Tree != nil {
		_ = opts.Tree.write("selection/choice.md", stripHeld(sel.Why, held)+"\nchose "+sel.ChosenAnon)
	}

	win := -1
	for i := range finals {
		if finals[i].label == sel.ChosenLabel {
			win = i
			break
		}
	}
	if win >= 0 && !sel.ChosenBase && finals[win].adopted && finals[win].succ.Hash != parent.Hash && win < len(cRolls) && cRolls[win].Delivered {
		res.Adopted = true
		res.KeptBase = false
		res.Hash = finals[win].succ.Hash
		res.Compose = finals[win].kind
		res.Reason = finals[win].notes
		if sel.Why != "" {
			res.Reason = sel.Why
		}
		res.INV = invs[win].prop
		res.DOM = doms[win].prop
	} else {
		res.Adopted = false
		res.KeptBase = true
		res.Hash = parent.Hash
		res.Compose = "keep_base"
		if res.Reason == "" {
			if k == 1 && sel.Why == "no candidate produced an artifact" {
				res.Reason = "c1 did not deliver"
			} else {
				res.Reason = "keep_base"
			}
		}
	}

	res.Spend.WallMs = time.Since(start).Milliseconds()
	if err := res.Ledger.IdentityOK(); err != nil {
		res.Adopted = false
		res.KeptBase = true
		res.Hash = parent.Hash
		res.LibraryStatus = LibraryNoEntry
		return overlayAbort(e, res, before, intentSeq, err)
	}
	if err := assertRefsFrozen(e.Refs, before); err != nil {
		res.Adopted = false
		res.KeptBase = true
		res.Hash = parent.Hash
		return overlayAbort(e, res, before, intentSeq, err)
	}
	res.R = res.Ledger.R()
	res.P = res.Ledger.P()
	res.SessionAttempts = res.Ledger.SessionAttempts()
	res.Budget = res.Ledger.View()
	if opts.Tree != nil {
		res.Evidence = opts.Tree.evidence()
	}
	e.overlayCommitLibrary(ctx, opts, &res, held)
	if e.Journal != nil {
		_, _ = e.Journal.Append("ppc.overlay.commit", map[string]any{
			"session": opts.SessionID, "adopted": res.Adopted, "hash": res.Hash,
			"kept_base": res.KeptBase, "R": res.R, "P": res.P, "k": res.K,
			"usd": res.Spend.USD, "library": res.LibraryStatus, "domain": res.Domain,
			"mode": res.Mode, "chosen": sel.ChosenLabel, "chosen_base": sel.ChosenBase,
			"session_attempts": res.SessionAttempts, "id_map": sel.IDMap,
			"diversity": res.Diversity, "fell_back": res.FellBack,
		})
		if intentSeq != 0 {
			_ = e.Journal.Commit(intentSeq)
		}
	}
	return res, nil
}

type overlayCand struct {
	prop  Proposal
	mats  MaterialSet
	fell  bool
	label string
}

type overlayFinal struct {
	idx     int
	label   string
	kind    string
	notes   string
	reason  string
	succ    MaterialSet
	adopted bool
}

type overlayProbeJob struct {
	label string
	mats  MaterialSet
}

func (e *Engine) applyOverlayCompose(origin, current MaterialSet, kind, notes string, inv, dom overlayCand) (MaterialSet, bool, string) {
	succ := current
	adopted := false
	reason := notes
	switch kind {
	case "adopt_inv":
		if !inv.fell && inv.mats.Hash != current.Hash {
			succ, adopted = inv.mats, true
		}
	case "adopt_dom":
		if !dom.fell && dom.mats.Hash != current.Hash {
			succ, adopted = dom.mats, true
		}
	case "merge":
		if !inv.fell && !dom.fell {
			if merged, ok := e.mergeOverlay(current, inv.prop, dom.prop); ok {
				succ, adopted = merged, true
			}
		}
	}
	if adopted && !artifact.LoopTopologyEqual(origin.Loop, succ.Loop) {
		return current, false, "l3 topology"
	}
	if !adopted {
		return current, false, reason
	}
	return succ, true, reason
}

func overlayAbort(e *Engine, res OverlayResult, before map[string]string, intentSeq uint64, err error) (OverlayResult, error) {
	res.Adopted = false
	res.KeptBase = true
	res.LibraryStatus = LibraryNoEntry
	if res.Parent != "" {
		res.Hash = res.Parent
	}
	_ = assertRefsFrozen(e.Refs, before)
	if e != nil && e.Journal != nil && intentSeq != 0 {
		_ = e.Journal.Commit(intentSeq)
	}
	return res, err
}

func overlayBases(ctx context.Context, opts OverlayOpts, parent MaterialSet, k int, res *OverlayResult) ([]OverlayRollout, error) {
	out := make([]OverlayRollout, 0, k)
	for i := 1; i <= k; i++ {
		label := "base"
		if k > 1 {
			label = fmt.Sprintf("base%d", i)
		}
		if i == 1 && opts.ReuseBase != nil {
			r := *opts.ReuseBase
			r.Label = label
			r.Cached = true
			overlayRecordRollout(res, r, nil, opts.Tree)
			out = append(out, r)
			continue
		}
		r, err := overlayExec(ctx, opts.Roller, label, parent)
		overlayRecordRollout(res, r, nil, opts.Tree)
		out = append(out, r)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func joinOverlayBases(bases []OverlayRollout) OverlayRollout {
	if len(bases) == 0 {
		return OverlayRollout{Label: "base"}
	}
	if len(bases) == 1 {
		return bases[0]
	}
	var traj strings.Builder
	delivered := false
	for _, b := range bases {
		if b.Delivered {
			delivered = true
		}
		if b.Trajectory != "" {
			traj.WriteString(b.Trajectory)
			traj.WriteByte('\n')
		}
	}
	out := bases[0]
	out.Label = "base"
	out.Delivered = delivered
	out.Trajectory = traj.String()
	return out
}

func overlayExec(ctx context.Context, roller OverlayRoller, label string, mats MaterialSet) (OverlayRollout, error) {
	r, err := roller(ctx, label, mats)
	if err != nil {
		r = OverlayRollout{Label: label, Error: err.Error()}
	}
	r.Label = label
	return r, nil
}

func overlayRecordRollout(res *OverlayResult, r OverlayRollout, held map[string]bool, tree *OverlayTree) {
	res.Ledger.rollout(r.Label, r.USD, r.Cached)
	res.Spend.USD += r.USD
	if held != nil && r.Trajectory != "" {
		r.Trajectory = stripHeld(r.Trajectory, held)
	}
	res.Rollouts = append(res.Rollouts, r)
	if tree != nil {
		tree.writeRollout(r)
	}
}

func overlayRunProbes(ctx context.Context, roller OverlayRoller, jobs []overlayProbeJob, res *OverlayResult, held map[string]bool, tree *OverlayTree) error {
	if len(jobs) == 0 {
		return nil
	}
	got := make([]OverlayRollout, len(jobs))
	if len(jobs) == 1 {
		r, _ := overlayExec(ctx, roller, jobs[0].label, jobs[0].mats)
		got[0] = r
	} else {
		ch := make(chan struct {
			i int
			r OverlayRollout
		}, len(jobs))
		for i := range jobs {
			go func(i int) {
				r, _ := overlayExec(ctx, roller, jobs[i].label, jobs[i].mats)
				ch <- struct {
					i int
					r OverlayRollout
				}{i: i, r: r}
			}(i)
		}
		for range jobs {
			item := <-ch
			got[item.i] = item.r
		}
	}
	for _, r := range got {
		overlayRecordRollout(res, r, held, tree)
	}
	return nil
}

func overlayPickRolls(all []OverlayRollout, k int, prefix string) []OverlayRollout {
	out := make([]OverlayRollout, k)
	for i := 1; i <= k; i++ {
		want := fmt.Sprintf("%s%d", prefix, i)
		for j := range all {
			if all[j].Label == want {
				out[i-1] = all[j]
				break
			}
		}
	}
	return out
}

func overlayCandidatePool(invs, doms []overlayCand, invR, domR []OverlayRollout) string {
	var b strings.Builder
	for i := range invs {
		fmt.Fprintf(&b, "inv%d delivered=%v fell=%v traj:\n%s\n", i+1, invR[i].Delivered, invs[i].fell, invR[i].Trajectory)
		fmt.Fprintf(&b, "dom%d delivered=%v fell=%v traj:\n%s\n", i+1, domR[i].Delivered, doms[i].fell, domR[i].Trajectory)
	}
	return b.String()
}

func (e *Engine) overlayStage(ctx context.Context, dim string, index int, opts OverlayOpts, parent MaterialSet, held map[string]bool, base OverlayRollout, overBudget func() bool) (Proposal, MaterialSet, bool, int, error) {
	used := 0
	var lastErr error
	for attempt := 1; attempt <= overlayStageAttempts; attempt++ {
		used = attempt
		if ctx.Err() != nil {
			return Proposal{ID: strings.ToLower(dim) + "-inherit", Surface: "keep_base", Audit: "inherit parent"}, parent, true, used, ctx.Err()
		}
		if overBudget() {
			return Proposal{ID: strings.ToLower(dim) + "-inherit", Surface: "keep_base", Audit: "inherit parent"}, parent, true, used, nil
		}
		p, err := proposeOverlay(ctx, opts, dim, index, parent, base, held)
		if err != nil {
			lastErr = err
			continue
		}
		p = sanitizeHeldOut(p, held)
		if p.Surface == "" {
			p.Surface = surfaceKey(p)
		}
		if err := AdmitOverlay(p, dim, parent.Playbook, parent.Loop.PlaybookTokens, held, opts.HeldInText); err != nil {
			lastErr = err
			continue
		}
		snap, hash, err := ApplyProposal(e.CAS, parent.Snap, parent.Hash, p)
		if err != nil {
			lastErr = err
			continue
		}
		mats := MaterialsFromSnapshot(e.CAS, snap, hash, parent)
		if !artifact.LoopTopologyEqual(parent.Loop, mats.Loop) {
			lastErr = fmt.Errorf("l3 topology")
			continue
		}
		return p, mats, false, used, nil
	}
	_ = lastErr
	return Proposal{ID: strings.ToLower(dim) + "-inherit", Surface: "keep_base", Audit: "inherit parent"}, parent, true, used, nil
}

func (e *Engine) mergeOverlay(parent MaterialSet, inv, dom Proposal) (MaterialSet, bool) {
	if surfaceKey(inv) == surfaceKey(dom) {
		return parent, false
	}
	snap, hash, err := MergeProposals(e.CAS, parent.Snap, parent.Hash, []Proposal{inv, dom})
	if err != nil {
		return parent, false
	}
	mats := MaterialsFromSnapshot(e.CAS, snap, hash, parent)
	if !artifact.LoopTopologyEqual(parent.Loop, mats.Loop) {
		return parent, false
	}
	return mats, true
}

func heuristicCompose(inv, dom Proposal, base, invR, domR OverlayRollout) (string, string) {
	invHelped := invR.Delivered && (!base.Delivered || (invR.Deliverable != "" && invR.Deliverable != base.Deliverable))
	domHelped := domR.Delivered && (!base.Delivered || (domR.Deliverable != "" && domR.Deliverable != base.Deliverable))
	if invHelped && domHelped && surfaceKey(inv) != surfaceKey(dom) && inv.ID != "" && dom.ID != "" {
		return "merge", "both probes delivered a different artifact; surfaces disjoint"
	}
	if invHelped {
		return "adopt_inv", "inv probe delivered relative to base"
	}
	if domHelped {
		return "adopt_dom", "dom probe delivered relative to base"
	}
	return "keep_base", "no supported behavioral contrast"
}

func (e *Engine) overlayCommitLibrary(ctx context.Context, opts OverlayOpts, res *OverlayResult, held map[string]bool) {
	res.LibraryStatus = LibraryNoEntry
	if !res.Adopted || opts.Library == nil {
		return
	}
	exp := heuristicExperience(*res)
	if opts.Client != nil {
		if got, noEntry, err := distilOverlayLLM(ctx, opts, *res, held); err == nil {
			if noEntry {
				return
			}
			if HasExperience(got) {
				exp = got
			}
		}
	}
	if experienceHasHeld(exp, held) {
		return
	}
	exp = SanitizeExperience(exp, held)
	if !HasExperience(exp) || !experienceTransferable(exp) || experienceBanned(exp) {
		return
	}
	taskID := strings.TrimSpace(opts.TaskID)
	if taskID == "" {
		taskID = opts.SessionID
	}
	ok, err := opts.Library.Add(taskID, res.Hash, res.Reason, exp, held)
	if err != nil || !ok {
		return
	}
	_ = opts.Library.WriteL1(e.CAS, taskID, res.Hash, held)
	res.Experience = &exp
	res.LibraryStatus = LibraryAdded
}

func snapshotGlobalRefs(r *artifact.Refs) map[string]string {
	out := map[string]string{}
	if r == nil {
		return out
	}
	for _, n := range []string{artifact.RefActive, artifact.RefCanary, artifact.RefStaging} {
		out[n] = r.GetOrEmpty(n)
	}
	return out
}

func assertRefsFrozen(r *artifact.Refs, before map[string]string) error {
	if r == nil {
		return nil
	}
	after := snapshotGlobalRefs(r)
	for _, n := range []string{artifact.RefActive, artifact.RefCanary, artifact.RefStaging} {
		if after[n] != before[n] {
			return fmt.Errorf("overlay wrote refs/%s", n)
		}
	}
	return nil
}

func stripHeld(s string, held map[string]bool) string {
	for id := range held {
		if id != "" {
			s = strings.ReplaceAll(s, id, "")
		}
	}
	return s
}

func overlayEvidenceBlob(res OverlayResult) string {
	var b strings.Builder
	for _, e := range res.Evidence {
		b.WriteString(e)
		b.WriteByte('\n')
	}
	b.WriteString(res.INV.Audit)
	b.WriteString(res.INV.Expected)
	b.WriteString(res.INV.SkillMD)
	b.WriteString(res.INV.InstructionText)
	b.WriteString(res.DOM.Audit)
	b.WriteString(res.DOM.Expected)
	b.WriteString(res.DOM.SkillMD)
	if res.Experience != nil {
		b.WriteString(res.Experience.Cue)
		b.WriteString(res.Experience.Lesson)
		b.WriteString(res.Experience.Evidence)
	}
	return b.String()
}
