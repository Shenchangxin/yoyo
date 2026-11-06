package superagent

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// TestDeliverPendingKeepsWordsWhenNoSlotIsFree 锁住槽满时人的话不被标成已送达，空出槽后下一次投递还能送出去。
func TestDeliverPendingKeepsWordsWhenNoSlotIsFree(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	model := scriptModel{fn: func(context.Context, dsh.Request) (dsh.Assistant, error) {
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"waiting","summary":"收到"}`}}}, nil
	}}
	slots := NewSlots(1)
	ex := NewExecutor(eng, slots, model)
	ex.mu.Lock()
	ex.agents[id] = dsh.New("ticket-"+id, model, dsh.NewRegistry(), dsh.Options{})
	ex.mu.Unlock()
	_, first := eng.PendingInputs(id)
	eng.AckInputs(id, first)
	if err := eng.Say(id, "用上海的数据"); err != nil {
		t.Fatal(err)
	}
	for _, other := range []string{"a", "b"} {
		if _, err := slots.Reserve(other, "person"); err != nil {
			t.Fatal(err)
		}
	}
	if err := ex.DeliverPending(id); !errors.Is(err, ErrBusy) {
		t.Fatalf("busy = %v", err)
	}
	if texts, _ := eng.PendingInputs(id); len(texts) != 1 || texts[0] != "用上海的数据" {
		t.Fatalf("the words were dropped: %v", texts)
	}
	slots.Free("a")
	if err := ex.DeliverPending(id); err != nil {
		t.Fatal(err)
	}
	if texts, _ := eng.PendingInputs(id); len(texts) != 0 {
		t.Fatalf("still pending: %v", texts)
	}
}

// TestPauseParksTheTurnAndTheNextWordsResumeIt 锁住先停下只停这一轮：工单不结束，不再催工人做下去，人再说一句就接着做。
func TestPauseParksTheTurnAndTheNextWordsResumeIt(t *testing.T) {
	eng := NewEngine(autoPolicy())
	started := make(chan struct{})
	var calls atomic.Int32
	model := scriptModel{fn: func(ctx context.Context, _ dsh.Request) (dsh.Assistant, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			return dsh.Assistant{}, ctx.Err()
		}
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"按新说法做完了"}`}}}, nil
	}}
	ex := NewExecutor(eng, NewSlots(1), model)
	id := acceptOn(t, eng, "m1", "room-a")
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("model did not start")
	}
	if !ex.Pause(id) {
		t.Fatal("pause should reach a running worker")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		ticket, _ := eng.Ticket(id)
		reports := ticket.Assignments[len(ticket.Assignments)-1].Reports
		ex.mu.Lock()
		live := ex.live[id]
		ex.mu.Unlock()
		if len(reports) > 0 && reports[len(reports)-1].Summary == pausedSummary && !live {
			if ticket.Ended != nil {
				t.Fatalf("pause ended the ticket: %+v", ticket.Ended)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("not parked: reports=%+v live=%v", reports, live)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("the worker was nudged on after pause: %d calls", n)
	}
	if ex.Pause(id) {
		t.Fatal("nothing is running to pause")
	}
	if !ex.Deliver(id, "改用上海的数据") {
		t.Fatal("deliver after pause")
	}
	waitOutcome(t, eng, id, "completed")
}

// TestNoteWakesTheWorker 锁住路由记的备注像别的输入一样送给工人，不等下一句人话。
func TestNoteWakesTheWorker(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	var woke []string
	eng.OnInput = func(_ string, text string) { woke = append(woke, text) }
	if err := eng.Note(id, "截止改到周五"); err != nil {
		t.Fatal(err)
	}
	if len(woke) != 1 || woke[0] != "截止改到周五" {
		t.Fatalf("woke = %v", woke)
	}
}

// TestInferredAssignmentNamesItsRules 锁住按规则主动接下的工单，派工正文写明规则；人明确交代的不写。任务页不显示这一行。
func TestInferredAssignmentNamesItsRules(t *testing.T) {
	policy := autoPolicy()
	policy.Rules = []Rule{{ID: "copy", Text: "文案小改可以直接做", Enabled: true}}
	eng := NewEngine(policy)
	msg := human("m1", "首页标题改成「浏览网页」")
	msg.Addressed = false
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	d, err := eng.DecideMatched("m1", "accept", "copy", []string{"copy"})
	if err != nil || d.TicketID == "" {
		t.Fatalf("decide %+v %v", d, err)
	}
	ex := NewExecutor(eng, NewSlots(1), nil)
	ticket, _ := eng.Ticket(d.TicketID)
	line := ex.mandateLine(ticket)
	if !strings.Contains(line, "文案小改可以直接做") || !strings.Contains(line, "dropped") {
		t.Fatalf("mandate = %q", line)
	}
	body := assignmentText(ticket, "", line)
	if !strings.Contains(body, line) {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(assignmentBody(body), mandatePrefix) {
		t.Fatal("the task page should not show the mandate line")
	}
	explicit := acceptOn(t, eng, "m2", "room-b")
	plain, _ := eng.Ticket(explicit)
	if ex.mandateLine(plain) != "" {
		t.Fatal("explicit work carries no rule line")
	}
}
