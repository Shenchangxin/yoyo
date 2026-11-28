package app

import (
	"os"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/lease"
)

// drainingDaemon 搭一个拿着租约的 daemon：只有租约和选主状态，不跑送达循环、例行和编排，
// 这样收尾的判断（drainStep / cancelDrain）可以一拍一拍地手动推进。busy 为真时放一个正在跑的会话回合。
func drainingDaemon(t *testing.T, busy bool) *App {
	t.Helper()
	a := &App{Home: &home.Dir{Root: t.TempDir()}}
	if err := os.MkdirAll(a.Home.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	h := lease.Holder{Role: "daemon", PID: os.Getpid(), Since: "2026-11-28T00:00:00Z"}
	l, err := lease.Try(a.leadPath(), h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(l.Release)
	a.lead.role = "daemon"
	a.lead.held, a.lead.heldAs, a.lead.leading = l, h, true
	if busy {
		a.mu.Lock()
		a.runs = map[string]*runSlot{"main": {started: time.Now()}}
		a.mu.Unlock()
	}
	return a
}

// finishRun 模拟那一轮会话回合跑完：放掉运行槽。
func finishRun(a *App) {
	a.mu.Lock()
	delete(a.runs, "main")
	a.mu.Unlock()
}

// TestDrainWaitsForRunningTurn 确认桌面打开时手上有回合在跑就先收尾：不让出租约，持有人记录标上 Draining；
// 回合跑完的下一拍才让出，记录随租约一起删掉。
func TestDrainWaitsForRunningTurn(t *testing.T) {
	a := drainingDaemon(t, true)
	now := time.Now()
	a.drainStep(now)
	if !a.leading() {
		t.Fatal("a running turn must keep the lead while draining")
	}
	if h, ok := lease.ReadHolder(a.leadPath()); !ok || !h.Draining || h.Since != "2026-11-28T00:00:00Z" {
		t.Fatalf("holder while draining = %+v %v", h, ok)
	}
	if s := a.LeadState(); s.Role != "lead" || !s.Draining {
		t.Fatalf("lead state while draining = %+v", s)
	}
	a.drainStep(now.Add(time.Minute))
	if !a.leading() {
		t.Fatal("still running after a minute: keep draining")
	}
	finishRun(a)
	a.drainStep(now.Add(2 * time.Minute))
	if a.leading() {
		t.Fatal("the turn finished: the lead must be handed over")
	}
	if _, ok := lease.ReadHolder(a.leadPath()); ok {
		t.Fatal("the holder record must go with the lease")
	}
	if s := a.LeadState(); s.Draining {
		t.Fatalf("not draining after stepping down: %+v", s)
	}
}

// TestDrainIdleStepsDownAtOnce 确认手上没有在跑的回合时，桌面一打开当拍就让出，和以前一样快。
func TestDrainIdleStepsDownAtOnce(t *testing.T) {
	a := drainingDaemon(t, false)
	a.drainStep(time.Now())
	if a.leading() {
		t.Fatal("idle daemon must step down on the first tick")
	}
}

// TestDrainGivesUpAfterMax 确认回合一直不结束时，等满 leadDrainMax 照样让出，不会让桌面一直接不过去。
func TestDrainGivesUpAfterMax(t *testing.T) {
	a := drainingDaemon(t, true)
	now := time.Now()
	a.drainStep(now)
	a.drainStep(now.Add(leadDrainMax - time.Second))
	if !a.leading() {
		t.Fatal("stepped down before the drain limit")
	}
	a.drainStep(now.Add(leadDrainMax))
	if a.leading() {
		t.Fatal("must step down once the drain limit is reached")
	}
}

// TestDrainCancelledWhenDesktopCloses 确认收尾中桌面又关掉时取消收尾：接着领头，持有人记录去掉 Draining；
// 之后桌面再打开时重新从第一拍开始计时。
func TestDrainCancelledWhenDesktopCloses(t *testing.T) {
	a := drainingDaemon(t, true)
	now := time.Now()
	a.drainStep(now)
	a.cancelDrain()
	// cancelDrain 把送达循环接了回来；测试里不让它跑下去。
	t.Cleanup(a.stopDeskLoop)
	t.Cleanup(a.stopSchedule)
	if !a.leading() {
		t.Fatal("cancelling the drain keeps the lead")
	}
	if h, ok := lease.ReadHolder(a.leadPath()); !ok || h.Draining {
		t.Fatalf("holder after cancel = %+v %v", h, ok)
	}
	if s := a.LeadState(); s.Draining {
		t.Fatalf("lead state after cancel = %+v", s)
	}
	// drainSince 归选主的锁管，不是会话槽的 a.mu。
	a.lead.mu.Lock()
	since := a.lead.drainSince
	a.lead.mu.Unlock()
	if !since.IsZero() {
		t.Fatal("drain clock must reset after cancel")
	}
	// 再收尾时从新的一拍开始算：离上一段开始已经超过上限也不该当拍让出。
	a.drainStep(now.Add(leadDrainMax + time.Minute))
	if !a.leading() {
		t.Fatal("a new drain must start its own clock")
	}
}
