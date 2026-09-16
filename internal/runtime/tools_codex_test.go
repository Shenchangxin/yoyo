package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSkillDirsOverlay(t *testing.T) {
	bundled := t.TempDir()
	home := t.TempDir()
	ws := t.TempDir()
	writeSkill(t, filepath.Join(bundled, "skill-creator", "SKILL.md"), "skill-creator", "bundled")
	writeSkill(t, filepath.Join(home, "skills", "skill-creator", "SKILL.md"), "skill-creator", "home overlay")
	writeSkill(t, filepath.Join(ws, ".yoyo", "skills", "extra", "SKILL.md"), "extra", "workspace extra")

	sk := LoadSkillDirs(SkillRoots(home, ws, bundled)...)
	if len(sk) < 2 {
		t.Fatalf("skills=%+v", sk)
	}
	var creator, extra bool
	for _, s := range sk {
		if s.Name == "skill-creator" {
			creator = true
			if !strings.Contains(s.Body, "home overlay") {
				t.Fatalf("overlay lost: %s", s.Body)
			}
		}
		if s.Name == "extra" {
			extra = true
		}
	}
	if !creator || !extra {
		t.Fatalf("missing names %+v", sk)
	}
}

func TestBlockedHost(t *testing.T) {
	if !blockedHost("localhost") || !blockedHost("127.0.0.1") || !blockedHost("10.0.0.1") || !blockedHost("::1") {
		t.Fatal("private hosts must block")
	}
	if blockedHost("8.8.8.8") {
		t.Fatal("public IP must pass the IP filter")
	}
}

func TestUpdatePlanAndListSkills(t *testing.T) {
	tools := &WorkspaceTools{Skills: map[string]string{"plan-first": "body"}}
	res := tools.Call("update_plan", `{"plan":[{"step":"a","status":"in_progress"},{"step":"b","status":"pending"}]}`)
	if res.Err != nil || !strings.Contains(res.Content, "in_progress") {
		t.Fatalf("%+v", res)
	}
	if tools.PlanText == "" {
		t.Fatal("plan text")
	}
	listed := tools.Call("list_skills", `{}`)
	if listed.Err != nil || listed.Content != "plan-first" {
		t.Fatalf("%+v", listed)
	}
}

func TestViewImageRejectsText(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("view_image", `{"path":"a.txt"}`)
	if res.Err == nil {
		t.Fatal("expected type error")
	}
}

func writeSkill(t *testing.T, path, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "---\nname: " + name + "\ndescription: test skill for " + name + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}
