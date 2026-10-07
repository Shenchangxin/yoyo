package runtime

import (
	"path"
	"path/filepath"
	"strings"
)

// parseUnixFindGlob maps `find <root> -name <pat>` onto a glob pattern.
// Windows FIND.exe is a text search; models keep emitting unix find(1).
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
			// expression noise
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
