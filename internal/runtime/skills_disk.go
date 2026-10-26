// 从磁盘根目录收集 SKILL.md，并按名字、显示名和目录别名建索引。包引导是否把正文放进提示由 pack_bootstrap.go 决定，本文件只加载文件。

package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// SkillRoots 按 bundled、home/skills、packDirs、工作区 .yoyo/skills、.agents/skills、skills 的顺序返回非空根。空的 home、workspace、bundled 或 pack 目录会跳过。
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

// LoadSkillDirs 遍历每个根下的 SKILL.md。同名技能后者覆盖前者，但 order 保留第一次出现的位置。解析失败的文件跳过。
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

// SkillPackFiles 列出目录下非点开头的文件，相对路径用斜杠，并按名字排序。空目录返回 nil。
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

// SkillAliasKeys 返回可用的加载名：frontmatter 的 name、显示名，以及磁盘目录的基名。重复项去掉。
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

// MapSkills 先按主名写入正文、目录和元数据，再把尚未占用的别名指到同一技能，使 load_skill 和 @skill 能用安装目录名。
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

// MergeSkills 以 base 为底，extra 里同名的项整份替换，新名字追加在后。
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

// FilterSkills 在 planMode 为假时原样返回。为真时丢掉 Compatibility 含 network 的技能。
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
