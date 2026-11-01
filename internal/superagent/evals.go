// evals.go 拥有线程上的评估轮次：开评、暂停顺延、过期重排队和超时。
// 不调用模型。截止时间从编排台领走这轮评估才开始，没领走的不过期。

package superagent

import (
	"sort"
	"time"
)

// requeues 是连续过期后还能重新排队的次数。再过一次就记成 unavailable。
const requeues = 2

// evaluateDueLocked 处理到期评估、唤醒暂停工单，并为还有队列的线程开一轮评估。
// 调用时要已持有锁。返回可以启动的工单编号。
func (e *Engine) evaluateDueLocked() []string {
	if e == nil {
		return nil
	}
	now := e.now()
	keys := e.threadKeys()
	for _, key := range keys {
		e.dueEval(e.threads[key], now)
	}
	if e.policy.Paused {
		return nil
	}
	var launches []string
	for _, id := range e.wakePausedLocked() {
		launches = append(launches, id)
	}
	for _, key := range keys {
		th := e.threads[key]
		if th == nil || th.Evaluating != nil || len(th.Queued) == 0 {
			continue
		}
		e.startEval(th, now)
	}
	return launches
}

// threadKeys 返回线程编号，按字典序。调用时要已持有锁。
func (e *Engine) threadKeys() []string {
	keys := make([]string, 0, len(e.threads))
	for key := range e.threads {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// dueEval 处理 th 上正在进行的评估。now 是引擎时钟。
// 暂停时记下 PausedAt。恢复时把 Deadline 往后推暂停的时长。已交给评估者且超过 Deadline 就过期。
func (e *Engine) dueEval(th *Thread, now time.Time) {
	if th == nil || th.Evaluating == nil {
		return
	}
	ev := th.Evaluating
	if e.policy.Paused {
		if ev.PausedAt.IsZero() {
			ev.PausedAt = now
		}
		return
	}
	if !ev.PausedAt.IsZero() {
		ev.Deadline = ev.Deadline.Add(now.Sub(ev.PausedAt))
		ev.PausedAt = time.Time{}
		return
	}
	if ev.Delivered.IsZero() || ev.Deadline.After(now) {
		return
	}
	e.expireEval(th)
}

// startEval 把 th 队列里还没人决定的触发收成一轮评估。now 写进评估编号。截止时间要等 DeliverOpen。
// 不准入的触发直接记 ignore。已经有工单在决定的人的后续消息离开队列，不进这一轮。
func (e *Engine) startEval(th *Thread, now time.Time) {
	ids := append([]string(nil), th.Queued...)
	var batch []*Trigger
	for _, id := range ids {
		tr := e.triggers[id]
		if tr == nil {
			continue
		}
		if len(e.owners(tr)) > 0 {
			e.unqueue(tr)
			continue
		}
		admitted, strength, why := e.policy.Gate(*tr)
		if !admitted {
			if why == "" {
				why = "not_admitted"
			}
			e.putDecision(tr, "ignore", why, "")
			e.unqueue(tr)
			continue
		}
		tr.StrengthHold = strength
		batch = append(batch, tr)
	}
	if len(batch) == 0 {
		return
	}
	best := batch[0]
	bestStrength := best.StrengthHold
	for _, tr := range batch[1:] {
		strength := tr.StrengthHold
		if strength == "explicit" && bestStrength != "explicit" {
			best = tr
			bestStrength = strength
			continue
		}
		if strength == bestStrength && tr.ID > best.ID {
			best = tr
		}
	}
	kept := make([]string, 0, len(batch))
	for _, tr := range batch {
		kept = append(kept, tr.ID)
		e.unqueue(tr)
	}
	th.Evaluating = &Evaluation{
		ID:            "eval-" + th.Key + "-" + now.UTC().Format("20060102T150405.000000000"),
		TriggerIDs:    kept,
		Primary:       best.ID,
		Strength:      bestStrength,
		PolicyVersion: e.policy.Version,
	}
}

// DeliverOpen 把还没交给评估者的评估标成已交付，并按现在起算截止时间。返回新交付的轮数。
func (e *Engine) DeliverOpen() int {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	now := e.now()
	n := 0
	for _, th := range e.threads {
		if th == nil || th.Evaluating == nil || !th.Evaluating.Delivered.IsZero() {
			continue
		}
		th.Evaluating.Delivered = now
		th.Evaluating.Deadline = now.Add(e.policy.evalTimeout())
		n++
	}
	return n
}

// expireEval 结束 th 上到期的评估。连续次数不超过 requeues 时把触发重新排队，否则记 unavailable，原因是 timeout。
func (e *Engine) expireEval(th *Thread) {
	ev := th.Evaluating
	if ev == nil {
		return
	}
	ids := append([]string(nil), ev.TriggerIDs...)
	th.Evaluating = nil
	th.Expired++
	if th.Expired > requeues {
		for _, id := range ids {
			tr := e.triggers[id]
			if tr == nil || tr.DecisionID != "" {
				continue
			}
			e.putDecision(tr, "unavailable", "timeout", "")
			e.unqueue(tr)
		}
		return
	}
	for _, id := range ids {
		tr := e.triggers[id]
		if tr == nil || tr.DecisionID != "" {
			continue
		}
		tr.Requeued = true
		e.queue(tr)
	}
}

// wakePausedLocked 重新打开有未送达输入、且不是被人停下的工单。暂停时返回空。调用时要已持有锁。
func (e *Engine) wakePausedLocked() []string {
	if e.policy.Paused {
		return nil
	}
	var out []string
	for _, ticket := range e.tickets {
		if ticket == nil || ticket.Ended == nil || ticket.Ended.By == "human" || len(ticket.Undelivered) == 0 {
			continue
		}
		if e.resume(ticket) {
			out = append(out, ticket.ID)
		}
	}
	sort.Strings(out)
	return out
}
