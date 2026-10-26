//go:build windows

// 本文件拥有 Windows 上 HideConsole 和 NoConsole。
// HideConsole 设置 HideWindow，并在 CreationFlags 上或上 CREATE_NO_WINDOW。
// NoConsole 只或上 CREATE_NO_WINDOW，不设置 HideWindow。非 Windows 的空函数在 hide_other.go。

package osutil

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// HideConsole 让桌面程序启动的控制台子进程不弹出可见控制台。管道仍然可用。
// cmd 为 nil 时直接返回。已有的 SysProcAttr 保留。HideWindow 设为真，CreationFlags 或上 CREATE_NO_WINDOW，原有标志不清掉。
// 用户必须看见的 GUI 不要走这里：HideWindow 会要求该进程以隐藏方式启动。那种进程用 NoConsole。
func HideConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	attr := cmd.SysProcAttr
	if attr == nil {
		attr = &syscall.SysProcAttr{}
		cmd.SysProcAttr = attr
	}
	attr.HideWindow = true
	attr.CreationFlags |= windows.CREATE_NO_WINDOW
}

// NoConsole 只避免 Windows 为子进程分配控制台，不要求 GUI 隐藏窗口。
// cmd 为 nil 时直接返回。已有的 SysProcAttr 保留。只把 CreationFlags 或上 CREATE_NO_WINDOW，不改 HideWindow。
// CREATE_NO_WINDOW 对 GUI 程序无效，所以重新启动 Yoyo.exe 仍然可见。
func NoConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	attr := cmd.SysProcAttr
	if attr == nil {
		attr = &syscall.SysProcAttr{}
		cmd.SysProcAttr = attr
	}
	attr.CreationFlags |= windows.CREATE_NO_WINDOW
}
