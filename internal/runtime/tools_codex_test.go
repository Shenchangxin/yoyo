// 锁住技能目录的后层覆盖、blockedHost 的放行和拒绝，以及计划文本和 view_image 对非图片的拒绝。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadSkillDirsOverlay 锁住 home 里的 skill-creator 盖过 bundled，工作区里的 extra 仍被加载。
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

// TestBlockedHost 锁住 localhost、127.0.0.1 和 ::1 放行，10.0.0.1、192.168.1.1 和 metadata.google.internal 拒绝，8.8.8.8 放行。
func TestBlockedHost(t *testing.T) {
	if blockedHost("localhost") || blockedHost("127.0.0.1") || blockedHost("::1") {
		t.Fatal("loopback must be fetchable on a desktop agent")
	}
	if !blockedHost("10.0.0.1") || !blockedHost("192.168.1.1") || !blockedHost("metadata.google.internal") {
		t.Fatal("private LAN and metadata must block")
	}
	if blockedHost("8.8.8.8") {
		t.Fatal("public IP must pass the IP filter")
	}
}

// TestUpdatePlanAndListSkills 锁住 update_plan 写入 PlanText，没有 SkillMeta 时 list_skills 只返回名字。
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

// TestViewImageRejectsText 锁住对 .txt 调用 view_image 返回错误。
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

// writeSkill 写一份带 name 和 description 的 SKILL.md。
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
