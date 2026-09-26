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

func TestExpandMentionsPacksDecoratesAndNames(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "arxiv-watcher")
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "search_arxiv.sh"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	skills := map[string]string{"arxiv-watcher": "Use scripts/search_arxiv.sh"}
	dirs := map[string]string{"arxiv-watcher": skillDir}
	inject, refs := ExpandMentionsPacks(dir, "@skill:arxiv-watcher 帮我查论文", "", skills, dirs, 3200)
	if len(MentionSkillNames(refs)) != 1 {
		t.Fatalf("names=%v", MentionSkillNames(refs))
	}
	if !strings.Contains(inject, "run_skill_script") || !strings.Contains(inject, skillDir) {
		t.Fatalf("missing pack routing:\n%s", inject)
	}
	idxRoute := strings.Index(inject, "run_skill_script")
	idxBody := strings.Index(inject, "Use scripts/search_arxiv.sh")
	if idxRoute < 0 || idxBody < 0 || idxRoute > idxBody {
		t.Fatalf("routing must lead body: %s", inject)
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
