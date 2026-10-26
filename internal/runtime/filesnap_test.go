// 锁住写前快照按时间戳从早到晚恢复，同一路径留下最早那一份。带创建标记的快照会删掉后来出现的文件。after 之前的时间戳目录不参与。

package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFileSnapshotRewindRestoresOldest 锁住两次快照之后恢复得到的是第一次写入前的 orig，而不是中间的 edit-1。
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

// TestRestoreFileSnapshotsAfterSkipsOlderStamps 锁住早于截止时间的快照被跳过，恢复得到的是之后那次写入前的 edit-1，不是最初的 orig。
func TestRestoreFileSnapshotsAfterSkipsOlderStamps(t *testing.T) {
	dir := t.TempDir()
	ws := t.TempDir()
	target := filepath.Join(ws, "note.txt")
	if err := os.WriteFile(target, []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: ws, Spill: NewSpill(dir)}
	tools.snapshotBeforeWrite("note.txt", target)
	// 起点快照和紧接着取的墙钟可能落在同一个 Windows 时钟刻度上，所以截止时间从快照自己的时间戳推出来。
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

// TestRestoreFileSnapshotsAfterDeletesCreatedFile 锁住写入前不存在的文件，恢复时被删除。
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

// fileStampNames 列出溢出目录 files 下的时间戳子目录名。
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

// fileStampCutoff 解析唯一的时间戳，并加上一纳秒，使该快照早于返回的截止时间。
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

// waitForNewerFileStamp 等到 UTC 紧凑时间字符串大于已有的最新戳，避免两个快照同名。
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

// TestParseRewindScope 锁住 files from、both，以及默认的 conversation 加 from。
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
