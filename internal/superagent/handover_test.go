// handover_test.go 锁住改派和离线兜底：换人之后旧工人被关掉、不再被催，它的报告也落不到新派工上；
// 新工人只读交接之后的日志，任务页仍然看得到全部；点名的 agent 离线时问人，没点名的退回 yoyo，要登录的等人登录。

package superagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// memLog 是内存里的工人日志，代替数据库。
type memLog struct {
	mu   sync.Mutex
	body map[string][]byte
}

// load 返回 id 已存日志的副本。
func (l *memLog) load(id string) []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]byte(nil), l.body[id]...)
}

// save 记下 id 的日志。
func (l *memLog) save(id string, body []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.body == nil {
		l.body = map[string][]byte{}
	}
	l.body[id] = append([]byte(nil), body...)
}

// lines 返回 id 已存日志收成的会话行，每行写成「角色 来源 名字 正文」，空着的来源和名字不写。
func (l *memLog) lines(id string) []string {
	events, _ := decodeLog(l.load(id))
	var out []string
	for _, line := range TranscriptLines(events) {
		parts := []string{line.Role}
		for _, part := range []string{line.Source, line.Name} {
			if part != "" {
				parts = append(parts, part)
			}
		}
		out = append(out, strings.Join(append(parts, line.Text), " "))
	}
	return out
}

// agentNames 是测试用的名字表。
func agentNames(agent string) string {
	switch agent {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude"
	}
	return ""
}

// TestReassignRestartsWithNewAgent 锁住改派：旧派工结束、新派工交给新 agent，交接说明带上进展；
// 旧工人这一轮停下后被关掉，不再被催，它迟到的报告被拒；新工人从交接那条消息开始，做完照常结束工单。
func TestReassignRestartsWithNewAgent(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(1), nil)
	ex.AgentName = agentNames
	logs := &memLog{}
	ex.LogLoad, ex.LogSave = logs.load, logs.save
	release := make(chan struct{})
	late := make(chan dsh.Result, 1)
	opened := make(chan *fakeWorker, 2)
	ex.Opener = func(_ context.Context, spec WorkerSpec) (Worker, error) {
		w := newFakeWorker(spec, func(w *fakeWorker, msg dsh.UserMessage) {
			if msg.Source != "assignment" {
				return
			}
			if spec.Agent == "codex" {
				w.say("codex 先改了一半")
				w.report("progress", "改了一半")
				<-release
				late <- w.report("ready", "codex 说做完了")
				return
			}
			w.report("ready", "claude 做完了")
		})
		opened <- w
		return w, nil
	}
	id := accept(t, eng, "m1")
	setAgent(eng, id, "codex")
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	codex := <-opened
	waitFor(t, "codex progress", func() bool { return lastSummary(eng, id) == "改了一半" })

	if err := eng.Reassign(id, Handover{Agent: "claude", Name: "Claude", Pinned: true}); err != nil {
		t.Fatal(err)
	}
	ex.Interrupt(id)
	ticket, _ := eng.Ticket(id)
	n := len(ticket.Assignments)
	old, cur := ticket.Assignments[n-2], ticket.Assignments[n-1]
	if ticket.Agent != "claude" || !ticket.AgentPinned || ticket.Pointer != "" ||
		old.State != "ended" || old.Agent != "codex" || cur.State != "pending" || cur.Agent != "claude" || cur.From != "codex" {
		t.Fatalf("ticket %+v", ticket)
	}
	note := ticket.Inputs[len(ticket.Inputs)-1].Text
	if !strings.HasPrefix(note, "改由 Claude 接手。") || !strings.Contains(note, "之前的进展：改了一半。") {
		t.Fatalf("note %q", note)
	}
	if err := eng.Reassign(id, Handover{Agent: "claude"}); err == nil || err.Error() != "sameAgent" {
		t.Fatalf("same agent: %v", err)
	}
	if err := ex.Launch(id); err == nil || err.Error() != "already" {
		t.Fatalf("the old run still holds the slot: %v", err)
	}
	snap, _ := eng.Export()
	view := buildView(snap, nil, "", "", 1)
	row := view.Tickets[0]
	if row.Agent != "claude" || !row.AgentPinned || len(row.Assignments) != 2 || row.Assignments[0].Agent != "codex" || row.Assignments[1].Agent != "claude" {
		t.Fatalf("row %+v", row)
	}
	loads := agentLoads(view.Tickets, ex.AgentRunning())
	if len(loads) != 2 || loads[0] != (AgentLoad{Agent: "claude", Queued: 1}) || loads[1] != (AgentLoad{Agent: "codex", Running: 1}) {
		t.Fatalf("loads %+v", loads)
	}

	close(release)
	if res := <-late; !res.IsError || res.Code != "handedOff" {
		t.Fatalf("the old worker's report landed: %+v", res)
	}
	waitFor(t, "old worker closed", func() bool { return codex.closed.Load() })
	waitFor(t, "claude launch", func() bool { return ex.Launch(id) == nil })
	claude := <-opened
	waitOutcome(t, eng, id, "completed")
	if got := codex.messages(); len(got) != 1 {
		t.Fatalf("the old worker was nudged: %+v", got)
	}
	events := claude.session.Events()
	first, _ := events[0].Data.(dsh.UserMessage)
	if events[0].Type != dsh.EventUser || first.Source != "handover" || first.Text != "改由 Claude 接手" {
		t.Fatalf("first event %+v", events[0])
	}
	got := claude.messages()
	if len(got) != 1 || got[0].Source != "assignment" || !strings.Contains(got[0].Text, "改由 Claude 接手。") {
		t.Fatalf("claude got %+v", got)
	}
	if lastSummary(eng, id) != "claude 做完了" {
		t.Fatalf("summary %q", lastSummary(eng, id))
	}
}

// TestHandoverPrefixKeepsDisplay 锁住交接之后新工人读不到旧 agent 的工具调用，存下的日志仍是全部；
// 同一次派工重启时不再写第二条交接，旧工人迟到的写盘不会把日志写重。
func TestHandoverPrefixKeepsDisplay(t *testing.T) {
	ex := NewExecutor(NewEngine(autoPolicy()), NewSlots(1), nil)
	logs := &memLog{}
	ex.LogLoad, ex.LogSave = logs.load, logs.save
	id := "t1"
	old := dsh.NewSession()
	old.Append(dsh.EventUser, 1, 0, dsh.UserMessage{Text: "工单 t1\n改 README\n完成后调用 orchestrator_report。", Source: "assignment"})
	old.Append(dsh.EventAssistant, 1, 1, dsh.Assistant{ToolCalls: []dsh.ToolCall{{ID: "t1-c1", Name: "read_file", Arguments: `{"path":"README.md"}`}}})
	old.Append(dsh.EventToolResult, 1, 1, dsh.ToolResultEvent{ID: "t1-c1", Name: "read_file", Text: "# readme"})
	old.Append(dsh.EventTurnEnd, 1, 1, dsh.TurnEnd{Kind: "completed"})
	body, _ := encodeLog(old.Events())
	logs.save(id, body)
	marker := &dsh.UserMessage{ID: handoverID("asg-2"), Source: "handover", Text: "改由 yoyo 接手"}

	first := newFakeWorker(WorkerSpec{}, nil)
	if ex.restoreAgent(first, id, marker) {
		t.Fatal("a fresh handover restores nothing")
	}
	if evs := first.session.Events(); len(evs) != 1 || !hasMessage(evs, marker.ID) {
		t.Fatalf("new worker starts with %+v", evs)
	}
	first.Followup(dsh.UserMessage{Text: "接着改", Source: "assignment"})
	first.say("改好了")
	ex.storeAgent(first, id)
	want := []string{"user assignment 改 README", "tool_call read_file " + `{"path":"README.md"}`, "tool_result read_file # readme", "user handover asg-2 改由 yoyo 接手", "user assignment 接着改", "assistant 改好了"}
	if got := logs.lines(id); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("lines:\n%s", strings.Join(got, "\n"))
	}
	if _, turn := decodeLog(logs.load(id)); turn != 2 {
		t.Fatalf("turn %d: the new worker keeps counting after the old one", turn)
	}

	again := newFakeWorker(WorkerSpec{}, nil)
	if !ex.restoreAgent(again, id, marker) {
		t.Fatal("a restart restores the new segment")
	}
	evs := again.session.Events()
	if len(evs) == 0 || !hasMessage(evs[:1], marker.ID) {
		t.Fatalf("restart starts with %+v", evs)
	}
	for _, ev := range evs {
		if a, ok := ev.Data.(dsh.Assistant); ok && len(a.ToolCalls) > 0 {
			t.Fatalf("the new worker read the old agent's calls: %+v", a)
		}
	}
	ex.storeAgent(first, id)
	ex.storeAgent(again, id)
	if got := logs.lines(id); len(got) != len(want) {
		t.Fatalf("the log was doubled:\n%s", strings.Join(got, "\n"))
	}

	later := newFakeWorker(WorkerSpec{}, nil)
	if !ex.restoreAgent(later, id, nil) || !hasMessage(later.session.Events()[:1], marker.ID) {
		t.Fatalf("a later assignment starts after the handover: %+v", later.session.Events())
	}
}

// TestOfflinePinnedAsks 锁住人点名的 agent 离线时不擅自换人：报 blocked 并问人；人选 yoyo 后改派过去，问题清掉。
func TestOfflinePinnedAsks(t *testing.T) {
	eng := NewEngine(autoPolicy())
	ex := NewExecutor(eng, NewSlots(1), nil)
	ex.AgentName = agentNames
	var mu sync.Mutex
	var relayed []string
	ex.OnReport = func(_, state, _, question string) {
		mu.Lock()
		relayed = append(relayed, state+"|"+question)
		mu.Unlock()
	}
	ex.Opener = func(_ context.Context, spec WorkerSpec) (Worker, error) {
		return nil, fmt.Errorf("%s: %w", spec.Agent, ErrAgentOffline)
	}
	if eng.Receive(human("m1", "@agent:codex 修一下登录页")) != "stored" {
		t.Fatal("stored")
	}
	d, err := eng.DecideBriefFor("m1", "accept", "修登录页", "修一下登录页", nil, "codex", true)
	if err != nil {
		t.Fatal(err)
	}
	id := d.TicketID
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	want := "Codex 离线。改用 yoyo，还是等它上线？"
	waitFor(t, "ask", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(relayed) == 1
	})
	waitFor(t, "stopped", func() bool { return notLive(ex, id) })
	ticket, _ := eng.Ticket(id)
	mu.Lock()
	got := relayed[0]
	mu.Unlock()
	if got != "blocked|"+want || ticket.Attention != want || ticket.Agent != "codex" || !ticket.AgentPinned || ticket.Ended != nil {
		t.Fatalf("relayed %q ticket %+v", got, ticket)
	}
	if run := ex.AgentRunning(); len(run) != 0 {
		t.Fatalf("running %v", run)
	}

	if err := eng.Reassign(id, Handover{Agent: YoyoAgent, Pinned: true}); err != nil {
		t.Fatal(err)
	}
	ticket, _ = eng.Ticket(id)
	cur := ticket.Assignments[len(ticket.Assignments)-1]
	if ticket.Agent != "" || ticket.AgentPinned || ticket.Attention != "" || cur.State != "pending" || cur.Agent != YoyoAgent || cur.From != "codex" {
		t.Fatalf("ticket %+v", ticket)
	}
	if note := ticket.Inputs[len(ticket.Inputs)-1].Text; !strings.HasPrefix(note, "改由 yoyo 接手。之前的进展：Codex 离线。") {
		t.Fatalf("note %q", note)
	}
}

// TestOfflineUnpinnedFallsBack 锁住没点名的 agent 离线时退回 yoyo：同一次派工改由 yoyo 做完，日志先写一条交接；
// agent 要先登录时不换人，报 blocked 等人登录。
func TestOfflineUnpinnedFallsBack(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var mu sync.Mutex
	var seen []string
	model := scriptModel{fn: func(_ context.Context, req dsh.Request) (dsh.Assistant, error) {
		mu.Lock()
		for _, msg := range req.Messages {
			seen = append(seen, msg.Text)
		}
		mu.Unlock()
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"yoyo 做完了"}`}}}, nil
	}}
	ex := NewExecutor(eng, NewSlots(1), model)
	ex.AgentName = agentNames
	logs := &memLog{}
	ex.LogLoad, ex.LogSave = logs.load, logs.save
	ex.Opener = func(_ context.Context, spec WorkerSpec) (Worker, error) {
		return nil, fmt.Errorf("%s: %w", spec.Agent, ErrAgentOffline)
	}
	id := accept(t, eng, "m1")
	setAgent(eng, id, "codex")
	if err := ex.Launch(id); err != nil {
		t.Fatal(err)
	}
	waitOutcome(t, eng, id, "completed")
	ticket, _ := eng.Ticket(id)
	asg := ticket.Assignments[len(ticket.Assignments)-1]
	if ticket.Agent != "" || len(ticket.Assignments) != 1 || asg.Agent != YoyoAgent || asg.From != "codex" || lastSummary(eng, id) != "yoyo 做完了" {
		t.Fatalf("ticket %+v", ticket)
	}
	if lines := logs.lines(id); len(lines) == 0 || lines[0] != "user handover "+asg.ID+" Codex 离线，改由 yoyo 做" {
		t.Fatalf("lines %q", lines)
	}
	mu.Lock()
	joined := strings.Join(seen, "\n")
	mu.Unlock()
	if !strings.Contains(joined, "Codex 离线，改由 yoyo 做") {
		t.Fatalf("yoyo was not told about the handover:\n%s", joined)
	}
	waitFor(t, "released", func() bool { return notLive(ex, id) && len(ex.AgentRunning()) == 0 })

	ex.Opener = func(_ context.Context, spec WorkerSpec) (Worker, error) {
		return nil, fmt.Errorf("%s: %s: %w", spec.Agent, ReasonAuthRequired, ErrAgentOffline)
	}
	other := acceptOn(t, eng, "m2", "room-b")
	setAgent(eng, other, "codex")
	if err := ex.Launch(other); err != nil {
		t.Fatal(err)
	}
	want := "Codex 需要登录。去 Buddy 登录后回复继续。"
	waitFor(t, "login", func() bool {
		got, _ := eng.Ticket(other)
		return got.Attention == want && notLive(ex, other)
	})
	got, _ := eng.Ticket(other)
	last := got.Assignments[len(got.Assignments)-1]
	report := last.Reports[len(last.Reports)-1]
	if got.Agent != "codex" || report.State != "blocked" || report.Need != "login" || got.Ended != nil {
		t.Fatalf("ticket %+v", got)
	}
}
