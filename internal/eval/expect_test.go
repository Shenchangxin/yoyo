package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunExpectFileContains(t *testing.T) {
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "alpha.txt"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	expect := filepath.Join(t.TempDir(), "expect.toml")
	if err := os.WriteFile(expect, []byte("[[file]]\npath = \"alpha.txt\"\ncontains = \"alpha\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, _, err := runExpect(work, expect)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestRunExpectMissingFile(t *testing.T) {
	expect := filepath.Join(t.TempDir(), "expect.toml")
	if err := os.WriteFile(expect, []byte("[[file]]\npath = \"missing.txt\"\ncontains = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, _, err := runExpect(t.TempDir(), expect)
	if ok || err == nil {
		t.Fatal("expected failure")
	}
}
