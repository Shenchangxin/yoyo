package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
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

func TestExpandMentionsPrimarySourceOutline(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("# 西游记\n\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&b, "第%d回 标题\n这段正文要写得很长以便超过主源阈值。%s\n", i, strings.Repeat("文", 400))
	}
	name := "novel.txt"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs := ExpandMentions(dir, "@file:novel.txt 做成游戏", "", 3200)
	if len(refs) != 1 {
		t.Fatalf("refs=%+v", refs)
	}
	if !strings.Contains(inject, "source of truth") || !strings.Contains(inject, "read_file") {
		t.Fatalf("inject=%s", inject)
	}
	if !strings.Contains(inject, "第1回") || !strings.Contains(inject, "Outline:") {
		t.Fatalf("missing outline: %s", inject)
	}
	if utf8.RuneCountInString(inject) < 1000 {
		t.Fatalf("primary inject too small: %d", utf8.RuneCountInString(inject))
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
	inject, refs = ExpandMentionsPacks(dir, "@skill:ARXIV-WATCHER", "", skills, dirs, 3200)
	if len(refs) != 1 || !strings.Contains(inject, "search_arxiv.sh") {
		t.Fatalf("case-insensitive mention failed: %s refs=%+v", inject, refs)
	}
}

func TestExpandBareAtPathAndSession(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pin.txt"), []byte("hello pin"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	inject, refs := ExpandMentions(dir, "see @pin.txt and @src/", "", 2400)
	if len(refs) != 2 {
		t.Fatalf("refs=%+v", refs)
	}
	if !strings.Contains(inject, "hello pin") || !strings.Contains(inject, "## @folder:src") {
		t.Fatalf("inject=%s", inject)
	}
	inject, refs = ExpandMentions(dir, "see @file:pin.txt and @pin.txt", "", 2400)
	if len(refs) != 1 {
		t.Fatalf("dedupe refs=%+v", refs)
	}
	inject, refs = ExpandMentions(dir, "mail a@b.test @missing", "", 2400)
	if len(refs) != 0 {
		t.Fatalf("email/missing should not expand: refs=%+v inject=%s", refs, inject)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "loop.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs = ExpandMentions(dir, "fix @loop.go", "", 2400)
	if len(refs) != 1 || refs[0].Ref != "src/loop.go" {
		t.Fatalf("unique basename refs=%+v", refs)
	}
	if !strings.Contains(inject, "package src") {
		t.Fatalf("unique basename inject=%s", inject)
	}
	if err := os.Mkdir(filepath.Join(dir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg", "loop.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs = ExpandMentions(dir, "fix @loop.go", "", 2400)
	if len(refs) != 0 {
		t.Fatalf("ambiguous basename should not expand: refs=%+v inject=%s", refs, inject)
	}
	if err := os.WriteFile(filepath.Join(dir, "loop.go"), []byte("package root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs = ExpandMentions(dir, "fix @loop.go", "", 2400)
	if len(refs) != 1 || refs[0].Ref != "loop.go" {
		t.Fatalf("exact root path must win over unique walk: refs=%+v", refs)
	}
	inject, refs = ExpandMentions(dir, `see @"pin.txt"`, "", 2400)
	if len(refs) != 1 || refs[0].Ref != "pin.txt" {
		t.Fatalf("quoted refs=%+v", refs)
	}
	inject, refs = ExpandMentions(dir, "continue @session:s2", "", 2400)
	if len(refs) != 1 || refs[0].Kind != "session" {
		t.Fatalf("session refs=%+v", refs)
	}
	tools := &WorkspaceTools{
		ListThreads: func() []ThreadRef { return []ThreadRef{{ID: "s2", Title: "Earlier chat"}} },
		Sessions: func(id string) []Message {
			return []Message{{Role: RoleUser, Content: "remember the parser"}, {Role: RoleAssistant, Content: "fixed loop.go"}}
		},
	}
	extra := FormatSessionMentions(refs, "s1", tools, 1600)
	if !strings.Contains(extra, "Earlier chat") || !strings.Contains(extra, "remember the parser") || !strings.Contains(extra, "read_thread") {
		t.Fatalf("session overlay=%s", extra)
	}
	self := FormatSessionMentions(refs, "s2", tools, 1600)
	if !strings.Contains(self, "current chat") || strings.Contains(self, "remember the parser") {
		t.Fatalf("current session should not dump: %s", self)
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
