// actions_test.go 锁住挂接、回复、交还和评估超时驳回。

package superagent

import "testing"

// TestAttachAndReply 锁住挂接会把触发送进工单，回复按键去重，交还后重新排队。
func TestAttachAndReply(t *testing.T) {
	eng := NewEngine(autoPolicy())
	origin := accept(t, eng, "m1")
	next := human("m2", "also this")
	next.Source = "github"
	next.Ref = "9"
	if eng.Receive(next) != "stored" {
		t.Fatal("stored")
	}
	if err := eng.Attach(origin, "m2", "see this"); err != nil {
		t.Fatal(err)
	}
	ticket, ok := eng.Ticket(origin)
	if !ok || len(ticket.Inputs) < 2 {
		t.Fatalf("ticket %+v", ticket)
	}
	msg, err := eng.Reply(origin, "acknowledgement", "m2", "got it")
	if err != nil || msg.Status != "pending" || msg.Ref != "9" {
		t.Fatalf("reply %+v %v", msg, err)
	}
	again, err := eng.Reply(origin, "acknowledgement", "m2", "got it")
	if err != nil || again.ID != msg.ID {
		t.Fatalf("dup %+v %v", again, err)
	}
	queued, err := eng.Handback(origin, "m2", "not this ticket")
	if err != nil || !queued {
		t.Fatalf("handback %v %v", queued, err)
	}
	tr, _ := eng.Trigger("m2")
	if !tr.Queued || !contains(tr.Handbacks, origin) {
		t.Fatalf("trigger %+v", tr)
	}
	if err := eng.Attach(origin, "m2", ""); err == nil || err.Error() != "handedBackHere" {
		t.Fatalf("attach %v", err)
	}
}

// TestDismissUnavailable 锁住只有 unavailable 的决定能被驳回。
func TestDismissUnavailable(t *testing.T) {
	eng := NewEngine(autoPolicy())
	eng.Receive(human("m1", "fix"))
	d, err := eng.Decide("m1", "ignore", "no")
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.Dismiss(d.ID); err == nil {
		t.Fatal("ignore dismissed")
	}
	d.Kind = "unavailable"
	eng.mu.Lock()
	eng.decisions[len(eng.decisions)-1].Kind = "unavailable"
	eng.mu.Unlock()
	if err := eng.Dismiss(d.ID); err != nil {
		t.Fatal(err)
	}
}
