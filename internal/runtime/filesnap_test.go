package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSnapshotRewindRestoresOldest(t *testing.T) {
	dir := t.TempDir()
	ws := t.TempDir()
	target := filepath.Join(ws, "note.txt")
	if err := os.WriteFile(target, []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: ws, Spill: NewSpill(dir)}
	tools.snapshotBeforeWrite("note.txt", target)
	time.Sleep(2 * time.Millisecond)
	if err := os.WriteFile(target, []byte("edit-1"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools.snapshotBeforeWrite("note.txt", target)
	if err := os.WriteFile(target, []byte("edit-2"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := RestoreFileSnapshots(dir, ws)
	if err != nil || n != 1 {
		t.Fatalf("restore n=%d err=%v", n, err)
	}
	raw, _ := os.ReadFile(target)
	if string(raw) != "orig" {
		t.Fatalf("got %q", raw)
	}
}

func TestParseRewindScope(t *testing.T) {
	scope, from := ParseRewindScope("files from hello")
	if scope != "files" || from != "hello" {
		t.Fatalf("%s %s", scope, from)
	}
	scope, from = ParseRewindScope("both")
	if scope != "both" || from != "" {
		t.Fatalf("%s %s", scope, from)
	}
	scope, from = ParseRewindScope("from msg")
	if scope != "conversation" || from != "msg" {
		t.Fatalf("%s %s", scope, from)
	}
}
