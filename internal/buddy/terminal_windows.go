//go:build windows

// terminal_windows.go 拥有 Windows 上的可见终端：用 CREATE_NEW_CONSOLE 开一个 cmd 窗口跑命令，结束后停在 pause，等它退出。

package buddy

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/procgroup"
)

// openTerminal 开一个新控制台跑 l，等窗口关掉。
func openTerminal(ctx context.Context, l acp.Launch) (TerminalResult, error) {
	comspec := os.Getenv("ComSpec")
	if comspec == "" {
		comspec = "cmd.exe"
	}
	line := acp.QuoteWindowsArg(comspec) + ` /d /s /c "` + terminalInner(l) + ` & echo. & pause"`
	cmd := exec.Command(comspec)
	cmd.Env = l.Env
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Dir = l.Dir
	acp.SetCmdLine(cmd, line)
	p, err := procgroup.Start(cmd, procgroup.Options{NewConsole: true})
	if err != nil {
		return TerminalResult{}, err
	}
	select {
	case <-p.Done():
	case <-ctx.Done():
		_ = p.Kill()
		return TerminalResult{}, ctx.Err()
	}
	return TerminalResult{Waited: true}, nil
}

// terminalInner 是在 cmd 里要跑的那一段。已经是 cmd /d /s /c 形式的直接取那一行。
func terminalInner(l acp.Launch) string {
	if len(l.Args) == 4 && strings.EqualFold(l.Args[0], "/d") && strings.EqualFold(l.Args[2], "/c") {
		return l.Args[3]
	}
	parts := []string{acp.QuoteWindowsArg(l.Command)}
	for _, a := range l.Args {
		parts = append(parts, acp.QuoteWindowsArg(a))
	}
	return strings.Join(parts, " ")
}
