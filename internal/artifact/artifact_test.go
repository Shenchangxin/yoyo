package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCASRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "cas"))
	h, err := s.Put(KindPromptFragment, "boot", PromptFragment{ID: "boot", Slot: "bootstrap", Text: "start small"})
	if err != nil {
		t.Fatal(err)
	}
	got, env, err := Decode[PromptFragment](s, h)
	if err != nil {
		t.Fatal(err)
	}
	if env.Kind != KindPromptFragment || got.Text != "start small" {
		t.Fatalf("%+v %+v", env, got)
	}
}

func TestRefsCheckout(t *testing.T) {
	dir := t.TempDir()
	r := NewRefs(filepath.Join(dir, "refs"))
	if err := r.Set(RefActive, "abc"); err != nil {
		t.Fatal(err)
	}
	if err := r.Set(RefCanary, "def"); err != nil {
		t.Fatal(err)
	}
	if err := r.Set(RefActive, "def"); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get(RefActive)
	if err != nil || got != "def" {
		t.Fatalf("got %q %v", got, err)
	}
	if err := r.Archive("defdefdefdef"); err != nil {
		t.Fatal(err)
	}
}

func TestSkillParse(t *testing.T) {
	raw := "---\nname: verify-artifact\ndescription: Always check required output files exist.\n---\n\nRead the task, then confirm the artifact path.\n"
	sk, err := ParseSkillMD(raw, "mem")
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "verify-artifact" {
		t.Fatal(sk.Name)
	}
}

func TestSkillParseNestedMetadataAndIcon(t *testing.T) {
	raw := "---\nname: maishou\ndescription: Compare prices across shops.\ndisplay_name: 买手\nicon: https://cdn.example.com/skill.png\nmetadata:\n  openclaw:\n    emoji: shop\nallowed-tools: Read, Bash\n---\n\nRun `uv run scripts/main.py search`.\n"
	sk, err := ParseSkillMD(raw, filepath.Join("skills", "maishou", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if sk.Icon != "https://cdn.example.com/skill.png" || sk.DisplayName != "买手" {
		t.Fatalf("%+v", sk)
	}
	if sk.Dir == "" || sk.Metadata["openclaw.emoji"] != "shop" {
		t.Fatalf("dir/meta %+v", sk)
	}
	if SafeIconURL("javascript:alert(1)") != "" || SafeIconURL("http://cdn.example.com/x.png") != "" {
		t.Fatal("unsafe icon")
	}
}

func TestPlaybookDeltaNoCollapse(t *testing.T) {
	p := Playbook{ID: "main", Bullets: []PlaybookBullet{{ID: "1", Text: "create output early", Helpful: 2}}}
	p = p.ApplyDelta([]PlaybookBullet{{ID: "2", Text: "do not retry the same failing command"}}, nil)
	if len(p.Bullets) != 2 {
		t.Fatalf("len=%d", len(p.Bullets))
	}
	p = p.ApplyDelta([]PlaybookBullet{{ID: "3", Text: "create output early", Helpful: 1}}, nil)
	if len(p.Bullets) != 2 {
		t.Fatalf("dedup failed %d", len(p.Bullets))
	}
	if p.Bullets[0].Helpful != 3 {
		t.Fatalf("helpful=%d", p.Bullets[0].Helpful)
	}
}

func TestSnapshotGet(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "cas"))
	h, err := s.PutSnapshot(HarnessSnapshot{ID: "h", ModelFingerprint: "m1", LoopPreset: "x"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSnapshot(h)
	if err != nil {
		t.Fatal(err)
	}
	if got.ModelFingerprint != "m1" {
		t.Fatalf("%+v", got)
	}
	_ = os.MkdirAll(dir, 0o755)
}
