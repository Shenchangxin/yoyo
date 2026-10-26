// 把 unix find 的 -name 译成 glob 模式，并识别不该交给 Windows FIND.exe 的写法。
// 不执行命令，也不调用 WorkspaceTools.resolve。Windows 上 tools.go 的 shell 在命中时改走 glob。

package runtime

import (
	"path"
	"path/filepath"
	"strings"
)

// parseUnixFindGlob 把 `find <根> -name <模式>` 映成 glob。命令不是 find，或没有 -name/-iname，则返回 false。
// 根为 . 或空时只返回模式。根本身含通配符则失败。Windows FIND.exe 的文本搜索（例如 find "needle" notes.txt）不改写。
// command 交给 SplitShellArgv，程序名是 find 时从中取根和 -name 或 -iname 后面的模式。
func parseUnixFindGlob(command string) (string, bool) {
	argv := SplitShellArgv(command)
	if len(argv) == 0 {
		return "", false
	}
	name := strings.ToLower(filepath.Base(argv[0]))
	name = strings.TrimSuffix(name, ".exe")
	if name != "find" {
		return "", false
	}
	root := ""
	pattern := ""
	for i := 1; i < len(argv); i++ {
		a := argv[i]
		switch strings.ToLower(a) {
		case "-name", "-iname":
			if i+1 >= len(argv) {
				return "", false
			}
			pattern = argv[i+1]
			i++
		case "-type", "-maxdepth", "-mindepth", "-mtime", "-size", "-user", "-group":
			i++
		case "-print", "-print0", "(", ")", "-o", "-or", "-a", "-and":
			// 表达式噪声，跳过。
		default:
			if strings.HasPrefix(a, "-") {
				continue
			}
			if root == "" {
				root = a
			}
		}
	}
	pattern = strings.Trim(pattern, `"'`)
	if pattern == "" {
		return "", false
	}
	root = strings.Trim(root, `"'`)
	root = strings.TrimPrefix(filepath.ToSlash(root), "./")
	if root == "" || root == "." {
		return pattern, true
	}
	if strings.ContainsAny(root, `*?[]`) {
		return "", false
	}
	return path.Join(root, "**", pattern), true
}

// unixFindMisuse 判断这是不是 unix find 的写法：含 -name、-type、-maxdepth 等，或以 . 或 ./ 为参数。
// Windows FIND.exe 的文本搜索返回 false。
// command 交给 SplitShellArgv；程序名是 find 时，再看后续参数里有没有 -name、-type、-maxdepth 等，或 .、./、.\ 开头的路径。
func unixFindMisuse(command string) bool {
	argv := SplitShellArgv(command)
	if len(argv) == 0 {
		return false
	}
	name := strings.ToLower(filepath.Base(argv[0]))
	name = strings.TrimSuffix(name, ".exe")
	if name != "find" {
		return false
	}
	for _, a := range argv[1:] {
		low := strings.ToLower(a)
		switch low {
		case "-name", "-iname", "-type", "-maxdepth", "-mindepth", "-print", "-print0":
			return true
		}
		if a == "." || strings.HasPrefix(a, "./") || strings.HasPrefix(a, ".\\") {
			return true
		}
	}
	return false
}
