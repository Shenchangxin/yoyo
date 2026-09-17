package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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

func TestIndexMatch(t *testing.T) {
	idx := OpenIndex(t.TempDir())
	idx.Put("a", "fix the parser in loop.go")
	idx.Put("b", "unrelated playbook")
	got := idx.Match("parser loop")
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("%v", got)
	}
}

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
