package runtime

import (
	"os"
	"path/filepath"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// SkillRoots returns discovery directories in overlay order (later wins).
func SkillRoots(home, workspace, bundled string) []string {
	var out []string
	if bundled != "" {
		out = append(out, bundled)
	}
	if home != "" {
		out = append(out, filepath.Join(home, "skills"))
	}
	if workspace != "" {
		out = append(out, filepath.Join(workspace, ".yoyo", "skills"))
		out = append(out, filepath.Join(workspace, ".agents", "skills"))
		out = append(out, filepath.Join(workspace, "skills"))
	}
	return out
}

// LoadSkillDirs walks SKILL.md files (Codex/Agent Skills layout).
func LoadSkillDirs(roots ...string) []artifact.Skill {
	seen := map[string]artifact.Skill{}
	order := []string{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if info.Name() != "SKILL.md" {
				return nil
			}
			sk, err := artifact.LoadSkillFile(path)
			if err != nil {
				return nil
			}
			if _, ok := seen[sk.Name]; !ok {
				order = append(order, sk.Name)
			}
			seen[sk.Name] = sk
			return nil
		})
	}
	out := make([]artifact.Skill, 0, len(order))
	for _, name := range order {
		out = append(out, seen[name])
	}
	// Re-apply overlays so later roots replace earlier ones while keeping
	// first-seen order for new names.
	for _, root := range roots {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() || info.Name() != "SKILL.md" {
				return nil
			}
			sk, err := artifact.LoadSkillFile(path)
			if err != nil {
				return nil
			}
			seen[sk.Name] = sk
			return nil
		})
	}
	for i, name := range order {
		out[i] = seen[name]
	}
	return out
}

// MergeSkills overlays extra onto base by name. Extra wins.
func MergeSkills(base, extra []artifact.Skill) []artifact.Skill {
	idx := map[string]int{}
	out := append([]artifact.Skill(nil), base...)
	for i, s := range out {
		idx[s.Name] = i
	}
	for _, s := range extra {
		if i, ok := idx[s.Name]; ok {
			out[i] = s
			continue
		}
		idx[s.Name] = len(out)
		out = append(out, s)
	}
	return out
}
