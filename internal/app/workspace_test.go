package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceReady(t *testing.T) {
	if WorkspaceReady("") || WorkspaceReady(".") || WorkspaceReady("./") {
		t.Fatal("placeholders must not be ready")
	}
	dir := t.TempDir()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !WorkspaceReady(abs) {
		t.Fatal(abs)
	}
	if WorkspaceReady(filepath.Join(abs, "missing")) {
		t.Fatal("missing dir")
	}
	if err := os.WriteFile(filepath.Join(abs, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if WorkspaceReady(filepath.Join(abs, "f.txt")) {
		t.Fatal("file is not a workspace")
	}
}
