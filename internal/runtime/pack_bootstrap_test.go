package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestSkillRootsOrder(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	bundled := t.TempDir()
	pack := t.TempDir()
	got := SkillRoots(home, ws, bundled, pack)
	want := []string{
		bundled,
		filepath.Join(home, "skills"),
		pack,
		filepath.Join(ws, ".yoyo", "skills"),
		filepath.Join(ws, ".agents", "skills"),
		filepath.Join(ws, "skills"),
	}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("i=%d got=%s want=%s", i, got[i], want[i])
		}
	}
}

func TestLoadSkillDirsLaterRootWins(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	writeSkillMD(t, a, "demo", "from-a")
	writeSkillMD(t, b, "demo", "from-b")
	got := LoadSkillDirs(a, b)
	if len(got) != 1 || got[0].Body != "from-b" {
		t.Fatalf("%+v", got)
	}
}

func writeSkillMD(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "---\nname: " + name + "\ndescription: Fixture.\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyPackBootstrapOnceAndSkipChild(t *testing.T) {
	tools := &WorkspaceTools{
		Skills: map[string]string{"using-superpowers": "You must load_skill before acting."},
		Packs: []PackSession{{
			ID:             "superpowers",
			BootstrapSkill: "using-superpowers",
			Mapping:        "use load_skill",
			Methodology:    true,
		}},
	}
	ApplyPackBootstrap(tools)
	ApplyPackBootstrap(tools)
	if !tools.Methodology {
		t.Fatal("methodology")
	}
	if n := len(tools.Loaded); n != 1 || tools.Loaded[0] != "using-superpowers" {
		t.Fatalf("loaded %v", tools.Loaded)
	}
	body := tools.BootstrapBodies["using-superpowers"]
	if !strings.Contains(body, "<EXTREMELY_IMPORTANT>") || !strings.Contains(body, "ALREADY LOADED") {
		t.Fatalf("%s", body)
	}
	if !strings.Contains(body, "use load_skill") {
		t.Fatal("mapping missing from wrap")
	}
	dup := tools.loadSkill("using-superpowers")
	if dup.Err != nil || !strings.Contains(dup.Content, "ALREADY LOADED") {
		t.Fatalf("%+v", dup)
	}
	child := *tools
	child.Depth = 1
	child.BootstrapBodies = nil
	child.Loaded = nil
	child.Methodology = false
	ApplyPackBootstrap(&child)
	if child.Methodology || len(child.BootstrapBodies) != 0 {
		t.Fatal("child must skip bootstrap")
	}
	StripPackBootstrap(tools)
	if tools.Methodology || tools.Packs != nil || tools.BootstrapBodies != nil {
		t.Fatal("strip failed")
	}
	for _, n := range tools.Loaded {
		if n == "using-superpowers" {
			t.Fatal("bootstrap name must leave Loaded")
		}
	}
}

func TestCapLoadedSkillsPinsBootstrap(t *testing.T) {
	boot := WrapBootstrap("using-superpowers", strings.Repeat("B", 8000), "mapping")
	other := "## Skill: other\n" + strings.Repeat("X", 20000)
	got := capLoadedSkills([]string{boot, other})
	if len(got) == 0 || !isBootstrapBody(got[0]) {
		t.Fatalf("%d %v", len(got), len(got) > 0)
	}
	joined := strings.Join(got, "")
	if !strings.Contains(joined, "<EXTREMELY_IMPORTANT>") {
		t.Fatal("bootstrap dropped")
	}
}

func TestChatConductMethodologyYieldsWriteNow(t *testing.T) {
	plain := fragmentText(ChatConduct(ConductOpts{}))
	if !strings.Contains(plain, "start writing those workspace artifacts this turn") {
		t.Fatal("default conduct must keep write-now")
	}
	if strings.Contains(plain, "invoke the matching process skill") {
		t.Fatal("default must not require process skills")
	}
	meth := fragmentText(ChatConduct(ConductOpts{Methodology: true}))
	if strings.Contains(meth, "start writing those workspace artifacts this turn") {
		t.Fatal("methodology must yield write-now")
	}
	if !strings.Contains(meth, "load_skill") || !strings.Contains(meth, "Do not skip a skill") {
		t.Fatal(meth)
	}
	if !strings.Contains(meth, "read_skill_file") {
		t.Fatal("methodology conduct must name read_skill_file")
	}
	child := OverlayConduct(ChatConduct(ConductOpts{Methodology: true}), ConductOpts{Methodology: false})
	joined := fragmentText(child)
	if strings.Contains(joined, "invoke the matching process skill") {
		t.Fatal("child must not inherit methodology conduct")
	}
	if !strings.Contains(joined, "start writing those workspace artifacts this turn") {
		t.Fatal("child must get write-now")
	}
}

func TestPlanFirstOverlaySkippedWhenMethodology(t *testing.T) {
	tools := &WorkspaceTools{
		ChatOverlay: true,
		Methodology: true,
		Skills:      map[string]string{"plan-first": "plan first body"},
		PlanText:    "1. [pending] step",
	}
	if got := planFirstOverlay(tools); got != "" {
		t.Fatalf("%q", got)
	}
	tools.Methodology = false
	if got := planFirstOverlay(tools); !strings.Contains(got, "plan first body") {
		t.Fatalf("%q", got)
	}
}

func TestReadSkillFileJail(t *testing.T) {
	skillDir := t.TempDir()
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "guide.md"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "secret.txt"), []byte("nope\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{
		Workspace: ws,
		SkillDirs: map[string]string{"demo": skillDir},
	}
	ok := tools.readSkillFile("demo", "guide.md", 0, 0)
	if ok.Err != nil || !strings.Contains(ok.Content, "alpha") {
		t.Fatalf("%+v", ok)
	}
	esc := tools.readSkillFile("demo", "../secret.txt", 0, 0)
	if esc.Err == nil {
		t.Fatal("escape")
	}
	missing := tools.readSkillFile("demo", "nope.md", 0, 0)
	if missing.Err == nil || !strings.Contains(missing.Err.Error(), "missing") {
		t.Fatalf("%+v", missing)
	}
	wsRead := tools.Call("read_file", `{"path":"secret.txt"}`)
	if wsRead.Err != nil && !strings.Contains(wsRead.Content+errString(wsRead.Err), "nope") {
		// workspace read of secret is allowed; pack jail must not leak it via skill path
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestRunSkillScriptExtensionless(t *testing.T) {
	skillDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(skillDir, "scripts", "start-server")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{
		Workspace: t.TempDir(),
		SkillDirs: map[string]string{"demo": skillDir},
		Loaded:    []string{"demo"},
	}
	if ext := filepath.Ext(script); ext != "" {
		t.Fatalf("fixture must be extensionless, got %q", ext)
	}
	if err := requireScriptInterpreter(script); err != nil && !strings.Contains(err.Error(), "Git Bash") && !strings.Contains(err.Error(), "bash") {
		t.Fatal(err)
	}
	interp := scriptInterpreter(script)
	if interp == "" {
		// Windows without Git Bash is an expected skip of execution, not of routing.
		if requireScriptInterpreter(script) == nil {
			t.Fatal("empty interpreter must fail require")
		}
		return
	}
	res := tools.runSkillScript("demo", "start-server", "")
	if res.Err != nil && strings.Contains(res.Err.Error(), "missing script") {
		t.Fatal(res.Err)
	}
}

func TestHasMethodologyPack(t *testing.T) {
	if HasMethodologyPack(nil) {
		t.Fatal("empty")
	}
	if !HasMethodologyPack([]PackSession{{Methodology: true}}) {
		t.Fatal("want true")
	}
}

func TestMergeSkillsKeepsArtifact(t *testing.T) {
	base := []artifact.Skill{{Name: "a", Body: "cas"}}
	extra := []artifact.Skill{{Name: "a", Body: "disk", Dir: "/pack"}}
	got := MergeSkills(base, extra)
	if got[0].Body != "disk" || got[0].Dir != "/pack" {
		t.Fatalf("%+v", got)
	}
}
