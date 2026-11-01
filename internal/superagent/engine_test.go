// engine_test.go 锁住接收、评估、报告和效果队列的契约。

package superagent

import (
	"errors"
	"testing"
	"time"
)

// TestReceiveDedupesAndFilters 锁住重复接收、空消息和不在听的来源。
func TestReceiveDedupesAndFilters(t *testing.T) {
	eng := NewEngine(autoPolicy())
	if eng.Receive(human("m1", "fix")) != "stored" {
		t.Fatal("stored")
	}
	if eng.Receive(human("m1", "fix")) != "duplicate" {
		t.Fatal("duplicate")
	}
	if eng.Receive(human("m2", "  ")) != "filtered" {
		t.Fatal("empty")
	}
	if eng.Receive(human("m2", "again")) != "duplicate" {
		t.Fatal("filtered id is consumed")
	}
	eng.Listen("other", true)
	skipped := human("m3", "nope")
	skipped.Source = "desk"
	if eng.Receive(skipped) != "filtered" {
		t.Fatal("not listened")
	}
}

// TestAcceptCreatesTicket 锁住显式接受会开工单、授权、待运行派工、订阅和回执。
func TestAcceptCreatesTicket(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var launched []string
	eng.OnLaunch = func(id string) { launched = append(launched, id) }
	if eng.Receive(human("m1", "fix the label")) != "stored" {
		t.Fatal("stored")
	}
	d, err := eng.Decide("m1", "accept", "ok")
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != "accept" || d.TicketID == "" || len(launched) != 1 || launched[0] != d.TicketID {
		t.Fatalf("decision %+v launches %v", d, launched)
	}
	again, err := eng.Decide("m1", "accept", "again")
	if err != nil || again.ID != d.ID {
		t.Fatalf("second accept %+v %v", again, err)
	}
	if len(launched) != 1 {
		t.Fatal("second accept launched")
	}
	ticket, ok := eng.Ticket(d.TicketID)
	if !ok || ticket.Mandate.Strength != "explicit" || ticket.Mandate.Sender != "ada" || ticket.Origin != "m1" {
		t.Fatalf("ticket %+v", ticket)
	}
	if len(ticket.Assignments) != 1 || ticket.Assignments[0].State != "pending" {
		t.Fatalf("assignment %+v", ticket.Assignments)
	}
	th, ok := eng.Thread("room")
	if !ok || !contains(th.Subs, ticket.ID) || len(th.Queued) != 0 {
		t.Fatalf("thread %+v", th)
	}
	box := eng.Outbox()
	if len(box) != 1 || box[0].Kind != "acknowledgement" || box[0].Text != "已接单" {
		t.Fatalf("outbox %+v", box)
	}
}

// TestFollowupStaysWithTicket 锁住人的后续消息不排队，主会话也不能再接成新工单。
func TestFollowupStaysWithTicket(t *testing.T) {
	eng := NewEngine(autoPolicy())
	eng.Receive(human("m1", "fix"))
	if _, err := eng.Decide("m1", "accept", "ok"); err != nil {
		t.Fatal(err)
	}
	if eng.Receive(human("m2", "also this")) != "stored" {
		t.Fatal("stored")
	}
	tr, _ := eng.Trigger("m2")
	if tr.Queued || len(tr.Reached) != 1 {
		t.Fatalf("follow-up %+v", tr)
	}
	th, _ := eng.Thread("room")
	if contains(th.Queued, "m2") {
		t.Fatal("queued")
	}
	if _, err := eng.Decide("m2", "accept", "no"); err == nil || err.Error() != "workerDecides" {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(tr.Reached[0])
	if len(ticket.Inputs) != 2 {
		t.Fatalf("inputs %+v", ticket.Inputs)
	}
}

// TestBotDoesNotReachEndedTicket 锁住机器人消息会排队，并且到不了已经结束的工单。
func TestBotDoesNotReachEndedTicket(t *testing.T) {
	eng := NewEngine(autoPolicy())
	eng.Receive(human("m1", "fix"))
	d, err := eng.Decide("m1", "accept", "ok")
	if err != nil {
		t.Fatal(err)
	}
	live := bot("b1", "ci")
	if eng.Receive(live) != "stored" {
		t.Fatal("stored")
	}
	tr, _ := eng.Trigger("b1")
	if !tr.Queued || len(tr.Reached) != 1 {
		t.Fatalf("live bot %+v", tr)
	}
	if err := eng.Report(d.TicketID, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	if eng.Receive(bot("b2", "ci again")) != "stored" {
		t.Fatal("stored")
	}
	ended, _ := eng.Trigger("b2")
	if !ended.Queued || len(ended.Reached) != 0 {
		t.Fatalf("ended bot %+v", ended)
	}
}

// TestReadyWithoutFollowsCompletes 锁住没有跟随线程的 ready 以 completed 结束。
func TestReadyWithoutFollowsCompletes(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.Report(id, Report{State: "progress", Summary: "looking"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if ticket.Ended != nil || TicketStage(&ticket).Name != "working" {
		t.Fatalf("progress %+v %+v", ticket.Ended, TicketStage(&ticket))
	}
	if err := eng.AssignmentStarted(id, ticket.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "waiting", Summary: "shell"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ = eng.Ticket(id)
	if TicketStage(&ticket).Name != "idle" {
		t.Fatalf("stage %+v", TicketStage(&ticket))
	}
	if err := eng.Report(id, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ = eng.Ticket(id)
	if ticket.Ended == nil || ticket.Ended.Outcome != "completed" || ticket.Ended.By != "executor" {
		t.Fatalf("end %+v", ticket.Ended)
	}
	if err := eng.Report(id, Report{State: "progress", Summary: "late"}); err == nil || err.Error() != "ticketEnded" {
		t.Fatal(err)
	}
}

// TestReadyFollowsThreadClose 锁住有自己打开的线程时，ready 要等它关闭才结束。
func TestReadyFollowsThreadClose(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.BindThread(id, "pr-1", true); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "ready", Summary: "pr open"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if ticket.Ended != nil || TicketStage(&ticket).Action != "review" {
		t.Fatalf("still open %+v %+v", ticket.Ended, TicketStage(&ticket))
	}
	eng.ThreadClosed("pr-1", "merged", "2026-10-31T00:00:00Z")
	ticket, _ = eng.Ticket(id)
	if ticket.Ended == nil || ticket.Ended.Outcome != "merged" {
		t.Fatalf("merged %+v", ticket.Ended)
	}
}

// TestBlockedThenHumanThenRetry 锁住失败报告等人，人的消息解开等待，重试新开派工。
func TestBlockedThenHumanThenRetry(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	ticket, _ := eng.Ticket(id)
	if err := eng.AssignmentStarted(id, ticket.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "blocked", Summary: "which color"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ = eng.Ticket(id)
	if !CanRetry(&ticket) || TicketStage(&ticket).Action != "blocked" {
		t.Fatalf("blocked %+v", TicketStage(&ticket))
	}
	eng.Receive(human("m2", "blue"))
	ticket, _ = eng.Ticket(id)
	if CanRetry(&ticket) || TicketStage(&ticket).Name != "working" {
		t.Fatalf("released %+v", TicketStage(&ticket))
	}
	if err := eng.Retry(id); err == nil || err.Error() != "notRetryable" {
		t.Fatal(err)
	}

	eng = NewEngine(autoPolicy())
	var launches int
	eng.OnLaunch = func(string) { launches++ }
	id = accept(t, eng, "m1")
	if launches != 1 {
		t.Fatal(launches)
	}
	ticket, _ = eng.Ticket(id)
	if err := eng.AssignmentStarted(id, ticket.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "failed", Summary: "build broke"}); err != nil {
		t.Fatal(err)
	}
	if err := eng.Retry(id); err != nil {
		t.Fatal(err)
	}
	if launches != 2 {
		t.Fatal(launches)
	}
	ticket, _ = eng.Ticket(id)
	if len(ticket.Assignments) != 2 || ticket.Assignments[0].State != "ended" || ticket.Assignments[1].State != "pending" {
		t.Fatalf("assignments %+v", ticket.Assignments)
	}
	if TicketStage(&ticket).Action != "queued" {
		t.Fatalf("stage %+v", TicketStage(&ticket))
	}
}

// TestObserveRecordsWouldAccept 锁住观察模式下的推断接受只记 observed，不开工单。
func TestObserveRecordsWouldAccept(t *testing.T) {
	eng := NewEngine(Policy{Version: 3, Mode: "observe"})
	msg := human("m1", "maybe fix")
	msg.Sender = "stranger"
	eng.Receive(msg)
	d, err := eng.Decide("m1", "accept", "would")
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != "ignore" || d.Reason != "observed" || d.TicketID != "" {
		t.Fatalf("decision %+v", d)
	}
	if _, ok := eng.Ticket("tkt-1"); ok {
		t.Fatal("ticket")
	}
}

// TestExplicitSurvivesModeOff 锁住名单内并且点名的消息在 off 模式下仍然可以接受。
func TestExplicitSurvivesModeOff(t *testing.T) {
	eng := NewEngine(Policy{Version: 1, Mode: "off", Allowed: map[string]bool{"ada": true}})
	eng.Receive(human("m1", "fix"))
	d, err := eng.Decide("m1", "accept", "ok")
	if err != nil || d.Kind != "accept" {
		t.Fatal(err, d)
	}
	other := human("m2", "please")
	other.Sender = "stranger"
	other.Threads = []string{"other"}
	other.Primary = "other"
	eng.Receive(other)
	if _, err := eng.Decide("m2", "accept", "no"); err == nil || err.Error() != "mode_off" {
		t.Fatal(err)
	}
}

// TestPauseSkipsLaunchAndResume 锁住暂停不启动派工，人的消息先留在已结束的工单上。
func TestPauseSkipsLaunchAndResume(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var launches int
	eng.OnLaunch = func(string) { launches++ }
	id := accept(t, eng, "m1")
	p := autoPolicy()
	p.Paused = true
	eng.SetPolicy(p)
	if err := eng.Report(id, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	before := launches
	eng.Receive(human("m2", "one more"))
	tr, _ := eng.Trigger("m2")
	ticket, _ := eng.Ticket(id)
	if tr.Queued || ticket.Ended == nil || launches != before {
		t.Fatalf("queued %v ended %+v launches %d", tr.Queued, ticket.Ended, launches)
	}
}

// TestQuestionCannotRideReady 锁住问题和 ready 不能写在同一次报告里。
func TestQuestionCannotRideReady(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	err := eng.Report(id, Report{State: "ready", Summary: "done", Question: "ship it?"})
	if err == nil || err.Error() != "questionWithResult" {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "progress", Summary: "need a call", Question: "ship it?"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if TicketStage(&ticket).Action != "answer" || ticket.Attention != "ship it?" {
		t.Fatalf("stage %+v attention %q", TicketStage(&ticket), ticket.Attention)
	}
}

// TestDroppedEndsTicket 锁住 dropped 结束工单。
func TestDroppedEndsTicket(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.Report(id, Report{State: "dropped", Summary: "not ours"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if ticket.Ended == nil || ticket.Ended.Outcome != "dropped" {
		t.Fatalf("%+v", ticket.Ended)
	}
}

// TestEffectCoalesceBackoffAndLater 锁住同键合并、失败退避，以及 ErrLater 不计失败。
func TestEffectCoalesceBackoffAndLater(t *testing.T) {
	q := NewQueue()
	var got []string
	q.Handle("launch", 1, func(payload any) error {
		got = append(got, payload.(string))
		return nil
	})
	q.Enqueue("launch", "t", "a")
	q.Enqueue("launch", "t", "b")
	q.Enqueue("launch", "u", "u")
	if q.Pending() != 2 {
		t.Fatal(q.Pending())
	}
	now := time.Now().Add(time.Second)
	q.Pump(now)
	if len(got) != 1 || got[0] != "b" {
		t.Fatalf("first pump %v", got)
	}
	q.Pump(now)
	if len(got) != 2 || got[1] != "u" {
		t.Fatalf("second pump %v", got)
	}

	later := NewQueue()
	calls := 0
	later.Handle("launch", 1, func(any) error {
		calls++
		return ErrLater
	})
	later.Enqueue("launch", "k", "a")
	at := time.Now().Add(time.Second)
	later.Pump(at)
	later.Pump(at)
	if calls != 1 || !later.Healthy() || later.Pending() != 1 {
		t.Fatalf("calls %d healthy %v pending %d", calls, later.Healthy(), later.Pending())
	}

	failing := NewQueue()
	failing.Handle("launch", 1, func(any) error { return errors.New("boom") })
	failing.Enqueue("launch", "k", "a")
	t0 := time.Now()
	failing.Pump(t0.Add(time.Second))
	if !failing.Healthy() {
		t.Fatal("one failure")
	}
	failing.Pump(t0.Add(4 * time.Second))
	failing.Pump(t0.Add(30 * time.Second))
	if failing.Healthy() {
		t.Fatal("three failures")
	}
}

// autoPolicy 返回测试用的自动策略。发送者 ada 在名单里。
func autoPolicy() Policy {
	return Policy{Version: 1, Mode: "auto", Allowed: map[string]bool{"ada": true}}
}

// human 造一条编号为 id、正文为 text 的人的消息。线程是 room，发送者是 ada。
func human(id, text string) Trigger {
	return Trigger{ID: id, Source: "desk", Threads: []string{"room"}, Primary: "room", Human: true, Addressed: true, Sender: "ada", Text: text}
}

// bot 造一条编号为 id、正文为 text 的机器人消息。线程是 room。
func bot(id, text string) Trigger {
	return Trigger{ID: id, Source: "desk", Threads: []string{"room"}, Primary: "room", Sender: "ci", Text: text}
}

// accept 在 eng 上接受编号为 id 的人的消息。线程是 room。返回工单编号。
func accept(t *testing.T, eng *Engine, id string) string {
	t.Helper()
	return acceptOn(t, eng, id, "room")
}

// acceptOn 在 eng 上接受编号为 id 的人的消息。线程是 thread。返回工单编号。
func acceptOn(t *testing.T, eng *Engine, id, thread string) string {
	t.Helper()
	msg := human(id, "fix")
	msg.Threads = []string{thread}
	msg.Primary = thread
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	d, err := eng.Decide(id, "accept", "ok")
	if err != nil {
		t.Fatal(err)
	}
	return d.TicketID
}

// TestReceiveStampsArrival 锁住收下的触发带上到达时刻，相同编号的第二次接收不改这个时刻。
func TestReceiveStampsArrival(t *testing.T) {
	eng := NewEngine(autoPolicy())
	if eng.Receive(human("m", "hello")) != "stored" {
		t.Fatal("stored")
	}
	snap, err := eng.Export()
	if err != nil {
		t.Fatal(err)
	}
	var first time.Time
	for _, tr := range snap.Triggers {
		if tr.ID == "m" {
			first = tr.At
		}
	}
	if first.IsZero() {
		t.Fatal("at")
	}
	if eng.Receive(human("m", "again")) != "duplicate" {
		t.Fatal("duplicate")
	}
	snap, err = eng.Export()
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range snap.Triggers {
		if tr.ID == "m" && !tr.At.Equal(first) {
			t.Fatal("kept")
		}
	}
}
