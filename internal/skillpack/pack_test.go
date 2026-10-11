package skillpack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileAllowedExtensionlessScripts(t *testing.T) {
	if !FileAllowed("scripts/start-server") {
		t.Fatal("extensionless scripts/ helpers must be allowed")
	}
	if FileAllowed("bin/start-server") {
		t.Fatal("extensionless outside scripts/ must be denied")
	}
	if FileAllowed("skills/../evil.sh") {
		t.Fatal("path escape must be denied")
	}
	if !FileAllowed("using-superpowers/references/yoyo-tools.md") {
		t.Fatal("markdown references must be allowed")
	}
	if FileAllowed("payload.exe") {
		t.Fatal("executables must be denied")
	}
	if !FileAllowed("LICENSE") {
		t.Fatal("LICENSE must be allowed")
	}
	if !FileAllowed("diagram.dot") {
		t.Fatal(".dot files must be allowed")
	}
}

func TestEnabledWorkspaceOverridesGlobal(t *testing.T) {
	global := map[string]Pref{"superpowers": {Enabled: true}}
	ws := map[string]WorkspacePref{"superpowers": {Enabled: false}}
	if Enabled("superpowers", global, ws) {
		t.Fatal("workspace false must win")
	}
	if !Enabled("superpowers", global, map[string]WorkspacePref{}) {
		t.Fatal("global true must apply when workspace is unset")
	}
	if Enabled("superpowers", nil, nil) {
		t.Fatal("default must be off")
	}
}

func TestSanitizeID(t *testing.T) {
	if got := SanitizeID(" Super Powers!! "); got != "superpowers" {
		t.Fatalf("%q", got)
	}
	if SanitizeID("../etc") != "etc" {
		t.Fatal("path fragments must strip")
	}
}

func packByID(list []Status, id string) (Status, bool) {
	for _, st := range list {
		if st.ID == id {
			return st, true
		}
	}
	return Status{}, false
}

func TestListKnownBeforeInstall(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	list := List(home, ws, nil)
	if len(list) < 2 {
		t.Fatalf("%+v", list)
	}
	if list[0].ID != SuperpowersID {
		t.Fatalf("catalog order: %+v", list)
	}
	if list[0].Installed || list[0].Enabled {
		t.Fatal("catalog entry must start uninstalled and off")
	}
	if !list[0].Methodology || list[0].BootstrapSkill != "using-superpowers" {
		t.Fatalf("%+v", list[0])
	}
	ng, ok := packByID(list, NovelToGameID)
	if !ok {
		t.Fatal("novel-to-game must be in the catalog")
	}
	if ng.Installed || ng.Enabled || ng.Methodology || ng.BootstrapSkill != "" {
		t.Fatalf("domain pack must be off and unbootstrapped: %+v", ng)
	}
	if ng.Origin.Repo != "zenstory-ai/novel-to-game" || ng.Origin.SkillsRel != "skills" {
		t.Fatalf("%+v", ng.Origin)
	}
}

func writeLocalPack(t *testing.T, root string) {
	t.Helper()
	skill := func(name, extra string) string {
		return "---\nname: " + name + "\ndescription: Fixture skill " + name + ".\n---\n\n" + extra + "\n"
	}
	files := map[string]string{
		"using-superpowers/SKILL.md":         skill("using-superpowers", "Bootstrap. Invoke matching skills before acting."),
		"brainstorming/SKILL.md":             skill("brainstorming", "Design before code."),
		"brainstorming/references/guide.md":  "# Guide\nAsk first.\n",
		"brainstorming/scripts/start-server": "#!/usr/bin/env bash\necho ok\n",
		"brainstorming/scripts/hello.py":     "print('ok')\n",
	}
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstallLocalAndEnable(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	src := t.TempDir()
	writeLocalPack(t, src)
	m, err := Install(home, SuperpowersID, InstallOptions{Origin: Origin{Kind: KindLocal, Path: src}, GOOS: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	if m.SkillCount < 2 || m.BootstrapSkill != "using-superpowers" || !m.Methodology {
		t.Fatalf("%+v", m)
	}
	mapPath := filepath.Join(SkillsDir(home, SuperpowersID), "using-superpowers", "references", "yoyo-tools.md")
	raw, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "load_skill") || !strings.Contains(string(raw), "read_skill_file") {
		t.Fatalf("mapping missing: %s", raw)
	}
	if !strings.Contains(string(raw), "Git Bash") {
		t.Fatal("windows mapping must mention Git Bash")
	}
	st := List(home, ws, nil)
	sp, ok := packByID(st, SuperpowersID)
	if !ok || !sp.Installed || sp.Enabled {
		t.Fatalf("installed but default-off: %+v", sp)
	}
	if err := SetWorkspaceEnabled(ws, SuperpowersID, true); err != nil {
		t.Fatal(err)
	}
	st = List(home, ws, nil)
	sp, ok = packByID(st, SuperpowersID)
	if !ok || !sp.Enabled {
		t.Fatal("workspace enable failed")
	}
	dirs := EnabledSkillDirs(home, ws, nil)
	if len(dirs) != 1 {
		t.Fatalf("dirs %v", dirs)
	}
	rt := RuntimeFor(home, ws, nil, "windows")
	if len(rt) != 1 || rt[0].BootstrapSkill != "using-superpowers" || !rt[0].Methodology {
		t.Fatalf("%+v", rt)
	}
	if !strings.Contains(rt[0].Mapping, "update_plan") {
		t.Fatal("runtime mapping missing")
	}
	if err := ClearWorkspaceEnabled(ws, SuperpowersID); err != nil {
		t.Fatal(err)
	}
	if Enabled(SuperpowersID, nil, ReadWorkspace(ws)) {
		t.Fatal("inherit after clear must be off")
	}
	if err := Uninstall(home, SuperpowersID); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadManifest(home, SuperpowersID); ok {
		t.Fatal("manifest survived uninstall")
	}
}

func TestToolMappingNeverNamesForeignTools(t *testing.T) {
	for _, id := range []string{SuperpowersID, NovelToGameID, "unknown-pack"} {
		got := ToolMapping(id, "linux")
		if !strings.Contains(got, "Never invent Claude/Codex names") {
			t.Fatal(id, got)
		}
		if !strings.Contains(got, "`load_skill`") || !strings.Contains(got, "`update_plan`") || !strings.Contains(got, "`task`") {
			t.Fatal(id, got)
		}
		if strings.Contains(got, "| `Bash` |") || strings.Contains(got, "| `TodoWrite` |") {
			t.Fatal(id, "must not map actions onto Claude/Codex tool names")
		}
	}
	sp := ToolMapping(SuperpowersID, "linux")
	if !strings.Contains(sp, "companion pulse") {
		t.Fatal("superpowers mapping must tell the model the operator sees skill names live")
	}
}

func TestToolMappingNovelToGameIsDomainNotBootstrap(t *testing.T) {
	got := ToolMapping(NovelToGameID, "linux")
	if strings.Contains(got, "using-superpowers") {
		t.Fatal("domain mapping must not claim the methodology bootstrap")
	}
	if strings.Contains(got, "You have superpowers") {
		t.Fatal(got)
	}
	for _, needle := range []string{
		"novel-game-analyze",
		"game-adaptations",
		"read_skill_file",
		"_progress.md",
		"browser_screenshot",
		"browser_click",
		"do **not** reload",
		"present_choices",
		"profile=explore",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("missing %q in:\n%s", needle, got)
		}
	}
}

func TestGenericMappingDoesNotClaimBootstrap(t *testing.T) {
	got := ToolMapping("some-other-pack", "linux")
	if strings.Contains(got, "using-superpowers is already in context") {
		t.Fatal(got)
	}
}

func writeNovelToGamePack(t *testing.T, root string) {
	t.Helper()
	skill := func(name, extra string) string {
		return "---\nname: " + name + "\ndescription: Fixture skill " + name + ".\n---\n\n" + extra + "\n"
	}
	files := map[string]string{
		"novel-to-game/SKILL.md":               skill("novel-to-game", "Orchestrate stages. Read references/pipeline-contract.md."),
		"novel-to-game/references/pipeline.md": "# Pipeline\nResume from _progress.md.\n",
		"novel-game-analyze/SKILL.md":          skill("novel-game-analyze", "Write SOURCE_BIBLE.md."),
	}
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstallNovelToGameWritesMappingAndStaysOff(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	src := t.TempDir()
	writeNovelToGamePack(t, src)
	m, err := Install(home, NovelToGameID, InstallOptions{Origin: Origin{Kind: KindLocal, Path: src}, GOOS: "linux"})
	if err != nil {
		t.Fatal(err)
	}
	if m.SkillCount < 2 || m.Methodology || m.BootstrapSkill != "" {
		t.Fatalf("%+v", m)
	}
	if m.Name != "NovelToGame" || m.License != "MIT" {
		t.Fatalf("%+v", m)
	}
	mapPath := filepath.Join(SkillsDir(home, NovelToGameID), "novel-to-game", "references", "yoyo-tools.md")
	raw, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "load_skill") || !strings.Contains(string(raw), "game-adaptations") {
		t.Fatalf("mapping missing: %s", raw)
	}
	if !strings.Contains(string(raw), "browser_screenshot") || !strings.Contains(string(raw), "Do **not** start") {
		t.Fatalf("playable evidence must prefer browser_screenshot over http.server: %s", raw)
	}
	if !strings.Contains(string(raw), "connect to CDP 9333") {
		t.Fatalf("playable evidence must forbid a second Chrome/CDP: %s", raw)
	}
	if strings.Contains(string(raw), "Git Bash") {
		t.Fatal("linux mapping must not mention Git Bash")
	}
	st, ok := packByID(List(home, ws, nil), NovelToGameID)
	if !ok || !st.Installed || st.Enabled {
		t.Fatalf("installed but default-off: %+v", st)
	}
	if err := SetWorkspaceEnabled(ws, NovelToGameID, true); err != nil {
		t.Fatal(err)
	}
	rt := RuntimeFor(home, ws, nil, "linux")
	if len(rt) != 0 {
		t.Fatalf("domain pack must not session-bootstrap: %+v", rt)
	}
	dirs := EnabledSkillDirs(home, ws, nil)
	if len(dirs) != 1 {
		t.Fatalf("dirs %v", dirs)
	}
}

func TestCallLabelShowsSkillId(t *testing.T) {
	if CallLabel("load_skill", `{"name":"brainstorming"}`) != "brainstorming" {
		t.Fatal("load_skill")
	}
	if CallLabel("read_skill_file", `{"skill":"brainstorming","path":"references/guide.md"}`) != "brainstorming · references/guide.md" {
		t.Fatal("read_skill_file")
	}
	if CallLabel("run_skill_script", `{"skill":"brainstorming","script":"scripts/start-server.sh"}`) != "brainstorming · scripts/start-server.sh" {
		t.Fatal("run_skill_script")
	}
	if CallLabel("web_search", `{"query":"x"}`) != "" {
		t.Fatal("non-skill tools stay unlabeled")
	}
	if CallLabel("load_skill", "") != "load_skill" {
		t.Fatal("bare load_skill still discloses the tool")
	}
}
