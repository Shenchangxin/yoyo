// shim.go 解析 npm 在 Windows 上生成的 .cmd 垫片，找出 JS 入口，好用 node.exe 直接启动、不经 cmd.exe。
// 解析和平台无关，任何系统上都能测；真正改写命令在 shim_windows.go，其他平台原样启动。

package buddy

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// reShimEntry 匹配垫片最后一行里 "%dp0%\..." %* 那个入口。
	reShimEntry = regexp.MustCompile(`"%dp0%\\([^"]+)"\s+%\*`)
	// reShimProg 匹配 SET "_prog=..." 里的解释器。
	reShimProg = regexp.MustCompile(`(?i)SET\s+"_prog=([^"]+)"`)
)

// parseShim 从垫片内容里取出 JS 入口的绝对路径。dir 是垫片所在目录。解释器不是 node 时返回假。
func parseShim(content, dir string) (string, bool) {
	m := reShimEntry.FindStringSubmatch(content)
	if m == nil {
		return "", false
	}
	progs := reShimProg.FindAllStringSubmatch(content, -1)
	if len(progs) == 0 {
		return "", false
	}
	for _, p := range progs {
		base := strings.ToLower(path.Base(strings.ReplaceAll(p[1], `\`, "/")))
		if base != "node" && base != "node.exe" {
			return "", false
		}
	}
	rel := strings.ReplaceAll(m[1], `\`, "/")
	return filepath.Join(dir, filepath.FromSlash(rel)), true
}

// nodeShebang 判断脚本开头是不是 node 的 shebang，或者根本没有 shebang（npm 默认按 node 跑）。
func nodeShebang(head []byte) bool {
	text := string(head)
	if !strings.HasPrefix(text, "#!") {
		return true
	}
	line := text
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		line = text[:i]
	}
	return strings.Contains(line, "node")
}
