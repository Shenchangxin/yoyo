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

func TestSecretTextKeepsPlainWordsAndCatchesKeys(t *testing.T) {
	kept := []string{
		"I prefer task-based summaries",
		"risk-averse plans first",
		"答复要省 token，越短越好",
		"working on design tokens this week",
		"keep the surprise secret until Friday",
		"password manager is 1Password",
		"项目 my-very-long-project-name-version-2026 还在改",
		"550e8400-e29b-41d4-a716-446655440000",
	}
	for _, text := range kept {
		if secretText(text) {
			t.Fatalf("plain text dropped: %q", text)
		}
	}
	dropped := []string{
		"sk-live-secret",
		"key sk-proj-abcdefgh12345678",
		"SECRET_TOKEN_XYZ",
		"export OPENAI_API_KEY",
		"ghp_" + strings.Repeat("a1", 18),
		"AKIAIOSFODNN7EXAMPLE",
		"password: hunter22",
		"my password is hunter22",
		"我的密码是 abc123",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0In0.c2ln",
		"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
	}
	for _, text := range dropped {
		if !secretText(text) {
			t.Fatalf("secret kept: %q", text)
		}
	}
}

func TestWriteMergesTheSameSentence(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first := s.Write(Item{Kind: KindProfile, Text: "我叫小明。", Source: "user"})
	again := s.Write(Item{Kind: KindProfile, Text: "  我叫小明 ", Source: "user"})
	if first.ID == "" || again.ID != first.ID || len(s.All()) != 1 {
		t.Fatalf("first %+v again %+v all %d", first, again, len(s.All()))
	}
	if other := s.Write(Item{Kind: KindProfile, Text: "我叫小红", Source: "user"}); other.ID == first.ID {
		t.Fatal("a different sentence must stay separate")
	}
}

func TestSearchScoresTermsAndHanPairs(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	s.mu.Lock()
	s.items = []Item{
		{ID: "a", Kind: KindSemantic, Text: "喜欢简短的回复", CreatedAt: base},
		{ID: "b", Kind: KindProject, Text: "yoyo 在做主动性", Project: "yoyo", CreatedAt: base.Add(time.Hour)},
		{ID: "c", Kind: KindEpisodic, Text: "周报交给了 Ada", CreatedAt: base.Add(2 * time.Hour)},
	}
	s.mu.Unlock()
	if got := s.Search("回复风格", "", 5); len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("han pairs = %+v", got)
	}
	if got := s.Search("YOYO 主动", "", 5); len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("terms = %+v", got)
	}
	if got := s.Search("ada 周报", KindEpisodic, 5); len(got) != 1 || got[0].ID != "c" {
		t.Fatalf("kind and terms = %+v", got)
	}
	if got := s.Search("", "", 2); len(got) != 2 || got[0].ID != "c" || got[1].ID != "b" {
		t.Fatalf("recency = %+v", got)
	}
	if got := s.Search("完全无关", "", 5); len(got) != 0 {
		t.Fatalf("no term should match = %+v", got)
	}
}

func TestOperatorPinCarriesWhatAWorkerNeeds(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.FileMachine([]MachineFact{
		{ID: "machine:host", Kind: "profile", Project: "device", Text: "DESK-01"},
		{ID: "machine:os", Kind: "profile", Project: "device", Text: "Windows 11"},
		{ID: MachineID("project", "yoyo"), Kind: "project", Project: "yoyo", Text: "yoyo", State: "moving"},
	})
	s.Write(Item{Kind: KindProfile, Text: "叫我阿杰", Source: "User confirmed in chat"})
	s.Write(Item{Kind: KindSemantic, Text: "报告用中文，先写结论", Source: "user"})
	s.Write(Item{Kind: KindEpisodic, Text: "周报交给了 Ada", Source: "worker"})
	pin := s.OperatorPin(0)
	for _, want := range []string{"Windows 11", "叫我阿杰", "报告用中文，先写结论"} {
		if !strings.Contains(pin, want) {
			t.Fatalf("pin misses %q: %q", want, pin)
		}
	}
	for _, leak := range []string{"DESK-01", "yoyo", "周报"} {
		if strings.Contains(pin, leak) {
			t.Fatalf("pin carries %q: %q", leak, pin)
		}
	}
	if short := s.OperatorPin(30); len(short) > 30 {
		t.Fatalf("budget ignored: %d", len(short))
	}
}

func TestEditRewritesAFactButNotTheMachine(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	it := s.Write(Item{Kind: KindSemantic, Text: "回复要长", Source: "model"})
	got, ok := s.Edit(it.ID, "回复要短")
	if !ok || got.Text != "回复要短" || got.Source != "user" || got.ID != it.ID {
		t.Fatalf("edit = %+v %v", got, ok)
	}
	if _, ok := s.Edit(it.ID, "token: abcdef123456"); ok {
		t.Fatal("a secret must not be written by an edit")
	}
	s.FileMachine([]MachineFact{{ID: "machine:os", Kind: "profile", Project: "device", Text: "Windows"}})
	if _, ok := s.Edit("machine:os", "Linux"); ok {
		t.Fatal("machine facts are rewritten by the census, not by hand")
	}
	if _, ok := s.Edit("missing", "x"); ok {
		t.Fatal("unknown id")
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
