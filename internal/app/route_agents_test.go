// route_agents_test.go 锁住路由这一侧的 agent 选择：@ 过的 agent 核对后记进触发、写进说明；task_route 按它开单；
// 没点名的 agent 被拒并说清能填什么；continue 换 agent 时改派，新工人从交接开始。也锁住应用关掉后编排不再被回调打开。

package app

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
	"github.com/Shenchangxin/yoyo/internal/buddy"
)

// openAgentsApp 打开应用，加上连假 agent 的自定义 agent ids，并等它们在线。名字是首字母大写的编号。
func openAgentsApp(t *testing.T, script acptest.Script, ids ...string) (*App, *fakeAgents) {
	t.Helper()
	f := &fakeAgents{script: script}
	a := openBuddyApp(t, t.TempDir(), f)
	t.Cleanup(func() { _ = a.Close() })
	m := a.Buddies()
	for _, id := range ids {
		if _, err := m.Add(buddy.CustomAgent{ID: id, Name: strings.ToUpper(id[:1]) + id[1:], Command: fakeCommand(t)}); err != nil {
			t.Fatal(err)
		}
		probeOnline(t, m, id)
	}
	return a, f
}

// route 调一次 task_route，返回回执。单项的错误也写在回执里。
func route(t *testing.T, a *App, item map[string]any) string {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"items": []any{item}})
	res := a.routerExtras()["task_route"].Call(string(raw))
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	return res.Content
}

// quietTurn 是假 agent 的一轮：说一句就停，不写报告。
func quietTurn(turn *acptest.Turn) acp.StopReason {
	turn.Say("在做。")
	return acp.StopEndTurn
}

// TestOrchestratorStaysClosed 锁住应用关掉之后编排不会被回调重新打开。外部工人在关闭途中失败时会回报，
// 重新打开会让数据库一直被占着。
func TestOrchestratorStaysClosed(t *testing.T) {
	a := openBuddyApp(t, t.TempDir(), &fakeAgents{})
	if a.orchestrator() == nil {
		t.Fatal("no orchestrator")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if rt := a.orchestrator(); rt != nil {
		rt.Stop()
		t.Fatal("orchestrator reopened after Close")
	}
}

// TestIntakeDeskNotesAgents 锁住 @ 过的 agent 用 Manager 核对：认得的记进触发并写明谁来做、附上可派名单；
// 认不出的丢掉并说一句；停用的照样记下，但让路由去问人；没有 @ 也没有默认设置时什么都不加。
func TestIntakeDeskNotesAgents(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps}, "fake", "lazy")
	off := false
	if _, err := a.Buddies().Configure("lazy", buddy.ConfigPatch{Enabled: &off}); err != nil {
		t.Fatal(err)
	}
	rt := a.orchestrator()

	note := a.intakeDesk("s1", "t1", "@agent:fake 修一下登录页")
	for _, want := range []string{"人指定 fake 来做。accept 时 agent 填 fake。", "可派的 agent：\nyoyo ", "fake Fake，在线"} {
		if !strings.Contains(note, want) {
			t.Fatalf("note lacks %q:\n%s", want, note)
		}
	}
	if got := rt.TriggerAgents("desk:s1:t1"); !slices.Equal(got, []string{"fake"}) {
		t.Fatalf("agents %v", got)
	}

	note = a.intakeDesk("s1", "t2", "@agent:nobody @agent:lazy 看看")
	for _, want := range []string{"人指定 lazy 来做", "@agent:nobody 不是已知的 agent，已忽略。", "lazy 现在离线（已停用）。告诉人，问是等它上线还是改用 yoyo。不要擅自换人。"} {
		if !strings.Contains(note, want) {
			t.Fatalf("note lacks %q:\n%s", want, note)
		}
	}
	if strings.Contains(note, "lazy Lazy") {
		t.Fatalf("a disabled agent is on the roster:\n%s", note)
	}
	if got := rt.TriggerAgents("desk:s1:t2"); !slices.Equal(got, []string{"lazy"}) {
		t.Fatalf("agents %v", got)
	}

	if note = a.intakeDesk("s1", "t3", "随便聊聊"); strings.Contains(note, "agent") {
		t.Fatalf("no mention adds no agent lines:\n%s", note)
	}
}

// TestRouteAcceptWithAgent 锁住只 @ 了一个 agent 时，accept 不填 agent 也交给它，并记成点名；没 @ 时用默认 agent，不算点名。
func TestRouteAcceptWithAgent(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps, OnPrompt: quietTurn}, "fake")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "@agent:fake 写个 hello.txt")
	got := route(t, a, map[string]any{"trigger": "desk:s1:t1", "action": "accept", "title": "hello", "brief": "在工单目录写 hello.txt。"})
	if got != "accepted desk:s1:t1 for fake" {
		t.Fatalf("route = %q", got)
	}
	row, ok := ticketFor(rt, "desk:s1:t1")
	if !ok || row.Agent != "fake" || !row.AgentPinned {
		t.Fatalf("row %+v", row)
	}

	a.buddyMu.Lock()
	a.Config.Buddies.DefaultAgent = "fake"
	a.buddyMu.Unlock()
	a.intakeDesk("s1", "t2", "再写个 bye.txt")
	pick, pinned, err := a.routeAgent(rt, "desk:s1:t2", "")
	if pick != "fake" || pinned || err != nil {
		t.Fatalf("default = %q %v %v", pick, pinned, err)
	}
	if pick, pinned, err = a.routeAgent(rt, "desk:s1:t2", "yoyo"); pick != "" || pinned || err != nil {
		t.Fatalf("yoyo = %q %v %v", pick, pinned, err)
	}
}

// TestRouteRejectsUnmentionedAgent 锁住路由只能把工单交给人 @ 过的 agent 或 yoyo：
// 填了别的 agent 被拒，回执说清能填什么，也不开单；@ 了两个却不填时要它选一个。
func TestRouteRejectsUnmentionedAgent(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps}, "fake", "other")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "写个 hello.txt")
	got := route(t, a, map[string]any{"trigger": "desk:s1:t1", "action": "accept", "title": "hello", "brief": "写 hello.txt。", "agent": "fake"})
	if got != "人没有点名 fake。人没有点名 agent，agent 只能填 yoyo 或留空。" {
		t.Fatalf("route = %q", got)
	}
	if _, ok := ticketFor(rt, "desk:s1:t1"); ok {
		t.Fatal("a rejected item opened a ticket")
	}

	a.intakeDesk("s1", "t2", "@agent:fake 写个 hello.txt")
	got = route(t, a, map[string]any{"trigger": "desk:s1:t2", "action": "accept", "title": "hello", "brief": "写 hello.txt。", "agent": "other"})
	if got != "人没有点名 other。agent 只能填人 @ 过的 fake，或 yoyo。" {
		t.Fatalf("route = %q", got)
	}

	a.intakeDesk("s1", "t3", "@agent:fake @agent:other 写个 hello.txt")
	if _, _, err := a.routeAgent(rt, "desk:s1:t3", ""); err == nil || err.Error() != "人点了 fake、other，agent 要填其中一个" {
		t.Fatalf("two mentions = %v", err)
	}
}

// TestRouteContinueReassigns 锁住 continue 填了和工单现在不同的 agent 时改派：工单换人并记成点名，新工人从交接那条开始；
// 再填同一个 agent 时照常继续，不再改派。
func TestRouteContinueReassigns(t *testing.T) {
	a, f := openAgentsApp(t, acptest.Script{Caps: fakeCaps, OnPrompt: quietTurn}, "fake", "other")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "@agent:fake 写个 hello.txt")
	if got := route(t, a, map[string]any{"trigger": "desk:s1:t1", "action": "accept", "title": "hello", "brief": "在工单目录写 hello.txt。"}); got != "accepted desk:s1:t1 for fake" {
		t.Fatalf("accept = %q", got)
	}
	row, _ := ticketFor(rt, "desk:s1:t1")
	waitFor(t, "fake worker", func() bool { return len(f.prompted()) >= 1 })

	a.intakeDesk("s1", "t2", "@agent:other 换 other 接着写")
	got := route(t, a, map[string]any{"trigger": "desk:s1:t2", "action": "continue", "ticket": row.ID, "brief": "接着写 hello.txt。", "agent": "other"})
	if got != "reassigned "+row.ID+" to other" {
		t.Fatalf("continue = %q", got)
	}
	handover := ""
	waitFor(t, "other worker", func() bool {
		for _, line := range rt.WorkerTranscript(row.ID) {
			if line.Source == "handover" && line.Text == "改由 Other 接手" {
				handover = line.Name
				return true
			}
		}
		return false
	})
	row, _ = ticketFor(rt, "desk:s1:t1")
	last := row.Assignments[len(row.Assignments)-1]
	if row.Agent != "other" || !row.AgentPinned || last.Agent != "other" || row.Assignments[0].Agent != "fake" || handover != last.ID {
		t.Fatalf("handover line names %q, row %+v", handover, row)
	}

	a.intakeDesk("s1", "t3", "@agent:other 再补一句")
	if got := route(t, a, map[string]any{"trigger": "desk:s1:t3", "action": "continue", "ticket": row.ID, "brief": "在末尾加一行。", "agent": "other"}); got != "continued "+row.ID {
		t.Fatalf("same agent = %q", got)
	}

	if err := a.OrchestratorAct("reassign", row.ID, "", "nobody", ""); err == nil || err.Error() != "nobody 不是已知的 agent" {
		t.Fatalf("unknown agent = %v", err)
	}
	if err := a.OrchestratorAct("reassign", row.ID, "", "Fake", ""); err != nil {
		t.Fatal(err)
	}
	if row, _ = ticketFor(rt, "desk:s1:t1"); row.Agent != "fake" || !row.AgentPinned {
		t.Fatalf("person's reassign: %+v", row)
	}
}
