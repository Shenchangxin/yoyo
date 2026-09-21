package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

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
	return out
}

func SkillPackFiles(dir string) []string {
	if dir == "" {
		return nil
	}
	var out []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out
}

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

func FilterSkills(skills []artifact.Skill, planMode bool) []artifact.Skill {
	if !planMode {
		return skills
	}
	var out []artifact.Skill
	for _, s := range skills {
		if strings.Contains(strings.ToLower(s.Compatibility), "network") {
			continue
		}
		out = append(out, s)
	}
	return out
}
