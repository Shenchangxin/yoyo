//go:build !windows && !darwin && !linux

// terminal_other.go 是没有已知终端的平台：把命令交给界面让人复制运行。

package buddy

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// openTerminal 不开终端，只返回要复制的命令。
func openTerminal(ctx context.Context, l acp.Launch) (TerminalResult, error) {
	return TerminalResult{Copy: shJoin(l)}, nil
}
