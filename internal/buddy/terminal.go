// terminal.go 拥有各平台终端共用的小工具：POSIX shell 的引号、把启动方式拼成一行，以及要不要把某个变量写进脚本。
// 真正打开终端在 terminal_windows.go、terminal_darwin.go、terminal_linux.go 和 terminal_other.go。

package buddy

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// shQuote 给 POSIX shell 加单引号。
func shQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n'\"\\$`!*?[]{}()<>|&;#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shJoin 把命令和参数拼成一行。已经是 sh -c 形式的直接取那一行。
func shJoin(l acp.Launch) string {
	if l.Command == "/bin/sh" && len(l.Args) == 2 && l.Args[0] == "-c" {
		return l.Args[1]
	}
	parts := []string{shQuote(l.Command)}
	for _, a := range l.Args {
		parts = append(parts, shQuote(a))
	}
	return strings.Join(parts, " ")
}

// scriptEnv 挑出可以写进终端脚本的变量：PATH 和不像密钥的变量。密钥绝不落盘。
func scriptEnv(env []string) []string {
	var out []string
	for _, kv := range env {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		name := kv[:i]
		if SecretLike(name) || !ValidEnvName(name) || strings.EqualFold(name, "TERM") || strings.EqualFold(name, "NO_COLOR") {
			continue
		}
		if strings.EqualFold(name, "PATH") || !envPassed(name) {
			out = append(out, "export "+name+"="+shQuote(kv[i+1:]))
		}
	}
	return out
}
