//go:build !windows

// 本文件在 Windows 以外的系统上提供 HideConsole 和 NoConsole。
// 两个函数都不修改 cmd。

package osutil

import "os/exec"

// HideConsole 留空，不修改 cmd。
func HideConsole(cmd *exec.Cmd) {}

// NoConsole 留空，不修改 cmd。
func NoConsole(cmd *exec.Cmd) {}
