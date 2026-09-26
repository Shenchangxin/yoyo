package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteRulesIndexIncludesPathScope(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("root agents"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "web")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "AGENTS.md"), []byte("web agents"), 0o644); err != nil {
		t.Fatal(err)
	}
	WriteRulesIndex(root)
	raw, err := os.ReadFile(filepath.Join(root, ".yoyo", "rules", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "when path matches") || !strings.Contains(s, "web/") {
		t.Fatalf("%s", s)
	}
}
