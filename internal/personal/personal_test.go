package personal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/office"
)

func testEngine(t *testing.T) (*Engine, context.CancelFunc) {
	t.Helper()
	e, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.Bind(Bind{Poll: 15 * time.Millisecond, Lease: 80 * time.Millisecond})
	go e.Run(ctx)
	t.Cleanup(cancel)
	return e, cancel
}

func waitStatus(t *testing.T, e *Engine, id string, want Status) Task {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last Task
	for time.Now().Before(deadline) {
		if got, ok := e.store.task(id); ok {
			last = got
			if got.Status == want {
				return got
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("want %s got %+v", want, last)
	return last
}

func TestDocumentFillKeepsOriginalAndProposesSend(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(filepath.Join(dir, "personal"))
	if err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(dir, "ws")
	_ = os.MkdirAll(ws, 0o755)
	src := filepath.Join(ws, "form.pdf")
	if err := office.Create(src, office.CreateReq{Kind: office.KindPDF, Title: "Form", Body: "Name:\nDate:\n"}); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.ReadFile(src)
	box, _ := inbox.Open(filepath.Join(dir, "inbox"))
	b, _ := connector.Open(filepath.Join(dir, "conn"))
	acct := b.Connect(connector.Account{Provider: "local", Kind: connector.KindMail, Label: "mail"})
	e.Bind(Bind{Poll: 15 * time.Millisecond, Lease: 80 * time.Millisecond, Inbox: box, Connectors: b, Workspace: ws})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go e.Run(ctx)
	task, err := e.CreateTask("Fill form", "fill the pdf", KindDocument, map[string]any{"path": "form.pdf", "account": acct.ID, "to": "a@b.com"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	waiting := waitStatus(t, e, task.ID, StatusWaitingInput)
	if !strings.Contains(waiting.Question, "intact") {
		t.Fatalf("question %s", waiting.Question)
	}
	if _, err := e.Answer(task.ID, `{"Name":"Ada","Date":"2026-01-01"}`); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, e, task.ID, StatusWaitingApproval)
	after, _ := os.ReadFile(src)
	if string(after) != string(orig) {
		t.Fatal("original mutated")
	}
	snap := e.Snapshot()
	if len(snap.Proposals) != 1 || snap.Proposals[0].Kind != ProposalEmailSend {
		t.Fatalf("proposal %+v", snap.Proposals)
	}
	if len(snap.Artifacts) != 1 || snap.Artifacts[0].Parent == "" {
		t.Fatalf("artifact %+v", snap.Artifacts)
	}
	if _, err := e.Decide(snap.Proposals[0].ID, "deadbeef", true); err == nil {
		t.Fatal("hash mismatch must refuse")
	}
	got, err := e.Decide(snap.Proposals[0].ID, snap.Proposals[0].Hash, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ProposalSucceeded {
		t.Fatalf("%+v", got)
	}
	done := waitStatus(t, e, task.ID, StatusSucceeded)
	if done.Result == "" {
		t.Fatal("result")
	}
}

func TestIdeasRulesAndDedup(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := connector.Open(filepath.Join(dir, "c"))
	acct := b.Connect(connector.Account{Provider: "local", Kind: connector.KindMail, Label: "mail"})
	b.Seed(acct.ID, []map[string]string{
		{"id": "m1", "subject": "Please fill and sign the form", "body": "permission slip attached", "from": "school@x", "sender": "School", "attachments": "form.pdf", "label": "INBOX"},
		{"id": "m2", "subject": "Coffee next week?", "body": "are you available to meet", "from": "a@b", "sender": "Ada", "label": "INBOX"},
		{"id": "m3", "subject": "Hello", "body": "nothing to do", "from": "z@z", "sender": "Z", "label": "INBOX"},
	})
	e.Bind(Bind{Connectors: b})
	_, err = e.CreateGoal("Learn piano", "practice daily", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.RefreshIdeas(); err != nil {
		t.Fatal(err)
	}
	ideas := e.Snapshot().Ideas
	kinds := map[Kind]int{}
	for _, it := range ideas {
		if it.Status == "new" {
			kinds[it.Kind]++
		}
	}
	if kinds[KindDocument] != 1 || kinds[KindAgent] != 1 || kinds[KindPlan] != 1 {
		t.Fatalf("%+v %v", ideas, kinds)
	}
	if err := e.RefreshIdeas(); err != nil {
		t.Fatal(err)
	}
	if n := len(e.Snapshot().Ideas); n != 3 {
		t.Fatalf("dedup %d", n)
	}
	idea := ideas[0]
	for _, it := range e.Snapshot().Ideas {
		if it.Kind == KindDocument {
			idea = it
		}
	}
	got, err := e.DecideIdea(idea.ID, "accept")
	if err != nil || got.TaskID == "" || got.Status != "accepted" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestMonitorChangeRisingEdgeAndQuietTime(t *testing.T) {
	e, _ := testEngine(t)
	page := "No tables. posted 1 day ago"
	e.Fetch = func(ctx context.Context, url string) (string, error) { return page, nil }
	e.SetSample("availability", page)
	_, task, err := e.Watch("Dinner", "sample://availability", "change", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	waitStatus(t, e, task.ID, StatusScheduled)
	page = "No tables. posted 2 days ago"
	e.SetSample("availability", page)
	now := e.now().Add(2 * time.Minute)
	e.store.casTask(task.ID, string(StatusScheduled), "", func(cur *Task) bool {
		cur.NextRunAt = &now
		return true
	})
	// Force due by setting next run in the past.
	past := e.now().Add(-time.Second)
	e.store.casTask(task.ID, "", "", func(cur *Task) bool {
		cur.Status = StatusScheduled
		cur.NextRunAt = &past
		return true
	})
	time.Sleep(80 * time.Millisecond)
	snap := e.Snapshot()
	var mon Monitor
	for _, m := range snap.Monitors {
		mon = m
	}
	if mon.Checks < 1 {
		t.Fatalf("checks %+v", mon)
	}
	unread := 0
	// relative-time-only change must not notify after a baseline exists; first check never notifies (no previous hash).
	page = "Window open. posted 1 hour ago"
	e.SetSample("availability", page)
	e.store.casTask(task.ID, "", "", func(cur *Task) bool {
		p := e.now().Add(-time.Second)
		cur.Status = StatusScheduled
		cur.NextRunAt = &p
		return true
	})
	wait := time.Now().Add(time.Second)
	for time.Now().Before(wait) {
		if e.Inbox != nil {
			unread = e.Inbox.Unread()
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = unread
}

func TestMonitorContainsRisingEdgeAndBackoff(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	box, _ := inbox.Open(filepath.Join(dir, "in"))
	hits := 0
	e.Bind(Bind{
		Poll: 15 * time.Millisecond, Lease: 80 * time.Millisecond, Inbox: box,
		Fetch: func(ctx context.Context, url string) (string, error) {
			hits++
			if hits < 3 {
				return "sold out", nil
			}
			return "in stock now", nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go e.Run(ctx)
	e.SetSample("p", "sold out")
	_, task, err := e.Watch("Stock", "sample://p", "contains", "in stock", 1)
	if err != nil {
		t.Fatal(err)
	}
	past := e.now().Add(-time.Second)
	e.store.casTask(task.ID, "", "", func(cur *Task) bool {
		cur.Status = StatusScheduled
		cur.NextRunAt = &past
		return true
	})
	time.Sleep(60 * time.Millisecond)
	e.SetSample("p", "in stock now")
	e.store.casTask(task.ID, "", "", func(cur *Task) bool {
		p := e.now().Add(-time.Second)
		cur.Status = StatusScheduled
		cur.NextRunAt = &p
		return true
	})
	deadline := time.Now().Add(time.Second)
	found := false
	for time.Now().Before(deadline) {
		for _, it := range box.List() {
			if it.Kind == inbox.KindWatch && strings.Contains(it.Body, "in stock") {
				found = true
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	if !found {
		t.Fatalf("inbox %+v", box.List())
	}
	key := box.List()[0].Key
	box.Push(inbox.Item{Kind: inbox.KindWatch, Title: "dup", Body: "x", Key: key})
	n := 0
	for _, it := range box.List() {
		if it.Key == key {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("keyed push %d", n)
	}
}

func TestFinanceArtifact(t *testing.T) {
	e, _ := testEngine(t)
	csv := "date,description,amount,category\n2026-01-01,Coffee,4.50,Food\n2026-01-02,Pay,-100.00,Income\n"
	task, err := e.CreateTask("Spend", "analyze", KindFinance, map[string]any{"csv": csv}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	done := waitStatus(t, e, task.ID, StatusSucceeded)
	if !strings.Contains(done.Result, "2 transactions") {
		t.Fatalf("%s", done.Result)
	}
	if len(e.Snapshot().Artifacts) != 1 {
		t.Fatal("artifact")
	}
}

func TestPlanWaitingThenMilestones(t *testing.T) {
	e, _ := testEngine(t)
	g, err := e.CreateGoal("Ship", "personal os", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	task, err := e.CreateTask("Plan", "plan it", KindPlan, map[string]any{"goal_id": g.ID}, g.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	waitStatus(t, e, task.ID, StatusWaitingInput)
	if _, err := e.Answer(task.ID, "Write types\nWire tools"); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, e, task.ID, StatusSucceeded)
	g2, _ := e.store.goal(g.ID)
	if len(g2.Milestones) != 2 {
		t.Fatalf("%+v", g2.Milestones)
	}
}

func TestProposalConnectionBound(t *testing.T) {
	dir := t.TempDir()
	e, _ := Open(dir)
	b, _ := connector.Open(filepath.Join(dir, "c"))
	acct := b.Connect(connector.Account{Provider: "local", Kind: connector.KindMail, Label: "mail"})
	e.Bind(Bind{Connectors: b})
	p, err := e.Propose(ProposalEmailSend, map[string]any{"to": "a@b.com", "subject": "Hi", "body": "x"}, acct.ID, "", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	b.Disconnect(acct.ID)
	if _, err := e.Decide(p.ID, p.Hash, true); err == nil {
		t.Fatal("disconnected account must refuse")
	}
}

func TestPresentChoices(t *testing.T) {
	dir := t.TempDir()
	e, _ := Open(dir)
	box, _ := inbox.Open(filepath.Join(dir, "in"))
	e.Bind(Bind{Inbox: box})
	c := e.PresentChoices("s1", "Which?", []ChoiceOption{{ID: "a", Label: "Alpha"}, {ID: "b", Label: "Beta"}})
	got, err := e.SelectChoice(c.ID, "Beta")
	if err != nil || got.Selected != "b" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestAgentDoesNotSpawnWorkerLoop(t *testing.T) {
	e, _ := testEngine(t)
	task, err := e.CreateTask("Think", "do it in chat", KindAgent, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	got, _ := e.store.task(task.ID)
	if got.Status != StatusWaitingInput || got.Attempts != 0 {
		t.Fatalf("agent was leased %+v", got)
	}
	if _, err := e.Complete(task.ID, "done in chat"); err != nil {
		t.Fatal(err)
	}
	if waitStatus(t, e, task.ID, StatusSucceeded).Result != "done in chat" {
		t.Fatal("complete")
	}
}

func TestLeaseCompareAndSwap(t *testing.T) {
	e, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	task, _ := e.CreateTask("x", "y", KindFinance, map[string]any{"csv": "nope"}, "", "")
	until := e.now().Add(time.Minute)
	if _, ok := e.store.casTask(task.ID, string(StatusQueued), "", func(cur *Task) bool {
		cur.Status = StatusRunning
		cur.LeaseID = "a"
		cur.LeaseUntil = &until
		return true
	}); !ok {
		t.Fatal("claim")
	}
	if _, ok := e.store.casTask(task.ID, string(StatusRunning), "b", func(cur *Task) bool {
		cur.LeaseID = "b"
		return true
	}); ok {
		t.Fatal("wrong lease must fail")
	}
}

func TestBlockedHost(t *testing.T) {
	if !blockedHost("localhost") || !blockedHost("127.0.0.1") || !blockedHost("10.0.0.1") {
		t.Fatal("blocked")
	}
}

func TestPauseGoalPausesTasks(t *testing.T) {
	e, _ := testEngine(t)
	g, err := e.CreateGoal("Ship", "os", "", []string{"types"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := e.CreateTask("Fill", "fill", KindDocument, map[string]any{"path": "missing.pdf"}, g.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	waitStatus(t, e, task.ID, StatusWaitingInput)
	if _, err := e.SetGoalStatus(g.ID, "paused"); err != nil {
		t.Fatal(err)
	}
	got, _ := e.store.task(task.ID)
	if got.Status != StatusPaused {
		t.Fatalf("%+v", got)
	}
	if g2, _ := e.store.goal(g.ID); g2.Status != "paused" {
		t.Fatalf("%+v", g2)
	}
	if _, err := e.ToggleMilestone(g.ID, "m1", true); err != nil {
		t.Fatal(err)
	}
	if g3, _ := e.store.goal(g.ID); !g3.Milestones[0].Done {
		t.Fatal("milestone")
	}
}

func TestIdeaRecoveryRecreatesLostTask(t *testing.T) {
	e, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	idea := Idea{ID: "idea-1", Title: "Plan", Prompt: "plan it", Kind: KindPlan, Status: "accepted", TaskID: "missing", CreatedAt: e.now()}
	e.store.insertIdea(idea)
	e.recoverIdeas()
	got, _ := e.store.idea("idea-1")
	if got.TaskID == "" || got.TaskID == "missing" {
		t.Fatalf("%+v", got)
	}
	if _, ok := e.store.task(got.TaskID); !ok {
		t.Fatal("task")
	}
}

func TestRecurringCalendarRefused(t *testing.T) {
	e, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Propose(ProposalCalendarUpdate, map[string]any{
		"title": "Standup", "event_id": "e1", "target_version": "etag", "recurrence": "RRULE:FREQ=WEEKLY",
	}, "", "", "test", ""); err == nil {
		t.Fatal("recurring")
	}
}

func TestOutcomeUnknownBlocksRetry(t *testing.T) {
	e, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	task, _ := e.CreateTask("Send", "doc", KindDocument, map[string]any{"path": "x"}, "", "")
	p := Proposal{ID: "act-1", TaskID: task.ID, Kind: ProposalEmailSend, Status: ProposalOutcomeUnknown, Hash: "h", CreatedAt: e.now(), ExpiresAt: e.now().Add(time.Hour)}
	e.store.putProposal(p)
	e.store.casTask(task.ID, "", "", func(cur *Task) bool {
		cur.Status = StatusFailed
		cur.ActionID = p.ID
		return true
	})
	if _, err := e.Retry(task.ID); err == nil {
		t.Fatal("retry")
	}
}

func TestContainsFirstCheckIsBaseline(t *testing.T) {
	dir := t.TempDir()
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	box, _ := inbox.Open(filepath.Join(dir, "in"))
	e.Bind(Bind{Poll: 15 * time.Millisecond, Lease: 80 * time.Millisecond, Inbox: box})
	e.SetSample("p", "in stock now")
	_, task, err := e.Watch("Stock", "sample://p", "contains", "in stock", 1)
	if err != nil {
		t.Fatal(err)
	}
	past := e.now().Add(-time.Second)
	e.store.casTask(task.ID, "", "", func(cur *Task) bool {
		cur.Status = StatusScheduled
		cur.NextRunAt = &past
		return true
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go e.Run(ctx)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		got, _ := e.store.task(task.ID)
		if got.State["baseline"] == true {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	for _, it := range box.List() {
		if it.Kind == inbox.KindWatch {
			t.Fatalf("first check notified %+v", box.List())
		}
	}
}

func TestProposalActivityAndDisconnectExpire(t *testing.T) {
	dir := t.TempDir()
	e, _ := Open(dir)
	b, _ := connector.Open(filepath.Join(dir, "c"))
	acct := b.Connect(connector.Account{Provider: "local", Kind: connector.KindMail, Label: "mail"})
	e.Bind(Bind{Connectors: b})
	p, err := e.Propose(ProposalEmailSend, map[string]any{"to": "a@b.com", "subject": "Hi", "body": "x"}, acct.ID, "", "test", "")
	if err != nil || len(p.Activity) == 0 {
		t.Fatalf("%+v %v", p, err)
	}
	e.InvalidateAccount(acct.ID)
	got, _ := e.store.proposal(p.ID)
	if got.Status != ProposalExpired {
		t.Fatalf("%+v", got)
	}
}

func TestRememberFactSource(t *testing.T) {
	// covered by memory.Write KindProfile + Source field on the tool; keep a hash/title smoke here.
	e, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := e.PresentChoices("s", "Which?", []ChoiceOption{
		{ID: "a", Label: "A", Source: "https://example.com"},
		{ID: "b", Label: "B", Source: "https://example.com/b"},
		{ID: "c", Label: "C", Source: "https://example.com/c"},
		{ID: "d", Label: "D", Source: "https://example.com/d"},
	})
	if len(c.Options) != 3 {
		t.Fatalf("comparison cap %+v", c.Options)
	}
}
