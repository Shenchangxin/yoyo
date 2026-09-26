package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSkillDecoratesPackPath(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "arxiv-watcher")
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "search_arxiv.sh"), []byte("#!/usr/bin/env bash\necho \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{
		Workspace: t.TempDir(),
		Skills:    map[string]string{"arxiv-watcher": "Use scripts/search_arxiv.sh"},
		SkillDirs: map[string]string{"arxiv-watcher": skillDir},
	}
	res := tools.Call("load_skill", `{"name":"arxiv-watcher"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, skillDir) || !strings.Contains(res.Content, "scripts/search_arxiv.sh") {
		t.Fatalf("missing pack footer: %s", res.Content)
	}
	if strings.Index(res.Content, "run_skill_script") > strings.Index(res.Content, "Use scripts/search_arxiv.sh") {
		t.Fatal("routing must lead the skill body")
	}
}

func TestRunSkillScriptFindsPackHelper(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(skillDir, "scripts", "hello.py")
	if err := os.WriteFile(script, []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	tools := &WorkspaceTools{
		Workspace: ws,
		SkillDirs: map[string]string{"demo": skillDir},
		Loaded:    []string{"demo"},
	}
	res := tools.Call("run_skill_script", `{"skill":"demo","script":"hello.py"}`)
	if res.Err != nil && strings.Contains(res.Err.Error(), "missing script") {
		t.Fatal(res.Err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Fatal(err)
	}
	missing := tools.Call("run_skill_script", `{"skill":"demo","script":"nope.py"}`)
	if missing.Err == nil || !strings.Contains(missing.Err.Error(), "missing script") {
		t.Fatalf("%+v", missing)
	}
}

func TestExpandSkillScriptsRewritesRelativePath(t *testing.T) {
	skillDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(skillDir, "scripts", "search_arxiv.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{
		Workspace: t.TempDir(),
		SkillDirs: map[string]string{"arxiv-watcher": skillDir},
		Loaded:    []string{"arxiv-watcher"},
	}
	got := tools.expandSkillScripts(`scripts/search_arxiv.sh "llm"`)
	if !strings.Contains(got, script) && !strings.Contains(got, filepath.ToSlash(script)) {
		t.Fatalf("rewrite failed: %s", got)
	}
	if err := ShellDenied(got, tools.Workspace, nil, skillDir); err != nil {
		t.Fatal(err)
	}
}

func TestShellAllowsSkillDirPath(t *testing.T) {
	ws := t.TempDir()
	skillDir := t.TempDir()
	script := filepath.Join(skillDir, "scripts", "x.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("echo"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := quoteShellArg(script)
	if err := ShellDenied(cmd, ws, nil); err == nil {
		t.Fatal("expected deny without extra root")
	}
	if err := ShellDenied(cmd, ws, nil, skillDir); err != nil {
		t.Fatal(err)
	}
}
