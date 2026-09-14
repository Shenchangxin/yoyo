package artifact

import (
	"bytes"
	"fmt"
	"strings"
)

func (s *Store) PutSnapshot(snap HarnessSnapshot) (string, error) {
	if snap.ID == "" {
		snap.ID = "harness"
	}
	return s.Put(KindHarnessSnapshot, snap.ID, snap)
}

func (s *Store) GetSnapshot(hash string) (HarnessSnapshot, error) {
	snap, _, err := Decode[HarnessSnapshot](s, hash)
	return snap, err
}

func SnapshotDiff(a, b HarnessSnapshot) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "model %s -> %s\n", a.ModelFingerprint, b.ModelFingerprint)
	fmt.Fprintf(&buf, "parent %s -> %s\n", a.Parent, b.Parent)
	fmt.Fprintf(&buf, "loop %s -> %s\n", a.LoopPreset, b.LoopPreset)
	fmt.Fprintf(&buf, "playbook %s -> %s\n", a.Playbook, b.Playbook)
	fmt.Fprintf(&buf, "policy %s -> %s\n", a.PolicyPack, b.PolicyPack)
	fmt.Fprintf(&buf, "eval %s -> %s\n", a.EvalSuite, b.EvalSuite)
	fmt.Fprintf(&buf, "prompts %s\n  -> %s\n", strings.Join(a.PromptFragments, ","), strings.Join(b.PromptFragments, ","))
	fmt.Fprintf(&buf, "skills %s\n  -> %s\n", strings.Join(a.Skills, ","), strings.Join(b.Skills, ","))
	fmt.Fprintf(&buf, "tools %s\n  -> %s\n", strings.Join(a.Tools, ","), strings.Join(b.Tools, ","))
	fmt.Fprintf(&buf, "wasm %s\n  -> %s\n", strings.Join(a.WASMPlugins, ","), strings.Join(b.WASMPlugins, ","))
	if a.Note != b.Note {
		fmt.Fprintf(&buf, "note %q -> %q\n", a.Note, b.Note)
	}
	return buf.String()
}

func CloneSnapshot(s HarnessSnapshot) HarnessSnapshot {
	s.PromptFragments = append([]string(nil), s.PromptFragments...)
	s.Skills = append([]string(nil), s.Skills...)
	s.Tools = append([]string(nil), s.Tools...)
	s.WASMPlugins = append([]string(nil), s.WASMPlugins...)
	return s
}
