// route_wake_test.go 锁住唤醒路由的一轮：没有用户行，任务组回报落成一条 wake 注入、回放时是一条用户消息；
// 会话在跑、槽没放、有排队或应用暂停时稍后再试，不动人排着的话；没有了的会话回到主会话；
// 做完了的任务组记下路由的最后一段话，别的状态不记。

package app

import (
	"errors"
	"os"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/superagent"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// openWakeApp 打开一个取不到真模型的应用和一条会话。唤醒那一轮的模型由测试直接给。
func openWakeApp(t *testing.T) (*App, SessionMeta) {
	t.Helper()
	a := openBuddyApp(t, t.TempDir(), &fakeAgents{})
	t.Cleanup(func() { _ = a.Close() })
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return a, sess
}

// reply 是只回一句 text 的模型。
func reply(text string) runtime.Client {
	return &runtime.ScriptedClient{Steps: []runtime.Message{{Role: runtime.RoleAssistant, Content: text}}}
}

// TestWakeTurnHasNoUserLine 锁住唤醒的一轮不写用户行：任务组回报落成一条带编号和状态的 wake 注入，路由的回复照常落下，
// 回放历史时回报是一条用户消息。
func TestWakeTurnHasNoUserLine(t *testing.T) {
	a, sess := openWakeApp(t)
	digest := "这是任务组 grp-1「两件事」的回报，不是人说的话。分工，2 份，现在都结束了。"
	if err := a.launchWake(sess.ID, "grp-1", superagent.GroupDone, digest, reply("两份都做完了。")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "wake turn", func() bool { return !a.runSlotHeld(sess.ID) })
	evs, err := a.Traces.Read(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	wakes, replies := 0, 0
	for _, ev := range evs {
		switch {
		case ev.Type == trace.TypeUser:
			t.Fatalf("a wake turn wrote a user line %+v", ev.Payload)
		case ev.Type == trace.TypeInject && ev.Source == "wake":
			wakes++
			if ev.Payload["text"] != digest || ev.Payload["group"] != "grp-1" || ev.Payload["state"] != superagent.GroupDone {
				t.Fatalf("wake %+v", ev.Payload)
			}
		case ev.Type == trace.TypeAssistant && ev.Payload["text"] == "两份都做完了。":
			replies++
		}
	}
	if wakes != 1 || replies != 1 {
		t.Fatalf("wakes %d replies %d", wakes, replies)
	}
	msgs := runtime.MessagesFromEvents(evs)
	if len(msgs) < 2 || msgs[len(msgs)-2].Role != runtime.RoleUser || msgs[len(msgs)-2].Content != digest {
		t.Fatalf("replay %+v", msgs)
	}
}

// TestWakeWaitsForBusySession 锁住会话在跑、跑完槽还没放、有排队的回合或应用暂停时唤醒稍后再试，排着的回合原样留着；
// 没有了的会话回到主会话，主会话也没有时丢掉这次唤醒。
func TestWakeWaitsForBusySession(t *testing.T) {
	a, sess := openWakeApp(t)
	later := func(what string) {
		t.Helper()
		if err := a.wakeRouter(sess.ID, "grp-1", superagent.GroupDone, "回报"); !errors.Is(err, superagent.ErrLater) {
			t.Fatalf("%s: %v", what, err)
		}
	}
	a.mu.Lock()
	a.runs[sess.ID] = &runSlot{cancel: func() {}}
	a.mu.Unlock()
	later("running")
	a.markRunDone(sess.ID)
	later("slot not released")
	a.mu.Lock()
	delete(a.runs, sess.ID)
	a.mu.Unlock()

	a.Enqueue(sess.ID, QueuedTurn{Text: "再看看"})
	later("queued")
	if q := a.QueueList(sess.ID); len(q) != 1 || q[0].Text != "再看看" {
		t.Fatalf("queue %+v", q)
	}
	a.clearQueue(sess.ID)

	if err := a.SetPaused(true); err != nil {
		t.Fatal(err)
	}
	later("paused")
	if err := a.SetPaused(false); err != nil {
		t.Fatal(err)
	}

	if got := a.wakeSession("gone"); got != "" {
		t.Fatalf("no main session, got %q", got)
	}
	if err := a.wakeRouter("gone", "grp-1", superagent.GroupDone, "回报"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.mainSessionPath(), []byte(sess.ID), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := a.wakeSession("gone"); got != sess.ID {
		t.Fatalf("fallback %q", got)
	}
}

// TestVerdictCaptured 锁住做完了的任务组把路由唤醒那一轮的最后一段话记成结论；要人处理的那次唤醒不记。
func TestVerdictCaptured(t *testing.T) {
	a, sess := openWakeApp(t)
	rt := a.orchestrator()
	trigger := "desk:" + sess.ID + ":t1"
	rt.Receive(superagent.Trigger{
		ID: trigger, Source: "desk", Threads: []string{"desk:" + sess.ID}, Primary: "desk:" + sess.ID,
		Human: true, Addressed: true, Sender: "operator", Text: "做两件事",
	})
	g, err := rt.Split(superagent.SplitRequest{Trigger: trigger, Mode: superagent.GroupSplit, Title: "两件事", Parts: []superagent.SplitPart{
		{Agent: "yoyo", Role: "甲", Brief: "做甲"},
		{Agent: "yoyo", Role: "乙", Brief: "做乙"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if g.Session != sess.ID {
		t.Fatalf("group session %q", g.Session)
	}
	if err := a.launchWake(sess.ID, g.ID, superagent.GroupAttention, "回报", reply("甲在等你回答。")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "attention wake", func() bool { return !a.runSlotHeld(sess.ID) })
	if got, _ := rt.Group(g.ID); got.Verdict != "" {
		t.Fatalf("attention kept a verdict %q", got.Verdict)
	}
	if err := a.launchWake(sess.ID, g.ID, superagent.GroupDone, "回报", reply("甲和乙都做完了，东西在工单目录里。")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "done wake", func() bool { return !a.runSlotHeld(sess.ID) })
	if got, _ := rt.Group(g.ID); got.Verdict != "甲和乙都做完了，东西在工单目录里。" {
		t.Fatalf("verdict %q", got.Verdict)
	}
}
