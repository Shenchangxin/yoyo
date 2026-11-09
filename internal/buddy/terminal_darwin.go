//go:build darwin

// terminal_darwin.go 拥有 macOS 上的可见终端：写一个临时 .command 脚本，用 open -a Terminal 打开。拿不到结束信号，调用方轮询。
// 脚本里只写 PATH 和不像密钥的变量，密钥不落盘。

package buddy

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// openTerminal 在 Terminal.app 里跑 l。
func openTerminal(ctx context.Context, l acp.Launch) (TerminalResult, error) {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	if l.Dir != "" {
		b.WriteString("cd " + shQuote(l.Dir) + " 2>/dev/null\n")
	}
	for _, line := range scriptEnv(l.Env) {
		b.WriteString(line + "\n")
	}
	b.WriteString(shJoin(l) + "\n")
	b.WriteString("echo\nprintf '%s' 'Press Return to close this window. '\nread -r _\n")
	f, err := os.CreateTemp("", "yoyo-*.command")
	if err != nil {
		return TerminalResult{}, err
	}
	name := f.Name()
	_, _ = f.WriteString(b.String())
	_ = f.Close()
	if err := os.Chmod(name, 0o700); err != nil {
		return TerminalResult{}, err
	}
	if err := exec.CommandContext(ctx, "open", "-a", "Terminal", name).Run(); err != nil {
		_ = os.Remove(name)
		return TerminalResult{Copy: shJoin(l)}, nil
	}
	time.AfterFunc(15*time.Minute, func() { _ = os.Remove(name) })
	return TerminalResult{}, nil
}
