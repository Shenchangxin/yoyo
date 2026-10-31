// pass.go 是两秒轮次里引擎自己做的那一段。
// 它不打模型。显式且模式为 auto 的排队触发会被接受。暂停期间什么都不启动。

package superagent

import "sort"

// EvaluateDue 恢复暂停期间攒下的工单，并在自动模式下接受显式触发。
func (e *Engine) EvaluateDue() {
	if e == nil {
		return
	}
	for _, id := range e.wakePaused() {
		if e.OnLaunch != nil {
			e.OnLaunch(id)
		}
	}
	for _, id := range e.explicitQueued() {
		_, _ = e.Decide(id, "accept", "due")
	}
}

// wakePaused 重新打开有未送达输入、且不是被人停下的工单。暂停时返回空。
func (e *Engine) wakePaused() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
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

// explicitQueued 返回自动模式下还排着队、而且准入强度是 explicit 的触发编号。暂停或不是 auto 时返回空。
func (e *Engine) explicitQueued() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.policy.Paused || e.policy.Mode != "auto" {
		return nil
	}
	var ids []string
	for _, tr := range e.triggers {
		if tr == nil || !tr.Queued {
			continue
		}
		admitted, strength, _ := e.policy.Gate(*tr)
		if admitted && strength == "explicit" {
			ids = append(ids, tr.ID)
		}
	}
	sort.Strings(ids)
	return ids
}
