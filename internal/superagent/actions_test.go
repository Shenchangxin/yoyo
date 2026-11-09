// actions_test.go 锁住挂接、回复、交还和评估超时驳回。

package superagent

import (
	"strings"
	"testing"
)

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

// TestAssignFoldsShortReply 锁住短回复不进工人正文，只留下一份任务说明。
func TestAssignFoldsShortReply(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.Report(id, Report{State: "blocked", Question: "要新建吗"}); err != nil {
		t.Fatal(err)
	}
	if eng.Receive(deskHuman("m2", "新建吧")) != "stored" {
		t.Fatal("stored")
	}
	brief := "背景：人决定新建，不要再确认。\n要做的：在工作目录新建 gugu-mart，并写初始文档。\n不要做的：不要移动已有文件。"
	var woke []string
	eng.OnInput = func(_, text string) { woke = append(woke, text) }
	if err := eng.Assign(id, "m2", brief); err != nil {
		t.Fatal(err)
	}
	ticket, ok := eng.Ticket(id)
	if !ok {
		t.Fatal("ticket")
	}
	sawHuman := false
	for _, in := range ticket.Inputs {
		if strings.Contains(in.Text, "新建吧") {
			t.Fatalf("raw reply forwarded: %+v", in)
		}
		if in.TriggerID != "m2" {
			continue
		}
		if in.Text != "" || !in.Human || contains(ticket.Undelivered, in.ID) {
			t.Fatalf("marker %+v pending %v", in, ticket.Undelivered)
		}
		sawHuman = true
	}
	if !sawHuman {
		t.Fatal("human marker")
	}
	if WaitsForPerson(&ticket) {
		t.Fatal("still waiting")
	}
	body := assignmentText(ticket, "", "", "")
	if strings.Contains(body, "新建吧") || strings.Count(body, "gugu-mart") != 1 {
		t.Fatal(body)
	}
	if len(woke) != 1 || woke[0] != brief {
		t.Fatalf("wake %+v", woke)
	}
}

// TestAssignWithoutBriefKeepsReply 锁住没有任务说明时，原话仍然送进工单。
func TestAssignWithoutBriefKeepsReply(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if eng.Receive(deskHuman("m2", "新建吧")) != "stored" {
		t.Fatal("stored")
	}
	if err := eng.Assign(id, "m2", "  "); err != nil {
		t.Fatal(err)
	}
	ticket, ok := eng.Ticket(id)
	if !ok {
		t.Fatal("ticket")
	}
	found := false
	for _, in := range ticket.Inputs {
		if in.Text == "新建吧" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%+v", ticket.Inputs)
	}
}

// TestDecideBriefReplacesWorkerText 锁住新工单的工人说明替换原话，原话仍留在触发上。
func TestDecideBriefReplacesWorkerText(t *testing.T) {
	eng := NewEngine(autoPolicy())
	if eng.Receive(human("m1", "新建吧")) != "stored" {
		t.Fatal("stored")
	}
	brief := "背景：人要新建咕咕便利店。\n要做的：建立 gugu-mart 并写初始文档。\n不要做的：不要再追问。"
	d, err := eng.DecideBrief("m1", "accept", "gugu", brief, nil)
	if err != nil || d.TicketID == "" {
		t.Fatal(err)
	}
	ticket, ok := eng.Ticket(d.TicketID)
	if !ok || len(ticket.Inputs) != 1 || ticket.Inputs[0].Text != brief {
		t.Fatalf("%+v", ticket.Inputs)
	}
	tr, ok := eng.Trigger("m1")
	if !ok || tr.Text != "新建吧" {
		t.Fatalf("%+v", tr)
	}
}

// TestDecideBriefForSetsAgentBeforeLaunch 锁住接受时新工单带上 agent 和点名，派工开始时已经写好；重复接受不改已开的工单。
func TestDecideBriefForSetsAgentBeforeLaunch(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var seen []string
	eng.OnLaunch = func(id string) {
		ticket, _ := eng.Ticket(id)
		seen = append(seen, ticket.Agent)
	}
	if eng.Receive(human("m1", "让 codex 修")) != "stored" {
		t.Fatal("stored")
	}
	d, err := eng.DecideBriefFor("m1", "accept", "fix", "修好登录页", nil, " codex ", true)
	if err != nil || d.TicketID == "" {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(d.TicketID)
	if ticket.Agent != "codex" || !ticket.AgentPinned || len(seen) != 1 || seen[0] != "codex" {
		t.Fatalf("ticket %+v, launched with %v", ticket, seen)
	}
	if _, err := eng.DecideBriefFor("m1", "accept", "fix", "", nil, "claude", false); err != nil {
		t.Fatal(err)
	}
	if again, _ := eng.Ticket(d.TicketID); again.Agent != "codex" || !again.AgentPinned {
		t.Fatalf("re-accept changed the agent: %+v", again)
	}
	other := human("m2", "随便谁")
	other.Threads, other.Primary = []string{"room2"}, "room2"
	if eng.Receive(other) != "stored" {
		t.Fatal("stored")
	}
	d2, err := eng.DecideBrief("m2", "accept", "any", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if plain, _ := eng.Ticket(d2.TicketID); plain.Agent != "" || plain.AgentPinned {
		t.Fatalf("DecideBrief set an agent: %+v", plain)
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

// deskHuman 造一条主会话消息。桌面线程不会在接收时自动抄进工单。
func deskHuman(id, text string) Trigger {
	msg := human(id, text)
	msg.Threads = []string{"desk:main"}
	msg.Primary = "desk:main"
	return msg
}
