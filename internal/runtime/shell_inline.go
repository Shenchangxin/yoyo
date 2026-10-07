package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// rewriteInlineInterpreters materializes `python -c` / `py -c` bodies to
// workspace temp files. cmd.exe quote stripping turns
// `python -c "import glob; print(glob.glob('*.pdf'))"` into a SyntaxError
// (`"import`); argv-direct execution is equally fragile once `-c` contains
// nested quotes. A file is the platform-stable contract.
func rewriteInlineInterpreters(command, workspace string) (string, func()) {
	command = strings.TrimSpace(command)
	if command == "" || !hasInlinePython(command) {
		return command, func() {}
	}
	var b strings.Builder
	var cleanups []func()
	rest := command
	changed := false
	for rest != "" {
		idx, interp, script, after := nextInlinePython(rest)
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		path, done, err := writeInlinePython(workspace, script)
		if err != nil {
			b.WriteString(rest)
			break
		}
		changed = true
		cleanups = append(cleanups, done)
		b.WriteString(rest[:idx])
		b.WriteString(interp)
		b.WriteByte(' ')
		b.WriteString(quoteShellPath(path))
		rest = after
	}
	if !changed {
		return command, func() {}
	}
	return b.String(), func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
}

func hasInlinePython(command string) bool {
	low := strings.ToLower(command)
	return strings.Contains(low, "python -c") || strings.Contains(low, "python.exe -c") ||
		strings.Contains(low, "python3 -c") || strings.Contains(low, "pythonw -c") ||
		strings.Contains(low, "py -c") || strings.Contains(low, "py.exe -c") ||
		strings.Contains(low, "py -3 -c")
}

func nextInlinePython(command string) (idx int, interp, script, after string) {
	for i := 0; i < len(command); i++ {
		if i > 0 && !isInlineBoundary(rune(command[i-1])) {
			continue
		}
		token, n := peekInterpreter(command[i:])
		if token == "" {
			continue
		}
		rest := strings.TrimLeft(command[i+n:], " \t")
		if isPyLauncher(token) {
			rest = stripPyLauncherVersion(rest)
		}
		if !hasDashC(rest) {
			continue
		}
		rest = strings.TrimLeft(rest[2:], " \t") // skip -c
		code, consumed := readInlineArg(rest)
		if strings.TrimSpace(code) == "" {
			continue
		}
		return i, token, code, rest[consumed:]
	}
	return -1, "", "", command
}

func peekInterpreter(s string) (string, int) {
	low := strings.ToLower(s)
	names := []string{"pythonw.exe", "python.exe", "python3", "pythonw", "python", "py.exe", "py"}
	for _, name := range names {
		if !strings.HasPrefix(low, name) {
			continue
		}
		n := len(name)
		if n < len(s) && s[n] != ' ' && s[n] != '\t' && !isInlineBoundary(rune(s[n])) {
			continue
		}
		return s[:n], n
	}
	return "", 0
}

func isPyLauncher(token string) bool {
	low := strings.ToLower(token)
	return low == "py" || low == "py.exe"
}

func stripPyLauncherVersion(s string) string {
	if len(s) < 2 || s[0] != '-' {
		return s
	}
	if s[1] < '0' || s[1] > '9' {
		return s
	}
	i := 2
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i < len(s) && s[i] != ' ' && s[i] != '\t' && !isInlineBoundary(rune(s[i])) {
		return s
	}
	return strings.TrimLeft(s[i:], " \t")
}

func hasDashC(s string) bool {
	if !strings.HasPrefix(s, "-c") {
		return false
	}
	if len(s) == 2 {
		return true
	}
	return isInlineBoundary(rune(s[2])) || s[2] == ' ' || s[2] == '\t'
}

func readInlineArg(s string) (string, int) {
	if s == "" {
		return "", 0
	}
	if s[0] == '"' || s[0] == '\'' {
		quote := s[0]
		var b strings.Builder
		esc := false
		for i := 1; i < len(s); i++ {
			c := s[i]
			if esc {
				b.WriteByte(c)
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == quote {
				return b.String(), i + 1
			}
			b.WriteByte(c)
		}
		return b.String(), len(s)
	}
	i := 0
	for i < len(s) {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' {
			break
		}
		if (s[i] == '&' || s[i] == '|') && i+1 < len(s) && s[i+1] == s[i] {
			break
		}
		if s[i] == '|' || s[i] == ';' {
			break
		}
		i++
	}
	return s[:i], i
}

func isInlineBoundary(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == ';' || r == '|' || r == '&' || r == '(' || unicode.IsSpace(r)
}

func writeInlinePython(workspace, script string) (string, func(), error) {
	dir := os.TempDir()
	if strings.TrimSpace(workspace) != "" {
		dir = filepath.Join(workspace, ".yoyo", "tmp")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			dir = os.TempDir()
		}
	}
	f, err := os.CreateTemp(dir, "yoyo-inline-*.py")
	if err != nil {
		return "", func() {}, err
	}
	path := f.Name()
	if _, err := f.WriteString(script); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", func() {}, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func quoteShellPath(p string) string {
	if p == "" {
		return p
	}
	if !strings.ContainsAny(p, " \t\"'") {
		return p
	}
	return `"` + strings.ReplaceAll(p, `"`, `\"`) + `"`
}
