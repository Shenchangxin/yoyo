//go:build windows

// cmdline_windows.go 拥有 Windows 上的原样命令行：经 cmd.exe 启动 npm 垫片时，参数不能再被 Go 转义一遍。

package acp

import (
	"os/exec"
	"syscall"
)

// SetCmdLine 在 line 非空时把它设为 cmd 的原样命令行，保留已有的进程属性。
func SetCmdLine(cmd *exec.Cmd, line string) {
	if line == "" {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CmdLine = line
}
