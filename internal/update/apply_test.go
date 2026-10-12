package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDownloadedPendingBin(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "yoyo.bin")
	stage := filepath.Join(dir, "pkg.bin")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stage, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := MarkPending(dir, Pending{Staging: stage, Kind: "bin", Version: "9.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyDownloaded(exe, dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("%q", got)
	}
}
