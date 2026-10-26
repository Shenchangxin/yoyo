// agg 与 seq 两种叠加日程。agg 的每一对都从父材料出发。seq 在采纳后把当前材料换成后继，下一对从当前材料提议。
// 两种都不写 refs。预算条目由调用方记入账本。

package evolve

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// overlayAgg 先对父材料提出全部 INV 与 DOM，再并行探针，然后各自相对父材料合成。合成会话的 Attempts 记 1。终稿探针的标签是 c1 起。
func (e *Engine) overlayAgg(ctx context.Context, opts OverlayOpts, parent MaterialSet, k int, joined OverlayRollout, held map[string]bool, overBudget func() bool, res *OverlayResult) ([]overlayCand, []overlayCand, []overlayFinal, error) {
	invs, doms, invAtt, domAtt, err := e.overlayProposeAll(ctx, opts, parent, k, joined, held, overBudget)
	if err != nil {
		return nil, nil, nil, err
	}
	for i := 0; i < k; i++ {
		overlayNoteFall(res, k, "inv", i+1, invs[i].fell)
		res.Ledger.session(overlaySessLabel(k, "inv", i+1), 0, invAtt[i])
		overlayNoteFall(res, k, "dom", i+1, doms[i].fell)
		res.Ledger.session(overlaySessLabel(k, "dom", i+1), 0, domAtt[i])
	}
	jobs := make([]overlayProbeJob, 0, 2*k)
	for i := 0; i < k; i++ {
		jobs = append(jobs, overlayProbeJob{label: invs[i].label, mats: invs[i].mats})
		jobs = append(jobs, overlayProbeJob{label: doms[i].label, mats: doms[i].mats})
	}
	if err := overlayRunProbes(ctx, opts.Roller, jobs, res, held, opts.Tree); err != nil {
		return nil, nil, nil, err
	}
	invRolls := overlayPickRolls(res.Rollouts, k, "inv")
	domRolls := overlayPickRolls(res.Rollouts, k, "dom")
	pool := overlayCandidatePool(invs, doms, invRolls, domRolls)
	if opts.Tree != nil {
		for i := 0; i < k; i++ {
			opts.Tree.writePool(invs[i].label, invs[i].mats, invRolls[i])
			opts.Tree.writePool(doms[i].label, doms[i].mats, domRolls[i])
		}
	}
	finals := make([]overlayFinal, k)
	for i := 0; i < k; i++ {
		kind, notes := heuristicCompose(invs[i].prop, doms[i].prop, joined, invRolls[i], domRolls[i])
		if opts.Client != nil && !overBudget() {
			if got, n, cerr := composeOverlayLLM(ctx, opts, parent, invs[i].prop, doms[i].prop, joined, invRolls[i], domRolls[i], pool); cerr == nil && got != "" {
				kind, notes = got, n
			}
		}
		sess := overlaySessLabel(k, "compose", i+1)
		res.Ledger.session(sess, 0, 1)
		if opts.Tree != nil {
			opts.Tree.writeDecision(sess, notes)
		}
		succ, adopted, reason := e.applyOverlayCompose(parent, parent, kind, notes, invs[i], doms[i])
		finals[i] = overlayFinal{
			idx: i + 1, kind: kind, notes: notes, succ: succ, adopted: adopted, reason: reason,
			label: fmt.Sprintf("c%d", i+1),
		}
	}
	cJobs := make([]overlayProbeJob, k)
	for i := 0; i < k; i++ {
		cJobs[i] = overlayProbeJob{label: finals[i].label, mats: finals[i].succ}
	}
	if err := overlayRunProbes(ctx, opts.Roller, cJobs, res, held, opts.Tree); err != nil {
		return nil, nil, nil, err
	}
	return invs, doms, finals, nil
}

// overlaySeq 按 1 到 k 逐步提议。合成作用在 current 上。该步 adopted 为真时，current 换成 succ，下一步从新的 current 出发。
func (e *Engine) overlaySeq(ctx context.Context, opts OverlayOpts, parent MaterialSet, k int, joined OverlayRollout, held map[string]bool, overBudget func() bool, res *OverlayResult) ([]overlayCand, []overlayCand, []overlayFinal, error) {
	invs := make([]overlayCand, k)
	doms := make([]overlayCand, k)
	finals := make([]overlayFinal, k)
	current := parent
	var history strings.Builder
	for t := 1; t <= k; t++ {
		pairInv, pairDom, invAtt, domAtt, err := e.overlayProposePair(ctx, opts, current, t, joined, held, overBudget)
		if err != nil {
			return nil, nil, nil, err
		}
		invs[t-1], doms[t-1] = pairInv, pairDom
		overlayNoteFall(res, k, "inv", t, invs[t-1].fell)
		res.Ledger.session(overlaySessLabel(k, "inv", t), 0, invAtt)
		overlayNoteFall(res, k, "dom", t, doms[t-1].fell)
		res.Ledger.session(overlaySessLabel(k, "dom", t), 0, domAtt)

		pair := []overlayProbeJob{
			{label: invs[t-1].label, mats: invs[t-1].mats},
			{label: doms[t-1].label, mats: doms[t-1].mats},
		}
		if err := overlayRunProbes(ctx, opts.Roller, pair, res, held, opts.Tree); err != nil {
			return nil, nil, nil, err
		}
		invRoll := overlayFindRoll(res.Rollouts, invs[t-1].label)
		domRoll := overlayFindRoll(res.Rollouts, doms[t-1].label)
		if opts.Tree != nil {
			opts.Tree.writePool(invs[t-1].label, invs[t-1].mats, invRoll)
			opts.Tree.writePool(doms[t-1].label, doms[t-1].mats, domRoll)
		}
		kind, notes := heuristicCompose(invs[t-1].prop, doms[t-1].prop, joined, invRoll, domRoll)
		pool := history.String() + overlayCandidatePool(invs[:t], doms[:t], overlayPickRolls(res.Rollouts, t, "inv"), overlayPickRolls(res.Rollouts, t, "dom"))
		if opts.Client != nil && !overBudget() {
			if got, n, cerr := composeOverlayLLM(ctx, opts, current, invs[t-1].prop, doms[t-1].prop, joined, invRoll, domRoll, pool); cerr == nil && got != "" {
				kind, notes = got, n
			}
		}
		sess := overlaySessLabel(k, "compose", t)
		res.Ledger.session(sess, 0, 1)
		if opts.Tree != nil {
			opts.Tree.writeDecision(sess, notes)
		}
		succ, adopted, reason := e.applyOverlayCompose(parent, current, kind, notes, invs[t-1], doms[t-1])
		finals[t-1] = overlayFinal{
			idx: t, kind: kind, notes: notes, succ: succ, adopted: adopted, reason: reason,
			label: fmt.Sprintf("c%d", t),
		}
		if err := overlayRunProbes(ctx, opts.Roller, []overlayProbeJob{{label: finals[t-1].label, mats: finals[t-1].succ}}, res, held, opts.Tree); err != nil {
			return nil, nil, nil, err
		}
		history.WriteString(notes)
		history.WriteByte('\n')
		if adopted {
			current = succ
		}
	}
	return invs, doms, finals, nil
}

// overlayStageJob 是一次待提议的维度和下标。
type overlayStageJob struct {
	dim string
	i   int
}

// overlayProposeAll 为每个下标提出 INV 和 DOM。ParallelPropose 为真且任务多于一条时并发。任一任务带回错误则整批失败。
func (e *Engine) overlayProposeAll(ctx context.Context, opts OverlayOpts, parent MaterialSet, k int, joined OverlayRollout, held map[string]bool, overBudget func() bool) ([]overlayCand, []overlayCand, []int, []int, error) {
	jobs := make([]overlayStageJob, 0, 2*k)
	for i := 1; i <= k; i++ {
		jobs = append(jobs, overlayStageJob{"INV", i}, overlayStageJob{"DOM", i})
	}
	type out struct {
		cand overlayCand
		att  int
		err  error
	}
	got := make([]out, len(jobs))
	run := func(j int) {
		dim, i := jobs[j].dim, jobs[j].i
		p, mats, fell, att, err := e.overlayStage(ctx, dim, i, opts, parent, held, joined, overBudget)
		got[j] = out{
			cand: overlayCand{prop: p, mats: mats, fell: fell, label: fmt.Sprintf("%s%d", strings.ToLower(dim[:3]), i)},
			att:  att,
			err:  err,
		}
		if dim == "INV" {
			got[j].cand.label = fmt.Sprintf("inv%d", i)
		} else {
			got[j].cand.label = fmt.Sprintf("dom%d", i)
		}
	}
	if opts.ParallelPropose && len(jobs) > 1 {
		var wg sync.WaitGroup
		for j := range jobs {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				run(j)
			}(j)
		}
		wg.Wait()
	} else {
		for j := range jobs {
			run(j)
		}
	}
	invs := make([]overlayCand, k)
	doms := make([]overlayCand, k)
	invAtt := make([]int, k)
	domAtt := make([]int, k)
	for j, job := range jobs {
		if got[j].err != nil {
			return nil, nil, nil, nil, got[j].err
		}
		if job.dim == "INV" {
			invs[job.i-1] = got[j].cand
			invAtt[job.i-1] = got[j].att
		} else {
			doms[job.i-1] = got[j].cand
			domAtt[job.i-1] = got[j].att
		}
	}
	return invs, doms, invAtt, domAtt, nil
}

// overlayProposePair 对当前材料提出一对 INV 和 DOM。ParallelPropose 为真时两维并发，按维度把结果放回 inv 与 dom。
func (e *Engine) overlayProposePair(ctx context.Context, opts OverlayOpts, current MaterialSet, i int, joined OverlayRollout, held map[string]bool, overBudget func() bool) (overlayCand, overlayCand, int, int, error) {
	type out struct {
		dim  string
		cand overlayCand
		att  int
		err  error
	}
	run := func(dim string) out {
		p, mats, fell, att, err := e.overlayStage(ctx, dim, i, opts, current, held, joined, overBudget)
		label := fmt.Sprintf("inv%d", i)
		if dim == "DOM" {
			label = fmt.Sprintf("dom%d", i)
		}
		return out{dim: dim, cand: overlayCand{prop: p, mats: mats, fell: fell, label: label}, att: att, err: err}
	}
	if opts.ParallelPropose {
		ch := make(chan out, 2)
		go func() { ch <- run("INV") }()
		go func() { ch <- run("DOM") }()
		a, b := <-ch, <-ch
		if a.err != nil {
			return overlayCand{}, overlayCand{}, 0, 0, a.err
		}
		if b.err != nil {
			return overlayCand{}, overlayCand{}, 0, 0, b.err
		}
		inv, dom := a.cand, b.cand
		invAtt, domAtt := a.att, b.att
		if a.dim == "DOM" {
			inv, dom = b.cand, a.cand
			invAtt, domAtt = b.att, a.att
		}
		return inv, dom, invAtt, domAtt, nil
	}
	invOut := run("INV")
	if invOut.err != nil {
		return overlayCand{}, overlayCand{}, 0, 0, invOut.err
	}
	domOut := run("DOM")
	if domOut.err != nil {
		return overlayCand{}, overlayCand{}, 0, 0, domOut.err
	}
	return invOut.cand, domOut.cand, invOut.att, domOut.att, nil
}

// overlayFindRoll 按标签找 rollout。找不到时返回只带该标签的零值。
func overlayFindRoll(all []OverlayRollout, label string) OverlayRollout {
	for i := range all {
		if all[i].Label == label {
			return all[i]
		}
	}
	return OverlayRollout{Label: label}
}
