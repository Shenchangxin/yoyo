package app

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSweepExpiredSessionsDeletesOldArchive(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sess.Archived = true
	sess.CreatedAt = time.Now().UTC().AddDate(0, 0, -40)
	if err := a.writeSession(sess); err != nil {
		t.Fatal(err)
	}
	keep, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.Config.SessionRetentionDays = 30
	if n := a.SweepExpiredSessions(); n != 1 {
		t.Fatalf("swept %d", n)
	}
	if _, err := a.GetSession(sess.ID); err == nil {
		t.Fatal("archived session should be gone")
	}
	if _, err := a.GetSession(keep.ID); err != nil {
		t.Fatal(err)
	}
}

func TestListSessionsMarksInterruptedQueue(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.Enqueue(sess.ID, QueuedTurn{Text: "later"})
	list, err := a.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	var hit SessionMeta
	for _, m := range list {
		if m.ID == sess.ID {
			hit = m
		}
	}
	if !hit.Interrupted || hit.Queued != 1 {
		t.Fatalf("%+v", hit)
	}
}
