// 未知会话、分叉、材料解码、打断和删除子会话。

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestSendUnknownSessionDoesNotMintID 向不存在的会话发送应失败，而不是新建一个 id。
func TestSendUnknownSessionDoesNotMintID(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	_, err = a.Send(context.Background(), "missing-session", "hello", runtime.HeuristicSolver{}, nil)
	if err == nil {
		t.Fatal("expected unknown session")
	}
}

// TestForkPinsHarness 确认分叉后的 harness 策略是钉住，哈希来自源会话或当前 active。
func TestForkPinsHarness(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dst, err := a.ForkSession(src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dst.HarnessPolicy != session.Pin {
		t.Fatalf("policy %s", dst.HarnessPolicy)
	}
	if dst.Harness != src.Harness && dst.Harness != a.ActiveHash() {
		t.Fatalf("harness %s", dst.Harness)
	}
}

// TestForkSessionFromKeepsFromUserMessage 确认从正文片段分叉时，新轨迹从那条用户消息开始。
func TestForkSessionFromKeepsFromUserMessage(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: src.ID, Payload: map[string]any{"text": "first turn"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeAssistant, SessionID: src.ID, Payload: map[string]any{"text": "ok"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: src.ID, Payload: map[string]any{"text": "branch here now"}})
	dst, err := a.ForkSessionFrom(src.ID, "branch here")
	if err != nil {
		t.Fatal(err)
	}
	evs, err := a.Traces.Read(dst.ID)
	if err != nil || len(evs) == 0 {
		t.Fatalf("%d %v", len(evs), err)
	}
	if evs[0].Type != trace.TypeUser {
		t.Fatalf("want user start, got %s", evs[0].Type)
	}
	text, _ := evs[0].Payload["text"].(string)
	if text != "branch here now" {
		t.Fatalf("got %q", text)
	}
}

// TestForkSessionFromSeq 确认按用户事件序号分叉时，新轨迹从第二条用户消息开始。
func TestForkSessionFromSeq(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: src.ID, Payload: map[string]any{"text": "first turn"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeAssistant, SessionID: src.ID, Payload: map[string]any{"text": "ok"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: src.ID, Payload: map[string]any{"text": "second turn"}})
	page, err := a.Traces.PageTurns(src.ID, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	var seq int64
	for _, ev := range page.Events {
		if ev.Type == trace.TypeUser {
			text, _ := ev.Payload["text"].(string)
			if text == "second turn" {
				seq = ev.Seq
			}
		}
	}
	if seq == 0 {
		t.Fatal("missing seq for second user")
	}
	dst, err := a.ForkSessionFrom(src.ID, fmt.Sprintf("seq:%d", seq))
	if err != nil {
		t.Fatal(err)
	}
	evs, err := a.Traces.Read(dst.ID)
	if err != nil || len(evs) == 0 {
		t.Fatalf("%d %v", len(evs), err)
	}
	text, _ := evs[0].Payload["text"].(string)
	if evs[0].Type != trace.TypeUser || text != "second turn" {
		t.Fatalf("got %s %q", evs[0].Type, text)
	}
}

// TestForkSessionFromUnknownTurnErrors 确认找不到分叉点时失败，并且不留下分叉会话。
func TestForkSessionFromUnknownTurnErrors(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: src.ID, Payload: map[string]any{"text": "only turn"}})
	if _, err := a.ForkSessionFrom(src.ID, "seq:999"); err == nil {
		t.Fatal("expected missing fork point")
	}
	list, err := a.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range list {
		if m.ID != src.ID && strings.Contains(strings.ToLower(m.Title), "fork") {
			t.Fatalf("orphan fork left behind %s", m.ID)
		}
	}
}

// TestForkSessionFromFirstUserKeepsHistory 确认从第一条用户消息分叉时，后面的历史还在。
func TestForkSessionFromFirstUserKeepsHistory(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: src.ID, Payload: map[string]any{"text": "first turn"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeAssistant, SessionID: src.ID, Payload: map[string]any{"text": "ok"}})
	dst, err := a.ForkSessionFrom(src.ID, "first turn")
	if err != nil {
		t.Fatal(err)
	}
	evs, err := a.Traces.Read(dst.ID)
	if err != nil || len(evs) < 2 {
		t.Fatalf("%d %v", len(evs), err)
	}
	text, _ := evs[0].Payload["text"].(string)
	if evs[0].Type != trace.TypeUser || text != "first turn" {
		t.Fatalf("got %s %q", evs[0].Type, text)
	}
}

// TestRestoreSessionFilesUnknownTurnErrors 确认按不存在的序号恢复文件会失败。
func TestRestoreSessionFilesUnknownTurnErrors(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: src.ID, Payload: map[string]any{"text": "only turn"}})
	if _, err := a.RestoreSessionFiles(src.ID, "seq:999"); err == nil {
		t.Fatal("expected missing turn")
	}
}

// TestMaterialsDecodeFailsVisible 确认快照里的循环哈希无法解码时，Materials 返回错误。
func TestMaterialsDecodeFailsVisible(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	snap, err := a.LoadSnapshot(a.ActiveHash())
	if err != nil {
		t.Fatal(err)
	}
	snap.LoopPreset = "not-a-hash"
	hash, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, _, err := a.Materials(hash); err == nil {
		t.Fatal("expected materials decode error")
	}
}

// TestHubSpillOverflow 确认订阅通道接不住时，事件会写进 Overflow 目录。
func TestHubSpillOverflow(t *testing.T) {
	dir := t.TempDir()
	h := NewHub()
	h.Overflow = dir
	ch := make(chan trace.Event)
	h.mu.Lock()
	h.subs["s"] = map[chan trace.Event]struct{}{ch: {}}
	h.mu.Unlock()
	h.Publish(trace.Event{SessionID: "s", Type: trace.TypeUser, Payload: map[string]any{"text": "x"}})
	if _, err := os.Stat(filepath.Join(dir, "s.jsonl")); err != nil {
		t.Fatal(err)
	}
	<-ch
}

// TestSeedAdvertisesHostTools 确认播种的快照带有宿主工具名。
func TestSeedAdvertisesHostTools(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	snap, err := a.LoadSnapshot(a.ActiveHash())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Tools) == 0 {
		t.Fatal("seed tools empty")
	}
	_ = artifact.KindToolSpec
}

// TestSendAppliesChatHorizon 发送一长串工具步骤后，回合仍能结束并得到 done。
func TestSendAppliesChatHorizon(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	steps := make([]runtime.Message, 0, 34)
	for i := 0; i < 33; i++ {
		steps = append(steps, runtime.Message{
			Role: runtime.RoleAssistant,
			ToolCalls: []runtime.ToolCall{{
				ID:        fmt.Sprintf("c%d", i),
				Name:      "list_dir",
				Arguments: fmt.Sprintf(`{"path":".","tag":%d}`, i),
			}},
		})
	}
	steps = append(steps, runtime.Message{Role: runtime.RoleAssistant, Content: "done"})
	out, err := a.Send(context.Background(), sess.ID, "list files", &runtime.ScriptedClient{Steps: steps}, nil)
	if err != nil {
		t.Fatalf("chat overlay should keep the loop alive past 32 turns: %v", err)
	}
	if out != "done" {
		t.Fatalf("out %q", out)
	}
}

// TestChatTurnContextHasNoDeadline 确认异步回合的上下文没有墙钟截止。
func TestChatTurnContextHasNoDeadline(t *testing.T) {
	ctx, cancel := chatTurnContext()
	defer cancel()
	if _, ok := ctx.Deadline(); ok {
		t.Fatal("async chat turns must not expire on a wall clock")
	}
}

// blockChat 是会一直等到上下文取消的假客户端。
type blockChat struct {
	started chan struct{}
}

// Chat 先通知已经进入模型调用，然后等到上下文取消。
func (b *blockChat) Chat(ctx context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	if b.started != nil {
		select {
		case <-b.started:
		default:
			close(b.started)
		}
	}
	<-ctx.Done()
	return runtime.Message{}, ctx.Err()
}

// TestInterruptCancelsInFlightSend 确认 Interrupt 会取消正在等待的发送，原因是取消。
func TestInterruptCancelsInFlightSend(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	hang := &blockChat{started: make(chan struct{})}
	errc := make(chan error, 1)
	go func() {
		_, e := a.Send(context.Background(), sess.ID, "hello", hang, nil)
		errc <- e
	}()
	select {
	case <-hang.started:
	case <-time.After(5 * time.Second):
		t.Fatal("model was never called")
	}
	if err := a.Interrupt(sess.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-errc:
		if e == nil {
			t.Fatal("expected cancel")
		}
		if runtime.ReasonOf(e) != runtime.StopCancelled {
			t.Fatalf("reason %s err=%v", runtime.ReasonOf(e), e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("send did not return after interrupt")
	}
}

// TestDeleteSessionCascadesTaskChildren 确认删除父会话时，任务前缀的子会话 meta 一并删除。
func TestDeleteSessionCascadesTaskChildren(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	childID := src.ID + "--task--deadbeef"
	child := SessionMeta{
		ID:        childID,
		ParentID:  src.ID,
		CreatedAt: time.Now().UTC(),
		Workspace: src.Workspace,
		Title:     "task · explore",
	}
	if err := a.writeSession(child); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteSession(src.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.Home.Sessions(), childID+".meta.json")); !os.IsNotExist(err) {
		t.Fatal("child meta remains")
	}
}
