package superagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// TestClaimBranchFollowsOwnPull 锁住短名对上的开着的工单会跟随那条拉取请求。
func TestClaimBranchFollowsOwnPull(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	eng.mu.Lock()
	eng.tickets[id].Name = "fix-build"
	eng.mu.Unlock()
	eng.ClaimBranch("github:acme/widgets!9", "fix-build")
	got := eng.FollowThreads()
	if len(got) != 1 || got[0] != "github:acme/widgets!9" {
		t.Fatalf("follows %v", got)
	}
}

// TestDeskRequestIsExplicit 锁住坐在机器前的人，在观察模式和关闭模式下都是明确请求。
func TestDeskRequestIsExplicit(t *testing.T) {
	for _, mode := range []string{"observe", "off"} {
		eng := NewEngine(Policy{Version: 1, Mode: mode})
		msg := Trigger{
			ID: "d1", Source: "desk", Threads: []string{"desk:s1"}, Primary: "desk:s1",
			Human: true, Addressed: true, Sender: "operator", Text: "fix the build",
		}
		if eng.Receive(msg) != "stored" {
			t.Fatal(mode)
		}
		d, err := eng.Decide("d1", "accept", "fix")
		if err != nil || d.Kind != "accept" || d.TicketID == "" {
			t.Fatalf("%s %+v %v", mode, d, err)
		}
	}
}

// TestRecoverHostSkipsSettled 锁住重启只重开被打断的派工，等待中的报告保持原样。
func TestRecoverHostSkipsSettled(t *testing.T) {
	eng := NewEngine(autoPolicy())
	waiting := acceptOn(t, eng, "m1", "room-a")
	running := acceptOn(t, eng, "m2", "room-b")
	hold, _ := eng.Ticket(waiting)
	if err := eng.AssignmentStarted(waiting, hold.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(waiting, Report{State: "waiting", Summary: "hold"}); err != nil {
		t.Fatal(err)
	}
	live, _ := eng.Ticket(running)
	if err := eng.AssignmentStarted(running, live.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	ids := eng.RecoverHost()
	if len(ids) != 1 || ids[0] != running {
		t.Fatalf("ids %v", ids)
	}
	again, _ := eng.Ticket(running)
	if again.Assignments[0].State != "ended" || again.Assignments[1].State != "pending" {
		t.Fatalf("assignments %+v", again.Assignments)
	}
	for _, in := range again.Inputs {
		if strings.Contains(in.Text, "宿主重启了") {
			t.Fatal("recovery note")
		}
	}
	if !again.Resume {
		t.Fatal("resume")
	}
	left, _ := eng.Ticket(waiting)
	if len(left.Assignments) != 1 || left.Assignments[0].State != "running" {
		t.Fatalf("waiting changed %+v", left.Assignments)
	}
}

// TestPrepareFailureDoesNotRetry 锁住准备命令失败会结束工单，再次启动不会再跑准备命令。
func TestPrepareFailureDoesNotRetry(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	ex := NewExecutor(eng, NewSlots(1), scriptModel{fn: func(context.Context, dsh.Request) (dsh.Assistant, error) {
		t.Errorf("model should not run")
		return dsh.Assistant{}, nil
	}})
	ex.Root = t.TempDir()
	ex.PrepareCmd = "exit 1"
	calls := 0
	ex.RunPrepare = func(context.Context, string, string, map[string]string) (string, error) {
		calls++
		return "boom", fmt.Errorf("exit")
	}
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ticket, _ := eng.Ticket(id)
		if ticket.Ended != nil && ticket.Ended.Outcome == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	ticket, _ := eng.Ticket(id)
	if ticket.Ended == nil || ticket.Ended.Outcome != "failed" || !strings.Contains(ticket.Ended.Reason, "boom") {
		t.Fatalf("ticket %+v", ticket.Ended)
	}
	if err := ex.Launch(id); err == nil {
		t.Fatal("relaunched")
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

// TestOutboxHoldsLaterMessage 锁住同一工单前面一条没发出时，后面的回复不先写到 GitHub。
func TestOutboxHoldsLaterMessage(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	msg := human("g1", "please fix")
	msg.Source = "github"
	msg.Threads = []string{"github:acme/widgets#7"}
	msg.Primary = "github:acme/widgets#7"
	msg.Ref = "7"
	if rt.eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	if _, err := rt.eng.Decide("g1", "accept", "fix"); err != nil {
		t.Fatal(err)
	}
	ticket := ""
	for _, row := range rt.eng.Outbox() {
		ticket = row.TicketID
	}
	if _, err := rt.eng.Reply(ticket, "answer", "g1", "later"); err != nil {
		t.Fatal(err)
	}
	rt.SetSources(SourceConfig{GitHubToken: "t", GitHubRepo: "acme/widgets", GitHubAPI: srv.URL})
	rt.flushOutbox()
	rt.flushOutbox()
	if len(bodies) != 2 {
		t.Fatalf("posts %d %v", len(bodies), bodies)
	}
	for _, body := range bodies {
		if strings.Contains(body, "later") {
			t.Fatalf("result went out early %s", body)
		}
	}
}

// TestOrderedRelayWaits 锁住前面一条报告没写成时，后面的报告不先写。
func TestOrderedRelayWaits(t *testing.T) {
	q := NewQueue()
	var got []string
	q.HandleOrdered("relay", 1, func(payload any) error {
		text := payload.(string)
		if text == "first" {
			return errors.New("no")
		}
		got = append(got, text)
		return nil
	})
	q.Enqueue("relay", "a", "first")
	q.Enqueue("relay", "b", "second")
	now := time.Now().Add(time.Second)
	q.Pump(now)
	q.Pump(now.Add(time.Minute))
	if len(got) != 0 {
		t.Fatalf("second ran %v", got)
	}
}

// TestHostRestartContinuesWithoutNewBrief 锁住宿主重启只接上已有日志，不再把工单说明重发一遍。
func TestHostRestartContinuesWithoutNewBrief(t *testing.T) {
	sess := dsh.NewSession()
	sess.Append(dsh.EventUser, 1, 1, dsh.UserMessage{Text: "已改过 README", Source: "assignment"})
	sess.Append(dsh.EventTurnEnd, 1, 0, dsh.TurnEnd{Kind: "completed"})
	body, _ := encodeLog(sess.Events())
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	ticket, _ := eng.Ticket(id)
	if err := eng.AssignmentStarted(id, ticket.Assignments[0].ID); err != nil {
		t.Fatal(err)
	}
	if ids := eng.RecoverHost(); len(ids) != 1 || ids[0] != id {
		t.Fatalf("ids %v", ids)
	}
	var seen []string
	ex := NewExecutor(eng, NewSlots(1), scriptModel{fn: func(_ context.Context, req dsh.Request) (dsh.Assistant, error) {
		for _, msg := range req.Messages {
			seen = append(seen, msg.Text)
		}
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{
			ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"ok"}`,
		}}}, nil
	}})
	ex.LogLoad = func(string) []byte { return body }
	ex.LogSave = func(string, []byte) {}
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	waitOutcome(t, eng, id, "completed")
	joined := strings.Join(seen, "\n")
	if !strings.Contains(joined, "已改过 README") || !strings.Contains(joined, "继续。") {
		t.Fatal(joined)
	}
	if strings.Contains(joined, "完成后调用 orchestrator_report。") {
		t.Fatal(joined)
	}
}

// TestRestartKeepsWorkerWords 锁住重启后的新派工还能看见上次会话里已经说过的话，回合号接着上次。
func TestRestartKeepsWorkerWords(t *testing.T) {
	sess := dsh.NewSession()
	sess.Append(dsh.EventUser, 1, 1, dsh.UserMessage{Text: "已改过 README", Source: "assignment"})
	sess.Append(dsh.EventTurnEnd, 1, 0, dsh.TurnEnd{Kind: "completed"})
	body, _ := encodeLog(sess.Events())
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	seen := ""
	ex := NewExecutor(eng, NewSlots(1), scriptModel{fn: func(_ context.Context, req dsh.Request) (dsh.Assistant, error) {
		for _, msg := range req.Messages {
			seen += msg.Text
		}
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{
			ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"ok"}`,
		}}}, nil
	}})
	ex.LogLoad = func(string) []byte { return body }
	var saved []byte
	ex.LogSave = func(_ string, raw []byte) { saved = append([]byte(nil), raw...) }
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	waitOutcome(t, eng, id, "completed")
	if !strings.Contains(seen, "已改过 README") {
		t.Fatal(seen)
	}
	_, turn := decodeLog(saved)
	if turn < 2 {
		t.Fatalf("turn %d", turn)
	}
}

// TestPollableKeepsRecentWork 锁住工单结束后两周内仍读它的线程，再晚就不再读。
func TestPollableKeepsRecentWork(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := acceptOn(t, eng, "m1", "github:acme/widgets#9")
	if err := eng.Report(id, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	got := eng.PollableThreads(time.Now())
	if len(got) != 1 || got[0] != "github:acme/widgets#9" {
		t.Fatalf("recent %v", got)
	}
	if later := eng.PollableThreads(time.Now().Add(15 * 24 * time.Hour)); len(later) != 0 {
		t.Fatalf("late %v", later)
	}
}

// TestClosedCommentResumes 锁住议题已经不在开放列表里时，两周内的人的评论仍能把工单重新打开。
func TestClosedCommentResumes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/widgets/issues":
			_, _ = w.Write([]byte("[]"))
		case "/repos/acme/widgets/issues/9/comments":
			_, _ = w.Write([]byte(`[{"id":41,"body":"please continue","user":{"login":"ada"}}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	id := acceptOn(t, rt.eng, "m1", "github:acme/widgets#9")
	if err := rt.eng.Report(id, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	rt.client = srv.Client()
	rt.SetSources(SourceConfig{GitHubToken: "t", GitHubRepo: "acme/widgets", GitHubAPI: srv.URL})
	rt.Pass(time.Now())
	ticket, ok := rt.eng.Ticket(id)
	if !ok || ticket.Ended != nil {
		t.Fatalf("ended %+v", ticket.Ended)
	}
	found := false
	for _, in := range ticket.Inputs {
		if strings.Contains(in.Text, "please continue") {
			found = true
		}
	}
	if !found {
		t.Fatalf("inputs %+v", ticket.Inputs)
	}
}

// TestWorkerLogSurvivesOpen 锁住工人会话日志写在快照旁边，重新打开还能读到。
func TestWorkerLogSurvivesOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "superagent.sqlite")
	rt, err := Open(path, autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.saveWorkerLog("tkt-1", []byte("已改过 README"))
	rt.Stop()
	again, err := Open(path, autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	defer again.Stop()
	if string(again.loadWorkerLog("tkt-1")) != "已改过 README" {
		t.Fatalf("log %q", again.loadWorkerLog("tkt-1"))
	}
}

// TestContextFollowDoesNotResume 锁住只为上下文跟随的线程，在工单结束后不能把它重新打开，也不算拥有这条消息。
func TestContextFollowDoesNotResume(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := acceptOn(t, eng, "m1", "room")
	if _, err := eng.FollowForContext(id, "side"); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "dropped", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	msg := human("m2", "please continue")
	msg.Threads = []string{"side"}
	msg.Primary = "side"
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	ticket, _ := eng.Ticket(id)
	if ticket.Ended == nil {
		t.Fatal("resumed")
	}
	for _, in := range ticket.Inputs {
		if strings.Contains(in.Text, "please continue") {
			t.Fatal("delivered to ended ticket")
		}
	}
}

// TestContextFollowReachesLiveWork 锁住还开着的工单能收到上下文线程的正文，但这条消息仍然可以另行评估。
func TestContextFollowReachesLiveWork(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := acceptOn(t, eng, "m1", "room")
	if _, err := eng.FollowForContext(id, "side"); err != nil {
		t.Fatal(err)
	}
	msg := human("m2", "see this")
	msg.Threads = []string{"side"}
	msg.Primary = "side"
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	ticket, _ := eng.Ticket(id)
	if ticket.Ended != nil {
		t.Fatal("ended")
	}
	found := false
	for _, in := range ticket.Inputs {
		if strings.Contains(in.Text, "see this") {
			found = true
		}
	}
	if !found {
		t.Fatalf("inputs %+v", ticket.Inputs)
	}
	snap, err := eng.Export()
	if err != nil {
		t.Fatal(err)
	}
	queued := false
	for _, tr := range snap.Triggers {
		if tr.ID != "m2" {
			continue
		}
		if contains(tr.Reached, id) {
			t.Fatal("context follow counted as owner")
		}
		queued = tr.Queued
	}
	if !queued {
		t.Fatal("message was not left for assessment")
	}
}

// TestBotDoesNotResume 锁住机器人的消息不能把已经结束的工单重新打开。
func TestBotDoesNotResume(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := acceptOn(t, eng, "m1", "room")
	if err := eng.Report(id, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	msg := Trigger{
		ID: "b1", Source: "github", Threads: []string{"room"}, Primary: "room",
		Human: false, Sender: "ci[bot]", Text: "checks failed",
	}
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	ticket, _ := eng.Ticket(id)
	if ticket.Ended == nil {
		t.Fatal("resumed")
	}
}

// TestAutoAcceptNeedsEnabledRule 锁住自动模式下的推断接受必须命中一条还启用的规则，停用后不能恢复。
func TestAutoAcceptNeedsEnabledRule(t *testing.T) {
	policy := Policy{Version: 1, Mode: "auto", Rules: []Rule{{ID: "bugs", Text: "修构建", Enabled: true}}}
	eng := NewEngine(policy)
	msg := Trigger{
		ID: "m", Source: "github", Threads: []string{"room"}, Primary: "room",
		Human: true, Addressed: true, Sender: "stranger", Text: "please",
	}
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	if _, err := eng.Decide("m", "accept", "ok"); err == nil || err.Error() != "needs_rule" {
		t.Fatal(err)
	}
	d, err := eng.DecideMatched("m", "accept", "ok", []string{"bugs"})
	if err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(d.TicketID)
	if len(ticket.Mandate.Rules) != 1 || ticket.Mandate.Rules[0] != "bugs" || ticket.Mandate.Strength != "inferred" {
		t.Fatalf("%+v", ticket.Mandate)
	}
	if err := eng.Report(d.TicketID, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	policy.Rules[0].Enabled = false
	eng.SetPolicy(policy)
	follow := Trigger{
		ID: "m2", Source: "github", Threads: []string{"room"}, Primary: "room",
		Human: true, Addressed: true, Sender: "stranger", Text: "continue",
	}
	if eng.Receive(follow) != "stored" {
		t.Fatal("stored")
	}
	again, _ := eng.Ticket(d.TicketID)
	if again.Ended == nil {
		t.Fatal("disabled rule resumed")
	}
}

// TestPersonConfirmOpensInObserve 锁住观察模式下人点接受仍开工，路由自己接受则只记本来会接。
func TestPersonConfirmOpensInObserve(t *testing.T) {
	eng := NewEngine(Policy{Version: 1, Mode: "observe"})
	msg := Trigger{
		ID: "m", Source: "github", Threads: []string{"room"}, Primary: "room",
		Human: true, Addressed: true, Sender: "stranger", Text: "please",
	}
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	if d, err := eng.Decide("m", "accept", "ok"); err != nil || d.Kind != "ignore" || d.Reason != "observed" {
		t.Fatalf("%+v %v", d, err)
	}
	next := Trigger{
		ID: "m2", Source: "github", Threads: []string{"room"}, Primary: "room",
		Human: true, Addressed: true, Sender: "stranger", Text: "please again",
	}
	if eng.Receive(next) != "stored" {
		t.Fatal("stored")
	}
	d, err := eng.Confirm("m2", "accept", "ok")
	if err != nil || d.Kind != "accept" || d.TicketID == "" {
		t.Fatalf("%+v %v", d, err)
	}
	ticket, _ := eng.Ticket(d.TicketID)
	if ticket.Mandate.Strength != "inferred" || len(ticket.Mandate.Rules) != 1 || ticket.Mandate.Rules[0] != "person" {
		t.Fatalf("%+v", ticket.Mandate)
	}
	if err := eng.Report(d.TicketID, Report{State: "ready", Summary: "done"}); err != nil {
		t.Fatal(err)
	}
	follow := Trigger{
		ID: "m3", Source: "github", Threads: []string{"room"}, Primary: "room",
		Human: true, Addressed: true, Sender: "stranger", Text: "continue",
	}
	if eng.Receive(follow) != "stored" {
		t.Fatal("stored")
	}
	again, _ := eng.Ticket(d.TicketID)
	if again.Ended != nil {
		t.Fatal("person confirmation did not resume")
	}
}

// TestCursorSurvivesOpen 锁住每个线程的读游标写进快照，重新打开后仍往前重叠五秒。
func TestCursorSurvivesOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "superagent.sqlite")
	rt, err := Open(path, autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)
	rt.markCursor("github:acme/widgets#9", at)
	rt.save()
	rt.Stop()
	again, err := Open(path, autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	defer again.Stop()
	got := again.sinceFor("github:acme/widgets#9")
	if !got.Equal(at.Add(-5 * time.Second)) {
		t.Fatalf("since %s", got)
	}
}

// TestFeishuBadItemStaysOnTopic 锁住一条读不懂的飞书消息只记在自己的话题上，同页的文字仍然收下。连续三次之后来源才报这个话题。
func TestFeishuBadItemStaysOnTopic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "tenant_access_token") {
			_, _ = w.Write([]byte(`{"code":0,"tenant_access_token":"tok"}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"message_id":"m1","body":{"content":"@yoyo hello"},"sender":{"id":"ada","sender_type":"user"}},{"message_id":"m2","body":{"content":"{\"image_key\":\"x\"}"},"sender":{"id":"ada","sender_type":"user"}}]}}`))
	}))
	defer srv.Close()
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.client = srv.Client()
	rt.SetSources(SourceConfig{FeishuAppID: "app", FeishuSecret: "sec", FeishuChat: "oc1", FeishuAPI: srv.URL, FeishuMention: "yoyo"})
	now := time.Now()
	rt.Pass(now)
	view := rt.View()
	var feishu SourceState
	for _, src := range view.Sources {
		if src.Name == "feishu" {
			feishu = src
		}
	}
	if feishu.State != "ok" {
		t.Fatalf("first %+v", feishu)
	}
	found := false
	for _, row := range view.Queued {
		if strings.Contains(row.Text, "hello") {
			found = true
		}
	}
	if !found {
		t.Fatalf("queued %+v", view.Queued)
	}
	rt.Pass(now.Add(31 * time.Second))
	rt.Pass(now.Add(62 * time.Second))
	view = rt.View()
	for _, src := range view.Sources {
		if src.Name == "feishu" {
			feishu = src
		}
	}
	if feishu.State != "error" || !strings.Contains(feishu.Detail, "feishu:oc1:m2") {
		t.Fatalf("after %+v", feishu)
	}
}

// TestReviewAndCheckUseOwnThreads 锁住还在看的拉取请求上，审阅和已完成的检查各自进自己的线程，并送到这件工作。
func TestReviewAndCheckUseOwnThreads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/widgets/issues":
			_, _ = w.Write([]byte("[]"))
		case "/repos/acme/widgets/issues/9/comments":
			_, _ = w.Write([]byte("[]"))
		case "/repos/acme/widgets/pulls/9/comments":
			_, _ = w.Write([]byte(`[{"id":7,"body":"please rename this","user":{"login":"ada","type":"User"}}]`))
		case "/repos/acme/widgets/pulls/9":
			_, _ = w.Write([]byte(`{"head":{"sha":"abc"}}`))
		case "/repos/acme/widgets/commits/abc/check-runs":
			_, _ = w.Write([]byte(`{"check_runs":[{"id":3,"name":"build","status":"completed","conclusion":"failure"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	id := acceptOn(t, rt.eng, "m1", "github:acme/widgets!9")
	rt.client = srv.Client()
	rt.SetSources(SourceConfig{GitHubToken: "t", GitHubRepo: "acme/widgets", GitHubAPI: srv.URL, GitHubAccount: "yoyo"})
	rt.Pass(time.Now())
	ticket, ok := rt.eng.Ticket(id)
	if !ok {
		t.Fatal("ticket")
	}
	var review, check bool
	for _, in := range ticket.Inputs {
		if strings.Contains(in.Text, "please rename") {
			review = true
		}
		if strings.Contains(in.Text, "build") {
			check = true
		}
	}
	if !review || !check {
		t.Fatalf("inputs %+v", ticket.Inputs)
	}
	snap, err := rt.eng.Export()
	if err != nil {
		t.Fatal(err)
	}
	var reviewThread, ciThread bool
	for _, th := range snap.Threads {
		if th.Key == "github:acme/widgets!9/review:7" {
			reviewThread = true
		}
		if th.Key == "github:acme/widgets:ci:build" {
			ciThread = true
		}
	}
	if !reviewThread || !ciThread {
		t.Fatalf("threads %+v", snap.Threads)
	}
}
