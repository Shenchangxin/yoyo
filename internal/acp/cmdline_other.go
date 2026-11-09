//go:build !windows

// cmdline_other.go 是非 Windows 平台的空实现：这些平台按参数数组启动，没有原样命令行。

package acp

import "os/exec"

// SetCmdLine 在非 Windows 平台上什么都不做。
func SetCmdLine(cmd *exec.Cmd, line string) {}
