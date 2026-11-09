//go:build !windows

// shim_other.go 是非 Windows 平台的命令改写：没有 .cmd 垫片，按原样启动。

package buddy

import "github.com/Shenchangxin/yoyo/internal/acp"

// commandFor 原样返回可执行文件和参数。
func commandFor(path string, args []string, node func() (string, error)) acp.Launch {
	return acp.Launch{Command: path, Args: args}
}
