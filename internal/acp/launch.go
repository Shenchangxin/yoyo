// launch.go 拥有启动一个 agent 进程所需的参数包。Buddy 的启动器产出它，探测、登录和工人拿它去 Dial。
// 不负责找命令、拼环境；那些在 internal/buddy。

package acp

import (
	"os"
	"strings"
)

// Launch 是启动一个 agent 进程需要的全部东西。Env 是完整环境，不再继承当前进程。
type Launch struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"-"`
	Dir     string   `json:"dir,omitempty"`
	// ViaCmd 只在 Windows 上用：为真时 Command 是一个 .cmd 或 .bat，经 cmd.exe /d /s /c 启动。
	ViaCmd bool `json:"viaCmd,omitempty"`
}

// Options 把启动参数转成 Dial 的选项。Handler、Version 等由调用方再填。
func (l Launch) Options() Options {
	opts := Options{Command: l.Command, Args: append([]string(nil), l.Args...), Env: l.Env, Dir: l.Dir}
	if l.ViaCmd {
		comspec := firstEnv(l.Env, "ComSpec")
		if comspec == "" {
			comspec = os.Getenv("ComSpec")
		}
		if comspec == "" {
			comspec = "cmd.exe"
		}
		opts.Command = comspec
		opts.Args = append([]string{"/d", "/s", "/c", l.Command}, l.Args...)
		opts.CmdLine = CmdLineViaCmd(comspec, l.Command, l.Args)
	}
	return opts
}

// WithArgs 返回在参数末尾追加 extra 的副本，登录时加上认证方法自带的参数用它。
func (l Launch) WithArgs(extra ...string) Launch {
	out := l
	out.Args = append(append([]string(nil), l.Args...), extra...)
	return out
}

// CmdLineViaCmd 拼出 cmd.exe /d /s /c "<target> <args>" 的原样命令行。/s 让 cmd 只去掉最外层那对引号。
func CmdLineViaCmd(comspec, target string, args []string) string {
	parts := []string{QuoteWindowsArg(target)}
	for _, a := range args {
		parts = append(parts, QuoteWindowsArg(a))
	}
	return QuoteWindowsArg(comspec) + ` /d /s /c "` + strings.Join(parts, " ") + `"`
}

// firstEnv 在 KEY=VALUE 列表里按不分大小写的名字取值。
func firstEnv(env []string, name string) string {
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i > 0 && strings.EqualFold(kv[:i], name) {
			return kv[i+1:]
		}
	}
	return ""
}

// QuoteWindowsArg 按 CommandLineToArgvW 的规则给一个参数加引号；不含空白和引号时原样返回。
func QuoteWindowsArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s
	}
	out := []byte{'"'}
	slashes := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			slashes++
		case '"':
			for ; slashes > 0; slashes-- {
				out = append(out, '\\')
			}
			out = append(out, '\\')
		default:
			slashes = 0
		}
		out = append(out, s[i])
	}
	for ; slashes > 0; slashes-- {
		out = append(out, '\\')
	}
	return string(append(out, '"'))
}
