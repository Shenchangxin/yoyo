package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/skillpack"
)

func packStatus(t *testing.T, list []skillpack.Status, id string) skillpack.Status {
	t.Helper()
	for _, st := range list {
		if st.ID == id {
			return st
		}
	}
	t.Fatalf("pack %s missing from %+v", id, list)
	return skillpack.Status{}
}

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
	sp := packStatus(t, list, skillpack.SuperpowersID)
	if !sp.Installed || !sp.Enabled {
		t.Fatalf("%+v", sp)
	}
	if sp.EnableWorkspace == nil || !*sp.EnableWorkspace {
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
	if packStatus(t, a.ListPacks(a.Workspace()), skillpack.SuperpowersID).Enabled {
		t.Fatal("workspace off")
	}
	if err := a.EnablePack(skillpack.SuperpowersID, skillpack.ScopeInherit, true); err != nil {
		t.Fatal(err)
	}
	if packStatus(t, a.ListPacks(a.Workspace()), skillpack.SuperpowersID).EnableWorkspace != nil {
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

func TestInstallNovelToGamePackIsDomain(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src := t.TempDir()
	skill := "---\nname: novel-to-game\ndescription: Fixture orchestrator.\n---\n\nBody.\n"
	p := filepath.Join(src, "novel-to-game", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := a.InstallPack(skillpack.NovelToGameID, src)
	if err != nil {
		t.Fatal(err)
	}
	if m.Methodology || m.BootstrapSkill != "" {
		t.Fatalf("%+v", m)
	}
	st := packStatus(t, a.ListPacks(a.Workspace()), skillpack.NovelToGameID)
	if !st.Installed || !st.Enabled || st.Methodology {
		t.Fatalf("%+v", st)
	}
	if len(a.packSessions(a.Workspace())) != 0 {
		t.Fatal("domain pack must not bootstrap a session")
	}
	roots := a.skillRoots(a.Workspace())
	found := false
	for _, r := range roots {
		if strings.Contains(filepath.ToSlash(r), "/packs/novel-to-game/") {
			found = true
		}
	}
	if !found {
		t.Fatalf("pack skills dir missing from roots: %v", roots)
	}
	sk := a.ListSkills(a.Workspace())
	var saw bool
	for _, row := range sk {
		if row["pack"] == skillpack.NovelToGameID {
			saw = true
			if row["source"] != "pack" {
				t.Fatalf("%+v", row)
			}
		}
	}
	if !saw {
		t.Fatal("enabled domain pack skills missing")
	}
	if err := a.UninstallPack(skillpack.NovelToGameID); err != nil {
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
