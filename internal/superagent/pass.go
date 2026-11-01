// pass.go 是两秒轮次里引擎自己做的那一段。
// 它不打模型，也不直接接受触发。暂停期间不唤醒工单，也不新开评估。

package superagent

// EvaluateDue 处理到期评估，唤醒暂停期间攒下的工单，并为还有队列的线程开一轮评估。
// 它不直接接受触发。接受仍走 Decide。
func (e *Engine) EvaluateDue() {
	if e == nil {
		return
	}
	e.mu.Lock()
	launches := e.evaluateDueLocked()
	e.mu.Unlock()
	if e.OnLaunch == nil {
		return
	}
	for _, id := range launches {
		e.OnLaunch(id)
	}
}
