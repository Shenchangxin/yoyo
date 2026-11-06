package superagent

import "testing"

// TestAcceptClosesOnlyAReview 锁住验收只收在等人看结果的工单，收下记成人确认做完；别的阶段拒绝，收过不能再收。
func TestAcceptClosesOnlyAReview(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.Accept(id, ""); err == nil || err.Error() != "notReview" {
		t.Fatalf("accept before any result = %v", err)
	}
	if err := eng.BindThread(id, "pr-1", true); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "ready", Summary: "pr open"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if TicketStage(&ticket).Action != "review" {
		t.Fatalf("stage = %+v", TicketStage(&ticket))
	}
	if err := eng.Accept(id, ""); err != nil {
		t.Fatal(err)
	}
	ticket, _ = eng.Ticket(id)
	if ticket.Ended == nil || ticket.Ended.Outcome != "completed" || ticket.Ended.By != "human" || ticket.Ended.Reason != "验收通过" {
		t.Fatalf("ended = %+v", ticket.Ended)
	}
	if err := eng.Accept(id, ""); err == nil {
		t.Fatal("accepting twice should fail")
	}
}

// TestContinuedFromTheDeskStillCompletes 锁住主会话继续过的工单，工人报做完就记成做完了。
// 主会话那条线程只是上下文，永远不会关，不能把工单挡在等验收。
func TestContinuedFromTheDeskStillCompletes(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	msg := human("d1", "标题改成新的")
	msg.Threads = []string{"desk:s1"}
	msg.Primary = "desk:s1"
	if got := eng.Receive(msg); got != "stored" {
		t.Fatalf("receive = %q", got)
	}
	if err := eng.Assign(id, "d1", "把首页标题改成新的"); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if len(ticket.Follows) != 1 || ticket.Follows[0].Thread != "desk:s1" || ticket.Follows[0].Own {
		t.Fatalf("follows = %+v", ticket.Follows)
	}
	if err := eng.Report(id, Report{State: "ready", Summary: "改好了"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ = eng.Ticket(id)
	if ticket.Ended == nil || ticket.Ended.Outcome != "completed" || ticket.Ended.By != "executor" {
		t.Fatalf("ended = %+v stage = %+v", ticket.Ended, TicketStage(&ticket))
	}
}

// TestRecoverSettlesStrandedReviews 锁住快照里只跟着上下文线程、却停在等验收的工单，启动时记成做完了；
// 等自己拉取请求合并的工单仍然留在等验收。
func TestRecoverSettlesStrandedReviews(t *testing.T) {
	eng := NewEngine(autoPolicy())
	stranded := acceptOn(t, eng, "m1", "room-a")
	pr := acceptOn(t, eng, "m2", "room-b")
	if err := eng.BindThread(pr, "pr-1", true); err != nil {
		t.Fatal(err)
	}
	if err := eng.AssignmentStarted(pr, latestAssignment(eng.tickets[pr]).ID); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(pr, Report{State: "ready", Summary: "pr open"}); err != nil {
		t.Fatal(err)
	}
	tk := eng.tickets[stranded]
	tk.Follows = append(tk.Follows, Follow{Thread: "desk:s1"})
	asg := latestAssignment(tk)
	asg.State = "running"
	eng.seq++
	asg.Reports = append(asg.Reports, Report{State: "ready", Summary: "改好了", Seq: eng.seq})
	if TicketStage(tk).Action != "review" {
		t.Fatalf("stage = %+v", TicketStage(tk))
	}
	if launched := eng.RecoverHost(); len(launched) != 0 {
		t.Fatalf("launched = %v", launched)
	}
	done, _ := eng.Ticket(stranded)
	if done.Ended == nil || done.Ended.Outcome != "completed" || done.Ended.Reason != "改好了" {
		t.Fatalf("stranded = %+v", done.Ended)
	}
	open, _ := eng.Ticket(pr)
	if open.Ended != nil || TicketStage(&open).Action != "review" {
		t.Fatalf("pr ticket = %+v %+v", open.Ended, TicketStage(&open))
	}
}

// TestSendingBackAReviewReopensTheRound 锁住验收没通过时，人的话让工单回到在做，而不是停在等验收。
func TestSendingBackAReviewReopensTheRound(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.BindThread(id, "pr-1", true); err != nil {
		t.Fatal(err)
	}
	if err := eng.AssignmentStarted(id, latestAssignment(eng.tickets[id]).ID); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "ready", Summary: "pr open"}); err != nil {
		t.Fatal(err)
	}
	if err := eng.Say(id, "验收没通过：标题还是旧的"); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if stage := TicketStage(&ticket); stage.Name != "working" || stage.Person {
		t.Fatalf("stage after sending back = %+v", stage)
	}
	if err := eng.Accept(id, ""); err == nil {
		t.Fatal("a ticket that is being redone is not up for review")
	}
}
