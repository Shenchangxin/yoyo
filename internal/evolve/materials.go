// 把库里的 harness 快照解成循环、片段、剧本和技能。解不出的字段沿用 fallback。不写 refs。

package evolve

import "github.com/Shenchangxin/yoyo/internal/artifact"

// MaterialSet 是一份解开的快照。Cycle 里父材料是变异起点，基线是对照用的那一份。
type MaterialSet struct {
	Snap      artifact.HarnessSnapshot
	Hash      string
	Loop      artifact.LoopPreset
	Fragments []artifact.PromptFragment
	Playbook  artifact.Playbook
	Skills    []artifact.Skill
}

// MaterialsFromSnapshot 用 snap 和 hash 覆盖 fallback。循环、剧本、片段、技能只在对应哈希非空且解码成功时替换；片段或技能一条都解不出时保留 fallback 里的列表。
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

// PlaybooksEqual 比较编号、条目数，以及每条的编号、文本、Helpful 和 Harmful。
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
