// 锁住 @ 提及的注入、越界拒绝、主文件提纲、裸路径的唯一基名，以及会话摘录。ShellDenied 的几条策略也在本文件。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseMentionsAgent 锁住 @agent:<编号> 认成 agent 提及：小写、按出现顺序去重，句末标点不算进编号，不合格式的不认。
func TestParseMentionsAgent(t *testing.T) {
	text := "@agent:codex 做后端，@agent:Claude-Code 做前端，再让 @agent:codex 复查。@agent:-bad @agent:gemini."
	if got := MentionAgents(text); strings.Join(got, ",") != "codex,claude-code,gemini" {
		t.Fatalf("agents %v", got)
	}
	var kinds []string
	for _, m := range ParseMentions("@agent:codex 看 @file:a.go") {
		kinds = append(kinds, m.Kind+":"+m.Ref)
	}
	if strings.Join(kinds, " ") != "file:a.go agent:codex" {
		t.Fatalf("mentions %v", kinds)
	}
	if got := MentionAgents("mail me at a@agent.test"); len(got) != 0 {
		t.Fatalf("not a mention: %v", got)
	}
}

// TestBareMentionsSkipAgent 锁住裸 @ 的路径解析跳过 agent 前缀，即使工作区里正好有同名的文件。
func TestBareMentionsSkipAgent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte("not an agent"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := parseBareWorkspaceMentions(dir, "@agent:codex 来做", map[string]bool{}); len(got) != 0 {
		t.Fatalf("bare mentions %+v", got)
	}
}

// TestExpandMentionsPacksDecoratesAndNames 锁住带技能目录的 @skill 会装饰包路径，MentionSkillNames 能取出技能名。
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

// TestShellPolicy 锁住未放行网络时 curl 被拒绝，NetworkAllow 含 * 时放行；rm -rf / 被拒绝；go test ./... 放行。
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
