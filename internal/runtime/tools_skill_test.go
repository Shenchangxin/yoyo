// 锁住 load_skill 会带上包路径和 yoyo-tools 适配说明，安装目录名能解析到规范名，scripts/ 相对路径会改写成包内文件。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestLoadSkillPointsAtGeneratedAdapter 锁住包里有 references/yoyo-tools.md 时，load_skill 的正文点名先读它。
func TestLoadSkillPointsAtGeneratedAdapter(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "novel-to-game")
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "references", "yoyo-tools.md"), []byte("# Yoyo Tool Mapping\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{
		Workspace: t.TempDir(),
		Skills:    map[string]string{"novel-to-game": "Orchestrate stages."},
		SkillDirs: map[string]string{"novel-to-game": skillDir},
	}
	res := tools.Call("load_skill", `{"name":"novel-to-game"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "references/yoyo-tools.md") {
		t.Fatalf("missing adapter pointer:\n%s", res.Content)
	}
	idxHint := strings.Index(res.Content, "Host adapter")
	idxBody := strings.Index(res.Content, "Orchestrate stages.")
	if idxHint < 0 || idxBody < 0 || idxHint > idxBody {
		t.Fatalf("adapter hint must lead the skill body: %s", res.Content)
	}
}

// TestLoadSkillAnalyzeHintsTaskExplore 锁住 novel-game-analyze 的正文追加 task profile=explore 的分章说明。
func TestLoadSkillAnalyzeHintsTaskExplore(t *testing.T) {
	tools := &WorkspaceTools{
		Workspace: t.TempDir(),
		Skills:    map[string]string{"novel-game-analyze": "Extract a SOURCE_BIBLE."},
	}
	res := tools.Call("load_skill", `{"name":"novel-game-analyze"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "profile=explore") || !strings.Contains(res.Content, "chapter") {
		t.Fatalf("missing task hint:\n%s", res.Content)
	}
}

// TestLoadSkillDecoratesPackPath 锁住有目录的技能正文以 Skill pack directory 开头，并列出包内文件。
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

// TestLoadSkillResolvesInstallSlug 锁住用安装目录基名调用 load_skill 仍能取到该技能正文。
func TestLoadSkillResolvesInstallSlug(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "dev-god")
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "hint.py"), []byte("print(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{
		Workspace: t.TempDir(),
		Skills:    map[string]string{"coding-god": "Use scripts/hint.py"},
		SkillDirs: map[string]string{"coding-god": skillDir},
		SkillMeta: map[string]artifact.Skill{
			"coding-god": {Name: "coding-god", DisplayName: "编程大神", Dir: skillDir},
		},
	}
	res := tools.Call("load_skill", `{"name":"dev-god"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "scripts/hint.py") {
		t.Fatalf("missing pack files:\n%s", res.Content)
	}
	listed := tools.Call("list_skills", `{}`)
	if listed.Err != nil || !strings.Contains(listed.Content, "coding-god") {
		t.Fatalf("%+v", listed)
	}
	if strings.Contains(listed.Content, "dev-god") {
		t.Fatalf("list_skills leaked alias: %s", listed.Content)
	}
}

// TestRunSkillScriptFindsPackHelper 锁住 run_skill_script 能在包的 scripts 目录里找到脚本，而不是报 missing script。
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

// TestExpandSkillScriptsRewritesRelativePath 锁住 scripts/ 相对路径被改成包内绝对路径，并且该路径在额外根放行后能通过 ShellDenied。
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

// TestShellAllowsSkillDirPath 锁住技能目录不在额外根里时 ShellDenied 拒绝该脚本，传了该目录则放行。
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
