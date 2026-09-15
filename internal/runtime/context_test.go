package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestShapeDoesNotMutateAndStubs(t *testing.T) {
	loop := DefaultLoop()
	loop.CompactionTokens = 200
	orig := strings.Repeat("x", 8000)
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
		{Role: RoleTool, ToolCallID: "t1", Name: "read_file", Content: orig},
		{Role: RoleAssistant, Content: "ok"},
	}
	out, note := Compact(msgs, loop)
	if note == "" {
		t.Fatalf("expected shaper note, tokens=%d", messagesTokens(out))
	}
	if msgs[2].Content != orig {
		t.Fatal("live transcript mutated")
	}
	joined := ""
	for _, m := range out {
		joined += m.Content
	}
	if !strings.Contains(joined, "elided") && !strings.Contains(joined, "truncated") && !strings.Contains(joined, "omitted") {
		t.Fatalf("expected elision in %q", joined[:min(200, len(joined))])
	}
}

func TestSpillRecall(t *testing.T) {
	dir := t.TempDir()
	sp := NewSpill(filepath.Join(dir, "spill"))
	id := sp.Put("abc", "full secret payload")
	got, err := sp.Get(id)
	if err != nil || got != "full secret payload" {
		t.Fatalf("%q %v", got, err)
	}
	tools := &WorkspaceTools{Workspace: dir, Spill: sp}
	res := tools.Call("recall_context", `{"id":"abc"}`)
	if res.Err != nil || !strings.Contains(res.Content, "full secret") {
		t.Fatalf("%+v", res)
	}
}

func TestWorkspaceRulesAssemble(t *testing.T) {
	s := Assemble(DefaultLoop(), nil, artifact.Playbook{}, nil, "always use tabs", "YOYO.md", nil)
	if !strings.Contains(s, "YOYO.md") || !strings.Contains(s, "always use tabs") {
		t.Fatal(s)
	}
}

func TestLoadWorkspaceRules(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "YOYO.md"), []byte("prefer grep over shell"), 0o644); err != nil {
		t.Fatal(err)
	}
	text, src := LoadWorkspaceRules(dir, 200)
	if src != "YOYO.md" || !strings.Contains(text, "prefer grep") {
		t.Fatalf("%s %q", src, text)
	}
}

func TestGitignoreGlob(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("secret.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("yes"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	g := tools.Call("glob", `{"pattern":"*.txt"}`)
	if g.Err != nil {
		t.Fatal(g.Err)
	}
	if strings.Contains(g.Content, "secret.txt") {
		t.Fatalf("gitignore leaked: %s", g.Content)
	}
	if !strings.Contains(g.Content, "ok.txt") {
		t.Fatalf("missing ok.txt: %s", g.Content)
	}
}

func TestApplyPatchAddAndUpdate(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	add := "*** Begin Patch\n*** Add File: n.txt\n+hello\n*** End Patch\n"
	res := tools.Call("apply_patch", `{"patch":`+jsonQuote(add)+`}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "n.txt"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("%q %v", b, err)
	}
	upd := "*** Begin Patch\n*** Update File: n.txt\n@@\n-hello\n+hello world\n*** End Patch\n"
	res = tools.Call("apply_patch", `{"patch":`+jsonQuote(upd)+`}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "n.txt"))
	if string(b) != "hello world" {
		t.Fatalf("%q", b)
	}
}

func jsonQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `"` + s + `"`
}
