package runtime

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type ignoreSet struct {
	names map[string]bool
	globs []string
}

func loadIgnore(root string) ignoreSet {
	s := ignoreSet{names: map[string]bool{}}
	for k, v := range skipDirNames {
		s.names[k] = v
	}
	b, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return s
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		line = strings.TrimSuffix(line, "/")
		if !strings.ContainsAny(line, "*?[") && !strings.Contains(line, "/") {
			s.names[line] = true
			continue
		}
		s.globs = append(s.globs, filepath.ToSlash(line))
	}
	return s
}

func (s ignoreSet) skipDir(name string) bool {
	return s.names[name]
}

func (s ignoreSet) skipFile(rel, name string) bool {
	if s.names[name] {
		return true
	}
	rel = filepath.ToSlash(rel)
	for _, g := range s.globs {
		if ok, _ := filepath.Match(g, name); ok {
			return true
		}
		if ok, _ := filepath.Match(g, rel); ok {
			return true
		}
		if strings.HasSuffix(g, "/**") {
			pref := strings.TrimSuffix(g, "/**")
			if rel == pref || strings.HasPrefix(rel, pref+"/") {
				return true
			}
		}
	}
	return false
}
