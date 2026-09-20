package runtime

import (
	"path/filepath"
	"strconv"
	"strings"
)

// parseShellRead maps a cat/sed/head/type invocation onto read_file.
// Pipelines and compound commands stay in the real shell.
func parseShellRead(command string) (rel string, offset, limit int, ok bool) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", 0, 0, false
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
		rel = lastNonFlag(argv[1:])
		return rel, 0, 0, rel != "" && !strings.HasPrefix(rel, "-")
	case "get-content":
		rel = lastNonFlag(argv[1:])
		return rel, 0, 0, rel != "" && !strings.HasPrefix(rel, "-")
	case "head":
		limit = 10
		rest := argv[1:]
		if n, consumed := parseDashN(rest); consumed > 0 {
			limit = n
			rest = rest[consumed:]
		}
		rel = lastNonFlag(rest)
		return rel, 1, limit, rel != ""
	case "sed":
		start, end, rest, okSed := parseSedPrint(argv[1:])
		if !okSed {
			return "", 0, 0, false
		}
		rel = lastNonFlag(rest)
		if rel == "" {
			return "", 0, 0, false
		}
		return rel, start, end - start + 1, true
	default:
		return "", 0, 0, false
	}
}

func lastNonFlag(argv []string) string {
	for i := len(argv) - 1; i >= 0; i-- {
		a := strings.TrimSpace(argv[i])
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}

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
