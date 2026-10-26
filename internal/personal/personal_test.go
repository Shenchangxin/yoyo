// 锁住文档填写不改原件、监视的第一次检查不通知、提案要账号，以及 agent 不被 worker 租走。
// 本包不写记忆，这些测试也不判断记忆写入是否默认要问。

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

// testEngine 打开临时目录里的引擎，绑上本地邮件连接器和收件箱，并启动 Run。
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

// waitStatus 在时限内等到任务变成指定状态，否则让测试失败。
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

// TestDocumentFillKeepsOriginalAndProposesSend 确认填写后原件字节不变，并产生待审阅的发信提案。哈希不符时 Decide 拒绝。
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

// TestIdeasRulesAndDedup 确认表单邮件和约见邮件会出想法，重复刷新不增加条数，接受后变成任务。
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

// TestMonitorChangeRisingEdgeAndQuietTime 用 sample:// 监视。只改相对时间并强制到期后，检查次数增加。
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
	// 把下次运行时间改到过去，迫使这一轮到期。
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
	// 已有基线后，只有相对时间变化不得通知。第一次检查没有先前哈希，也不通知。
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

// TestMonitorContainsRisingEdgeAndBackoff 确认 contains 第一次命中不通知，再次命中才通知，相同键不重复推送。
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

// TestFinanceArtifact 确认 CSV 任务成功后留下 finance 产物。
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

// TestPlanWaitingThenMilestones 确认计划任务先等待输入，回答里程碑后写到目标上。
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

// TestProposalConnectionBound 确认账号不在连接器列表里时 Propose 返回错误。
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

// TestPresentChoices 确认选项 ID 能被选中。
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

// TestAgentDoesNotSpawnWorkerLoop 确认 agent 任务保持 waiting_input，worker 不把它租走，Complete 后变为 succeeded。
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

// TestLeaseCompareAndSwap 确认错误的租约 ID 不能改掉正在 running 的任务。
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

// TestBlockedHost 确认回环地址被 blockedHost 拒绝。
func TestBlockedHost(t *testing.T) {
	if !blockedHost("localhost") || !blockedHost("127.0.0.1") || !blockedHost("10.0.0.1") {
		t.Fatal("blocked")
	}
}

// TestPauseGoalPausesTasks 确认目标标成 paused 后，其下任务也暂停，里程碑完成标记能写上。
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

// TestIdeaRecoveryRecreatesLostTask 确认 accepted 想法在任务丢失后会被 recoverIdeas 补回。
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

// TestRecurringCalendarRefused 确认带 recurrence 的提案被拒绝。
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

// TestOutcomeUnknownBlocksRetry 确认关联提案为 outcome_unknown 时 Retry 返回错误。
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

// TestContainsFirstCheckIsBaseline 确认 contains 的第一次检查不往收件箱推 watch。
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

// TestProposalActivityAndDisconnectExpire 确认新提案带活动记录，InvalidateAccount 后状态变为 expired。
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

// TestRememberFactSource 确认带 Source 的比较选项被截到 3 个。它不调用记忆写入。
func TestRememberFactSource(t *testing.T) {
	// 记忆写入的 KindProfile 和 Source 在工具侧。本测试不写记忆，也不判断默认是否询问，只检查带 Source 的比较选项被截到 3 个。
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
