// 识别会在捕获标准输出时挂住的 cmd start，并改写成 PowerShell 的 Start-Process。
// 不创建进程，也不做路径 resolve。本文件没有 Windows Job Object。tools.go 的 shell 在 Windows 上调用这里。

package runtime

import (
	"fmt"
	"regexp"
	"strings"
)

// cmd start 在捕获标准输出时到不了 EOF：子进程握着管道句柄。这是进程生命周期问题，不写进系统提示。把子进程拆出去，shell 才能结束。

const cmdStartHangMsg = "cmd start waits for the child while stdout is captured, and piping it to head/tail never reaches EOF — the tool hangs until timeout. Use Start-Process -WindowStyle Hidden (no pipe to head/tail). Then sleep and Get-Process. PowerShell $ variables are passed as argv and must not be wrapped in Git Bash."

var (
	reCmdStartHead = regexp.MustCompile(`(?i)\bcmd(?:\.exe)?\s+/{1,2}c\s+`)
	reBareStart    = regexp.MustCompile(`(?i)(?:^|[;&\n]|&&|\|\|)\s*start\s+(?:/|"|')`)
	reHeadPipe     = regexp.MustCompile(`(?i)\s*(?:2>&1\s*)?(?:\|\s*(?:head|tail)\b[^;|&]*)`)
)

// cmdStartWouldHang 判断命令里有没有会挂住的 cmd start 或裸 start。
// command 交给 cmdStartPattern。
func cmdStartWouldHang(command string) bool {
	return cmdStartPattern(command)
}

// cmdStartPattern 在裸 start 或 cmd /c 后面的 start 载荷会挂住时返回真。Start-Process 不算。
// command 先匹配裸 start；否则从 cmd /c 之后交给 takeStartPayload，失败时再看剩余文本是不是以 start 开头。
func cmdStartPattern(command string) bool {
	if reBareStart.MatchString(command) {
		return true
	}
	loc := reCmdStartHead.FindStringIndex(command)
	if loc == nil {
		return false
	}
	rest := strings.TrimLeft(command[loc[1]:], " \t")
	inner, _, ok := takeStartPayload(rest)
	if !ok {
		low := strings.ToLower(rest)
		return strings.HasPrefix(low, "start ") || strings.HasPrefix(low, "start\"")
	}
	return startPayloadHangs(inner)
}

// startPayloadHangs 判断载荷是 start 而不是 Start-Process。
// inner 去掉空白并转成小写后，以 start-process 开头则返回假；以 start 加空格或引号开头，或等于 start，则返回真。
func startPayloadHangs(inner string) bool {
	low := strings.ToLower(strings.TrimSpace(inner))
	if strings.HasPrefix(low, "start-process") {
		return false
	}
	return strings.HasPrefix(low, "start ") || strings.HasPrefix(low, "start\"") || low == "start"
}

// rewriteCmdStart 把 `cmd /c start ...` 以及紧跟的 2>&1 | head/tail 换成 PowerShell Start-Process。Start-Process 在标准输出被捕获时仍会返回。
// 文件名为空或带 /WAIT 时不改写。
// command 在 cmd /c 之后把载荷交给 takeStartPayload 和 parseStartInvocation；文件名为空或带 /WAIT 时原样返回。
func rewriteCmdStart(command string) (string, bool) {
	loc := reCmdStartHead.FindStringIndex(command)
	if loc == nil {
		return command, false
	}
	prefix := command[:loc[0]]
	padded := command[loc[1]:]
	nspace := len(padded) - len(strings.TrimLeft(padded, " \t"))
	rest := padded[nspace:]
	inner, consumed, ok := takeStartPayload(rest)
	if !ok || !startPayloadHangs(inner) {
		return command, false
	}
	file, args, wait := parseStartInvocation(inner)
	args = stripCmdRedirects(args)
	if file == "" || wait {
		return command, false
	}
	after := rest[consumed:]
	if m := reHeadPipe.FindStringIndex(after); m != nil && m[0] == 0 {
		after = after[m[1]:]
	}
	return strings.TrimSpace(prefix + startProcessCommand(file, args) + after), true
}

// rewriteAnyStart 先尝试改写 cmd /c start，失败再改写裸 start。
// command 先交给 rewriteCmdStart，没有改写再交给 rewriteBareStart。
func rewriteAnyStart(command string) (string, bool) {
	if next, ok := rewriteCmdStart(command); ok {
		return next, true
	}
	return rewriteBareStart(command)
}

// rewriteBareStart 改写不在 cmd /c 里的 start。带 /WAIT 或没有文件名时不改。
// command 在裸 start 处切开，载荷交给 takeStartPayload 和 parseStartInvocation；文件名为空或带 /WAIT 时原样返回。
func rewriteBareStart(command string) (string, bool) {
	loc := reBareStart.FindStringIndex(command)
	if loc == nil {
		return command, false
	}
	head := command[loc[0]:]
	idx := strings.Index(strings.ToLower(head), "start")
	if idx < 0 {
		return command, false
	}
	prefix := command[:loc[0]+idx]
	rest := head[idx:]
	inner, consumed, ok := takeStartPayload(rest)
	if !ok || !startPayloadHangs(inner) {
		return command, false
	}
	file, args, wait := parseStartInvocation(inner)
	args = stripCmdRedirects(args)
	if file == "" || wait {
		return command, false
	}
	after := rest[consumed:]
	if m := reHeadPipe.FindStringIndex(after); m != nil && m[0] == 0 {
		after = after[m[1]:]
	}
	return strings.TrimSpace(prefix + startProcessCommand(file, args) + after), true
}

// stripCmdRedirects 从 start 参数里去掉重定向和 NUL 目标，避免它们进 Start-Process 的参数列表。
// args 逐项去掉空白并转成小写后识别重定向；>、>>、1>、2> 这类还跳过下一项。
func stripCmdRedirects(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := strings.ToLower(strings.TrimSpace(args[i]))
		switch {
		case a == ">" || a == ">>" || a == "1>" || a == "2>" || a == "1>>" || a == "2>>":
			i++
		case a == "2>&1" || a == ">&1" || a == ">nul" || a == "1>nul" || a == "2>nul" ||
			strings.HasPrefix(a, ">") || strings.HasPrefix(a, "2>"):
		default:
			out = append(out, args[i])
		}
	}
	return out
}

// takeStartPayload 取出 start 后面的一段。引号外的分号、换行、管道和单独的 & 截断。`>&` 里的 & 不算语句分隔。
// rest 为空时失败；不以引号开头时在分号、换行、管道或单独的 & 处截断，以引号开头时读到闭合引号。
func takeStartPayload(rest string) (inner string, consumed int, ok bool) {
	if rest == "" {
		return "", 0, false
	}
	if rest[0] != '"' {
		end := len(rest)
		for i := 0; i < len(rest); i++ {
			if rest[i] == ';' || rest[i] == '\n' || rest[i] == '|' {
				end = i
				break
			}
			if rest[i] == '&' {
				if i > 0 && rest[i-1] == '>' {
					continue
				}
				end = i
				break
			}
		}
		return strings.TrimSpace(rest[:end]), end, true
	}
	var b strings.Builder
	i := 1
	for i < len(rest) {
		if rest[i] == '"' && i+1 < len(rest) && rest[i+1] == '"' {
			b.WriteByte('"')
			i += 2
			continue
		}
		if rest[i] == '\\' && i+1 < len(rest) {
			b.WriteByte(rest[i+1])
			i += 2
			continue
		}
		if rest[i] == '"' {
			return b.String(), i + 1, true
		}
		b.WriteByte(rest[i])
		i++
	}
	return "", 0, false
}

// parseStartInvocation 从 start 载荷里取出文件、参数，以及是否带 /WAIT。引号里的标题被丢掉。/D 的下一个词是目录，不进参数。
// inner 去掉空白后，若开头五个字节不区分大小写等于 start 就剥掉，再跳过引号标题和 / 开关。
func parseStartInvocation(inner string) (file string, args []string, wait bool) {
	s := strings.TrimSpace(inner)
	if len(s) >= 5 && strings.EqualFold(s[:5], "start") {
		s = strings.TrimSpace(s[5:])
	}
	if strings.HasPrefix(s, `"`) {
		_, rest, ok := readQuotedToken(s)
		if !ok {
			return "", nil, false
		}
		s = strings.TrimSpace(rest)
	}
	for {
		s = strings.TrimSpace(s)
		if !strings.HasPrefix(s, "/") {
			break
		}
		sw, rest := nextShellToken(s)
		if strings.EqualFold(sw, "/WAIT") {
			wait = true
		}
		s = strings.TrimSpace(rest)
		if strings.EqualFold(sw, "/D") {
			_, s = nextShellToken(s)
			s = strings.TrimSpace(s)
		}
	}
	file, rest := nextShellToken(s)
	if file == "" {
		return "", nil, wait
	}
	for rest = strings.TrimSpace(rest); rest != ""; rest = strings.TrimSpace(rest) {
		tok, next := nextShellToken(rest)
		if tok == "" {
			break
		}
		args = append(args, tok)
		rest = next
	}
	return file, args, wait
}

// readQuotedToken 读取引号字符串。成对的 "" 是一个引号，反斜杠吃掉下一个字节。引号未闭合则失败。
// s 为空或不以引号开头时失败；成对的 "" 收成一个引号，反斜杠吃掉下一个字节。
func readQuotedToken(s string) (tok, rest string, ok bool) {
	if s == "" || s[0] != '"' {
		return "", s, false
	}
	var b strings.Builder
	i := 1
	for i < len(s) {
		if s[i] == '"' && i+1 < len(s) && s[i+1] == '"' {
			b.WriteByte('"')
			i += 2
			continue
		}
		if s[i] == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == '"' {
			return b.String(), s[i+1:], true
		}
		b.WriteByte(s[i])
		i++
	}
	return "", s, false
}

// nextShellToken 取下一个词。以引号开头且能闭合时返回引号内的内容。
// s 先去掉空白，空串时返回两个空串；以引号开头且 readQuotedToken 成功时采用它的结果，否则取到空白为止。
func nextShellToken(s string) (tok, rest string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if s[0] == '"' {
		tok, rest, ok := readQuotedToken(s)
		if ok {
			return tok, rest
		}
	}
	i := 0
	for i < len(s) && s[i] != ' ' && s[i] != '\t' {
		i++
	}
	return s[:i], s[i:]
}

// startProcessCommand 拼出 powershell -NoProfile -Command "Start-Process -WindowStyle Hidden ..."。参数用单引号包住。
// args 长度大于 0 时逐项交给 psSingle，写成 -ArgumentList。
func startProcessCommand(file string, args []string) string {
	var b strings.Builder
	b.WriteString(`powershell -NoProfile -Command "Start-Process -WindowStyle Hidden -FilePath `)
	b.WriteString(psSingle(file))
	if len(args) > 0 {
		b.WriteString(` -ArgumentList `)
		for i, a := range args {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(psSingle(a))
		}
	}
	b.WriteString(`"`)
	return b.String()
}

// psSingle 用单引号包裹字符串，内部单引号写成两个。
func psSingle(s string) string {
	return "'" + strings.ReplaceAll(s, `'`, `''`) + "'"
}

// cmdStartHangError 返回挂住说明，文案是 cmdStartHangMsg。
func cmdStartHangError() error {
	return fmt.Errorf("%s", cmdStartHangMsg)
}
