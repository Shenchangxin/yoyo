// executor_test.go 锁住工人槽、派工启动，以及后台命令完成后的报告。

package superagent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// TestHumanSlotAndBusyLaunch 锁住普通人槽满后，人还可以多占一个，第二张普通派工会返回 ErrBusy。
func TestHumanSlotAndBusyLaunch(t *testing.T) {
	slots := NewSlots(1)
	if fresh, err := slots.Reserve("a", "launch"); err != nil || !fresh {
		t.Fatal(err, fresh)
	}
	slots.Running("a")
	if _, err := slots.Reserve("b", "launch"); !errors.Is(err, ErrBusy) || !errors.Is(err, ErrLater) {
		t.Fatal(err)
	}
	if fresh, err := slots.Reserve("c", "person"); err != nil || !fresh {
		t.Fatal(err, fresh)
	}
	if _, err := slots.Reserve("d", "person"); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	slots.Free("c")
	if slots.Used() != 1 {
		t.Fatal(slots.Used())
	}

	eng := NewEngine(autoPolicy())
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	model := scriptModel{fn: func(context.Context, dsh.Request) (dsh.Assistant, error) {
		once.Do(func() { close(started) })
		<-release
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{
			ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"ok"}`,
		}}}, nil
	}}
	ex := NewExecutor(eng, NewSlots(1), model)
	first := acceptOn(t, eng, "m1", "room-a")
	second := acceptOn(t, eng, "m2", "room-b")
	if err := ex.Launch(first); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("model did not start")
	}
	if err := ex.Launch(second); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	close(release)
	waitOutcome(t, eng, first, "completed")
	ticket, _ := eng.Ticket(second)
	if ticket.Ended != nil || ticket.Assignments[0].State != "pending" {
		t.Fatalf("second %+v", ticket)
	}
}

// TestLaunchReportEndsTicket 锁住接受后的派工会跑到 ready，工单以 completed 结束。
func TestLaunchReportEndsTicket(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(2), scriptModel{fn: func(context.Context, dsh.Request) (dsh.Assistant, error) {
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{
			ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"done"}`,
		}}}, nil
	}})
	eng.OnLaunch = func(id string) {
		if err := ex.Launch(id); err != nil {
			t.Errorf("launch %s: %v", id, err)
		}
	}
	id := accept(t, eng, "m1")
	waitOutcome(t, eng, id, "completed")
}

// TestBackgroundShellThenReport 锁住超过前台预算的命令会在完成后把输出送回，再由报告结束工单。
func TestBackgroundShellThenReport(t *testing.T) {
	eng := NewEngine(autoPolicy())
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	ex := NewExecutor(eng, NewSlots(1), scriptModel{fn: func(_ context.Context, req dsh.Request) (dsh.Assistant, error) {
		last := dsh.Message{}
		if len(req.Messages) > 0 {
			last = req.Messages[len(req.Messages)-1]
		}
		if last.Role == "user" && strings.Contains(last.Text, "finished") {
			return dsh.Assistant{ToolCalls: []dsh.ToolCall{{
				ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"saw job"}`,
			}}}, nil
		}
		if last.Role == "tool" && strings.Contains(last.Text, "output") {
			return dsh.Assistant{ToolCalls: []dsh.ToolCall{{
				ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"saw output"}`,
			}}}, nil
		}
		if last.Role == "tool" && strings.Contains(last.Text, "background") {
			return dsh.Assistant{Text: "ack"}, nil
		}
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{
			ID: "s1", Name: "shell", Arguments: `{"command":"echo hi"}`,
		}}}, nil
	}})
	ex.Foreground = 30 * time.Millisecond
	ex.Shell = func(context.Context, string) (string, error) {
		once.Do(func() { close(started) })
		<-release
		return "output", nil
	}
	eng.OnLaunch = func(id string) { _ = ex.Launch(id) }
	id := accept(t, eng, "m1")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("shell did not start")
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	waitOutcome(t, eng, id, "completed")
}

// waitOutcome 等到 eng 里 ticketID 的结束结果是 outcome。超过三秒就失败。
func waitOutcome(t *testing.T, eng *Engine, ticketID, outcome string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ticket, ok := eng.Ticket(ticketID)
		if ok && ticket.Ended != nil && ticket.Ended.Outcome == outcome {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	ticket, _ := eng.Ticket(ticketID)
	t.Fatalf("ticket %+v", ticket)
}

// scriptModel 把测试函数当成一次模型完成。
type scriptModel struct {
	fn func(context.Context, dsh.Request) (dsh.Assistant, error)
}

// Complete 调用 fn。ctx 与 req 原样传进去。
func (m scriptModel) Complete(ctx context.Context, req dsh.Request) (dsh.Assistant, error) {
	return m.fn(ctx, req)
}
