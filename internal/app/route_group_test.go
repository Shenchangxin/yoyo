// route_group_test.go 锁住路由这一侧的任务组：task_split 校验每一份交给谁，坏的拆法给出能照着改的说明且什么都不开；
// 合规时开出任务组和共享目录、回执写明每一份的状态并提醒没分到的 agent、写一次任务组卡片，成员的回报带上任务组和分工；
// task_read 读到工人最后一段话；task_recommend 记下对比推荐哪一份；名单里写明任务组和在等谁；人点了几个 agent 时说明里提到 task_split，像要对比时点明 compare。

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// split 在会话 s1 上调一次 task_split，返回回执和错误。
func split(a *App, args map[string]any) (string, error) {
	raw, _ := json.Marshal(args)
	res := a.routerExtrasFor("s1")["task_split"].Call(string(raw))
	return res.Content, res.Err
}

// todoSplit 是 trigger 上两份的分工：后端交给 fake，前端交给 yoyo，等后端做完。
func todoSplit(trigger, title string) map[string]any {
	return map[string]any{"trigger": trigger, "mode": "split", "title": title, "parts": []any{
		map[string]any{"agent": "fake", "role": "后端", "brief": "在工单目录写后端接口。"},
		map[string]any{"agent": "yoyo", "role": "前端", "brief": "写前端页面，接口见共享目录。", "depends_on": []any{0}},
	}}
}

// groupCards 是会话 sessionID 里写下的任务组卡片。
func groupCards(t *testing.T, a *App, sessionID string) []trace.Event {
	t.Helper()
	evs, err := a.Traces.Read(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	var out []trace.Event
	for _, ev := range evs {
		if ev.Type == trace.TypeAssistant && ev.Payload["kind"] == "group" {
			out = append(out, ev)
		}
	}
	return out
}

// TestTaskSplitValidation 锁住 task_split 的校验：没点名的 agent、坏的前置、份数、模式、对比写了前置或重复 agent、不认得的触发都被拒，
// 说明能照着改，也不开任务组和工单；合规时回执写明每一份，建好共享目录和 NOTES.md，共享目录进可写目录；人 @ 过却没分到的 agent 提醒一句。
func TestTaskSplitValidation(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps, OnPrompt: quietTurn}, "fake", "other")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "@agent:fake 做个 todo 应用，前后端分开")
	trig := "desk:s1:t1"
	yoyo := func(brief string, deps ...any) map[string]any {
		return map[string]any{"agent": "yoyo", "brief": brief, "depends_on": deps}
	}
	bad := []struct {
		name string
		args map[string]any
		want string
	}{
		{"unmentioned agent", map[string]any{"trigger": trig, "mode": "split", "title": "todo", "parts": []any{map[string]any{"agent": "other", "brief": "a"}, yoyo("b")}},
			"parts[0]：人没有点名 other。agent 只能填人 @ 过的 fake，或 yoyo。"},
		{"forward deps", map[string]any{"trigger": trig, "mode": "split", "title": "todo", "parts": []any{yoyo("a", 1), yoyo("b")}},
			"depends_on 只能填排在前面的份在 parts 里的序号，从 0 开始，不能指向自己或后面的份"},
		{"not an index", map[string]any{"trigger": trig, "mode": "split", "title": "todo", "parts": []any{yoyo("a"), yoyo("b", "前端")}},
			"depends_on 只能填排在前面的份在 parts 里的序号，从 0 开始，不能指向自己或后面的份"},
		{"one part", map[string]any{"trigger": trig, "mode": "split", "title": "todo", "parts": []any{yoyo("a")}},
			"split 要拆成 2 到 6 份。只有一份就用 task_route accept"},
		{"bad mode", map[string]any{"trigger": trig, "mode": "race", "title": "todo", "parts": []any{yoyo("a"), yoyo("b")}},
			"mode 只能是 split（按分工拆成几份）或 compare（几个 agent 分别做同一件事再比较）"},
		{"compare with deps", map[string]any{"trigger": trig, "mode": "compare", "title": "todo", "brief": "做", "parts": []any{map[string]any{"agent": "fake"}, yoyo("", 0)}},
			"compare 不能写 depends_on，各自独立做"},
		{"compare twice", map[string]any{"trigger": trig, "mode": "compare", "title": "todo", "brief": "做", "agents": []any{"fake", "FAKE"}},
			"compare 的 agent 不能重复，每个 agent 只做一份"},
		{"compare without brief", map[string]any{"trigger": trig, "mode": "compare", "title": "todo", "agents": "fake, yoyo"},
			"compare 要写 brief：交给每个 agent 的同一份完整说明"},
		{"unknown trigger", map[string]any{"trigger": "desk:s1:nope", "mode": "split", "title": "todo", "parts": []any{yoyo("a"), yoyo("b")}},
			"没有这条触发 desk:s1:nope。trigger 要填这一轮材料里给的编号"},
	}
	for _, c := range bad {
		if got, err := split(a, c.args); err == nil || got != c.want {
			t.Fatalf("%s: %q %v", c.name, got, err)
		}
	}
	if len(rt.View().Groups) != 0 {
		t.Fatal("a rejected split opened a group")
	}
	if _, ok := ticketFor(rt, trig); ok {
		t.Fatal("a rejected split opened a ticket")
	}

	got, err := split(a, todoSplit(trig, "todo"))
	if err != nil {
		t.Fatal(err)
	}
	g := rt.View().Groups[0]
	back, front := g.Tickets[0], g.Tickets[1]
	for _, want := range []string{"开了任务组 " + g.ID + "「todo」（分工，2 份）：", back + " Fake · 后端：开工了", front + " yoyo · 前端：等 " + back + " 做完再开工", "对人只说一两句"} {
		if !strings.Contains(got, want) {
			t.Fatalf("receipt lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "没有分到") {
		t.Fatalf("everyone mentioned got a part:\n%s", got)
	}
	if g.SharedDir == "" {
		t.Fatalf("group %+v", g)
	}
	if _, err := os.Stat(filepath.Join(g.SharedDir, "NOTES.md")); err != nil {
		t.Fatal(err)
	}
	if roots := a.namedWorkRoots(front, a.Home.Root); !slices.Contains(roots, g.SharedDir) {
		t.Fatalf("roots %v lack %s", roots, g.SharedDir)
	}

	a.intakeDesk("s1", "t2", "@agent:fake @agent:other 再做一个 todo 应用")
	got, err = split(a, todoSplit("desk:s1:t2", "todo2"))
	if err != nil || !strings.Contains(got, "人 @ 过的 other 没有分到。") {
		t.Fatalf("receipt %q %v", got, err)
	}
}

// TestGroupCardEmitted 锁住开出任务组后在发起的会话里写一次卡片：编号、标题、模式和每个成员；同一条触发再拆一次不再写，回执说已经开过；
// 成员的回报写回会话时带上 agent、任务组和分工。
func TestGroupCardEmitted(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps, OnPrompt: quietTurn}, "fake")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "@agent:fake 做个 todo 应用")
	if _, err := split(a, todoSplit("desk:s1:t1", "todo")); err != nil {
		t.Fatal(err)
	}
	g := rt.View().Groups[0]
	cards := groupCards(t, a, "s1")
	if len(cards) != 1 {
		t.Fatalf("cards %d", len(cards))
	}
	p := cards[0].Payload
	if cards[0].Source != "worker" || p["id"] != "group:"+g.ID || p["group"] != g.ID || p["title"] != "todo" || p["mode"] != "split" {
		t.Fatalf("card %+v", p)
	}
	members, _ := p["members"].([]any)
	if len(members) != 2 {
		t.Fatalf("members %+v", p["members"])
	}
	back, _ := members[0].(map[string]any)
	front, _ := members[1].(map[string]any)
	deps, _ := front["dependsOn"].([]any)
	if back["ticket"] != g.Tickets[0] || back["agent"] != "fake" || back["agentName"] != "Fake" || back["role"] != "后端" ||
		front["agent"] != "yoyo" || len(deps) != 1 || deps[0] != g.Tickets[0] {
		t.Fatalf("members %+v", members)
	}
	if text, _ := p["text"].(string); !strings.HasPrefix(text, "任务组「todo」· 分工 2 份\n- "+g.Tickets[0]+" Fake · 后端") {
		t.Fatalf("text %q", text)
	}

	got, err := split(a, todoSplit("desk:s1:t1", "todo"))
	if err != nil || !strings.HasPrefix(got, "这条触发已经开过任务组 "+g.ID) || len(groupCards(t, a, "s1")) != 1 || len(rt.View().Groups) != 1 {
		t.Fatalf("again %q %v", got, err)
	}

	a.relayReport(g.Tickets[0], "ready", "接口写好了。", "")
	evs, _ := a.Traces.Read("s1")
	found := false
	for _, ev := range evs {
		if ev.Type == trace.TypeAssistant && ev.Payload["ticket"] == g.Tickets[0] {
			found = ev.Payload["group"] == g.ID && ev.Payload["role"] == "后端" && ev.Payload["agent"] == "fake"
		}
	}
	if !found {
		t.Fatal("the member's report lacks its group")
	}
}

// TestTaskReadTool 锁住 task_read 读到工单的状态、任务组、目录和工人最后一段话；没有的工单说清楚。
func TestTaskReadTool(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps, OnPrompt: quietTurn}, "fake")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "@agent:fake 写个登录页，和 yoyo 比一比")
	if _, err := split(a, map[string]any{"trigger": "desk:s1:t1", "mode": "compare", "title": "登录页", "brief": "在工单目录写 login.html。", "agents": []any{"fake", "yoyo"}}); err != nil {
		t.Fatal(err)
	}
	g := rt.View().Groups[0]
	id := g.Tickets[0]
	waitFor(t, "fake worker words", func() bool {
		for _, line := range rt.WorkerTranscript(id) {
			if line.Role == "assistant" && line.Text == "在做。" {
				return true
			}
		}
		return false
	})
	read := a.routerExtras()["task_read"]
	if !read.ReadOnly {
		t.Fatal("task_read must be read-only")
	}
	res := read.Call(`{"ticket":"` + id + `"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	for _, want := range []string{id + "「登录页」 · Fake · ", "\n任务组 " + g.ID + "\n", "\n目录 ", "工人最后说：\n在做。"} {
		if !strings.Contains(res.Content, want) {
			t.Fatalf("read lacks %q:\n%s", want, res.Content)
		}
	}
	if res = read.Call(`{"ticket":"tkt-404"}`); res.Err == nil || res.Content != "没有工单 tkt-404" {
		t.Fatalf("unknown ticket %+v", res)
	}
}

// TestTaskRecommendTool 锁住 task_recommend 记下对比推荐的那一份而不结束任务组，视图带出推荐；不在组里的工单给出能照着改的说明。
func TestTaskRecommendTool(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps, OnPrompt: quietTurn}, "fake")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "@agent:fake 写个登录页，和 yoyo 比一比")
	if _, err := split(a, map[string]any{"trigger": "desk:s1:t1", "mode": "compare", "title": "登录页", "brief": "在工单目录写 login.html。", "agents": []any{"fake", "yoyo"}}); err != nil {
		t.Fatal(err)
	}
	g := rt.View().Groups[0]
	recommend := a.routerExtras()["task_recommend"]
	if recommend.ReadOnly {
		t.Fatal("task_recommend writes to the group")
	}
	if res := recommend.Call(`{"group":"` + g.ID + `","ticket":"tkt-404"}`); res.Err == nil || res.Content != "tkt-404 不在任务组 "+g.ID+" 里" {
		t.Fatalf("not a member %+v", res)
	}
	if res := recommend.Call(`{"group":"` + g.ID + `","ticket":"` + g.Tickets[1] + `"}`); res.Err != nil || !strings.Contains(res.Content, "推荐 "+g.Tickets[1]) {
		t.Fatalf("recommend %+v", res)
	}
	if row := rt.View().Groups[0]; row.Recommended != g.Tickets[1] || row.Winner != "" || row.State == "stopped" {
		t.Fatalf("row %+v", row)
	}
}

// TestRosterShowsGroup 锁住名单里任务组成员写明在哪一组、做完了几份、负责什么，挂着的写明在等谁；
// 有任务组在做时，人没 @ 也给路由看可派名单。
func TestRosterShowsGroup(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps, OnPrompt: quietTurn}, "fake")
	rt := a.orchestrator()
	a.intakeDesk("s1", "t1", "@agent:fake 做个 todo 应用")
	if _, err := split(a, todoSplit("desk:s1:t1", "todo")); err != nil {
		t.Fatal(err)
	}
	g := rt.View().Groups[0]
	roster := a.taskRoster(rt)
	for _, want := range []string{
		"  任务组 " + g.ID + "（分工，0/2 做完） · 后端\n",
		"  任务组 " + g.ID + "（分工，0/2 做完） · 前端\n  等 " + g.Tickets[0] + " 做完才开工",
	} {
		if !strings.Contains(roster, want) {
			t.Fatalf("roster lacks %q:\n%s", want, roster)
		}
	}
	if status := a.routerExtras()["task_status"].Call("{}"); !strings.Contains(status.Content, "  任务组 "+g.ID+"（分工，0/2 做完） · 前端\n") {
		t.Fatalf("status %q", status.Content)
	}
	if note := a.intakeDesk("s1", "t2", "进展怎么样"); !strings.Contains(note, "可派的 agent：") || !strings.Contains(note, "fake Fake") {
		t.Fatalf("no roster while a group runs:\n%s", note)
	}
}

// TestIntakeSuggestsSplit 锁住人点了几个 agent 时说明里提到 task_split；话里像要对比时点明 compare，否则不点。
func TestIntakeSuggestsSplit(t *testing.T) {
	a, _ := openAgentsApp(t, acptest.Script{Caps: fakeCaps}, "fake", "other")
	note := a.intakeDesk("s1", "t1", "@agent:fake @agent:other 分别实现快速排序")
	for _, want := range []string{"人点了 fake、other。", "task_split", "人似乎要对比，多半用 compare。"} {
		if !strings.Contains(note, want) {
			t.Fatalf("note lacks %q:\n%s", want, note)
		}
	}
	if note = a.intakeDesk("s1", "t2", "@agent:fake @agent:other 做个 todo 应用，前后端分开"); !strings.Contains(note, "task_split") || strings.Contains(note, "人似乎要对比") {
		t.Fatalf("split note:\n%s", note)
	}
	if !compareWords.MatchString("codex vs claude") || !compareWords.MatchString("Let each agent try") || compareWords.MatchString("teach me") {
		t.Fatal("compare words")
	}
}
