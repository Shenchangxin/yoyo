// 给 glob、grep 和提及扫描用的忽略集。合并 skipDirNames 和根目录 .gitignore 里的简单规则。
// 不实现 git 的否定和多级目录语义。以 ! 开头的行直接跳过，不会把已忽略的名字重新纳入。

package runtime

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ignoreSet 保存按目录名跳过的表，以及带通配或斜杠的规则。
type ignoreSet struct {
	names map[string]bool
	globs []string
}

// loadIgnore 读取根目录的 .gitignore。空行、# 注释和以 ! 开头的行忽略。
// 不含通配也不含斜杠的名字进入 names，其余按斜杠形式放进 globs。读失败时只保留 skipDirNames。
// root 与 .gitignore 拼成要读的路径。
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

// skipDir 只按目录名判断，不看相对路径。
// name 作为 names 的键。
func (s ignoreSet) skipDir(name string) bool {
	return s.names[name]
}

// skipFile 按文件名、相对路径匹配。以 /** 结尾的规则再按目录前缀匹配。
// rel 转成斜杠后与规则做 Match，并以 /** 的目录前缀比较。
// name 先查 names，再与规则做 Match。
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
