// spin.go 认出没有副作用的打印命令。OpenCode 把报告工具藏进 Code Mode 时，较弱的模型会用
// shell 里的 console.log 假装调用，而且每次都成功，静默计时就被一直清掉，这一轮不会自己结束。

package acpworker

import (
	"regexp"
	"strings"
)

// probeStreakLimit 是连续多少次纯打印之后认定这一轮在空转。中间夹一次改文件或真正的命令就重计。
const probeStreakLimit = 6

var (
	reNodeEval = regexp.MustCompile(`(?i)^(?:node|nodejs)(?:\.exe)?\s+(?:--[a-z0-9-]+\s+)*-e\s+([\s\S]+)$`)
	rePyEval   = regexp.MustCompile(`(?i)^(?:python|python3|py)(?:\.exe)?\s+-c\s+([\s\S]+)$`)
	reEcho     = regexp.MustCompile(`(?i)^(?:echo|printf|write-output|write-host)\s+(\S[\s\S]*)$`)
	rePrintJS  = regexp.MustCompile(`^(?:console\.(?:log|info|debug|warn)|process\.stdout\.write)\(\s*(?:'(?:\\'|[^'])*'|"(?:\\"|[^"])*"|` + "`[^`]*`" + `)\s*\)\s*;?$`)
	rePrintPy  = regexp.MustCompile(`^print\(\s*(?:'(?:\\'|[^'])*'|"(?:\\"|[^"])*")\s*\)\s*$`)
)

// isPrintProbe 在 command 只是把一段短字打出来、不写文件也不跑别的程序时为真。
func isPrintProbe(command string) bool {
	s := strings.TrimSpace(command)
	if s == "" || strings.ContainsAny(s, "><|&") {
		return false
	}
	if rePrintJS.MatchString(s) {
		return true
	}
	if m := reNodeEval.FindStringSubmatch(s); m != nil {
		return rePrintJS.MatchString(strings.TrimSpace(unquoteShell(m[1])))
	}
	if m := rePyEval.FindStringSubmatch(s); m != nil {
		return rePrintPy.MatchString(strings.TrimSpace(unquoteShell(m[1])))
	}
	if m := reEcho.FindStringSubmatch(s); m != nil {
		msg := strings.TrimSpace(m[1])
		return len([]rune(msg)) <= 200 && !strings.Contains(msg, "\n")
	}
	return false
}

// unquoteShell 去掉包住整段脚本的一层引号。
func unquoteShell(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
