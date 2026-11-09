// 忙时入队、重试不入队，回合结束后槽仍占着时继续排队，以及一轮跑完放掉槽之后才取下一条排队的回合。

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/vault"
)

// TestQueueWhenBusy 在槽还占着时发送，应返回 ErrQueued 且队列里有这一轮。
func TestQueueWhenBusy(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.runs[sess.ID] = &runSlot{cancel: func() {}}
	a.mu.Unlock()
	if err := a.StartSendOpts(sess.ID, "hello", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatalf("want queued, got %v", err)
	}
	q := a.QueueList(sess.ID)
	if len(q) != 1 || q[0].Text != "hello" {
		t.Fatalf("%+v", q)
	}
}

// TestRetryDoesNotQueue 在会话正忙时重试，应返回正忙，且不入队。
func TestRetryDoesNotQueue(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.runs[sess.ID] = &runSlot{cancel: func() {}}
	a.mu.Unlock()
	err = a.RetrySession(sess.ID)
	if err == nil || err.Error() != string(errBusy) {
		t.Fatalf("want busy, got %v", err)
	}
	if q := a.QueueList(sess.ID); len(q) != 0 {
		t.Fatalf("retry queued: %+v", q)
	}
}

// TestQueueCancelAndReorder 调整排队顺序后再取消一条，剩下的条数正确。
func TestQueueCancelAndReorder(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.runs[sess.ID] = &runSlot{cancel: func() {}}
	a.mu.Unlock()
	if err := a.StartSendOpts(sess.ID, "one", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatal(err)
	}
	if err := a.StartSendOpts(sess.ID, "two", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatal(err)
	}
	if err := a.StartSendOpts(sess.ID, "three", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatal(err)
	}
	q := a.QueueList(sess.ID)
	if len(q) != 3 || q[0].ID == "" {
		t.Fatalf("%+v", q)
	}
	if !a.QueueReorder(sess.ID, q[2].ID, -1) {
		t.Fatal("reorder")
	}
	q = a.QueueList(sess.ID)
	if q[1].Text != "three" {
		t.Fatalf("reorder %+v", q)
	}
	if !a.QueueCancel(sess.ID, q[0].ID) {
		t.Fatal("cancel")
	}
	if got := a.QueueList(sess.ID); len(got) != 2 {
		t.Fatalf("after cancel %+v", got)
	}
}

// TestDoneSlotIsNotRunningButStillQueues 确认 done 的槽不再算正在跑，但槽还在时下一次发送仍然入队。
func TestDoneSlotIsNotRunningButStillQueues(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.runs[sess.ID] = &runSlot{cancel: func() {}}
	a.mu.Unlock()
	if !a.Running(sess.ID) {
		t.Fatal("slot should be running before done")
	}
	a.markRunDone(sess.ID)
	if a.Running(sess.ID) {
		t.Fatal("done slot must not report running")
	}
	if ids := a.RunningIDs(); len(ids) != 0 {
		t.Fatalf("done slot leaked into RunningIDs: %v", ids)
	}
	if st := a.RunningStatus(); len(st) != 0 {
		t.Fatalf("done slot leaked into RunningStatus: %+v", st)
	}
	if err := a.StartSendOpts(sess.ID, "hello", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatalf("want queued while slot held, got %v", err)
	}
}

// TestQueuedTurnRunsAfterTurnEnds 锁住一轮跑完后排着的下一轮会跑：先放掉槽再取，取出来的那条不会因为槽还占着又被塞回队尾。
// 连接表换成只读测试 vault 的一份，环境里的钥匙也清掉，开发机上的钥匙串和环境变量都拿不到，
// 取不到模型，每轮都在 sendLocked 里早退；早退也要清掉进行中的标记，不然重启后会再续一次。
func TestQueuedTurnRunsAfterTurnEnds(t *testing.T) {
	for _, name := range []string{vault.EnvAPIKey, "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY"} {
		t.Setenv(name, "")
	}
	a, sess := openWakeApp(t)
	reg, err := connection.Open(t.TempDir(), a.Vault)
	if err != nil {
		t.Fatal(err)
	}
	a.Video.Conn = reg
	a.mu.Lock()
	a.runs[sess.ID] = &runSlot{cancel: func() {}}
	a.mu.Unlock()
	if err := a.StartSendOpts(sess.ID, "第二句", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatalf("want queued, got %v", err)
	}
	a.mu.Lock()
	delete(a.runs, sess.ID)
	a.mu.Unlock()
	if err := a.launchSend(sess.ID, "第一句", false, nil, false); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "queue drained", func() bool { return len(a.QueueList(sess.ID)) == 0 && !a.runSlotHeld(sess.ID) })
	evs, err := a.Traces.Read(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	for _, ev := range evs {
		if ev.Type == trace.TypeUser {
			said = append(said, fmt.Sprint(ev.Payload["text"]))
		}
	}
	if strings.Join(said, "|") != "第一句|第二句" {
		t.Fatalf("user lines %q", said)
	}
	if a.Threads.Load(sess.ID).ResumeInFlight {
		t.Fatal("an early return left the turn marked in flight")
	}
}

// TestPinArchiveDelete 覆盖置顶、归档、取消置顶，并确认没有 meta 的 jsonl 不会出现在列表里，删除后也不再列出。
func TestPinArchiveDelete(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.PinSession(sess.ID, true); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetSession(sess.ID)
	if err != nil || !got.Pinned {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := a.ArchiveSession(sess.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := a.PinSession(sess.ID, false); err != nil {
		t.Fatal(err)
	}
	got, err = a.GetSession(sess.ID)
	if err != nil || got.Pinned {
		t.Fatalf("unpin %+v %v", got, err)
	}
	ghost := "ghost-jsonl-only"
	if err := os.WriteFile(filepath.Join(a.Home.Sessions(), ghost+".jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := a.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range list {
		if m.ID == ghost {
			t.Fatal("jsonl without meta should not list")
		}
	}
	if err := a.DeleteSession(sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetSession(sess.ID); err == nil {
		t.Fatal("expected missing")
	}
	list, err = a.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range list {
		if m.ID == sess.ID {
			t.Fatal("deleted session still listed")
		}
	}
}

// TestInboxEnqueueRecord 确认忙时入队会在轨迹里留下 next-turn 的 enqueue。
func TestInboxEnqueueRecord(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.runs[sess.ID] = &runSlot{cancel: func() {}}
	a.mu.Unlock()
	if err := a.StartSendOpts(sess.ID, "hello", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatalf("want queued, got %v", err)
	}
	evs, err := a.Traces.Read(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range evs {
		if ev.Type != "inbox" {
			continue
		}
		if ev.Payload["kind"] == "enqueue" && ev.Payload["lane"] == "next-turn" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing next-turn enqueue in %+v", evs)
	}
}
