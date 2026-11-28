// hold.go 让引擎在交接收尾时停止领新任务，并告诉调用方手上还有几件在跑。
// 收尾的规则在 app/lead.go：daemon 看到桌面打开，先 Hold(true)，等 Busy() 归零（或等满上限）再停掉引擎让出；
// 收尾取消或重新领头时 Hold(false)。以前收尾不看个人任务，让出时直接取消正在跑的那一件，下次从头再跑。

package personal

// Hold 打开或关上「不领新任务」。在跑的任务不受影响。
func (e *Engine) Hold(on bool) {
	if e == nil {
		return
	}
	e.hold.Store(on)
}

// Held 在 Hold(true) 之后为真。
func (e *Engine) Held() bool {
	return e != nil && e.hold.Load()
}

// Busy 是手上正在跑的任务数（已领取、还没结束的）。
func (e *Engine) Busy() int {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.inFlight)
}
