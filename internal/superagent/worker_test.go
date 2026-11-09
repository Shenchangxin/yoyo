// worker_test.go 锁住 Worker 缝：外部工人经 Opener 打开、用 orchestrator 工具报告、只催一次、插话退回排队、按 agent 限并行、工单结束后关掉。

package superagent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// fakeWorker 是测试用的外部工人。每条消息同步跑成一轮：记下消息，交给 onTurn，再写回合结束。
type fakeWorker struct {
	mu      sync.Mutex
	session *dsh.Session
	tools   map[string]dsh.Tool
	turn    int
	got     []dsh.UserMessage
	steers  int
	closed  atomic.Bool
	onTurn  func(w *fakeWorker, msg dsh.UserMessage)
}

// newFakeWorker 用 spec 里的工具建一个假工人。onTurn 为 nil 时这一轮什么都不做。
func newFakeWorker(spec WorkerSpec, onTurn func(w *fakeWorker, msg dsh.UserMessage)) *fakeWorker {
	tools := map[string]dsh.Tool{}
	for _, tool := range spec.Tools {
		tools[tool.Name] = tool
	}
	return &fakeWorker{session: dsh.NewSession(), tools: tools, onTurn: onTurn}
}

// Followup 把 msg 跑成一轮。
func (w *fakeWorker) Followup(msg dsh.UserMessage) {
	w.mu.Lock()
	w.turn++
	turn := w.turn
	w.got = append(w.got, msg)
	w.mu.Unlock()
	w.session.Append(dsh.EventUser, turn, 0, msg)
	if w.onTurn != nil {
		w.onTurn(w, msg)
	}
	w.session.Append(dsh.EventTurnEnd, turn, 0, dsh.TurnEnd{Kind: "completed"})
}

// Steer 记一次插话并返回假，像 ACP 一样不支持。
func (w *fakeWorker) Steer(dsh.UserMessage) bool {
	w.mu.Lock()
	w.steers++
	w.mu.Unlock()
	return false
}

// Cancel 什么都不做；假工人的一轮是同步的。
func (w *fakeWorker) Cancel(dsh.CancelCause, bool) {}

// WhenIdle 立刻返回。
func (w *fakeWorker) WhenIdle() {}

// Status 总是 idle。
func (w *fakeWorker) Status() string { return "idle" }

// Session 返回假工人的日志。
func (w *fakeWorker) Session() *dsh.Session { return w.session }

// Restore 把 events 灌进日志，回合号接在 turn 后面。
func (w *fakeWorker) Restore(events []dsh.Event, turn int) {
	w.session.Restore(events)
	w.mu.Lock()
	w.turn = turn
	w.mu.Unlock()
}

// Caps 是外部工人的能力：只催一次，不查计划。
func (w *fakeWorker) Caps() WorkerCaps { return WorkerCaps{Nudges: 1, External: true} }

// Close 记下已经关掉。
func (w *fakeWorker) Close() error {
	w.closed.Store(true)
	return nil
}

// report 调用工单的 orchestrator_report。
func (w *fakeWorker) report(state, summary string) dsh.Result {
	tool := w.tools["orchestrator_report"]
	res, _ := tool.Run(context.Background(), dsh.ToolCall{ID: "r1", Name: tool.Name, Arguments: fmt.Sprintf(`{"state":%q,"summary":%q}`, state, summary)})
	return res
}

// messages 返回假工人收到的全部消息。
func (w *fakeWorker) messages() []dsh.UserMessage {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]dsh.UserMessage(nil), w.got...)
}

// setAgent 把 eng 里 ticketID 交给 agent。
func setAgent(eng *Engine, ticketID, agent string) {
	eng.mu.Lock()
	defer eng.mu.Unlock()
	eng.tickets[ticketID].Agent = agent
}

// openWith 返回一个 Opener，每次打开都用 onTurn 建假工人，并把它送进 out。
func openWith(out chan<- *fakeWorker, onTurn func(w *fakeWorker, msg dsh.UserMessage)) Opener {
	return func(_ context.Context, spec WorkerSpec) (Worker, error) {
		w := newFakeWorker(spec, onTurn)
		if out != nil {
			out <- w
		}
		return w, nil
	}
}

// waitFor 等到 ok 为真，超过三秒就失败。
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for " + what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// lastSummary 返回 ticketID 最后一份报告的摘要。
func lastSummary(eng *Engine, ticketID string) string {
	ticket, _ := eng.Ticket(ticketID)
	if len(ticket.Assignments) == 0 {
		return ""
	}
	reports := ticket.Assignments[len(ticket.Assignments)-1].Reports
	if len(reports) == 0 {
		return ""
	}
	return reports[len(reports)-1].Summary
}

// TestOpenerWorkerReportsReady 锁住外部 agent 的工单走 Opener，拿到五个 orchestrator 工具，ready 不受计划文件挡住并结束工单。
func TestOpenerWorkerReportsReady(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(1), nil)
	ex.Root = t.TempDir()
	specs := make(chan WorkerSpec, 1)
	ex.Opener = func(_ context.Context, spec WorkerSpec) (Worker, error) {
		specs <- spec
		return newFakeWorker(spec, func(w *fakeWorker, msg dsh.UserMessage) {
			if msg.Source == "assignment" {
				if res := w.report("ready", "做完了"); res.IsError {
					t.Errorf("report: %s", res.Text)
				}
			}
		}), nil
	}
	id := accept(t, eng, "m1")
	setAgent(eng, id, "codex")
	dir, err := ex.ensureWorkDir(id)
	if err != nil {
		t.Fatal(err)
	}
	runtime.WritePlanFile(dir, "ticket-"+id, "1. [in_progress] 外部工人不写这个文件\n")
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	waitOutcome(t, eng, id, "completed")
	spec := <-specs
	if spec.Agent != "codex" || spec.TicketID != id || spec.Dir != dir || len(spec.Tools) != 5 || spec.Tools[0].Name != "orchestrator_report" {
		t.Fatalf("spec = %+v", spec)
	}
}

// TestExternalWorkerNudgedOnce 锁住外部工人一轮没写报告时只被催一次，然后记成在等下一条消息。
func TestExternalWorkerNudgedOnce(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(1), nil)
	opened := make(chan *fakeWorker, 1)
	ex.Opener = openWith(opened, nil)
	id := accept(t, eng, "m1")
	setAgent(eng, id, "codex")
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	w := <-opened
	waitFor(t, "settle", func() bool {
		ex.mu.Lock()
		live := ex.live[id]
		ex.mu.Unlock()
		return !live && lastSummary(eng, id) == "这一轮做完了，在等下一条消息"
	})
	got := w.messages()
	if len(got) != 2 || got[0].Source != "assignment" || got[1].Source != "retry" || got[1].Text != workerContinueNudge {
		t.Fatalf("messages = %+v", got)
	}
}

// TestDeliverFallsBackToFollowup 锁住工人不支持插话时，人的下一句改为排在这一轮后面，仍然送到。
func TestDeliverFallsBackToFollowup(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(1), nil)
	opened := make(chan *fakeWorker, 1)
	ex.Opener = openWith(opened, func(w *fakeWorker, msg dsh.UserMessage) {
		switch msg.Source {
		case "assignment":
			w.report("waiting", "收到")
		case "human":
			w.report("ready", "按新说法做完了")
		}
	})
	id := accept(t, eng, "m1")
	setAgent(eng, id, "codex")
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	w := <-opened
	waitFor(t, "first turn", func() bool {
		ex.mu.Lock()
		defer ex.mu.Unlock()
		return !ex.live[id] && len(w.messages()) > 0
	})
	if !ex.Deliver(id, "改用上海的数据") {
		t.Fatal("deliver")
	}
	waitOutcome(t, eng, id, "completed")
	got := w.messages()
	last := got[len(got)-1]
	w.mu.Lock()
	steers := w.steers
	w.mu.Unlock()
	if steers != 1 || last.Source != "human" || last.Text != "改用上海的数据" {
		t.Fatalf("steers %d last %+v", steers, last)
	}
}

// TestAgentLimitBusy 锁住同一个 agent 跑满上限后 Launch 返回 ErrBusy，别的 agent 不受影响，空出来后能再开。
func TestAgentLimitBusy(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(3), nil)
	ex.AgentLimit = func(agent string) int {
		if agent == "codex" {
			return 1
		}
		return 0
	}
	release := make(chan struct{})
	ex.Opener = openWith(nil, func(w *fakeWorker, msg dsh.UserMessage) {
		if msg.Source == "assignment" {
			<-release
			w.report("ready", "ok")
		}
	})
	first := acceptOn(t, eng, "m1", "room-a")
	second := acceptOn(t, eng, "m2", "room-b")
	third := acceptOn(t, eng, "m3", "room-c")
	setAgent(eng, first, "codex")
	setAgent(eng, second, "codex")
	setAgent(eng, third, "claude")
	if err := ex.Launch(first); err != nil {
		t.Fatal(err)
	}
	if err := ex.Launch(second); !errors.Is(err, ErrBusy) {
		t.Fatalf("second = %v", err)
	}
	if err := ex.Launch(third); err != nil {
		t.Fatalf("another agent is limited separately: %v", err)
	}
	if run := ex.AgentRunning(); run["codex"] != 1 || run["claude"] != 1 {
		t.Fatalf("running = %v", run)
	}
	close(release)
	waitOutcome(t, eng, first, "completed")
	waitFor(t, "codex slot", func() bool { return ex.Launch(second) == nil })
	waitOutcome(t, eng, second, "completed")
	waitOutcome(t, eng, third, "completed")
}

// TestClosedWhenTicketEnds 锁住工单做完后外部工人被关掉并移出表；空闲时被人停掉的，下一次清扫也会关掉。
func TestClosedWhenTicketEnds(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(2), nil)
	opened := make(chan *fakeWorker, 2)
	done := acceptOn(t, eng, "m1", "room-a")
	idle := acceptOn(t, eng, "m2", "room-b")
	setAgent(eng, done, "codex")
	setAgent(eng, idle, "codex")

	ex.Opener = openWith(opened, func(w *fakeWorker, msg dsh.UserMessage) {
		if msg.Source == "assignment" {
			w.report("ready", "ok")
		}
	})
	if err := ex.Launch(done); err != nil {
		t.Fatal(err)
	}
	finished := <-opened
	waitOutcome(t, eng, done, "completed")
	waitFor(t, "close after ready", func() bool { return finished.closed.Load() && !ex.HasAgent(done) })

	ex.Opener = openWith(opened, func(w *fakeWorker, msg dsh.UserMessage) {
		if msg.Source == "assignment" {
			w.report("waiting", "收到")
		}
	})
	if err := ex.Launch(idle); err != nil {
		t.Fatal(err)
	}
	parked := <-opened
	waitFor(t, "parked", func() bool {
		ex.mu.Lock()
		defer ex.mu.Unlock()
		return !ex.live[idle] && ex.workers[idle] != nil
	})
	if parked.closed.Load() {
		t.Fatal("a waiting ticket keeps its worker")
	}
	if err := eng.Stop(idle, "不用了"); err != nil {
		t.Fatal(err)
	}
	ex.Sweep()
	if !parked.closed.Load() || ex.HasAgent(idle) {
		t.Fatal("the sweep should close a stopped ticket's worker")
	}
}
