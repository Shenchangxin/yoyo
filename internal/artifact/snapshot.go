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

type DiffField struct {
	Field   string `json:"field"`
	From    string `json:"from"`
	To      string `json:"to"`
	Changed bool   `json:"changed"`
}

func SnapshotDiffFields(a, b HarnessSnapshot) []DiffField {
	join := func(xs []string) string { return strings.Join(xs, ",") }
	rows := []DiffField{
		{"model", a.ModelFingerprint, b.ModelFingerprint, a.ModelFingerprint != b.ModelFingerprint},
		{"parent", a.Parent, b.Parent, a.Parent != b.Parent},
		{"loop", a.LoopPreset, b.LoopPreset, a.LoopPreset != b.LoopPreset},
		{"playbook", a.Playbook, b.Playbook, a.Playbook != b.Playbook},
		{"policy", a.PolicyPack, b.PolicyPack, a.PolicyPack != b.PolicyPack},
		{"eval", a.EvalSuite, b.EvalSuite, a.EvalSuite != b.EvalSuite},
		{"prompts", join(a.PromptFragments), join(b.PromptFragments), join(a.PromptFragments) != join(b.PromptFragments)},
		{"skills", join(a.Skills), join(b.Skills), join(a.Skills) != join(b.Skills)},
		{"tools", join(a.Tools), join(b.Tools), join(a.Tools) != join(b.Tools)},
		{"eval_tools", join(a.EvalTools), join(b.EvalTools), join(a.EvalTools) != join(b.EvalTools)},
		{"wasm", join(a.WASMPlugins), join(b.WASMPlugins), join(a.WASMPlugins) != join(b.WASMPlugins)},
		{"note", a.Note, b.Note, a.Note != b.Note},
	}
	return rows
}

func SnapshotDiff(a, b HarnessSnapshot) string {
	var buf bytes.Buffer
	for _, f := range SnapshotDiffFields(a, b) {
		if !f.Changed {
			fmt.Fprintf(&buf, "%s %s\n", f.Field, f.From)
			continue
		}
		fmt.Fprintf(&buf, "%s %s -> %s\n", f.Field, f.From, f.To)
	}
	return buf.String()
}

func CloneSnapshot(s HarnessSnapshot) HarnessSnapshot {
	s.PromptFragments = append([]string(nil), s.PromptFragments...)
	s.Skills = append([]string(nil), s.Skills...)
	s.Tools = append([]string(nil), s.Tools...)
	s.EvalTools = append([]string(nil), s.EvalTools...)
	s.WASMPlugins = append([]string(nil), s.WASMPlugins...)
	return s
}
