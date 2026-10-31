// jobs.go 让前台命令在预算用尽后转入后台。
// 预算内结束的结果交给调用方；超时后仍在跑的作业，完成时由 Agent 用 steer 投递。

package dsh

import (
	"context"
	"sync"
	"time"
)

// DefaultForeground 是前台等待上限，与 DeepSeek Harness 工人预设的 5 秒一致。
const DefaultForeground = 5 * time.Second

// JobFunc 在 ctx 取消时停下。返回的字符串是完成通知里的正文。
type JobFunc func(ctx context.Context) (string, error)

// JobView 是一次 RunJob 的即时结果。Background 为真时 Output 还没有。
type JobView struct {
	ID         string
	Output     string
	Err        error
	Background bool
}

type jobSlot struct {
	mu         sync.Mutex
	done       bool
	background bool
	output     string
	err        error
	finished   chan struct{}
}

// RunForeground 跑 fn。budget 小于等于 0 时用 DefaultForeground。
// 在预算内结束则 Background 为假并带上输出。否则返回 Background 为真，fn 继续在 ctx 上跑，完成后调用 onDone 一次。
func RunForeground(ctx context.Context, id string, budget time.Duration, fn JobFunc, onDone func(output string, err error)) JobView {
	if budget <= 0 {
		budget = DefaultForeground
	}
	if fn == nil {
		return JobView{ID: id, Err: errInbox("empty job")}
	}
	slot := &jobSlot{finished: make(chan struct{})}
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		out, err := fn(runCtx)
		slot.mu.Lock()
		slot.done = true
		slot.output = out
		slot.err = err
		bg := slot.background
		slot.mu.Unlock()
		close(slot.finished)
		if bg && onDone != nil {
			onDone(out, err)
		}
		_ = cancel
	}()
	timer := time.NewTimer(budget)
	defer timer.Stop()
	select {
	case <-slot.finished:
		cancel()
		slot.mu.Lock()
		defer slot.mu.Unlock()
		return JobView{ID: id, Output: slot.output, Err: slot.err}
	case <-timer.C:
		slot.mu.Lock()
		if slot.done {
			out, err := slot.output, slot.err
			slot.mu.Unlock()
			cancel()
			return JobView{ID: id, Output: out, Err: err}
		}
		slot.background = true
		slot.mu.Unlock()
		return JobView{ID: id, Background: true}
	case <-ctx.Done():
		cancel()
		return JobView{ID: id, Err: ctx.Err()}
	}
}
