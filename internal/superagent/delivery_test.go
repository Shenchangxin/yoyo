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

// TestPauseParksARunningAssignmentWithNoLiveWorker 锁住派工还标着在做、工人却已经不在时，暂停直接记下停下，工单不结束。
func TestPauseParksARunningAssignmentWithNoLiveWorker(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(1), scriptModel{fn: func(context.Context, dsh.Request) (dsh.Assistant, error) {
		return dsh.Assistant{}, nil
	}})
	id := acceptOn(t, eng, "m-orphan", "room-a")
	if err := eng.AssignmentStarted(id, eng.LatestAssignment(id)); err != nil {
		t.Fatal(err)
	}
	if !ex.Pause(id) {
		t.Fatal("pause should park a running assignment with no live worker")
	}
	ticket, ok := eng.Ticket(id)
	if !ok || ticket.Ended != nil {
		t.Fatalf("ticket ended or missing: %+v", ticket.Ended)
	}
	reports := ticket.Assignments[len(ticket.Assignments)-1].Reports
	if len(reports) != 1 || reports[0].State != "waiting" || reports[0].Summary != pausedSummary {
		t.Fatalf("reports = %+v", reports)
	}
	if stage := TicketStage(&ticket); stage.Name != "idle" {
		t.Fatalf("stage = %+v", stage)
	}
	if ex.Pause(id) {
		t.Fatal("nothing is running to pause")
	}
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
	body := assignmentText(ticket, "", line, "")
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

// TestFollowUpOnAParkedTaskStaysInTheSameSession 锁住工人进程已经不在时，人的下一句回到原来的会话，只送这句，不重开派工，也不重发整段任务说明。
// 话还没送进去时，停止不会把这句记成停下。同一进程里的再下一句仍走这个工人。
func TestFollowUpOnAParkedTaskStaysInTheSameSession(t *testing.T) {
	sess := dsh.NewSession()
	sess.Append(dsh.EventUser, 1, 0, dsh.UserMessage{Text: "先做一版首页", Source: "assignment"})
	sess.Append(dsh.EventAssistant, 1, 1, dsh.Assistant{Text: "首页已经放好了"})
	sess.Append(dsh.EventTurnEnd, 1, 0, dsh.TurnEnd{Kind: "completed"})
	body, _ := encodeLog(sess.Events())

	eng := NewEngine(autoPolicy())
	var launches atomic.Int32
	eng.OnLaunch = func(string) { launches.Add(1) }
	id := accept(t, eng, "m1")
	setAgent(eng, id, "opencode")
	eng.SetPointer(id, "opencode", "sess-63")
	if err := eng.AssignmentStarted(id, eng.LatestAssignment(id)); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "waiting", Summary: "先停一下"}); err != nil {
		t.Fatal(err)
	}
	_, delivered := eng.PendingInputs(id)
	eng.AckInputs(id, delivered)
	opened := make(chan *fakeWorker, 1)
	ex := NewExecutor(eng, NewSlots(1), nil)
	ex.LogLoad = func(string) []byte { return body }
	ex.LogSave = func(string, []byte) {}
	ex.Opener = func(ctx context.Context, spec WorkerSpec) (Worker, error) {
		if spec.Agent != "opencode" || spec.Pointer != "sess-63" {
			t.Errorf("spec agent %q pointer %q", spec.Agent, spec.Pointer)
		}
		return openWith(opened, func(w *fakeWorker, msg dsh.UserMessage) {
			if msg.Source == "human" {
				w.say("接着做了")
			}
		})(ctx, spec)
	}
	rt := &Runtime{eng: eng, exec: ex}
	before := launches.Load()
	if err := eng.Say(id, "继续"); err != nil {
		t.Fatal(err)
	}
	if ex.Pause(id) {
		t.Fatal("a message that has not been delivered is not a running turn")
	}
	ticket, _ := eng.Ticket(id)
	if reports := ticket.Assignments[0].Reports; len(reports) != 1 || reports[0].Summary != "先停一下" {
		t.Fatalf("pause swallowed the new line: %+v", reports)
	}
	if err := rt.runDelivery(id); err != nil {
		t.Fatal(err)
	}
	if launches.Load() != before {
		t.Fatal("follow-up reopened the assignment")
	}
	w := <-opened
	waitFor(t, "reply", func() bool {
		ex.mu.Lock()
		live := ex.live[id]
		ex.mu.Unlock()
		ex.Slots.mu.Lock()
		held := ex.Slots.running[id] || ex.Slots.starting[id] != ""
		ex.Slots.mu.Unlock()
		return !live && !held && lastSummary(eng, id) == "接着做了"
	})
	if got := w.messages(); len(got) != 1 || got[0].Source != "human" || got[0].Text != "继续" {
		t.Fatalf("messages = %+v", got)
	}
	ticket, _ = eng.Ticket(id)
	if len(ticket.Assignments) != 1 || ticket.Ended != nil {
		t.Fatalf("assignments %d ended %+v", len(ticket.Assignments), ticket.Ended)
	}
	if texts, _ := eng.PendingInputs(id); len(texts) != 0 {
		t.Fatalf("still pending: %v", texts)
	}

	if err := eng.Say(id, "把标题改短"); err != nil {
		t.Fatal(err)
	}
	if err := rt.runDelivery(id); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "second line", func() bool {
		for _, msg := range w.messages() {
			if msg.Text == "把标题改短" && msg.Source == "human" {
				return true
			}
		}
		return false
	})
	select {
	case extra := <-opened:
		t.Fatalf("opened another worker with %+v", extra.messages())
	default:
	}
	ticket, _ = eng.Ticket(id)
	if launches.Load() != before || len(ticket.Assignments) != 1 || ticket.Ended != nil {
		t.Fatalf("second line split the session: launches %d assignments %d ended %+v", launches.Load()-before, len(ticket.Assignments), ticket.Ended)
	}
}

// TestFollowUpWithoutALogKeepsTheAssignment 锁住没有会话日志时，下一句仍留在原来的派工里，任务说明里带上这句，不另外重开一轮。
func TestFollowUpWithoutALogKeepsTheAssignment(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var launches atomic.Int32
	eng.OnLaunch = func(string) { launches.Add(1) }
	id := accept(t, eng, "m1")
	setAgent(eng, id, "opencode")
	if err := eng.AssignmentStarted(id, eng.LatestAssignment(id)); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "waiting", Summary: "先停一下"}); err != nil {
		t.Fatal(err)
	}
	opened := make(chan *fakeWorker, 1)
	ex := NewExecutor(eng, NewSlots(1), nil)
	ex.Opener = openWith(opened, func(w *fakeWorker, msg dsh.UserMessage) {
		if msg.Source == "assignment" {
			w.say("收到")
		}
	})
	before := launches.Load()
	if err := eng.Say(id, "继续"); err != nil {
		t.Fatal(err)
	}
	if err := (&Runtime{eng: eng, exec: ex}).runDelivery(id); err != nil {
		t.Fatal(err)
	}
	w := <-opened
	waitFor(t, "cold reply", func() bool {
		ex.mu.Lock()
		live := ex.live[id]
		ex.mu.Unlock()
		return !live && lastSummary(eng, id) == "收到"
	})
	got := w.messages()
	if launches.Load() != before || len(got) != 1 || got[0].Source != "assignment" || !strings.Contains(got[0].Text, "继续") || !strings.Contains(got[0].Text, "完成后调用 orchestrator_report。") {
		t.Fatalf("launches %d messages %+v", launches.Load()-before, got)
	}
	ticket, _ := eng.Ticket(id)
	if len(ticket.Assignments) != 1 || ticket.Ended != nil {
		t.Fatalf("assignments %d ended %+v", len(ticket.Assignments), ticket.Ended)
	}
}
