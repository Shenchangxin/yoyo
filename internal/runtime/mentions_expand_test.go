// 锁住桌面发送用的 @ 提及展开（ExpandMentionsPacks）：agent 提及不注入，@file 和 @harness 注入正文，越界路径报 escapes，
// 大的主文件只给提纲，裸 @路径按唯一基名展开、同名两份不展开、根上的精确路径优先。

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// expand 是没有技能目录时的展开。
func expand(dir, text, note string, skills map[string]string, budget int) (string, []Mention) {
	return ExpandMentionsPacks(dir, text, note, skills, nil, budget)
}

// TestExpandMentionsIgnoresAgent 锁住 agent 提及不注入任何内容，也不挤掉同一句里的文件注入。
func TestExpandMentionsIgnoresAgent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pin.txt"), []byte("hello pin"), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs := expand(dir, "@agent:codex 读 @file:pin.txt", "", nil, 2400)
	if strings.Contains(inject, "codex") || !strings.Contains(inject, "hello pin") || len(refs) != 2 {
		t.Fatalf("inject=%q refs=%+v", inject, refs)
	}
	if inject, _ := expand(dir, "@agent:codex", "", nil, 2400); inject != "" {
		t.Fatalf("agent alone injected %q", inject)
	}
}

// TestExpandMentionsFileAndJail 锁住 @file 和 @harness 会注入正文，@skill 能展开技能，逃出工作区的路径会报 escapes。
func TestExpandMentionsFileAndJail(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pin.txt"), []byte("hello pin"), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs := expand(dir, "see @file:pin.txt and @harness", "snap-abc note", nil, 2400)
	if len(refs) != 2 || !strings.Contains(inject, "hello pin") || !strings.Contains(inject, "snap-abc") {
		t.Fatalf("inject=%s refs=%+v", inject, refs)
	}
	inject, refs = expand(dir, "@skill:plan-first", "", map[string]string{"plan-first": "keep a plan"}, 2400)
	if len(refs) != 1 || !strings.Contains(inject, "keep a plan") {
		t.Fatalf("skill inject=%s refs=%+v", inject, refs)
	}
	outside := filepath.Join(dir, "..", "nope.txt")
	if inject, _ = expand(dir, "@file:"+filepath.ToSlash(outside), "", nil, 2400); !strings.Contains(inject, "escapes") {
		t.Fatalf("expected jail, got %s", inject)
	}
}

// TestExpandMentionsPrimarySourceOutline 锁住超过主文件门槛的正文只注入提纲和字节数，不把全文塞进注入块。
func TestExpandMentionsPrimarySourceOutline(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("# 西游记\n\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&b, "第%d回 标题\n这段正文要写得很长以便超过主源阈值。%s\n", i, strings.Repeat("文", 400))
	}
	if err := os.WriteFile(filepath.Join(dir, "novel.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	inject, refs := expand(dir, "@file:novel.txt 做成游戏", "", nil, 3200)
	if len(refs) != 1 {
		t.Fatalf("refs=%+v", refs)
	}
	for _, want := range []string{"source of truth", "read_file", "第1回", "Outline:"} {
		if !strings.Contains(inject, want) {
			t.Fatalf("missing %q: %s", want, inject)
		}
	}
	if utf8.RuneCountInString(inject) < 1000 {
		t.Fatalf("primary inject too small: %d", utf8.RuneCountInString(inject))
	}
}

// TestExpandBareAtPath 锁住裸 @路径：文件和目录都展开，和 @file 重复的只算一次，邮箱和不存在的不展开；
// 唯一基名会展开，两个同名则不展开，根上的精确路径优先。
func TestExpandBareAtPath(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("pin.txt", "hello pin")
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	inject, refs := expand(dir, "see @pin.txt and @src/", "", nil, 2400)
	if len(refs) != 2 || !strings.Contains(inject, "hello pin") || !strings.Contains(inject, "## @folder:src") {
		t.Fatalf("inject=%s refs=%+v", inject, refs)
	}
	if _, refs = expand(dir, "see @file:pin.txt and @pin.txt", "", nil, 2400); len(refs) != 1 {
		t.Fatalf("dedupe refs=%+v", refs)
	}
	if inject, refs = expand(dir, "mail a@b.test @missing", "", nil, 2400); len(refs) != 0 {
		t.Fatalf("email/missing should not expand: refs=%+v inject=%s", refs, inject)
	}
	write("src/loop.go", "package src\n")
	inject, refs = expand(dir, "fix @loop.go", "", nil, 2400)
	if len(refs) != 1 || refs[0].Ref != "src/loop.go" || !strings.Contains(inject, "package src") {
		t.Fatalf("unique basename refs=%+v inject=%s", refs, inject)
	}
	write("pkg/loop.go", "package pkg\n")
	if inject, refs = expand(dir, "fix @loop.go", "", nil, 2400); len(refs) != 0 {
		t.Fatalf("ambiguous basename should not expand: refs=%+v inject=%s", refs, inject)
	}
	write("loop.go", "package root\n")
	inject, refs = expand(dir, "fix @loop.go", "", nil, 2400)
	if len(refs) != 1 || refs[0].Ref != "loop.go" || !strings.Contains(inject, "package root") {
		t.Fatalf("exact root path should win: refs=%+v inject=%s", refs, inject)
	}
}
