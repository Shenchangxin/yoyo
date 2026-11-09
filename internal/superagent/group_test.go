// group_test.go 锁住任务组：拆分开出成员、共享目录和成员说明，坏的拆法什么都不改；有前置的成员挂着，前置做完带上结果放行，
// 前置没做成就挂起、重试做成后再放行；整组的状态、同一个状态只唤醒一次和唤醒上限；停掉整组、对比采用一份；
// 快照来回、旧快照照常能读；挂着的成员不会被启动；两份 yoyo 前后接着做完、只唤醒一次的验收；以及写给路由的摘要和 task_read 的详情。

package superagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// deskAsk 造一条桌面会话 s1 里操作者的话。agents 是人 @ 过的 agent。
func deskAsk(id, text string, agents ...string) Trigger {
	return Trigger{
		ID: id, Source: "desk", Threads: []string{"desk:s1"}, Primary: "desk:s1",
		Human: true, Addressed: true, Sender: "operator", Text: text, Agents: agents,
	}
}

// launchLog 记下引擎经 OnLaunch 启动过的工单。
type launchLog struct {
	ids []string
}

// watch 接到 eng 的 OnLaunch 上。
func (l *launchLog) watch(eng *Engine) {
	eng.OnLaunch = func(id string) { l.ids = append(l.ids, id) }
}

// receiveSplit 在 eng 上收下 msg，再按 req 拆分。req.Trigger 用 msg 的编号。
func receiveSplit(t *testing.T, eng *Engine, msg Trigger, req SplitRequest) (Group, []string) {
	t.Helper()
	if eng.Receive(msg) != "stored" {
		t.Fatal("stored")
	}
	req.Trigger = msg.ID
	g, launches, err := eng.Split(req)
	if err != nil {
		t.Fatal(err)
	}
	return g, launches
}

// backFront 是两份的分工：后端交给 codex，前端交给 yoyo，前端等后端做完。共享目录放在 root 下。
func backFront(root string) SplitRequest {
	return SplitRequest{Mode: GroupSplit, Title: "做 todo 应用", Root: root, Names: map[string]string{"codex": "Codex"}, Parts: []SplitPart{
		{Agent: "codex", Role: "后端", Brief: "写后端接口"},
		{Agent: "yoyo", Role: "前端", Brief: "写前端页面", DependsOn: []int{0}},
	}}
}

// twoParts 是两份互不相干、都交给 yoyo 的分工。
func twoParts() SplitRequest {
	return SplitRequest{Mode: GroupSplit, Title: "两件事", Parts: []SplitPart{
		{Agent: "yoyo", Role: "甲", Brief: "做甲"},
		{Agent: "yoyo", Role: "乙", Brief: "做乙"},
	}}
}

// lastInput 返回 ticketID 最后一条输入的正文。
func lastInput(eng *Engine, ticketID string) string {
	ticket, _ := eng.Ticket(ticketID)
	if len(ticket.Inputs) == 0 {
		return ""
	}
	return ticket.Inputs[len(ticket.Inputs)-1].Text
}

// lastReportOf 返回 ticketID 最后一份不是进度的报告。
func lastReportOf(eng *Engine, ticketID string) Report {
	ticket, _ := eng.Ticket(ticketID)
	rep, _ := lastReport(&ticket)
	return rep
}

// TestSplitOpensGroupAndTickets 锁住拆分：成员编号连着，人点名的 agent 记成点名，有前置的挂着，只有没前置的启动；
// 成员说明写明共享目录、同组成员和要等谁；只回一次已接单；决定记着任务组；同一条触发再拆一次拿回原来的任务组，不再启动。
func TestSplitOpensGroupAndTickets(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var started launchLog
	started.watch(eng)
	root := t.TempDir()
	g, launches := receiveSplit(t, eng, deskAsk("m1", "做个 todo 应用，后端给 @agent:codex", "codex"), backFront(root))
	if !strings.HasPrefix(g.ID, "grp-") || g.Mode != GroupSplit || g.Title != "做 todo 应用" || g.Session != "s1" || g.Trigger != "m1" ||
		len(g.Tickets) != 2 || g.SharedDir != filepath.Join(root, "groups", g.ID) {
		t.Fatalf("group %+v", g)
	}
	back, front := g.Tickets[0], g.Tickets[1]
	if ticketOrder(front) != ticketOrder(back)+1 || ticketOrder(back) != groupOrder(g.ID)+1 {
		t.Fatalf("members %s %v", g.ID, g.Tickets)
	}
	if len(launches) != 1 || launches[0] != back || len(started.ids) != 1 || started.ids[0] != back {
		t.Fatalf("launches %v started %v", launches, started.ids)
	}
	b, _ := eng.Ticket(back)
	f, _ := eng.Ticket(front)
	if b.Agent != "codex" || !b.AgentPinned || b.Group != g.ID || b.Role != "后端" || b.Title != "后端" || b.Held != "" || len(b.DependsOn) != 0 {
		t.Fatalf("back %+v", b)
	}
	if f.Agent != "" || f.AgentPinned || f.Group != g.ID || f.Role != "前端" || f.Held != HeldDeps || len(f.DependsOn) != 1 || f.DependsOn[0] != back {
		t.Fatalf("front %+v", f)
	}
	brief := f.Inputs[0].Text
	for _, want := range []string{"写前端页面", "\n" + g.SharedDir + "\n", "NOTES.md", back + " Codex · 后端", back + " 做完后你才开工"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("front brief misses %q:\n%s", want, brief)
		}
	}
	if got := b.Inputs[0].Text; !strings.Contains(got, front+" yoyo · 前端") || strings.Contains(got, "做完后你才开工") {
		t.Fatalf("back brief %q", got)
	}
	acks := 0
	for _, msg := range eng.Outbox() {
		if msg.Kind == "acknowledgement" {
			acks++
		}
	}
	if acks != 1 {
		t.Fatalf("acks %d", acks)
	}
	snap, _ := eng.Export()
	d := snap.Decisions[len(snap.Decisions)-1]
	if d.Kind != "accept" || d.Group != g.ID || d.TicketID != back || d.TriggerID != "m1" {
		t.Fatalf("decision %+v", d)
	}
	again, more, err := eng.Split(SplitRequest{Trigger: "m1", Mode: GroupSplit, Title: "x", Parts: twoParts().Parts})
	if err != nil || again.ID != g.ID || len(more) != 0 || len(started.ids) != 1 || len(eng.Groups()) != 1 {
		t.Fatalf("again %+v %v %v", again, more, err)
	}
}

// TestSplitRejectsBadDeps 锁住前置只能指向前面的项：指向自己、后面或越界都拒绝；份数、说明、agent、标题和模式不对也拒绝，什么都不改。
func TestSplitRejectsBadDeps(t *testing.T) {
	eng := NewEngine(autoPolicy())
	if eng.Receive(deskAsk("m1", "做个应用")) != "stored" {
		t.Fatal("stored")
	}
	part := func(brief string, deps ...int) SplitPart {
		return SplitPart{Agent: "yoyo", Brief: brief, DependsOn: deps}
	}
	cases := []struct {
		name  string
		req   SplitRequest
		code  string
		parts []SplitPart
	}{
		{name: "self", code: "bad_deps", parts: []SplitPart{part("a"), part("b", 1)}},
		{name: "forward", code: "bad_deps", parts: []SplitPart{part("a", 1), part("b")}},
		{name: "negative", code: "bad_deps", parts: []SplitPart{part("a"), part("b", -1)}},
		{name: "one part", code: "bad_parts", parts: []SplitPart{part("a")}},
		{name: "seven parts", code: "bad_parts", parts: []SplitPart{part("a"), part("b"), part("c"), part("d"), part("e"), part("f"), part("g")}},
		{name: "empty brief", code: "empty_brief", parts: []SplitPart{part("a"), part(" ")}},
		{name: "empty agent", code: "empty_agent", parts: []SplitPart{part("a"), {Brief: "b"}}},
		{name: "empty title", code: "empty_title", req: SplitRequest{Mode: GroupSplit, Title: " "}, parts: []SplitPart{part("a"), part("b")}},
		{name: "bad mode", code: "bad_mode", req: SplitRequest{Mode: "race", Title: "应用"}, parts: []SplitPart{part("a"), part("b")}},
	}
	for _, c := range cases {
		req := c.req
		if req.Mode == "" {
			req = SplitRequest{Mode: GroupSplit, Title: "应用"}
		}
		req.Trigger, req.Parts = "m1", c.parts
		if _, _, err := eng.Split(req); err == nil || err.Error() != c.code {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
	tr, _ := eng.Trigger("m1")
	snap, _ := eng.Export()
	if len(eng.Groups()) != 0 || len(snap.Tickets) != 0 || len(snap.Decisions) != 0 || tr.DecisionID != "" {
		t.Fatalf("a bad split changed the engine: %+v", snap)
	}
}

// TestCompareRejectsDeps 锁住对比：不能写前置、agent 不能重复、最多四个、要有说明；合规时每份拿到同一份说明，
// 写明还交给了谁、独立完成，没有共享目录，全部马上启动。
func TestCompareRejectsDeps(t *testing.T) {
	eng := NewEngine(autoPolicy())
	if eng.Receive(deskAsk("m1", "codex 和 claude 都做一遍，比一比", "codex", "claude")) != "stored" {
		t.Fatal("stored")
	}
	base := SplitRequest{Trigger: "m1", Mode: GroupCompare, Title: "登录页", Brief: "做一个登录页"}
	try := func(code string, parts ...SplitPart) {
		t.Helper()
		req := base
		req.Parts = parts
		if code == "empty_brief" {
			req.Brief = ""
		}
		if _, _, err := eng.Split(req); err == nil || err.Error() != code {
			t.Fatalf("%s: %v", code, err)
		}
	}
	try("compare_deps", SplitPart{Agent: "codex"}, SplitPart{Agent: "claude", DependsOn: []int{0}})
	try("dup_agents", SplitPart{Agent: "codex"}, SplitPart{Agent: "CODEX"})
	try("bad_parts", SplitPart{Agent: "a"}, SplitPart{Agent: "b"}, SplitPart{Agent: "c"}, SplitPart{Agent: "d"}, SplitPart{Agent: "e"})
	try("empty_brief", SplitPart{Agent: "codex"}, SplitPart{Agent: "claude"})
	req := base
	req.Root = t.TempDir()
	req.Names = map[string]string{"codex": "Codex", "claude": "Claude"}
	req.Parts = []SplitPart{{Agent: "codex"}, {Agent: "claude"}, {Agent: "yoyo"}}
	g, launches, err := eng.Split(req)
	if err != nil || g.SharedDir != "" || len(launches) != 3 {
		t.Fatalf("compare %+v %v %v", g, launches, err)
	}
	for i, id := range g.Tickets {
		ticket, _ := eng.Ticket(id)
		brief := ticket.Inputs[0].Text
		if !strings.HasPrefix(brief, "做一个登录页") || !strings.Contains(brief, "同一份说明也交给了") || !strings.Contains(brief, "独立完成，不要看别人的目录") || ticket.Title != "登录页" {
			t.Fatalf("member %d brief %q title %q", i, brief, ticket.Title)
		}
	}
	codex, _ := eng.Ticket(g.Tickets[0])
	if !strings.Contains(codex.Inputs[0].Text, "Claude（"+g.Tickets[1]+"）、yoyo（"+g.Tickets[2]+"）") || strings.Contains(codex.Inputs[0].Text, "Codex（") {
		t.Fatalf("codex brief %q", codex.Inputs[0].Text)
	}
}

// TestSplitLaunchesRootsOnly 锁住只启动没有前置的成员；暂停时人明确交代的照常启动，推断的一份都不启动。
func TestSplitLaunchesRootsOnly(t *testing.T) {
	parts := []SplitPart{
		{Agent: "yoyo", Role: "接口", Brief: "a"},
		{Agent: "yoyo", Role: "页面", Brief: "b", DependsOn: []int{0}},
		{Agent: "yoyo", Role: "文档", Brief: "c"},
	}
	eng := NewEngine(autoPolicy())
	g, launches := receiveSplit(t, eng, deskAsk("m1", "做"), SplitRequest{Mode: GroupSplit, Title: "应用", Parts: parts})
	if len(launches) != 2 || launches[0] != g.Tickets[0] || launches[1] != g.Tickets[2] {
		t.Fatalf("launches %v of %v", launches, g.Tickets)
	}
	policy := autoPolicy()
	policy.Paused = true
	policy.Rules = []Rule{{ID: "copy", Text: "文案小改可以直接做", Enabled: true}}
	paused := NewEngine(policy)
	if _, launches = receiveSplit(t, paused, deskAsk("m2", "做"), SplitRequest{Mode: GroupSplit, Title: "应用", Parts: parts}); len(launches) != 2 {
		t.Fatalf("explicit work while paused %v", launches)
	}
	msg := human("m3", "首页标题改一下")
	msg.Addressed = false
	g, launches = receiveSplit(t, paused, msg, SplitRequest{Mode: GroupSplit, Title: "文案", Rules: []string{"copy"}, Parts: parts})
	if len(launches) != 0 {
		t.Fatalf("inferred work while paused %v", launches)
	}
	if ticket, _ := paused.Ticket(g.Tickets[0]); ticket.Mandate.Strength != "inferred" || len(ticket.Mandate.Rules) != 1 {
		t.Fatalf("mandate %+v", ticket.Mandate)
	}
}

// TestDependentReleasedWithPredecessorSummary 锁住前置做完后放行：清掉挂着，追加一条写明谁做的、做成了什么、目录在哪的输入，再启动；
// 派工正文带任务组那一行，任务页去掉它；外部工人拿到前置目录只读。
func TestDependentReleasedWithPredecessorSummary(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var started launchLog
	started.watch(eng)
	g, _ := receiveSplit(t, eng, deskAsk("m1", "做个 todo 应用", "codex"), backFront(t.TempDir()))
	back, front := g.Tickets[0], g.Tickets[1]
	names := map[string]string{"codex": "Codex"}
	if launches, wakes := eng.WatchGroups(names); len(launches) != 0 || len(wakes) != 0 || eng.DepsState(front) != DepsWaiting {
		t.Fatalf("released too early %v %v", launches, wakes)
	}
	dir := `D:\work\tickets\` + back
	eng.SetDirectory(back, dir, "todo-api")
	if err := eng.Report(back, Report{State: "ready", Summary: "接口写好了，在 /api/todos。"}); err != nil {
		t.Fatal(err)
	}
	if eng.DepsState(front) != DepsReady {
		t.Fatal(eng.DepsState(front))
	}
	launches, wakes := eng.WatchGroups(names)
	if len(launches) != 1 || launches[0] != front || started.ids[len(started.ids)-1] != front || len(wakes) != 0 {
		t.Fatalf("launches %v started %v wakes %v", launches, started.ids, wakes)
	}
	f, _ := eng.Ticket(front)
	want := "前置任务都完成了：\n" + back + " Codex · 后端：接口写好了，在 /api/todos。目录 " + dir
	if f.Held != "" || lastInput(eng, front) != want || f.Inputs[len(f.Inputs)-1].Human {
		t.Fatalf("held %q note %q", f.Held, lastInput(eng, front))
	}
	if again, _ := eng.WatchGroups(names); len(again) != 0 {
		t.Fatalf("released twice %v", again)
	}
	ex := NewExecutor(eng, NewSlots(1), nil)
	line := ex.groupLine(f)
	if line != "任务组 "+g.ID+"「做 todo 应用」（分工） · 你负责：前端" {
		t.Fatalf("group line %q", line)
	}
	body := assignmentText(f, "", "", line)
	shown := assignmentBody(body)
	if !strings.Contains(body, "\n"+line+"\n") || strings.Contains(shown, groupLinePrefix) || !strings.Contains(shown, "前置任务都完成了") {
		t.Fatalf("body %q shown %q", body, shown)
	}
	shared, reads := eng.MemberDirs(front)
	if shared != g.SharedDir || len(reads) != 1 || reads[0] != dir {
		t.Fatalf("dirs %q %v", shared, reads)
	}
	if shared, reads := eng.MemberDirs(back); shared != g.SharedDir || len(reads) != 0 {
		t.Fatalf("back dirs %q %v", shared, reads)
	}
}

// TestDependentHeldWhenDepFails 锁住前置没做成时挂起后面的成员：Held 记成 deps_failed，记一份不带问题的 blocked，
// 整组要人处理，只唤醒一次；前置是开工失败结束的也一样。
func TestDependentHeldWhenDepFails(t *testing.T) {
	eng := NewEngine(autoPolicy())
	g, _ := receiveSplit(t, eng, deskAsk("m1", "做个 todo 应用", "codex"), backFront(""))
	back, front := g.Tickets[0], g.Tickets[1]
	if err := eng.Report(back, Report{State: "failed", Summary: "依赖装不上"}); err != nil {
		t.Fatal(err)
	}
	if eng.DepsState(front) != DepsFailed {
		t.Fatal(eng.DepsState(front))
	}
	launches, wakes := eng.WatchGroups(nil)
	f, _ := eng.Ticket(front)
	rep := lastReportOf(eng, front)
	if len(launches) != 0 || f.Held != HeldDepsFailed || rep.State != "blocked" || rep.Question != "" || !strings.Contains(rep.Summary, back+" codex · 后端") {
		t.Fatalf("launches %v held %q report %+v", launches, f.Held, rep)
	}
	if eng.GroupState(g.ID) != GroupAttention || len(wakes) != 1 || wakes[0].State != GroupAttention || wakes[0].Session != "s1" || wakes[0].Count != 1 || wakes[0].Group != g.ID {
		t.Fatalf("state %s wakes %+v", eng.GroupState(g.ID), wakes)
	}
	n := len(f.Assignments[len(f.Assignments)-1].Reports)
	if _, again := eng.WatchGroups(nil); len(again) != 0 {
		t.Fatalf("woke twice %+v", again)
	}
	if f, _ = eng.Ticket(front); len(f.Assignments[len(f.Assignments)-1].Reports) != n {
		t.Fatal("held twice")
	}

	eng2 := NewEngine(autoPolicy())
	g2, _ := receiveSplit(t, eng2, deskAsk("m1", "做个 todo 应用", "codex"), backFront(""))
	if err := eng2.FailStart(g2.Tickets[0], "目录建不出来"); err != nil {
		t.Fatal(err)
	}
	eng2.WatchGroups(nil)
	if f2, _ := eng2.Ticket(g2.Tickets[1]); f2.Held != HeldDepsFailed {
		t.Fatalf("after a failed start %+v", f2)
	}
}

// TestHeldReleasedAfterRetry 锁住挂起的成员在前置重试做成后照样放行，重试进行中还挂着。
func TestHeldReleasedAfterRetry(t *testing.T) {
	eng := NewEngine(autoPolicy())
	var started launchLog
	started.watch(eng)
	g, _ := receiveSplit(t, eng, deskAsk("m1", "做个 todo 应用", "codex"), backFront(""))
	back, front := g.Tickets[0], g.Tickets[1]
	_ = eng.Report(back, Report{State: "failed", Summary: "依赖装不上"})
	eng.WatchGroups(nil)
	if err := eng.Retry(back); err != nil {
		t.Fatal(err)
	}
	if eng.DepsState(front) != DepsWaiting || eng.GroupState(g.ID) != GroupAttention {
		t.Fatalf("deps %s state %s", eng.DepsState(front), eng.GroupState(g.ID))
	}
	if launches, _ := eng.WatchGroups(nil); len(launches) != 0 {
		t.Fatalf("released while the retry runs %v", launches)
	}
	if err := eng.Report(back, Report{State: "ready", Summary: "换了源，装好了"}); err != nil {
		t.Fatal(err)
	}
	launches, _ := eng.WatchGroups(nil)
	f, _ := eng.Ticket(front)
	if len(launches) != 1 || launches[0] != front || f.Held != "" || !strings.Contains(lastInput(eng, front), "换了源，装好了") {
		t.Fatalf("launches %v held %q note %q", launches, f.Held, lastInput(eng, front))
	}
	if stage := TicketStage(&f); stage.Person || eng.GroupState(g.ID) != GroupRunning {
		t.Fatalf("stage %+v state %s", stage, eng.GroupState(g.ID))
	}
}

// TestGroupStates 锁住整组状态怎样从成员算出，以及任务页的任务组行。
func TestGroupStates(t *testing.T) {
	ready := func(eng *Engine, id string) { _ = eng.Report(id, Report{State: "ready", Summary: "好了"}) }
	cases := []struct {
		name, mode, want string
		act              func(eng *Engine, g Group)
	}{
		{"running", GroupSplit, GroupRunning, func(*Engine, Group) {}},
		{"question", GroupSplit, GroupAttention, func(eng *Engine, g Group) {
			_ = eng.Report(g.Tickets[0], Report{State: "blocked", Summary: "要选库", Question: "用 SQLite 还是 Postgres？"})
		}},
		{"all done", GroupSplit, GroupDone, func(eng *Engine, g Group) { ready(eng, g.Tickets[0]); ready(eng, g.Tickets[1]) }},
		{"one failed", GroupSplit, GroupFailed, func(eng *Engine, g Group) {
			ready(eng, g.Tickets[0])
			_ = eng.FailStart(g.Tickets[1], "目录建不出来")
		}},
		{"one stopped by the person", GroupSplit, GroupStopped, func(eng *Engine, g Group) {
			ready(eng, g.Tickets[0])
			_ = eng.Stop(g.Tickets[1], "不要了")
		}},
		{"compare with one result", GroupCompare, GroupDone, func(eng *Engine, g Group) {
			ready(eng, g.Tickets[0])
			_ = eng.FailStart(g.Tickets[1], "连不上")
		}},
		{"compare without results", GroupCompare, GroupFailed, func(eng *Engine, g Group) {
			_ = eng.FailStart(g.Tickets[0], "连不上")
			_ = eng.FailStart(g.Tickets[1], "连不上")
		}},
		{"group stopped", GroupSplit, GroupStopped, func(eng *Engine, g Group) { _, _ = eng.StopGroup(g.ID, "") }},
		{"picked", GroupCompare, GroupDone, func(eng *Engine, g Group) { _, _ = eng.Pick(g.ID, g.Tickets[0], "") }},
	}
	for _, c := range cases {
		eng := NewEngine(autoPolicy())
		req := SplitRequest{Mode: c.mode, Title: "两份", Brief: "做", Parts: []SplitPart{{Agent: "yoyo", Brief: "a"}, {Agent: "codex", Brief: "b"}}}
		g, _ := receiveSplit(t, eng, deskAsk("m1", "做", "codex"), req)
		c.act(eng, g)
		if got := eng.GroupState(g.ID); got != c.want {
			t.Fatalf("%s: %s", c.name, got)
		}
		snap, _ := eng.Export()
		view := buildView(snap, nil, "", "", 1)
		row := view.Groups[0]
		done := 0
		if c.name == "all done" {
			done = 2
		} else if c.name == "one failed" || c.name == "one stopped by the person" || c.name == "compare with one result" {
			done = 1
		}
		if row.ID != g.ID || row.State != c.want || row.Total != 2 || row.Done != done || row.Mode != c.mode || row.OpenedAt == "" || row.UpdatedAt < row.OpenedAt {
			t.Fatalf("%s: row %+v", c.name, row)
		}
		for _, ticket := range view.Tickets {
			if ticket.Group != g.ID {
				t.Fatalf("%s: ticket row %+v", c.name, ticket)
			}
		}
	}
	if got := groupState(nil, nil); got != "" {
		t.Fatal(got)
	}
}

// TestWakeOncePerSettle 锁住整组结束只唤醒一次，摘要说明这是任务组的回报并带上该怎么做；
// 排队时的标记已经过时的唤醒作废；回调没接上时稍后再试。
func TestWakeOncePerSettle(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.mu.Lock()
	rt.wireEffects()
	rt.mu.Unlock()
	type woke struct{ session, group, state, digest string }
	var got []woke
	rt.SetOnWake(func(session, group, state, digest string) error {
		got = append(got, woke{session, group, state, digest})
		return nil
	})
	rt.Receive(deskAsk("m1", "做两件事"))
	req := twoParts()
	req.Trigger = "m1"
	g, err := rt.Split(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range g.Tickets {
		if err := rt.eng.Report(id, Report{State: "ready", Summary: "做好了 " + id}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		rt.watchGroups()
		rt.effects.Pump(time.Now())
	}
	if len(got) != 1 || got[0].session != "s1" || got[0].group != g.ID || got[0].state != GroupDone {
		t.Fatalf("woke %+v", got)
	}
	for _, want := range []string{"不是人说的话", "做好了 " + g.Tickets[0], "分工都做完了"} {
		if !strings.Contains(got[0].digest, want) {
			t.Fatalf("digest misses %q:\n%s", want, got[0].digest)
		}
	}
	stale, _ := json.Marshal(wakeBody{Session: "s1", Group: g.ID, State: GroupDone, Mark: "done:1"})
	if err := rt.runWake(string(stale)); err != nil || len(got) != 1 {
		t.Fatalf("stale wake ran: %v %d", err, len(got))
	}
	rt.SetOnWake(nil)
	current, _ := json.Marshal(wakeBody{Session: "s1", Group: g.ID, State: GroupDone, Mark: rt.eng.GroupMark(g.ID)})
	if err := rt.runWake(string(current)); !errors.Is(err, ErrLater) {
		t.Fatalf("unwired wake: %v", err)
	}
}

// TestWakeCap 锁住每个任务组最多唤醒六次，之后只写一句通知；同一会话一小时里唤醒满十次后也只写通知。
func TestWakeCap(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.Receive(deskAsk("m1", "做两件事"))
	req := twoParts()
	req.Trigger = "m1"
	g, err := rt.Split(req)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < groupWakeCap+3; i++ {
		if err := rt.eng.Report(g.Tickets[0], Report{State: "blocked", Summary: "卡住了", Question: fmt.Sprintf("第 %d 个问题？", i)}); err != nil {
			t.Fatal(err)
		}
		rt.watchGroups()
	}
	count := func(rt *Runtime) (wakes, notices int) {
		for _, row := range rt.effects.ExportPending() {
			switch row.Kind {
			case "wake":
				wakes++
			case "notice":
				notices++
				if !strings.Contains(row.Payload, "停下了") {
					t.Fatalf("notice %s", row.Payload)
				}
			}
		}
		return wakes, notices
	}
	if wakes, notices := count(rt); wakes != groupWakeCap || notices != 1 {
		t.Fatalf("wakes %d notices %d", wakes, notices)
	}

	busy, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	busy.Receive(deskAsk("m1", "做两件事"))
	g, err = busy.Split(req)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	busy.wakes = map[string][]time.Time{"s1": make([]time.Time, sessionWakeCap)}
	for i := range busy.wakes["s1"] {
		busy.wakes["s1"][i] = now
	}
	_ = busy.eng.Report(g.Tickets[0], Report{State: "blocked", Summary: "卡住了", Question: "怎么办？"})
	busy.watchGroups()
	if wakes, notices := count(busy); wakes != 0 || notices != 1 {
		t.Fatalf("hourly cap: wakes %d notices %d", wakes, notices)
	}
}

// TestStopGroup 锁住停掉整组：没结束的成员都记成人停下，做完的不动，任务组记成 stopped，不再唤醒；
// 经编排台动作停时打断这些成员，并各写回一句。
func TestStopGroup(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	eng := rt.eng
	parts := []SplitPart{{Agent: "yoyo", Brief: "a"}, {Agent: "yoyo", Brief: "b", DependsOn: []int{0}}, {Agent: "yoyo", Brief: "c"}}
	g, _ := receiveSplit(t, eng, deskAsk("m1", "做"), SplitRequest{Mode: GroupSplit, Title: "应用", Parts: parts})
	_ = eng.Report(g.Tickets[2], Report{State: "ready", Summary: "好了"})
	if err := rt.Act("stopGroup", "", "", g.ID, "不做了"); err != nil {
		t.Fatal(err)
	}
	for _, id := range g.Tickets[:2] {
		ticket, _ := eng.Ticket(id)
		if ticket.Ended == nil || ticket.Ended.By != "human" || ticket.Ended.Outcome != "stopped" || ticket.Ended.Reason != "不做了" {
			t.Fatalf("%s %+v", id, ticket.Ended)
		}
	}
	if done, _ := eng.Ticket(g.Tickets[2]); done.Ended == nil || done.Ended.Outcome != "completed" {
		t.Fatalf("finished member %+v", done.Ended)
	}
	got, _ := eng.Group(g.ID)
	if got.Ended == nil || got.Ended.Outcome != "stopped" || eng.GroupState(g.ID) != GroupStopped {
		t.Fatalf("group %+v", got)
	}
	if _, err := eng.StopGroup(g.ID, ""); err == nil || err.Error() != "groupEnded" {
		t.Fatal(err)
	}
	if _, err := eng.StopGroup("grp-404", ""); err == nil || err.Error() != "unknownGroup" {
		t.Fatal(err)
	}
	if _, wakes := eng.WatchGroups(nil); len(wakes) != 0 {
		t.Fatalf("woke a stopped group %+v", wakes)
	}
	relays := 0
	for _, row := range rt.effects.ExportPending() {
		if row.Kind == "relay" && strings.Contains(row.Payload, `"state":"stopped"`) && strings.Contains(row.Payload, "不做了") {
			relays++
		}
	}
	if relays != 2 {
		t.Fatalf("relays %d", relays)
	}
}

// TestPickStopsOthers 锁住对比采用一份：其他没结束的成员停下，任务组记下采用者并结束；分工不能采用，不在组里的工单也不行。
func TestPickStopsOthers(t *testing.T) {
	eng := NewEngine(autoPolicy())
	req := SplitRequest{Mode: GroupCompare, Title: "登录页", Brief: "做一个登录页", Parts: []SplitPart{{Agent: "codex"}, {Agent: "claude"}, {Agent: "yoyo"}}}
	g, _ := receiveSplit(t, eng, deskAsk("m1", "比一比", "codex", "claude"), req)
	_ = eng.Report(g.Tickets[0], Report{State: "ready", Summary: "codex 版好了"})
	if _, err := eng.Pick(g.ID, "tkt-404", ""); err == nil || err.Error() != "notMember" {
		t.Fatal(err)
	}
	stopped, err := eng.Pick(g.ID, g.Tickets[0], "")
	if err != nil || len(stopped) != 2 || stopped[0] != g.Tickets[1] || stopped[1] != g.Tickets[2] {
		t.Fatalf("stopped %v %v", stopped, err)
	}
	got, _ := eng.Group(g.ID)
	if got.Winner != g.Tickets[0] || got.Ended == nil || got.Ended.Outcome != "picked" || got.Ended.Reason != "采用了 "+g.Tickets[0] || eng.GroupState(g.ID) != GroupDone {
		t.Fatalf("group %+v", got)
	}
	for _, id := range stopped {
		if ticket, _ := eng.Ticket(id); ticket.Ended == nil || ticket.Ended.Outcome != "stopped" {
			t.Fatalf("%s %+v", id, ticket.Ended)
		}
	}
	if _, err := eng.Pick(g.ID, g.Tickets[1], ""); err == nil || err.Error() != "groupEnded" {
		t.Fatal(err)
	}
	split, _ := receiveSplit(t, eng, deskAsk("m2", "分工"), twoParts())
	if _, err := eng.Pick(split.ID, split.Tickets[0], ""); err == nil || err.Error() != "notCompare" {
		t.Fatal(err)
	}
}

// TestSnapshotRoundTripGroups 锁住任务组和成员字段经快照来回不变，编号接着往下发。
func TestSnapshotRoundTripGroups(t *testing.T) {
	eng := NewEngine(autoPolicy())
	g, _ := receiveSplit(t, eng, deskAsk("m1", "做个 todo 应用", "codex"), backFront(t.TempDir()))
	if err := eng.SetGroupVerdict(g.ID, "后端好了，前端在做"); err != nil {
		t.Fatal(err)
	}
	if !eng.MarkWoke(g.ID, "attention:3") || eng.MarkWoke(g.ID, "attention:3") {
		t.Fatal("mark")
	}
	snap, err := eng.Export()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(snap)
	var back Snapshot
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	restored := NewEngine(autoPolicy())
	restored.Import(back)
	got, ok := restored.Group(g.ID)
	if !ok || got.Mode != g.Mode || got.Title != g.Title || got.Session != g.Session || got.SharedDir != g.SharedDir || got.Trigger != g.Trigger ||
		strings.Join(got.Tickets, ",") != strings.Join(g.Tickets, ",") || got.Verdict != "后端好了，前端在做" || len(got.Woke) != 1 || !got.Opened.Equal(g.Opened) {
		t.Fatalf("group %+v want %+v", got, g)
	}
	front, _ := restored.Ticket(g.Tickets[1])
	if front.Group != g.ID || front.Role != "前端" || front.Held != HeldDeps || len(front.DependsOn) != 1 || front.DependsOn[0] != g.Tickets[0] {
		t.Fatalf("front %+v", front)
	}
	if restored.DepsState(front.ID) != DepsWaiting {
		t.Fatal(restored.DepsState(front.ID))
	}
	next, _ := receiveSplit(t, restored, deskAsk("m2", "再来"), twoParts())
	if groupOrder(next.ID) <= ticketOrder(g.Tickets[1]) {
		t.Fatalf("ids restarted: %s after %v", next.ID, g.Tickets)
	}
}

// TestOldSnapshotLoads 锁住没有任务组字段的旧快照照常能读，视图里任务组是空列表，之后照常能拆分。
func TestOldSnapshotLoads(t *testing.T) {
	old := `{"Policy":{"Version":1,"Mode":"auto"},` +
		`"Triggers":[{"ID":"m1","Source":"desk","Threads":["desk:s1"],"Primary":"desk:s1","Human":true,"Addressed":true,"Sender":"operator","Text":"修一下","DecisionID":"dec-3"}],` +
		`"Tickets":[{"ID":"tkt-1","Origin":"m1","Mandate":{"Version":1,"Strength":"explicit"},"Assignments":[{"ID":"asg-2","State":"pending"}]}],` +
		`"Decisions":[{"ID":"dec-3","TriggerID":"m1","Kind":"accept","TicketID":"tkt-1"}],"Seq":1,"N":3}`
	var snap Snapshot
	if err := json.Unmarshal([]byte(old), &snap); err != nil {
		t.Fatal(err)
	}
	eng := NewEngine(autoPolicy())
	eng.Import(snap)
	ticket, ok := eng.Ticket("tkt-1")
	if !ok || ticket.Group != "" || ticket.Held != "" || eng.DepsState("tkt-1") != DepsReady || len(eng.Groups()) != 0 {
		t.Fatalf("ticket %+v", ticket)
	}
	view := buildView(snap, nil, "", "", 1)
	raw, _ := json.Marshal(view)
	if view.Groups == nil || !strings.Contains(string(raw), `"groups":[]`) {
		t.Fatalf("view groups %v", view.Groups)
	}
	g, _ := receiveSplit(t, eng, deskAsk("m2", "再来"), twoParts())
	if g.ID != "grp-4" || g.Tickets[0] != "tkt-5" {
		t.Fatalf("group %+v", g)
	}
}

// TestRunLaunchSkipsHeld 锁住挂着的成员即使被排进启动也不开工人，没有前置的照常开。
func TestRunLaunchSkipsHeld(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	ex := NewExecutor(rt.eng, NewSlots(2), nil)
	ex.Root = t.TempDir()
	opened := make(chan WorkerSpec, 2)
	ex.Opener = func(_ context.Context, spec WorkerSpec) (Worker, error) {
		opened <- spec
		return newFakeWorker(spec, func(w *fakeWorker, msg dsh.UserMessage) {
			if msg.Source == "assignment" {
				w.report("ready", "好了")
			}
		}), nil
	}
	rt.exec = ex
	rt.Receive(deskAsk("m1", "做", "codex"))
	req := SplitRequest{Trigger: "m1", Mode: GroupSplit, Title: "应用", Parts: []SplitPart{
		{Agent: "codex", Role: "后端", Brief: "a"},
		{Agent: "codex", Role: "前端", Brief: "b", DependsOn: []int{0}},
	}}
	g, err := rt.Split(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.runLaunch(g.Tickets[1]); err != nil {
		t.Fatal(err)
	}
	select {
	case <-opened:
		t.Fatal("a held member got a worker")
	case <-time.After(100 * time.Millisecond):
	}
	if ticket, _ := rt.eng.Ticket(g.Tickets[1]); ticket.Assignments[len(ticket.Assignments)-1].State != "pending" {
		t.Fatalf("held assignment %+v", ticket.Assignments)
	}
	if err := rt.runLaunch(g.Tickets[0]); err != nil {
		t.Fatal(err)
	}
	select {
	case spec := <-opened:
		if spec.TicketID != g.Tickets[0] || len(spec.ReadRoots) != 0 {
			t.Fatalf("opened %+v", spec)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the root member never started")
	}
	waitOutcome(t, rt.eng, g.Tickets[0], "completed")
}

// TestTwoYoyoPartsRunInOrder 是只用 yoyo 的验收：第二份等第一份，第一份做完后第二份自己开工，派工里带着第一份的摘要；
// 两份都做完后只唤醒路由一次，摘要里有两份的结果。走的是 Bind 接上的执行器和两秒轮次的 Pass，不是直接改引擎。
func TestTwoYoyoPartsRunInOrder(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	fronts := make(chan string, 4)
	model := scriptModel{fn: func(_ context.Context, req dsh.Request) (dsh.Assistant, error) {
		var b strings.Builder
		for _, msg := range req.Messages {
			b.WriteString(msg.Text + "\n")
		}
		summary := "后端接口写好了"
		if text := b.String(); strings.Contains(text, "写前端页面") {
			summary = "前端页面写好了"
			fronts <- text
		}
		return dsh.Assistant{ToolCalls: []dsh.ToolCall{{ID: "r1", Name: "orchestrator_report", Arguments: `{"state":"ready","summary":"` + summary + `"}`}}}, nil
	}}
	rt.SetWorkRoot(t.TempDir())
	rt.Bind(nil, model, nil)
	woke := make(chan string, 4)
	rt.SetOnWake(func(session, group, state, digest string) error {
		woke <- session + "\n" + group + "\n" + state + "\n" + digest
		return nil
	})
	rt.Receive(deskAsk("m1", "做 todo 应用"))
	g, err := rt.Split(SplitRequest{Trigger: "m1", Mode: GroupSplit, Title: "做 todo 应用", Parts: []SplitPart{
		{Agent: "yoyo", Role: "后端", Brief: "写后端接口"},
		{Agent: "yoyo", Role: "前端", Brief: "写前端页面", DependsOn: []int{0}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var got string
	deadline := time.Now().Add(5 * time.Second)
	for got == "" {
		if time.Now().After(deadline) {
			t.Fatalf("the router was never woken; group %+v", rt.View().Groups)
		}
		rt.Pass(time.Now())
		select {
		case got = <-woke:
		case <-time.After(20 * time.Millisecond):
		}
	}
	select {
	case front := <-fronts:
		for _, want := range []string{"前置任务都完成了：", g.Tickets[0], "后端接口写好了"} {
			if !strings.Contains(front, want) {
				t.Fatalf("the second part did not get %q:\n%s", want, front)
			}
		}
	default:
		t.Fatal("the second part never ran")
	}
	if !strings.HasPrefix(got, "s1\n"+g.ID+"\n"+GroupDone+"\n") {
		t.Fatalf("woke %q", got)
	}
	for _, want := range []string{"后端接口写好了", "前端页面写好了", "分工都做完了"} {
		if !strings.Contains(got, want) {
			t.Fatalf("digest misses %q:\n%s", want, got)
		}
	}
	for i := 0; i < 3; i++ {
		rt.Pass(time.Now())
	}
	select {
	case again := <-woke:
		t.Fatalf("woke twice:\n%s", again)
	default:
	}
	waitFor(t, "workers released", func() bool { return notLive(rt.exec, g.Tickets[0]) && notLive(rt.exec, g.Tickets[1]) })
	rt.Stop()
}

// TestGroupDigestStaysShort 锁住任务组摘要最多三千字，成员太多时省掉后面的，行动规则总在最后。
func TestGroupDigestStaysShort(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.Receive(deskAsk("m1", "做六件事"))
	var parts []SplitPart
	for i := 0; i < splitMax; i++ {
		parts = append(parts, SplitPart{Agent: "yoyo", Role: fmt.Sprintf("第%d份", i), Brief: "做"})
	}
	g, err := rt.Split(SplitRequest{Trigger: "m1", Mode: GroupSplit, Title: "六件事", Parts: parts})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range g.Tickets {
		rt.eng.SetDirectory(id, strings.Repeat(`D:\很长的目录\`, 60)+id, "")
		_ = rt.eng.Report(id, Report{State: "ready", Summary: strings.Repeat("做了很多事。", 60)})
	}
	digest := rt.GroupDigest(g.ID)
	if n := len([]rune(digest)); n > digestMax {
		t.Fatalf("digest has %d runes", n)
	}
	if !strings.HasPrefix(digest, "这是任务组 "+g.ID+"「六件事」的回报") || !strings.HasSuffix(digest, "除非人之前要求过。") || !strings.Contains(digest, "- "+g.Tickets[0]+" yoyo · 第0份 · 做完了") {
		t.Fatalf("digest:\n%s", digest)
	}
	if rt.GroupDigest("grp-404") != "" {
		t.Fatal("unknown group")
	}
}

// TestReadTicketForRouter 锁住 task_read 读到的详情：工人最后一段话、最后一份报告、写过的文件（新的在前、去重，命令行输出和失败的不算）、目录和用量。
func TestReadTicketForRouter(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.Receive(deskAsk("m1", "比一比", "codex"))
	g, err := rt.Split(SplitRequest{Trigger: "m1", Mode: GroupCompare, Title: "登录页", Brief: "做一个登录页", Parts: []SplitPart{{Agent: "codex"}, {Agent: "yoyo"}}})
	if err != nil {
		t.Fatal(err)
	}
	id := g.Tickets[0]
	sess := dsh.NewSession()
	sess.Append(dsh.EventUser, 1, 0, dsh.UserMessage{Text: "工单 " + id + "\n做一个登录页", Source: "assignment"})
	sess.Append(dsh.EventAssistant, 1, 1, dsh.Assistant{Text: "先写页面"})
	sess.Append(dsh.EventToolResult, 1, 1, dsh.ToolResultEvent{ID: "c1", Name: "write_file", Text: "wrote login.html"})
	sess.Append(dsh.EventToolResult, 1, 1, dsh.ToolResultEvent{ID: "c2", Name: "shell", Text: "wrote fake.txt"})
	sess.Append(dsh.EventToolResult, 1, 1, dsh.ToolResultEvent{ID: "c3", Name: "write_file", Text: "wrote broken.css", IsError: true})
	sess.Append(dsh.EventToolResult, 1, 1, dsh.ToolResultEvent{ID: "c4", Name: "edit_file", Text: "wrote login.css\ndiff login.css: 1 -> 2 lines"})
	sess.Append(dsh.EventToolResult, 1, 1, dsh.ToolResultEvent{ID: "c5", Name: "write_file", Text: "wrote login.html"})
	sess.Append(dsh.EventAssistant, 1, 2, dsh.Assistant{Text: strings.Repeat("页面做好了。", 500)})
	sess.Append(dsh.EventTurnEnd, 1, 2, dsh.TurnEnd{Kind: "completed"})
	body, _ := encodeLog(sess.Events())
	rt.saveWorkerLog(id, body)
	rt.eng.SetDirectory(id, `D:\work\`+id, "")
	rt.eng.SetUsage(id, Usage{Used: 53000, Size: 200000, Cost: 0.05, Currency: "USD"})
	_ = rt.eng.Report(id, Report{State: "ready", Summary: "登录页做好了"})
	read, ok := rt.ReadTicket(id)
	if !ok || read.Agent != "codex" || read.Group != g.ID || read.Title != "登录页" || read.Directory != `D:\work\`+id || read.State != "做完了" {
		t.Fatalf("read %+v", read)
	}
	if n := len([]rune(read.Text)); n != 2000 || !strings.HasPrefix(read.Text, "页面做好了。") {
		t.Fatalf("text has %d runes", n)
	}
	if read.Report == nil || read.Report.Summary != "登录页做好了" || strings.Join(read.Files, ",") != "login.html,login.css" {
		t.Fatalf("report %+v files %v", read.Report, read.Files)
	}
	if read.Usage != "上下文 53k/200k · 0.05 USD" {
		t.Fatalf("usage %q", read.Usage)
	}
	if _, ok := rt.ReadTicket("tkt-404"); ok {
		t.Fatal("unknown ticket")
	}
}

// TestSplitWritesNotes 锁住拆分时建好共享目录并写一份 NOTES.md，写明成员、分工和要等谁；已有的 NOTES.md 不覆盖。
func TestSplitWritesNotes(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.SetAgentName(func(agent string) string { return agentNames(agent) })
	root := t.TempDir()
	rt.SetWorkRoot(root)
	rt.Receive(deskAsk("m1", "做个 todo 应用", "codex"))
	req := backFront("")
	req.Trigger = "m1"
	req.Names = nil
	g, err := rt.Split(req)
	if err != nil {
		t.Fatal(err)
	}
	if g.SharedDir != filepath.Join(root, "groups", g.ID) {
		t.Fatalf("shared %q", g.SharedDir)
	}
	raw, err := os.ReadFile(filepath.Join(g.SharedDir, "NOTES.md"))
	if err != nil {
		t.Fatal(err)
	}
	notes := string(raw)
	for _, want := range []string{"# 任务组 " + g.ID + "：做 todo 应用", "- " + g.Tickets[0] + " Codex · 后端", "- " + g.Tickets[1] + " yoyo · 前端（等 " + g.Tickets[0] + " 做完再开始）"} {
		if !strings.Contains(notes, want) {
			t.Fatalf("notes miss %q:\n%s", want, notes)
		}
	}
	if front, _ := rt.eng.Ticket(g.Tickets[1]); !strings.Contains(front.Inputs[0].Text, g.Tickets[0]+" Codex · 后端") {
		t.Fatalf("names did not reach the brief: %q", front.Inputs[0].Text)
	}
	_ = os.WriteFile(filepath.Join(g.SharedDir, "NOTES.md"), []byte("人改过"), 0o644)
	rt.writeNotes(g, nil)
	if raw, _ := os.ReadFile(filepath.Join(g.SharedDir, "NOTES.md")); string(raw) != "人改过" {
		t.Fatalf("notes overwritten: %q", raw)
	}
}

// TestTicketFilesMatchesTaskPage 锁住写过的文件和任务页 taskFiles 认法一致。
func TestTicketFilesMatchesTaskPage(t *testing.T) {
	lines := []TranscriptLine{
		{Role: "tool_result", Name: "write_file", Text: "wrote report.md"},
		{Role: "tool_result", Name: "shell", Text: "wrote fake.txt"},
		{Role: "tool_result", Name: "write_file", Text: "wrote broken.md", Error: true},
		{Role: "tool_result", Name: "str_replace", Text: "replaced 2 occurrence(s) in notes.md"},
		{Role: "tool_result", Name: "append_file", Text: "appended 12 bytes to log.txt (file is now 40 bytes)"},
		{Role: "tool_result", Name: "move_file", Text: "moved a.txt -> b.txt"},
		{Role: "tool_result", Name: "write_file", Text: "wrote D:\\proj\\out.html\nThis overwrote an existing page."},
		{Role: "assistant", Text: "wrote nothing.md"},
		{Role: "tool_result", Name: "write_file", Text: "wrote report.md"},
	}
	got := strings.Join(TicketFiles(lines), ",")
	if got != `report.md,D:\proj\out.html,b.txt,log.txt,notes.md` {
		t.Fatal(got)
	}
}
