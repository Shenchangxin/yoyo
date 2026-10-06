package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/skillpack"
)

func writePackFixture(t *testing.T, root string) {
	t.Helper()
	skill := func(name string) string {
		return "---\nname: " + name + "\ndescription: Fixture " + name + ".\n---\n\nBody.\n"
	}
	files := map[string]string{
		"using-superpowers/SKILL.md": skill("using-superpowers"),
		"brainstorming/SKILL.md":     skill("brainstorming"),
		"brainstorming/scripts/go":   "#!/usr/bin/env bash\necho ok\n",
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

func TestInstallPackLocalEnablesWorkspace(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src := t.TempDir()
	writePackFixture(t, src)
	m, err := a.InstallPack(skillpack.SuperpowersID, src)
	if err != nil {
		t.Fatal(err)
	}
	if m.SkillCount < 2 {
		t.Fatalf("%+v", m)
	}
	list := a.ListPacks(a.Workspace())
	if len(list) == 0 || !list[0].Installed || !list[0].Enabled {
		t.Fatalf("%+v", list)
	}
	if list[0].EnableWorkspace == nil || !*list[0].EnableWorkspace {
		t.Fatal("install must set workspace enable")
	}
	roots := a.skillRoots(a.Workspace())
	found := false
	for _, r := range roots {
		if strings.Contains(filepath.ToSlash(r), "/packs/superpowers/") {
			found = true
		}
	}
	if !found {
		t.Fatalf("pack skills dir missing from roots: %v", roots)
	}
	sess := a.packSessions(a.Workspace())
	if len(sess) != 1 || sess[0].BootstrapSkill != "using-superpowers" || !sess[0].Methodology {
		t.Fatalf("%+v", sess)
	}
	if err := a.EnablePack(skillpack.SuperpowersID, skillpack.ScopeWorkspace, false); err != nil {
		t.Fatal(err)
	}
	if a.ListPacks(a.Workspace())[0].Enabled {
		t.Fatal("workspace off")
	}
	if err := a.EnablePack(skillpack.SuperpowersID, skillpack.ScopeInherit, true); err != nil {
		t.Fatal(err)
	}
	if a.ListPacks(a.Workspace())[0].EnableWorkspace != nil {
		t.Fatal("inherit must clear workspace override")
	}
	sk := a.ListSkills(a.Workspace())
	for _, row := range sk {
		if row["pack"] == skillpack.SuperpowersID {
			t.Fatal("disabled pack skills must not appear")
		}
	}
	if err := a.EnablePack(skillpack.SuperpowersID, skillpack.ScopeWorkspace, true); err != nil {
		t.Fatal(err)
	}
	sk = a.ListSkills(a.Workspace())
	var sawPack bool
	for _, row := range sk {
		if row["pack"] == skillpack.SuperpowersID {
			sawPack = true
			if row["source"] != "pack" {
				t.Fatalf("%+v", row)
			}
		}
	}
	if !sawPack {
		t.Fatal("enabled pack skills missing")
	}
	if err := a.UninstallPack(skillpack.SuperpowersID); err != nil {
		t.Fatal(err)
	}
}

func TestHarborMaterialsStayPackOff(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if len(a.packSessions(a.Workspace())) != 0 {
		t.Fatal("fresh home must not bootstrap packs")
	}
}
