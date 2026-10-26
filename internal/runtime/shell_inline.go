// 把 python -c 和 node -e 的内联代码写成工作区临时脚本，并整理 Windows 引号与多余的 cd。
// 不执行解释器，也不调用 WorkspaceTools.resolve。本文件没有 Windows Job Object。tools.go 的 shell 在策略检查前调用这里。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// rewriteInlineInterpreters 把 `python -c` 和 `py -c` 的正文落到工作区临时文件。cmd.exe 剥引号会把 `python -c "import glob; print(glob.glob('*.pdf'))"` 变成语法错误（`"import`）；-c 里再嵌引号时，直接按 argv 执行同样不稳。文件才是各平台都稳的约定。
// 同时处理 node -e、-p 和 --eval。写文件失败则保留尚未改写的剩余命令。
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

// hasInlineCode 判断命令里有没有 python -c、py -c、node -e、node -p 或 node --eval。
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

// nextInlinePython 找出下一处内联解释器、代码和其后的尾巴。解释器必须落在边界上。py 启动器可以带 -3 这类版本号。
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

// peekInterpreter 从当前位置匹配 python 或 node 的可执行名，并要求名字在边界处结束。
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

// isPyLauncher 判断标记是 py 或 py.exe。
func isPyLauncher(token string) bool {
	low := strings.ToLower(token)
	return low == "py" || low == "py.exe"
}

// stripPyLauncherVersion 去掉 py 启动器后面的 -3 或 -3.11 这类版本号。后面不是空白或边界则不剥。
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

// inlineCodeFlag 识别 --eval、--print、-e、-p 或 -c，并要求开关在边界处结束。
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

// inlineExt 给 node 用 .js，其余用 .py。
func inlineExt(token string) string {
	low := strings.ToLower(strings.TrimSpace(token))
	if strings.HasPrefix(low, "node") {
		return ".js"
	}
	return ".py"
}

// readInlineArg 读取 -c 或 -e 后面的代码。引号内只解开外层引号和反斜杠本身。无引号时读到空白、换行、管道、分号或 &&、||。
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
				// 只有包住代码的引号和 \\ 是 shell 转义。\s、\/、\n 原样留下，JS 正则和 Python 字符串才能进文件。
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

// isInlineBoundary 判断字符是空白、分号、管道、& 或左括号。
func isInlineBoundary(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == ';' || r == '|' || r == '&' || r == '(' || unicode.IsSpace(r)
}

// writeInlineScript 把脚本写到工作区 .yoyo/tmp 的 yoyo-inline-* 文件。工作区为空或建目录失败时改用系统临时目录。返回的函数删除该文件。
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

// quoteShellPath 给会被 shell 拆开或吃掉的路径加引号。Windows 路径即使没有空格也要加引号：Git Bash 把未加引号的反斜杠当转义，C:\Users\x 到 node 时会变成 C:Usersx。工作区临时脚本在 Windows 上总是带反斜杠。
// 路径不含空格、制表符、引号或反斜杠时原样返回。
func quoteShellPath(p string) string {
	if p == "" {
		return p
	}
	if !strings.ContainsAny(p, " \t\"'\\") {
		return p
	}
	return `"` + strings.ReplaceAll(p, `"`, `\"`) + `"`
}

// normalizeWindowsQuotes 整理 Windows 命令的引号。将在 Git Bash 下运行的命令保留引号，因为 bash 会吃掉未加引号路径里的反斜杠。其余交给 cmd，使用放宽后的形式。
func normalizeWindowsQuotes(command string) string {
	if looksPosixUnix(command) {
		return command
	}
	return relaxRedundantShellQuotes(command)
}

// relaxRedundantShellQuotes 去掉不含空白或 cmd 元字符的引号。否则 cmd /S /C 加上 Go 额外的引号，会把 `node "rel\file.mjs"` 变成当前目录加上 `"\"rel\file.mjs"`。只给 cmd 命令调用；见 normalizeWindowsQuotes。
func relaxRedundantShellQuotes(command string) string {
	if command == "" || !strings.Contains(command, `"`) {
		return command
	}
	var b strings.Builder
	b.Grow(len(command))
	for i := 0; i < len(command); {
		if command[i] != '"' {
			b.WriteByte(command[i])
			i++
			continue
		}
		j := i + 1
		for j < len(command) && command[j] != '"' {
			j++
		}
		if j >= len(command) {
			b.WriteString(command[i:])
			break
		}
		inner := command[i+1 : j]
		if inner != "" && !strings.ContainsAny(inner, " \t&|<>^") {
			b.WriteString(inner)
		} else {
			b.WriteString(command[i : j+1])
		}
		i = j + 1
	}
	return b.String()
}

// unescapeWinPathEscapes 把 Windows 路径里的 JSON 风格 \t、\n、\r 换成真正的制表符和换行。模型写出 cd "C:\Users\...\Desktop\test" 时，JSON 的 "\test" 会变成制表符，CreateProcess 返回 ERROR_INVALID_NAME。
func unescapeWinPathEscapes(p string) string {
	return strings.NewReplacer(`\t`, "\t", `\n`, "\n", `\r`, "\r").Replace(p)
}

// repairWorkspacePathEscapes 把命令里被转义弄坏的工作区路径换回原始 workspace。两边相同或命令里没有坏路径时原样返回。
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

// sameFilePath 在去掉引号、统一斜杠、去掉末尾斜杠并忽略大小写后比较两条路径。
func sameFilePath(a, b string) bool {
	canon := func(p string) string {
		p = strings.TrimSpace(strings.Trim(p, "`'\""))
		p = strings.ReplaceAll(p, `\`, `/`)
		p = strings.TrimRight(p, `/`)
		return strings.ToLower(p)
	}
	return canon(a) == canon(b)
}

// isRedundantCwd 判断路径是空、.，或与工作区及其被转义弄坏的形式相同。工具已经以该目录为工作目录。
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

// readShellPathToken 读取 cd 后面的路径。引号、单引号和反引号成对剥掉。无引号时读到空白、&、| 或分号。
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

// stripOneWorkspaceCd 去掉开头一次指向工作区的 cd、pushd 或 Set-Location，后面必须是 &&、; 或 &。
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

// stripRedundantWorkspaceCd 去掉 `cd <工作区> &&`。工具已经以该目录为当前目录。带引号的 `C:\Users\...` 也是 Git Bash 把 `\t` 变成制表符的那种路径。最多剥四层。
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
