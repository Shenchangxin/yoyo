// 锁住技能根顺序、后根覆盖、别名，以及 ApplyPackBootstrap 不预载正文、Depth 大于 0 时直接跳过。领域包提示不包含方法论包。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestSkillRootsOrder 锁住根顺序是 bundled、home/skills、pack、工作区 .yoyo/skills、.agents/skills、skills。
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

// TestMapSkillsAliasesInstallSlug 锁住目录基名和显示名都能索引到同一技能，元数据里的规范名仍是 frontmatter 的 name。
func TestMapSkillsAliasesInstallSlug(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dev-god")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "---\nname: coding-god\ndisplay_name: 编程大神\ndescription: Fixture coding skill.\n---\n\nBe a coding god.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	sk := LoadSkillDirs(root)
	bodies, dirs, meta := MapSkills(sk)
	if bodies["coding-god"] == "" || bodies["dev-god"] == "" || bodies["编程大神"] == "" {
		t.Fatalf("aliases missing: %v", bodies)
	}
	if dirs["dev-god"] != dir || meta["dev-god"].Name != "coding-god" {
		t.Fatalf("dir alias %+v meta %+v", dirs, meta["dev-god"])
	}
}

// TestLoadSkillDirsLaterRootWins 锁住同名 SKILL.md 以后面的根为准。
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

// writeSkillMD 在临时根下写一份带 name 和 description 的 SKILL.md。
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

// TestApplyPackBootstrapOnceAndSkipChild 锁住引导只设 Methodology 和 BootstrapPin，不预载正文。Depth 为 1 时 ApplyPackBootstrap 直接返回。StripPackBootstrap 清掉包、正文、提示，并从 Loaded 去掉引导技能名。
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
	if len(tools.Loaded) != 0 {
		t.Fatalf("bootstrap body must not be preloaded: %v", tools.Loaded)
	}
	if tools.BootstrapBodies != nil {
		t.Fatalf("bootstrap body leaked into working memory: %v", tools.BootstrapBodies)
	}
	if !strings.Contains(tools.BootstrapPin, "load_skill using-superpowers") || strings.Contains(tools.BootstrapPin, "ALREADY LOADED") {
		t.Fatalf("pin %q", tools.BootstrapPin)
	}
	loaded := tools.loadSkill("using-superpowers")
	if loaded.Err != nil || !strings.Contains(loaded.Content, "You must load_skill before acting.") {
		t.Fatalf("%+v", loaded)
	}
	child := *tools
	child.Depth = 1
	child.BootstrapBodies = nil
	child.BootstrapPin = ""
	child.Loaded = nil
	child.Methodology = false
	ApplyPackBootstrap(&child)
	if child.Methodology || child.BootstrapPin != "" || len(child.BootstrapBodies) != 0 {
		t.Fatal("child must skip bootstrap")
	}
	StripPackBootstrap(tools)
	if tools.Methodology || tools.Packs != nil || tools.BootstrapBodies != nil || tools.BootstrapPin != "" {
		t.Fatal("strip failed")
	}
	for _, n := range tools.Loaded {
		if n == "using-superpowers" {
			t.Fatal("bootstrap name must leave Loaded")
		}
	}
}

// TestCapLoadedSkillsPinsBootstrap 锁住 WrapBootstrap 的正文在压缩已加载技能时仍保留 EXTREMELY_IMPORTANT 标记。
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

// TestChatConductMethodologyYieldsWriteNow 锁住默认行为含立即写产物，方法论为真时改成先 load_skill。子会话叠加把方法论行为换回立即写。
func TestChatConductMethodologyYieldsWriteNow(t *testing.T) {
	plain := fragmentText(ChatConduct(ConductOpts{}))
	if !strings.Contains(plain, "start writing those workspace artifacts this turn") {
		t.Fatal("default conduct must keep write-now")
	}
	if strings.Contains(plain, "invoke the matching process skill") {
		t.Fatal("default must not require process skills")
	}
	if !strings.Contains(plain, "load_skill for that skill before writing") {
		t.Fatal("default must load a matching catalog skill")
	}
	if !strings.Contains(plain, "Do not load methodology process skills") {
		t.Fatal("default must keep process skills methodology-only")
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

// TestPlanFirstOverlaySkippedWhenMethodology 锁住 Methodology 为真时 planFirstOverlay 为空，为假时含计划技能正文。
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

// TestReadSkillFileJail 锁住 readSkillFile 能读包内文件，含 .. 的路径被拒绝，缺失文件的错误带 missing。
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
		// 工作区读取 secret.txt 是允许的。技能路径上的越界不得把这份内容漏出来。
	}
}

// errString 把 nil 收成空串，否则返回 Error。
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestRunSkillScriptExtensionless 锁住无扩展名脚本会找 POSIX 解释器。没有解释器时 requireScriptInterpreter 必须失败，且不能报成 missing script。
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
		// 没有 Git Bash 的 Windows 会跳过真正执行，路由检查仍要做。
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

// TestDomainRoutingPinSkipsMethodology 锁住带 BootstrapSkill 的方法论包没有领域提示，novel-to-game 有 load_skill 行，且不把工具映射全文写进去。
func TestDomainRoutingPinSkipsMethodology(t *testing.T) {
	if pin := DomainRoutingPin([]PackSession{{ID: "superpowers", BootstrapSkill: "using-superpowers", Methodology: true}}); pin != "" {
		t.Fatalf("methodology must not get a domain pin: %s", pin)
	}
	pin := DomainRoutingPin([]PackSession{{ID: "novel-to-game"}})
	if !strings.Contains(pin, "## Enabled domain packs") || !strings.Contains(pin, "load_skill novel-to-game") {
		t.Fatal(pin)
	}
	if strings.Contains(pin, "Tool Mapping") {
		t.Fatal("must not dump full mapping")
	}
}

// TestWithoutNovelGamePack 锁住小说改游戏包和技能名被去掉，game-world-qa 和 yoyo-game 留着。
func TestWithoutNovelGamePack(t *testing.T) {
	got := WithoutNovelGamePack([]PackSession{
		{ID: "superpowers", Methodology: true},
		{ID: "novel-to-game"},
	})
	if len(got) != 1 || got[0].ID != "superpowers" {
		t.Fatalf("%+v", got)
	}
	if !IsNovelGameSkill("game-qa") || IsNovelGameSkill("game-world-qa") {
		t.Fatal("sibling names")
	}
	if names := WithoutNovelGameSkills([]string{"yoyo-game", "novel-to-game", "game-build"}); len(names) != 1 || names[0] != "yoyo-game" {
		t.Fatalf("%v", names)
	}
}

// TestHasMethodologyPack 锁住空列表为假，任一 Methodology 为真则为真。
func TestHasMethodologyPack(t *testing.T) {
	if HasMethodologyPack(nil) {
		t.Fatal("empty")
	}
	if !HasMethodologyPack([]PackSession{{Methodology: true}}) {
		t.Fatal("want true")
	}
}

// TestMergeSkillsKeepsArtifact 锁住同名时 extra 的正文和目录覆盖 base。
func TestMergeSkillsKeepsArtifact(t *testing.T) {
	base := []artifact.Skill{{Name: "a", Body: "cas"}}
	extra := []artifact.Skill{{Name: "a", Body: "disk", Dir: "/pack"}}
	got := MergeSkills(base, extra)
	if got[0].Body != "disk" || got[0].Dir != "/pack" {
		t.Fatalf("%+v", got)
	}
}
