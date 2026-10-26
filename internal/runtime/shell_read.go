// 把单条 cat、type、Get-Content、head 或 sed -n 映射成 read_file 的路径和行范围。
// 含管道或重定向的命令留在真正的 shell。不调用 WorkspaceTools.resolve；聊天叠层命中后由 tools.go 再走 readFile。

package runtime

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// parseShellRead 把 cat、sed、head、type 或 Get-Content 映到相对路径、起始行和行数。
// 管道留在真正的 shell。单独的 `cd 目录 && <读取>` 会改写路径，避免聊天经 bash 倾倒源码。
func parseShellRead(command string) (rel string, offset, limit int, ok bool) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", 0, 0, false
	}
	prefixDir := ""
	if dir, rest, stripped := splitCdAnd(cmd); stripped {
		prefixDir = dir
		cmd = rest
	}
	if strings.ContainsAny(cmd, "|&;><") {
		return "", 0, 0, false
	}
	argv := SplitShellArgv(cmd)
	if len(argv) < 2 {
		return "", 0, 0, false
	}
	bin := strings.ToLower(filepath.Base(argv[0]))
	bin = strings.TrimSuffix(bin, ".exe")
	switch bin {
	case "cat", "type":
		files := nonFlagFiles(argv[1:])
		if len(files) != 1 {
			return "", 0, 0, false
		}
		rel, ok = files[0], true
	case "get-content":
		files := nonFlagFiles(argv[1:])
		if len(files) != 1 {
			return "", 0, 0, false
		}
		rel, ok = files[0], true
	case "head":
		limit = 10
		rest := argv[1:]
		if n, consumed := parseDashN(rest); consumed > 0 {
			limit = n
			rest = rest[consumed:]
		}
		rel = lastNonFlag(rest)
		offset, ok = 1, rel != ""
	case "sed":
		start, end, rest, okSed := parseSedPrint(argv[1:])
		if !okSed {
			return "", 0, 0, false
		}
		rel = lastNonFlag(rest)
		if rel == "" {
			return "", 0, 0, false
		}
		return joinReadPath(prefixDir, rel), start, end - start + 1, true
	default:
		return "", 0, 0, false
	}
	if !ok {
		return "", 0, 0, false
	}
	return joinReadPath(prefixDir, rel), offset, limit, true
}

// splitCdAnd 只接受恰好一个 &&，且左边是 cd。两边再出现管道或重定向则不拆。/d 和 -d 不当成目录。
func splitCdAnd(cmd string) (dir, rest string, ok bool) {
	if strings.Count(cmd, "&&") != 1 {
		return "", "", false
	}
	left, right, _ := strings.Cut(cmd, "&&")
	if strings.ContainsAny(left, "|&;><") || strings.ContainsAny(right, "|&;><") {
		return "", "", false
	}
	argv := SplitShellArgv(strings.TrimSpace(left))
	if len(argv) < 2 {
		return "", "", false
	}
	bin := strings.ToLower(filepath.Base(argv[0]))
	bin = strings.TrimSuffix(bin, ".exe")
	if bin != "cd" {
		return "", "", false
	}
	dir = argv[len(argv)-1]
	if dir == "/d" || dir == "-d" {
		return "", "", false
	}
	rest = strings.TrimSpace(right)
	if dir == "" || rest == "" {
		return "", "", false
	}
	return dir, rest, true
}

// joinReadPath 把目录和相对路径拼起来。绝对路径和已能译成盘符的 MSYS 路径不再拼接。不调用 WorkspaceTools.resolve。
func joinReadPath(dir, rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return rel
	}
	if cap := capability.CanonicalizeToolPath(rel); filepath.IsAbs(cap) {
		return cap
	}
	if dir == "" {
		return rel
	}
	dir = capability.CanonicalizeToolPath(strings.TrimSpace(dir))
	if filepath.IsAbs(rel) || (strings.HasPrefix(rel, "/") && !msysDriveRel(rel)) {
		return rel
	}
	return filepath.ToSlash(filepath.Join(dir, rel))
}

// msysDriveRel 判断路径能否被 MSYSToWindowsPath 译成盘符路径。
func msysDriveRel(p string) bool {
	_, ok := capability.MSYSToWindowsPath(p)
	return ok
}

// lastNonFlag 返回参数里最后一个非开关文件。
func lastNonFlag(argv []string) string {
	files := nonFlagFiles(argv)
	if len(files) == 0 {
		return ""
	}
	return files[len(files)-1]
}

// nonFlagFiles 丢掉空串和以 - 开头的参数。
func nonFlagFiles(argv []string) []string {
	var out []string
	for _, a := range argv {
		a = strings.TrimSpace(a)
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// parseDashN 解析 head 的 -n N、-nN 或 -N。非正数返回 0。
func parseDashN(argv []string) (n int, consumed int) {
	if len(argv) == 0 {
		return 0, 0
	}
	if argv[0] == "-n" && len(argv) > 1 {
		v, err := strconv.Atoi(argv[1])
		if err != nil || v <= 0 {
			return 0, 0
		}
		return v, 2
	}
	if strings.HasPrefix(argv[0], "-n") && len(argv[0]) > 2 {
		v, err := strconv.Atoi(argv[0][2:])
		if err != nil || v <= 0 {
			return 0, 0
		}
		return v, 1
	}
	if len(argv[0]) > 1 && argv[0][0] == '-' {
		v, err := strconv.Atoi(argv[0][1:])
		if err != nil || v <= 0 {
			return 0, 0
		}
		return v, 1
	}
	return 0, 0
}

// parseSedPrint 只接受带 -n 的 `起,止p`。起始必须为正，结束不小于起始。
func parseSedPrint(argv []string) (start, end int, rest []string, ok bool) {
	quiet := false
	i := 0
	for i < len(argv) {
		a := argv[i]
		if a == "-n" || a == "-E" || a == "-r" {
			if a == "-n" {
				quiet = true
			}
			i++
			continue
		}
		break
	}
	if !quiet || i >= len(argv) {
		return 0, 0, nil, false
	}
	expr := strings.TrimSpace(argv[i])
	i++
	expr = strings.Trim(expr, `"'`)
	expr = strings.TrimSuffix(expr, "p")
	parts := strings.Split(expr, ",")
	if len(parts) != 2 {
		return 0, 0, nil, false
	}
	s, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	e, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || s <= 0 || e < s {
		return 0, 0, nil, false
	}
	return s, e, argv[i:], true
}
