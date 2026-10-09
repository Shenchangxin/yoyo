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
	if command == "" || !hasInlineCode(command) {
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
		path, done, err := writeInlineScript(workspace, script, inlineExt(interp))
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

func hasInlineCode(command string) bool {
	low := strings.ToLower(command)
	return strings.Contains(low, "python -c") || strings.Contains(low, "python.exe -c") ||
		strings.Contains(low, "python3 -c") || strings.Contains(low, "pythonw -c") ||
		strings.Contains(low, "py -c") || strings.Contains(low, "py.exe -c") ||
		strings.Contains(low, "py -3 -c") ||
		strings.Contains(low, "node -e") || strings.Contains(low, "node.exe -e") ||
		strings.Contains(low, "nodejs -e") || strings.Contains(low, "node --eval") ||
		strings.Contains(low, "node -p") || strings.Contains(low, "node.exe -p")
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
		flagLen, ok := inlineCodeFlag(rest)
		if !ok {
			continue
		}
		rest = strings.TrimLeft(rest[flagLen:], " \t")
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
	names := []string{"pythonw.exe", "python.exe", "python3", "pythonw", "python", "py.exe", "py", "nodejs.exe", "node.exe", "nodejs", "node"}
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

func inlineCodeFlag(s string) (int, bool) {
	for _, fl := range []string{"--eval", "--print", "-e", "-p", "-c"} {
		if !strings.HasPrefix(s, fl) {
			continue
		}
		n := len(fl)
		if n == len(s) || isInlineBoundary(rune(s[n])) || s[n] == ' ' || s[n] == '\t' {
			return n, true
		}
	}
	return 0, false
}

func inlineExt(token string) string {
	low := strings.ToLower(strings.TrimSpace(token))
	if strings.HasPrefix(low, "node") {
		return ".js"
	}
	return ".py"
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
				// Only the wrapping quote and \\ are shell escapes. Keep
				// \s \/ \n so JS regex and Python strings survive the file.
				switch c {
				case quote, '\\':
					b.WriteByte(c)
				default:
					b.WriteByte('\\')
					b.WriteByte(c)
				}
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
		if esc {
			b.WriteByte('\\')
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

func writeInlineScript(workspace, script, ext string) (string, func(), error) {
	if ext == "" {
		ext = ".py"
	}
	dir := os.TempDir()
	if strings.TrimSpace(workspace) != "" {
		dir = filepath.Join(workspace, ".yoyo", "tmp")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			dir = os.TempDir()
		}
	}
	f, err := os.CreateTemp(dir, "yoyo-inline-*"+ext)
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

// unescapeWinPathEscapes applies JSON-style \t \n \r inside a Windows path.
// Models emit cd "C:\Users\...\Desktop\test"; JSON "\test" becomes a tab and
// CreateProcess returns ERROR_INVALID_NAME.
func unescapeWinPathEscapes(p string) string {
	return strings.NewReplacer(`\t`, "\t", `\n`, "\n", `\r`, "\r").Replace(p)
}

func repairWorkspacePathEscapes(command, workspace string) string {
	if command == "" || workspace == "" {
		return command
	}
	broken := unescapeWinPathEscapes(workspace)
	if broken == workspace || !strings.Contains(command, broken) {
		return command
	}
	return strings.ReplaceAll(command, broken, workspace)
}

func sameFilePath(a, b string) bool {
	canon := func(p string) string {
		p = strings.TrimSpace(strings.Trim(p, "`'\""))
		p = strings.ReplaceAll(p, `\`, `/`)
		p = strings.TrimRight(p, `/`)
		return strings.ToLower(p)
	}
	return canon(a) == canon(b)
}

func isRedundantCwd(path, workspace string) bool {
	path = strings.TrimSpace(path)
	if path == "" || path == "." || path == "./" || path == `.\` {
		return true
	}
	if workspace == "" {
		return false
	}
	return sameFilePath(path, workspace) || sameFilePath(path, unescapeWinPathEscapes(workspace))
}

func readShellPathToken(s string) (string, int) {
	if s == "" {
		return "", 0
	}
	switch s[0] {
	case '"', '\'', '`':
		q := s[0]
		i := 1
		for i < len(s) && s[i] != q {
			i++
		}
		if i >= len(s) {
			return s[1:], len(s)
		}
		return s[1:i], i + 1
	}
	i := 0
	for i < len(s) {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '&' || s[i] == '|' || s[i] == ';' {
			break
		}
		i++
	}
	return s[:i], i
}

func stripOneWorkspaceCd(command, workspace string) (string, bool) {
	s := strings.TrimLeft(command, " \t")
	low := strings.ToLower(s)
	verbs := []string{
		"set-location -literalpath ",
		"set-location -path ",
		"set-location ",
		"pushd ",
		"cd /d ",
		"cd ",
	}
	rest := ""
	found := false
	for _, v := range verbs {
		if strings.HasPrefix(low, v) {
			rest = strings.TrimLeft(s[len(v):], " \t")
			found = true
			break
		}
	}
	if !found {
		return command, false
	}
	path, n := readShellPathToken(rest)
	if n == 0 {
		return command, false
	}
	after := strings.TrimLeft(rest[n:], " \t")
	sep := 0
	switch {
	case strings.HasPrefix(after, "&&"):
		sep = 2
	case strings.HasPrefix(after, ";"):
		sep = 1
	case strings.HasPrefix(after, "&"):
		sep = 1
	default:
		return command, false
	}
	tail := strings.TrimLeft(after[sep:], " \t")
	if tail == "" || !isRedundantCwd(path, workspace) {
		return command, false
	}
	return tail, true
}

// stripRedundantWorkspaceCd drops `cd <workspace> &&` — the tool already
// runs with that cwd. Quoted `C:\Users\...` paths are also how Git Bash
// turns `\t` into a tab.
func stripRedundantWorkspaceCd(command, workspace string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return command
	}
	for i := 0; i < 4; i++ {
		next, ok := stripOneWorkspaceCd(command, workspace)
		if !ok {
			break
		}
		command = next
	}
	return command
}
