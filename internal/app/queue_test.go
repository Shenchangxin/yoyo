package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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
	a.runs[sess.ID] = func() {}
	a.mu.Unlock()
	if err := a.StartSendOpts(sess.ID, "hello", false, nil); !errors.Is(err, ErrQueued) {
		t.Fatalf("want queued, got %v", err)
	}
	q := a.QueueList(sess.ID)
	if len(q) != 1 || q[0].Text != "hello" {
		t.Fatalf("%+v", q)
	}
}

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
