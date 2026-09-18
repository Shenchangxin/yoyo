package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/inbox"
)

func TestSessionIsolateAndPins(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "note.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.SetSessionIsolate(sess.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Isolate || got.Worktree == "" || got.ToolRoot() == got.ProjectRoot() {
		t.Fatalf("isolate %+v", got)
	}
	if _, err := os.Stat(filepath.Join(got.ToolRoot(), "note.txt")); err != nil {
		t.Fatal(err)
	}
	off, err := a.SetSessionIsolate(sess.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if off.Isolate || off.Workspace != ws {
		t.Fatalf("origin should stay, isolate off: %+v", off)
	}
	skillDir := filepath.Join(ws, ".agents", "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := "---\nname: demo\ndescription: demo skill\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	pinned, err := a.SetSessionPinnedSkills(sess.ID, []string{"demo", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pinned.PinnedSkills) != 1 || pinned.PinnedSkills[0] != "demo" {
		t.Fatalf("pins %+v", pinned.PinnedSkills)
	}
	sk := a.GetSkill(ws, "demo")
	if sk == nil || sk["body"] == "" {
		t.Fatalf("skill %+v", sk)
	}
	if err := a.DeleteSession(sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(got.Worktree); !os.IsNotExist(err) {
		t.Fatalf("worktree leftover: %v", err)
	}
}

func TestInboxDismiss(t *testing.T) {
	dir := t.TempDir()
	box, err := inbox.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	it := box.Push(inbox.Item{Title: "job", Body: "x"})
	if !box.Dismiss(it.ID) {
		t.Fatal("dismiss")
	}
	if len(box.List()) != 0 {
		t.Fatalf("%+v", box.List())
	}
}
