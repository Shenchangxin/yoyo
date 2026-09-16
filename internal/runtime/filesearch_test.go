package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFuzzySearch(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "loop.go"), []byte("package runtime\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hits := FuzzySearch(dir, "loop", 20)
	if len(hits) == 0 || hits[0].Path != "src/loop.go" {
		t.Fatalf("%+v", hits)
	}
	all := FuzzySearch(dir, "", 20)
	if len(all) < 2 {
		t.Fatalf("%+v", all)
	}
}
