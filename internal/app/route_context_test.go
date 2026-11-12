// route_context_test.go 锁住主会话的上下文：两轮之间直接写进会话的工人回报和通知，路由下一轮就看得到，
// 不用等重启；它们是一条主机记录，不是路由自己说过的话；同一次运行和重启之后，模型看到的是同一段字。

package app

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// watchingClient 记下每次模型调用收到的全部消息，回一句固定的话。
type watchingClient struct {
	mu   sync.Mutex
	seen [][]runtime.Message
}

// Chat 记录 req 的消息并回一句好的。
func (c *watchingClient) Chat(_ context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	c.mu.Lock()
	c.seen = append(c.seen, append([]runtime.Message(nil), req.Messages...))
	c.mu.Unlock()
	return runtime.Message{Role: runtime.RoleAssistant, Content: "好的。"}, nil
}

// last 是最近一次模型调用收到的消息。
func (c *watchingClient) last() []runtime.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.seen) == 0 {
		return nil
	}
	return c.seen[len(c.seen)-1]
}

// hostNotes 取出 msgs 里的主机记录。
func hostNotes(msgs []runtime.Message) []string {
	var out []string
	for _, m := range msgs {
		if m.Role == runtime.RoleUser && runtime.IsHostNote(m.Content) {
			out = append(out, m.Content)
		}
	}
	return out
}

// TestRouterSeesWhatThePersonWasShown 锁住主会话路由的上下文和屏幕上的对话一致。
func TestRouterSeesWhatThePersonWasShown(t *testing.T) {
	a, sess := openWakeApp(t)
	client := &watchingClient{}
	ctx := context.Background()
	send := func(text string) []runtime.Message {
		t.Helper()
		if _, err := a.SendOpts(ctx, sess.ID, text, client, nil, false); err != nil {
			t.Fatal(err)
		}
		return client.last()
	}
	send("把 #162 的 CI 失败修一下")

	// 两轮之间：工人的回报和一条通知直接写进会话，和 relayReport、relayNotice 的写法一样。
	const report = "失败来自 desk_push.go 的时区判断，已改成本地日期。"
	if err := a.Traces.AppendDurable(trace.Event{
		Type: trace.TypeAssistant, Source: "worker", SessionID: sess.ID,
		Payload: map[string]any{"text": report, "id": "handoff:tkt-1:abc", "kind": "status", "ticket": "tkt-1", "state": "ready"},
	}); err != nil {
		t.Fatal(err)
	}
	a.relayNotice(sess.ID, "更新已下载。")

	live := send("它改了哪里？")
	notes := hostNotes(live)
	if len(notes) != 1 || !strings.Contains(notes[0], report) || !strings.Contains(notes[0], "更新已下载。") {
		t.Fatalf("the live router did not see what was shown: %q", notes)
	}
	for _, m := range live {
		if m.Role == runtime.RoleAssistant && strings.Contains(m.Content, report) {
			t.Fatalf("the worker's report reached the router as its own words: %+v", m)
		}
	}

	// 下一轮不再重复这条记录。
	if again := hostNotes(send("知道了")); len(again) != 1 || again[0] != notes[0] {
		t.Fatalf("the note was repeated or rewritten: %q", again)
	}

	// 模拟重启：丢掉常驻代理，按轨迹重灌。模型看到的记录和重启前一字不差。
	a.agentDesk().Forget(sess.ID)
	reseeded := hostNotes(send("再说一遍"))
	if len(reseeded) != 1 || reseeded[0] != notes[0] {
		t.Fatalf("after a restart the router sees different text:\nlive   %q\nreseed %q", notes, reseeded)
	}
}
