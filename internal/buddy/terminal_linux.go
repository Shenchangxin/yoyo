//go:build linux

// terminal_linux.go 拥有 Linux 上的可见终端：依次试 x-terminal-emulator、gnome-terminal、konsole、xterm；都没有时把命令交给界面让人复制。

package buddy

import (
	"context"
	"os"
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// openTerminal 在找到的第一个终端里跑 l。
func openTerminal(ctx context.Context, l acp.Launch) (TerminalResult, error) {
	line := shJoin(l) + "; echo; printf '%s' 'Press Return to close this window. '; read -r _"
	candidates := [][]string{
		{"x-terminal-emulator", "-e", "sh", "-c", line},
		{"gnome-terminal", "--wait", "--", "sh", "-c", line},
		{"konsole", "-e", "sh", "-c", line},
		{"xterm", "-e", "sh", "-c", line},
	}
	for _, c := range candidates {
		path, err := exec.LookPath(c[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, c[1:]...)
		cmd.Env = l.Env
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		cmd.Dir = l.Dir
		if err := cmd.Start(); err != nil {
			continue
		}
		go func() { _ = cmd.Wait() }()
		return TerminalResult{}, nil
	}
	return TerminalResult{Copy: shJoin(l)}, nil
}
