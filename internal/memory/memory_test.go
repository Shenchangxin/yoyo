// 本文件锁住：事实直接留下，密钥不收，本机清点按稳定 id 替换，画像不把机器事实说成当事人讲过的话。

package memory

import (
	"strings"
	"testing"
	"time"
)

func TestForgetRemoves(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	it := s.Write(Item{Kind: KindEpisodic, Text: "SECRET_TOKEN_XYZ"})
	if it.ID != "" {
		t.Fatal("secret-like text must not be kept")
	}
	it = s.Write(Item{Kind: KindEpisodic, Text: "met at the desk"})
	if it.Staging {
		t.Fatal("writes are kept")
	}
	if !s.Forget(it.ID) {
		t.Fatal("forget")
	}
	if len(s.Search("desk", "", 10)) != 0 {
		t.Fatal("still present")
	}
}

func TestProfilePinSkipsSecretsAndKeepsConfirmed(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	it := s.Write(Item{Kind: KindProfile, Text: "call me Ada", Source: "User confirmed in chat"})
	if it.Staging || it.Source != "User confirmed in chat" {
		t.Fatalf("%+v", it)
	}
	if s.Write(Item{Kind: KindProfile, Text: "sk-live-secret"}).ID != "" {
		t.Fatal("secret kept")
	}
	again := s.Write(Item{Kind: KindProfile, Text: "call me Ada", Source: "user"})
	if again.ID != it.ID {
		t.Fatal("duplicate should reuse the same item")
	}
	pin := s.ProfilePin(400)
	if !strings.Contains(pin, "Ada") || strings.Contains(pin, "sk-") {
		t.Fatalf("pin %q", pin)
	}
}

func TestFileMachineReplacesLinesWithoutMovingTheClock(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	s.mu.Lock()
	s.items = []Item{{
		ID: "machine:project:old", Kind: KindProject, Text: "old", Source: "machine", Project: "old", CreatedAt: when,
	}}
	s.mu.Unlock()
	facts := []MachineFact{
		{ID: "machine:device", Kind: "profile", Project: "device", Text: "scx"},
		{ID: MachineID("project", "yoyo"), Kind: "project", Project: "yoyo", Text: "yoyo", State: "moving", Evidence: "main.go", When: "2026-10-10"},
	}
	if !s.FileMachine(facts) {
		t.Fatal("first file should write")
	}
	if s.FileMachine(facts) {
		t.Fatal("same census should not rewrite")
	}
	if len(s.Search("old", KindProject, 10)) != 0 {
		t.Fatal("stale line stayed")
	}
	pin := s.ProfilePin(500)
	if !strings.Contains(pin, "user: scx") || !strings.Contains(pin, "yoyo") || strings.Contains(pin, "Said by the operator") {
		t.Fatalf("pin %q", pin)
	}
	s.Write(Item{Kind: KindProfile, Text: "caca", Source: "user"})
	got := s.Portrait()
	if got.Device != "scx" || len(got.Lines) != 1 || got.Lines[0].Evidence != "main.go" || len(got.Said) != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestAbsorbClearsStaging(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.items = []Item{{ID: "a", Kind: KindProject, Text: "yoyo", Staging: true, CreatedAt: time.Now().UTC()}}
	if err := s.flush(); err != nil {
		s.mu.Unlock()
		t.Fatal(err)
	}
	s.mu.Unlock()
	again, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := again.All()
	if len(got) != 1 || got[0].Staging {
		t.Fatalf("%+v", got)
	}
}
