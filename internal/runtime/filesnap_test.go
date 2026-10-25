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
	waitForNewerFileStamp(t, dir)
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

func TestRestoreFileSnapshotsAfterSkipsOlderStamps(t *testing.T) {
	dir := t.TempDir()
	ws := t.TempDir()
	target := filepath.Join(ws, "note.txt")
	if err := os.WriteFile(target, []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: ws, Spill: NewSpill(dir)}
	tools.snapshotBeforeWrite("note.txt", target)
	// The origin snapshot and a wall-clock mark taken immediately after it can
	// share one Windows tick. Derive the cutoff from the snapshot itself.
	mark := fileStampCutoff(t, dir)
	waitForNewerFileStamp(t, dir)
	if err := os.WriteFile(target, []byte("edit-1"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools.snapshotBeforeWrite("note.txt", target)
	if err := os.WriteFile(target, []byte("edit-2"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := RestoreFileSnapshotsAfter(dir, ws, mark)
	if err != nil || n != 1 {
		t.Fatalf("restore n=%d err=%v", n, err)
	}
	raw, _ := os.ReadFile(target)
	if string(raw) != "edit-1" {
		t.Fatalf("got %q want edit-1 (turn snapshot, not session origin)", raw)
	}
}

func TestRestoreFileSnapshotsAfterDeletesCreatedFile(t *testing.T) {
	dir := t.TempDir()
	ws := t.TempDir()
	target := filepath.Join(ws, "fresh.txt")
	tools := &WorkspaceTools{Workspace: ws, Spill: NewSpill(dir)}
	tools.snapshotBeforeWrite("fresh.txt", target)
	if err := os.WriteFile(target, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := RestoreFileSnapshotsAfter(dir, ws, time.Time{})
	if err != nil || n != 1 {
		t.Fatalf("restore n=%d err=%v", n, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("created file should be removed, err=%v", err)
	}
}

func fileStampNames(t *testing.T, spill string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(spill, "files"))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

func fileStampCutoff(t *testing.T, spill string) time.Time {
	t.Helper()
	names := fileStampNames(t, spill)
	if len(names) != 1 {
		t.Fatalf("stamps=%v", names)
	}
	stamp, ok := ParseStampTime(names[0])
	if !ok {
		t.Fatalf("stamp %s", names[0])
	}
	return stamp.Add(time.Nanosecond)
}

func waitForNewerFileStamp(t *testing.T, spill string) {
	t.Helper()
	names := fileStampNames(t, spill)
	if len(names) == 0 {
		t.Fatal("no snapshot stamp")
	}
	prev := names[0]
	for _, name := range names[1:] {
		if name > prev {
			prev = name
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if time.Now().UTC().Format("20060102T150405.000000000") > prev {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("clock did not advance past %s", prev)
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
