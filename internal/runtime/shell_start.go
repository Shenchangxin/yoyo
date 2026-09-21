package runtime

import (
	"fmt"
	"regexp"
	"strings"
)

// cmd `start` with captured stdout never reaches EOF: the child keeps the
// pipe handle. That is a process-lifetime bug, not something to teach in
// the system prompt. Detach the child so the shell can finish.

const cmdStartHangMsg = "cmd start waits for the child while stdout is captured, and piping it to head/tail never reaches EOF — the tool hangs until timeout. Use Start-Process -WindowStyle Hidden (no pipe to head/tail). Then sleep and Get-Process. Inside bash double quotes write PowerShell $ as \\$ so bash does not expand it."

var (
	reCmdStartHead = regexp.MustCompile(`(?i)\bcmd(?:\.exe)?\s+/{1,2}c\s+`)
	reBareStart    = regexp.MustCompile(`(?i)(?:^|[;&\n]|&&|\|\|)\s*start\s+(?:/|"|')`)
	reHeadPipe     = regexp.MustCompile(`(?i)\s*(?:2>&1\s*)?(?:\|\s*(?:head|tail)\b[^;|&]*)`)
)

func cmdStartWouldHang(command string) bool {
	return cmdStartPattern(command)
}

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

func startPayloadHangs(inner string) bool {
	low := strings.ToLower(strings.TrimSpace(inner))
	if strings.HasPrefix(low, "start-process") {
		return false
	}
	return strings.HasPrefix(low, "start ") || strings.HasPrefix(low, "start\"") || low == "start"
}

// rewriteCmdStart replaces `cmd /c start ...` (and a trailing 2>&1 | head/tail)
// with PowerShell Start-Process, which returns while stdout is captured.
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
	if file == "" || wait {
		return command, false
	}
	after := rest[consumed:]
	if m := reHeadPipe.FindStringIndex(after); m != nil && m[0] == 0 {
		after = after[m[1]:]
	}
	return strings.TrimSpace(prefix + startProcessCommand(file, args) + after), true
}

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
			if i+1 < len(rest) && rest[i] == '&' && rest[i+1] == '&' {
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

func psSingle(s string) string {
	return "'" + strings.ReplaceAll(s, `'`, `''`) + "'"
}

func cmdStartHangError() error {
	return fmt.Errorf("%s", cmdStartHangMsg)
}
