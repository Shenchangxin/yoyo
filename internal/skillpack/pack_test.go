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

func TestListKnownBeforeInstall(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	list := List(home, ws, nil)
	if len(list) == 0 || list[0].ID != SuperpowersID {
		t.Fatalf("%+v", list)
	}
	if list[0].Installed || list[0].Enabled {
		t.Fatal("catalog entry must start uninstalled and off")
	}
	if !list[0].Methodology || list[0].BootstrapSkill != "using-superpowers" {
		t.Fatalf("%+v", list[0])
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
	if len(st) != 1 || !st[0].Installed || st[0].Enabled {
		t.Fatalf("installed but default-off: %+v", st[0])
	}
	if err := SetWorkspaceEnabled(ws, SuperpowersID, true); err != nil {
		t.Fatal(err)
	}
	st = List(home, ws, nil)
	if !st[0].Enabled {
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
	got := ToolMapping(SuperpowersID, "linux")
	if !strings.Contains(got, "Never invent Claude/Codex names") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "`load_skill`") || !strings.Contains(got, "`update_plan`") || !strings.Contains(got, "`task`") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "companion pulse") {
		t.Fatal("mapping must tell the model the operator sees skill names live")
	}
	if strings.Contains(got, "| `Bash` |") || strings.Contains(got, "| `TodoWrite` |") {
		t.Fatal("must not map actions onto Claude/Codex tool names")
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
