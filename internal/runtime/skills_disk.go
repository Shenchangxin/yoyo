package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func SkillRoots(home, workspace, bundled string, packDirs ...string) []string {
	var out []string
	if bundled != "" {
		out = append(out, bundled)
	}
	if home != "" {
		out = append(out, filepath.Join(home, "skills"))
	}
	for _, d := range packDirs {
		d = strings.TrimSpace(d)
		if d != "" {
			out = append(out, d)
		}
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

// SkillAliasKeys are names the operator or model may use to load a skill:
// frontmatter name, display name, and the on-disk directory slug.
func SkillAliasKeys(s artifact.Skill) []string {
	var keys []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		for _, e := range keys {
			if e == k {
				return
			}
		}
		keys = append(keys, k)
	}
	add(s.Name)
	add(s.DisplayName)
	if s.Dir != "" {
		add(filepath.Base(s.Dir))
	}
	return keys
}

// MapSkills indexes skills by primary name, then by unused aliases so
// load_skill / @skill can use the install slug when it differs from YAML name.
func MapSkills(skills []artifact.Skill) (bodies map[string]string, dirs map[string]string, meta map[string]artifact.Skill) {
	bodies = map[string]string{}
	dirs = map[string]string{}
	meta = map[string]artifact.Skill{}
	for _, s := range skills {
		if s.Name == "" {
			continue
		}
		bodies[s.Name] = s.Body
		meta[s.Name] = s
		if s.Dir != "" {
			dirs[s.Name] = s.Dir
		}
	}
	for _, s := range skills {
		if s.Name == "" {
			continue
		}
		for _, k := range SkillAliasKeys(s) {
			if k == s.Name {
				continue
			}
			if _, taken := bodies[k]; taken {
				continue
			}
			bodies[k] = s.Body
			meta[k] = s
			if s.Dir != "" {
				dirs[k] = s.Dir
			}
		}
	}
	return bodies, dirs, meta
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
