// 从任务说明里认出点名的目录。这些目录可以读写，过浅的路径和家目录不放行。

package runtime

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

var (
	quotedWinPath = regexp.MustCompile("(?i)[`'\"]([a-z]:\\\\[^`'\"]+)[`'\"]")
	bareWinPath   = regexp.MustCompile(`(?i)\b([a-z]:\\[^\s"'` + "`" + `<>|*?]+)`)
	quotedUnixAbs = regexp.MustCompile("(?i)[`'\"](/(?:[^/`'\"]+/)+[^/`'\"]*)[`'\"]")
)

// NamedRoots 返回 text 里可以读写的目录。home 以及它下面的路径不返回。文件路径收成它所在的目录。
func NamedRoots(text, home string) []string {
	seen := map[string]bool{}
	var out []string
	for _, raw := range absPaths(text) {
		dir := rootDir(raw)
		if dir == "" || !deepEnough(dir) || underHome(dir, home) || hasYoyoSegment(dir) {
			continue
		}
		key := strings.ToLower(filepath.Clean(dir))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, filepath.Clean(dir))
	}
	return out
}

func absPaths(text string) []string {
	var out []string
	for _, re := range []*regexp.Regexp{quotedWinPath, bareWinPath, quotedUnixAbs} {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			if len(m) > 1 {
				out = append(out, strings.TrimSpace(m[1]))
			}
		}
	}
	return out
}

func rootDir(raw string) string {
	p := strings.TrimSpace(raw)
	p = strings.TrimRight(p, ".,;，。、)）]】")
	p = capability.CanonicalizeToolPath(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	if !filepath.IsAbs(p) {
		return ""
	}
	base := filepath.Base(p)
	if strings.Contains(base, ".") && base != "." && base != ".." {
		p = filepath.Dir(p)
	}
	if !filepath.IsAbs(p) {
		return ""
	}
	return p
}

func deepEnough(p string) bool {
	vol := filepath.VolumeName(p)
	rest := strings.TrimPrefix(p, vol)
	rest = strings.ReplaceAll(rest, `\`, "/")
	n := 0
	for _, part := range strings.Split(rest, "/") {
		if part != "" && part != "." {
			n++
		}
	}
	return n >= 3
}

func underHome(dir, home string) bool {
	home = strings.TrimSpace(home)
	if home == "" {
		return false
	}
	home = filepath.Clean(capability.CanonicalizeToolPath(home))
	return strings.EqualFold(filepath.Clean(dir), home) || capability.WithinWorkspace(home, dir)
}

func hasYoyoSegment(p string) bool {
	slash := strings.ToLower(filepath.ToSlash(p))
	for _, part := range strings.Split(slash, "/") {
		if part == ".yoyo" {
			return true
		}
	}
	return false
}
