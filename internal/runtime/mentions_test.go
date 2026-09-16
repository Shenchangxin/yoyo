package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandMentionsFileAndJail(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pin.txt"), []byte("hello pin"), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs := ExpandMentions(dir, "see @file:pin.txt and @harness", "snap-abc note", 2400)
	if len(refs) != 2 {
		t.Fatalf("refs=%+v", refs)
	}
	if !strings.Contains(inject, "hello pin") || !strings.Contains(inject, "snap-abc") {
		t.Fatalf("inject=%s", inject)
	}
	skills := map[string]string{"plan-first": "keep a plan"}
	inject, refs = ExpandMentionsSkills(dir, "@skill:plan-first", "", skills, 2400)
	if len(refs) != 1 || !strings.Contains(inject, "keep a plan") {
		t.Fatalf("skill inject=%s refs=%+v", inject, refs)
	}
	outside := filepath.Join(dir, "..", "nope.txt")
	inject, _ = ExpandMentions(dir, "@file:"+filepath.ToSlash(outside), "", 2400)
	if !strings.Contains(inject, "escapes") {
		t.Fatalf("expected jail, got %s", inject)
	}
}

func TestShellPolicy(t *testing.T) {
	dir := t.TempDir()
	if err := ShellDenied("curl https://example.com", dir, nil); err == nil {
		t.Fatal("network should deny")
	}
	if err := ShellDenied("curl https://example.com", dir, []string{"*"}); err != nil {
		t.Fatal(err)
	}
	if err := ShellDenied("rm -rf /", dir, nil); err == nil {
		t.Fatal("destructive should deny")
	}
	if err := ShellDenied("go test ./...", dir, nil); err != nil {
		t.Fatal(err)
	}
}
