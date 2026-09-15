package evolve

import "github.com/Shenchangxin/yoyo/internal/artifact"

// MaterialSet is a decoded harness snapshot used by the evolve loop.
// Parent is the DGM mutation origin; Baseline is production (refs/active).
type MaterialSet struct {
	Snap      artifact.HarnessSnapshot
	Hash      string
	Loop      artifact.LoopPreset
	Fragments []artifact.PromptFragment
	Playbook  artifact.Playbook
	Skills    []artifact.Skill
}

func MaterialsFromSnapshot(cas *artifact.Store, snap artifact.HarnessSnapshot, hash string, fallback MaterialSet) MaterialSet {
	out := fallback
	out.Snap = snap
	out.Hash = hash
	if snap.LoopPreset != "" {
		if v, _, err := artifact.Decode[artifact.LoopPreset](cas, snap.LoopPreset); err == nil {
			out.Loop = v
		}
	}
	if snap.Playbook != "" {
		if v, _, err := artifact.Decode[artifact.Playbook](cas, snap.Playbook); err == nil {
			out.Playbook = v
		}
	}
	if len(snap.PromptFragments) > 0 {
		var frags []artifact.PromptFragment
		for _, h := range snap.PromptFragments {
			if v, _, err := artifact.Decode[artifact.PromptFragment](cas, h); err == nil {
				frags = append(frags, v)
			}
		}
		if len(frags) > 0 {
			out.Fragments = frags
		}
	}
	if len(snap.Skills) > 0 {
		var skills []artifact.Skill
		for _, h := range snap.Skills {
			if v, _, err := artifact.Decode[artifact.Skill](cas, h); err == nil {
				skills = append(skills, v)
			}
		}
		if len(skills) > 0 {
			out.Skills = skills
		}
	}
	return out
}

func PlaybooksEqual(a, b artifact.Playbook) bool {
	if a.ID != b.ID || len(a.Bullets) != len(b.Bullets) {
		return false
	}
	for i := range a.Bullets {
		if a.Bullets[i].ID != b.Bullets[i].ID || a.Bullets[i].Text != b.Bullets[i].Text ||
			a.Bullets[i].Helpful != b.Bullets[i].Helpful || a.Bullets[i].Harmful != b.Bullets[i].Harmful {
			return false
		}
	}
	return true
}
