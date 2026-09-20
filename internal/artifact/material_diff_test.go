package artifact

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterialDiffPlaybookAndLoop(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "cas"))
	pb1, err := s.Put(KindPlaybook, "main", Playbook{ID: "main", Bullets: []PlaybookBullet{{ID: "b1", Text: "check files", Helpful: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	pb2, err := s.Put(KindPlaybook, "main", Playbook{ID: "main", Bullets: []PlaybookBullet{
		{ID: "b1", Text: "check files", Helpful: 2},
		{ID: "b2", Text: "never claim success without a verifier"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	loop1, err := s.Put(KindLoopPreset, "default", LoopPreset{ID: "default", MaxTurns: 8, MaxToolMessages: 16, CompactionKeep: 4})
	if err != nil {
		t.Fatal(err)
	}
	loop2, err := s.Put(KindLoopPreset, "default", LoopPreset{ID: "default", MaxTurns: 12, MaxToolMessages: 16, CompactionKeep: 4, Bootstrap: "go slow"})
	if err != nil {
		t.Fatal(err)
	}
	a := HarnessSnapshot{Playbook: pb1, LoopPreset: loop1, Note: "seed"}
	b := HarnessSnapshot{Playbook: pb2, LoopPreset: loop2, Note: "evolve"}
	got := MaterialDiff(s, a, b)
	text := MaterialDiffText(got)
	if !strings.Contains(text, "playbook added b2") {
		t.Fatalf("missing bullet add:\n%s", text)
	}
	if !strings.Contains(text, "playbook changed b1") {
		t.Fatalf("missing score change:\n%s", text)
	}
	if !strings.Contains(text, "loop changed max_turns") || !strings.Contains(text, "[L3]") {
		t.Fatalf("missing L3 loop change:\n%s", text)
	}
	if !strings.Contains(text, "bootstrap") {
		t.Fatalf("missing L1 instruction:\n%s", text)
	}
	if !strings.Contains(text, "note changed") {
		t.Fatalf("missing note:\n%s", text)
	}
}

func TestMaterialDiffSkillsAndPrompts(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "cas"))
	sk1, err := s.Put(KindSkill, "verify", Skill{Name: "verify-artifact", Description: "check outputs"})
	if err != nil {
		t.Fatal(err)
	}
	sk2, err := s.Put(KindSkill, "plan", Skill{Name: "plan-first", Description: "write a plan"})
	if err != nil {
		t.Fatal(err)
	}
	p1, err := s.Put(KindPromptFragment, "boot", PromptFragment{ID: "boot", Slot: "bootstrap", Text: "start small"})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := s.Put(KindPromptFragment, "boot", PromptFragment{ID: "boot", Slot: "bootstrap", Text: "inspect the workspace first"})
	if err != nil {
		t.Fatal(err)
	}
	a := HarnessSnapshot{Skills: []string{sk1}, PromptFragments: []string{p1}, Tools: []string{"read", "shell"}}
	b := HarnessSnapshot{Skills: []string{sk1, sk2}, PromptFragments: []string{p2}, Tools: []string{"read"}}
	text := MaterialDiffText(MaterialDiff(s, a, b))
	if !strings.Contains(text, "skills added plan-first") {
		t.Fatalf("missing skill:\n%s", text)
	}
	if !strings.Contains(text, "prompts changed boot") {
		t.Fatalf("missing prompt:\n%s", text)
	}
	if !strings.Contains(text, "tools removed shell") {
		t.Fatalf("missing tool:\n%s", text)
	}
}

func TestMaterialDiffUnchanged(t *testing.T) {
	snap := HarnessSnapshot{Note: "same", Tools: []string{"read"}}
	if got := MaterialDiff(nil, snap, snap); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}
