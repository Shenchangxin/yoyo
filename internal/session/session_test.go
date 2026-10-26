// 本文件锁住马具钉住、载入时丢掉 high_risk、Acquire 忙、搜索重建，以及审批清空后的状态。由 go test 调用。

package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveHarnessPin 锁住 pin 返回钉住的哈希，follow_active 和空策略在 active 非空时返回 active。
func TestResolveHarnessPin(t *testing.T) {
	if got := ResolveHarness(Pin, "abc", "xyz"); got != "abc" {
		t.Fatalf("pin: %s", got)
	}
	if got := ResolveHarness(FollowActive, "abc", "xyz"); got != "xyz" {
		t.Fatalf("follow: %s", got)
	}
	if got := ResolveHarness("", "abc", "xyz"); got != "xyz" {
		t.Fatalf("default follow: %s", got)
	}
}

// TestRunStateDropsHighRiskOnLoad 锁住保存时的 running 载入后变成 idle，high_risk 被丢掉，队列仍在。
func TestRunStateDropsHighRiskOnLoad(t *testing.T) {
	dir := t.TempDir()
	st := RunState{
		Status:      StatusRunning,
		SessionCaps: []string{"shell", "high_risk", "network"},
		Queue:       []QueuedTurn{{Text: "hello"}},
	}
	if err := SaveRun(dir, "s1", st); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRun(dir, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusIdle {
		t.Fatalf("status %s", got.Status)
	}
	for _, c := range got.SessionCaps {
		if c == "high_risk" {
			t.Fatal("high_risk restored")
		}
	}
	if len(got.Queue) != 1 {
		t.Fatalf("queue %+v", got.Queue)
	}
}

// TestManagerAcquireBusy 锁住第一次 Acquire 成功且 Running 为真，同一 id 再 Acquire 返回 ErrBusy。
func TestManagerAcquireBusy(t *testing.T) {
	m := NewManager(t.TempDir())
	ctx, release, err := m.Acquire("s", context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if !m.Running("s") {
		t.Fatal("expected running")
	}
	if _, _, err := m.Acquire("s", ctx); err != ErrBusy {
		t.Fatalf("want busy, got %v", err)
	}
}

// TestIndexMatch 锁住两个词都要出现才命中，不相关的文档被排除。
func TestIndexMatch(t *testing.T) {
	idx := OpenIndex(t.TempDir())
	idx.Put("a", "fix the parser in loop.go")
	idx.Put("b", "unrelated playbook")
	got := idx.Match("parser loop")
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("%v", got)
	}
}

// TestIndexRebuildFile 锁住 Flush 写出 _search.json，再次 OpenIndex 还能按词命中。
func TestIndexRebuildFile(t *testing.T) {
	dir := t.TempDir()
	idx := OpenIndex(dir)
	idx.Put("x", "hello world")
	if err := idx.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_search.json")); err != nil {
		t.Fatal(err)
	}
	idx2 := OpenIndex(dir)
	if len(idx2.Match("hello")) != 1 {
		t.Fatal("rebuild failed")
	}
}

// TestSetApprovalsClearsAwaiting 锁住有审批时状态是 awaiting_approval，清空后在没有 Actor 时回到 idle，待审批列表为空。
func TestSetApprovalsClearsAwaiting(t *testing.T) {
	m := NewManager(t.TempDir())
	m.SetApprovals("s", []PendingApproval{{ID: "o1", Action: "shell"}})
	st := m.Load("s")
	if st.Status != StatusAwaitingApproval || len(st.PendingApprovals) != 1 {
		t.Fatalf("%+v", st)
	}
	m.SetApprovals("s", nil)
	st = m.Load("s")
	if st.Status != StatusIdle {
		t.Fatalf("status %s", st.Status)
	}
	if len(st.PendingApprovals) != 0 {
		t.Fatalf("pending %+v", st.PendingApprovals)
	}
}

// TestSetApprovalsRunningWhenActorLive 锁住 Actor 还在时，清空审批后状态回到 running。
func TestSetApprovalsRunningWhenActorLive(t *testing.T) {
	m := NewManager(t.TempDir())
	_, release, err := m.Acquire("s", context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	m.SetApprovals("s", []PendingApproval{{ID: "o1"}})
	m.SetApprovals("s", nil)
	st := m.Load("s")
	if st.Status != StatusRunning {
		t.Fatalf("status %s", st.Status)
	}
}
