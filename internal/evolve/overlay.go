// 会话上的 harness 叠加：按 k 做 rollout 和改进会话，只在本轮材料上采纳或保留父快照。
// 不写 refs/active、refs/canary、refs/staging。结束时 assertRefsFrozen 对照进入前的这三枚引用。KeptBase 一开始为真。

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
	// OverlayK 是默认预算倍数，值为 1。
	OverlayK = 1
	// OverlayMaxK 是 ClampOverlayK 允许的最大倍数，值为 4。
	OverlayMaxK = 4
	// OverlayR 是每个倍数对应的 rollout 次数，值为 4。OverlayBudget 返回它乘以 k。
	OverlayR = 4
	// OverlayP 是每个倍数对应的改进会话次数，值为 3。OverlayBudget 返回它乘以 k。
	OverlayP = 3
	// OverlayObsFloor 的值是 10000。会话记录单条观察时按 rune 截到这个数。本文件的 R/P 账本不读它。
	OverlayObsFloor = 10_000
	// overlayStageAttempts 是同一维提议最多尝试的次数，值为 2。
	overlayStageAttempts = 2
)

// OverlayBudget 先 ClampOverlayK，再返回 4×k 与 3×k，也就是 OverlayR×k 次 rollout 和 OverlayP×k 次改进会话。k 为 1 时是 4 与 3。
func OverlayBudget(k int) (r, p int) {
	k = ClampOverlayK(k)
	return 4 * k, 3 * k
}

// ClampOverlayK 把 k 限制在 1 和 OverlayMaxK 之间。小于 1 时返回 1，大于 OverlayMaxK 时返回 OverlayMaxK。
func ClampOverlayK(k int) int {
	if k < 1 {
		return 1
	}
	if k > OverlayMaxK {
		return OverlayMaxK
	}
	return k
}

// OverlayRollout 是一次求解执行。它不带 Harbor 分数。
type OverlayRollout struct {
	Label       string  `json:"label"`
	Delivered   bool    `json:"delivered"`
	Trajectory  string  `json:"trajectory,omitempty"`
	Deliverable string  `json:"deliverable,omitempty"`
	Cached      bool    `json:"cached,omitempty"`
	USD         float64 `json:"usd,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// OverlayRoller 跑一次隔离探针。它不写全局引用。
type OverlayRoller func(ctx context.Context, label string, mats MaterialSet) (OverlayRollout, error)

// overlayLedgerEntry 是账本里的一条 rollout 或 session。
type overlayLedgerEntry struct {
	Kind     string
	Label    string
	Cached   bool
	USD      float64
	Attempts int
}

// OverlayLedger 是配对预算。身份是 R 为 OverlayR×k、P 为 OverlayP×k。JSON 不直接输出本结构，对外用 View。
type OverlayLedger struct {
	K       int
	entries []overlayLedgerEntry
}

// rollout 记一条 rollout。
// label 写入条目的 Label。
// usd 写入条目的 USD。
// cached 写入条目的 Cached。
func (l *OverlayLedger) rollout(label string, usd float64, cached bool) {
	l.entries = append(l.entries, overlayLedgerEntry{Kind: "rollout", Label: label, Cached: cached, USD: usd})
}

// session 记一条改进会话。attempts 小于 1 时按 1 记。
// label 写入条目的 Label。
// usd 写入条目的 USD。
func (l *OverlayLedger) session(label string, usd float64, attempts int) {
	if attempts < 1 {
		attempts = 1
	}
	l.entries = append(l.entries, overlayLedgerEntry{Kind: "session", Label: label, USD: usd, Attempts: attempts})
}

// SessionAttempts 把 session 条目的 Attempts 相加。条目上的 Attempts 小于 1 时该条按 1 计。
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

// R 返回 rollout 条目数。
func (l OverlayLedger) R() int {
	n := 0
	for _, e := range l.entries {
		if e.Kind == "rollout" {
			n++
		}
	}
	return n
}

// P 返回 session 条目数。
func (l OverlayLedger) P() int {
	n := 0
	for _, e := range l.entries {
		if e.Kind == "session" {
			n++
		}
	}
	return n
}

// labels 按种类收集标签。
// kind 与条目的 Kind 相等时，收集该条的 Label。
func (l OverlayLedger) labels(kind string) []string {
	var out []string
	for _, e := range l.entries {
		if e.Kind == kind {
			out = append(out, e.Label)
		}
	}
	return out
}

// View 把账本收成可序列化的一行。K 经过 ClampOverlayK。
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

// IdentityOK 要求 R、P 等于 OverlayBudget(ClampOverlayK(K))，且同一种类里没有重复标签。否则返回带实际次数和期望次数的错误。
func (l OverlayLedger) IdentityOK() error {
	k := ClampOverlayK(l.K)
	wantR, wantP := OverlayBudget(k)
	rl, sl := l.labels("rollout"), l.labels("session")
	var dupes []string
	seen := map[string]int{}
	for _, label := range append(append([]string{}, rl...), sl...) {
		seen[label]++
	}
	// 重复按种类分开数，同一标签可以既是会话又是 rollout。k 为 1 时会话标签互不相同。
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

// OverlayLedgerView 是给操作者看的预算行。
type OverlayLedgerView struct {
	K               int      `json:"k"`
	R               int      `json:"r"`
	P               int      `json:"p"`
	SessionAttempts int      `json:"session_attempts,omitempty"`
	Rollouts        []string `json:"rollouts,omitempty"`
	Sessions        []string `json:"sessions,omitempty"`
}

// OverlayPair 记录两条终稿哈希是否相同，以及公共前缀的字节长度。
type OverlayPair struct {
	A      string `json:"a"`
	B      string `json:"b"`
	Equal  bool   `json:"equal"`
	Prefix int    `json:"prefix"`
}

// OverlayOpts 是一次叠加的父材料、任务和预算。K 会经 ClampOverlayK。Mode 会经 ClampOverlayMode。
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

// OverlayResult 是一次叠加的采纳结果。Adopted 为真时哈希是选中的子快照；否则哈希回到父快照。本结果不表示 refs/active 被改写。
type OverlayResult struct {
	Adopted bool `json:"adopted"`
	// KeptBase 在 Overlay 开头写成 true。只有选中终稿已采纳、哈希不同于父快照且对应 c 卷已交付时才改成 false。账本身份失败或引用被改写时改回 true。
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

// Overlay 跑完基线、INV、DOM 和合成。开始时 KeptBase 为 true，哈希等于父快照。
// 只有选中的终稿不是基线、已经采纳、哈希不同于父快照，且对应的 c 卷 Delivered 为真时，才把 Adopted 写成 true、KeptBase 写成 false。
// 账本 IdentityOK 失败，或 assertRefsFrozen 发现 active、canary、staging 有变化时，采纳撤销，KeptBase 回到 true，哈希回到父快照。
// 本方法不调用 Refs.Set，不把结果晋升为 active。
// ctx 传给 overlayBases、overlaySeq 或 overlayAgg、overlayPickFinal 和 overlayCommitLibrary。
// opts 的 Roller 为 nil 或 Parent.Hash 为空时直接返回错误；还读取 K、Mode、Domain、HeldOut、SessionID、ReuseBase、MaxUSD、MaxWall、Scratch、KeepEvidence、Task、Library。MaxUSD 大于 0 且已花到，或 MaxWall 大于 0 且已用时间达到，就视为超预算。
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

// overlayCand 是一维提议及其材料和是否退回父快照。
type overlayCand struct {
	prop  Proposal
	mats  MaterialSet
	fell  bool
	label string
}

// overlayFinal 是一轮合成后的后继材料。adopted 为假时 succ 仍是合成前的当前材料。
type overlayFinal struct {
	idx     int
	label   string
	kind    string
	notes   string
	reason  string
	succ    MaterialSet
	adopted bool
}

// overlayProbeJob 是一次待执行的探针：标签和材料。
type overlayProbeJob struct {
	label string
	mats  MaterialSet
}

// applyOverlayCompose 按 keep 以外的决定换成 INV、DOM 或合并结果。采纳后若循环拓扑与 origin 不同，退回 current，原因是 l3 topology。未采纳时返回 current。
// kind 为 adopt_inv、adopt_dom 或 merge 时才可能采纳。
// notes 先当作原因；只有采纳后拓扑与 origin 不同时才改成 l3 topology。
// inv 在 kind 为 adopt_inv 且未 fell、材料哈希不同于 current 时，用它的 mats；kind 为 merge 时还要求未 fell，并把 prop 交给 mergeOverlay。
// dom 在 kind 为 adopt_dom 且未 fell、材料哈希不同于 current 时，用它的 mats；merge 时同样要求未 fell，prop 交给 mergeOverlay。
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

// overlayAbort 把采纳改回 false、KeptBase 改回 true，库状态改成 LibraryNoEntry，哈希改回父快照，并再查一次引用是否被改写。意图序号非 0 时提交日志。
// res 把 Adopted 改成 false，KeptBase 改成 true，LibraryStatus 改成 LibraryNoEntry；Parent 非空时 Hash 改回 Parent。
// before 交给 assertRefsFrozen。
// intentSeq 非 0 且 Journal 非空时交给 Commit。
// err 原样作为第二个返回值。
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

// overlayBases 跑 k 次基线。k 大于 1 时标签为 base1 起。第一次且 ReuseBase 非空时记成缓存，不再调用 roller。
// ctx 传给 overlayExec。
// opts 的 ReuseBase 在第一次非空时复制并标上 Cached，不再调用 roller；否则用 Roller。Tree 传给 overlayRecordRollout。
// parent 作为材料传给 overlayExec。
// res 传给 overlayRecordRollout。
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

// joinOverlayBases 把多份基线轨迹接成标签为 base 的一条。任一交付则 Delivered 为真。空表返回标签 base 的零值。只有一条时原样返回。
// bases 长度为 0 时返回标签 base 的零值，长度为 1 时原样返回；更多时拼接 Trajectory，任一 Delivered 则结果为真，标签改成 base。
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

// overlayExec 调用 roller。出错时仍返回带该标签和错误文本的 rollout，并带回原错误。
// ctx 传给 roller。
// label 传给 roller，返回前再写入 rollout 的 Label；出错时 Error 写成该错误文本。
// mats 传给 roller。
func overlayExec(ctx context.Context, roller OverlayRoller, label string, mats MaterialSet) (OverlayRollout, error) {
	r, err := roller(ctx, label, mats)
	if err != nil {
		r = OverlayRollout{Label: label, Error: err.Error()}
	}
	r.Label = label
	return r, nil
}

// overlayRecordRollout 记入账本和用量。held 非空时先从轨迹里去掉那些编号。
// res 的账本记一条 rollout，Spend.USD 加上这次的 USD，并把该条追加进 Rollouts。
// tree 非空时 writeRollout。
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

// overlayRunProbes 执行探针并按原顺序入账。多于一条时并发，结果仍按任务下标写回。
// ctx 传给每次 overlayExec。
// roller 传给 overlayExec。
// jobs 长度为 0 时直接返回；否则用每条的 label 和 mats 调用 overlayExec。
// res 传给 overlayRecordRollout。
// held 传给 overlayRecordRollout。
// tree 传给 overlayRecordRollout。
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

// overlayPickRolls 按前缀加 1 到 k 取出标签匹配的 rollout。没有匹配时该位是零值。
// all 里 Label 等于 prefix 加序号的那条写入对应位置。
// prefix 和从 1 到 k 的序号拼成要找的 Label。
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

// overlayCandidatePool 把每一对 INV 与 DOM 的交付、退回和轨迹写成一段文本。
// invs 决定循环次数，每一项的 fell 写进 inv 行。
// doms 按下标取 fell，写进 dom 行。
// invR 按下标取 Delivered 和 Trajectory，写进 inv 行。
// domR 按下标取 Delivered 和 Trajectory，写进 dom 行。
func overlayCandidatePool(invs, doms []overlayCand, invR, domR []OverlayRollout) string {
	var b strings.Builder
	for i := range invs {
		fmt.Fprintf(&b, "inv%d delivered=%v fell=%v traj:\n%s\n", i+1, invR[i].Delivered, invs[i].fell, invR[i].Trajectory)
		fmt.Fprintf(&b, "dom%d delivered=%v fell=%v traj:\n%s\n", i+1, domR[i].Delivered, doms[i].fell, domR[i].Trajectory)
	}
	return b.String()
}

// overlayStage 最多尝试 overlayStageAttempts 次。上下文取消时返回父材料和该错误。超预算、提议失败、准入失败、写入失败或循环拓扑变化时继续下一次。用尽后返回表面为 keep_base 的父材料，fell 为 true，错误为 nil。
// ctx 的 Err 非空时返回父材料和该错误；否则传给 proposeOverlay。
// dim 转成小写后拼进继承提议的 ID，并交给 proposeOverlay 和 AdmitOverlay。
// index 传给 proposeOverlay。
// opts 传给 proposeOverlay，HeldInText 交给 AdmitOverlay。
// parent 在取消、超预算或用尽时返回；Snap 和 Hash 交给 ApplyProposal，Playbook 与 Loop.PlaybookTokens 交给 AdmitOverlay，Loop 用来比较拓扑。
// held 交给 proposeOverlay、sanitizeHeldOut 和 AdmitOverlay。
// overBudget 返回真时返回父材料，错误为 nil。
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

// mergeOverlay 在两个提议的 surfaceKey 相同时返回父材料。合并后循环拓扑与父材料不同时也退回父材料。
// parent 在表面相同、合并失败或循环拓扑不同时原样返回；Snap 和 Hash 交给 MergeProposals，Loop 用来比较拓扑。
// inv 与 dom 的 surfaceKey 相同则返回 parent 和 false，否则和 dom 一起交给 MergeProposals。
// dom 与 inv 一起交给 MergeProposals。
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

// heuristicCompose 在两侧都相对基线交付了不同产物、表面不同且编号都非空时返回 merge。否则优先 adopt_inv，再 adopt_dom，都没有则 keep_base。
// invR 必须 Delivered 为真，并且 base 未交付，或 Deliverable 非空且不同于 base.Deliverable，才算 INV 有帮助。
// domR 用同样的条件和 base 比较。
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

// overlayCommitLibrary 先把库状态写成 LibraryNoEntry。未采纳或库为 nil 时返回。蒸馏结果标明不入库、经验含 held、不可迁移或命中禁用词时也不写。写入成功才把状态改成 LibraryAdded。
// ctx 在 Client 非空时传给 distilOverlayLLM。
// opts 的 Library 为 nil 或未采纳时直接返回；Client 非空时用于蒸馏；TaskID 去掉空白后为空则改用 SessionID。
// res 先把 LibraryStatus 写成 LibraryNoEntry；Adopted 为假则返回。写入成功后 Experience 指向该经验，LibraryStatus 改成 LibraryAdded，Hash 和 Reason 交给 Library.Add。
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

// snapshotGlobalRefs 记下 refs/active、refs/canary、refs/staging 的当前值。Refs 为 nil 时返回空映射。
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

// assertRefsFrozen 对照进入 Overlay 前的快照，检查 refs/active、refs/canary、refs/staging 都没变。有任一不同就返回错误。Refs 为 nil 时返回 nil。
// before 与重新读到的 active、canary、staging 逐项比较，有任一不同就返回 overlay wrote refs/ 加该名字。
func assertRefsFrozen(r *artifact.Refs, before map[string]string) error {
	if r == nil {
		return nil
	}
	after := snapshotGlobalRefs(r)
	for _, n := range []string{artifact.RefActive, artifact.RefCanary, artifact.RefStaging} {
		// 叠加只钉住这一次会话。三枚全局指针有任何一枚变了，这次采纳就撤销。
		if after[n] != before[n] {
			return fmt.Errorf("overlay wrote refs/%s", n)
		}
	}
	return nil
}

// stripHeld 从文本里删掉 held 中的非空编号。
func stripHeld(s string, held map[string]bool) string {
	for id := range held {
		if id != "" {
			s = strings.ReplaceAll(s, id, "")
		}
	}
	return s
}

// overlayEvidenceBlob 把证据路径、INV、DOM 和经验字段接成一段，供测试查看 held 编号是否还在。
// res 的 Evidence 每条一行，再接上 INV 的 Audit、Expected、SkillMD、InstructionText，以及 DOM 的 Audit、Expected、SkillMD；Experience 非空时再接 Cue、Lesson、Evidence。
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
