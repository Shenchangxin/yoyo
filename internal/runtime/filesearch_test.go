package runtime

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	tree := ListTree(dir, 20)
	if len(tree) < 2 {
		t.Fatalf("tree %+v", tree)
	}
}

func TestFuzzySearchListsDirectoryChildrenBeyondTypeaheadCap(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60; i++ {
		name := filepath.Join(src, "f"+string(rune('A'+i%26))+strconv.Itoa(i)+".go")
		if err := os.WriteFile(name, []byte("package p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hits := FuzzySearch(dir, "src/", 40)
	if len(hits) < 60 {
		t.Fatalf("dir listing capped at typeahead: %d", len(hits))
	}
	filtered := FuzzySearch(dir, "src/fA", 40)
	if len(filtered) == 0 {
		t.Fatalf("expected children matching fA")
	}
	for _, h := range filtered {
		if !strings.HasPrefix(h.Path, "src/") {
			t.Fatalf("leaked %s", h.Path)
		}
	}
	nested := filepath.Join(dir, "src", "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "deep.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := ListTree(dir, 4000)
	foundDeep := false
	for _, h := range tree {
		if h.Path == "src/pkg/deep.go" {
			foundDeep = true
			break
		}
	}
	if !foundDeep {
		t.Fatalf("ListTree empty query must still walk nested files: %+v", tree)
	}
}
