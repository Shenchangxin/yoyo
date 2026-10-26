//go:build windows

// 本文件在 Windows 上处理控制台附着。双击控制台子系统程序时系统会附上控制台；生产构建用 -H windowsgui 链接，没有控制台。
// 从已有终端启动时与父进程共用控制台。YOYO_CONSOLE=1 保留私有控制台。

package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// init 在进程启动时调用 hideOwnConsole。
func init() {
	hideOwnConsole()
}

// hideOwnConsole 在 YOYO_CONSOLE=1 时直接返回，保留私有控制台。控制台进程数不是 1 时也返回，从已有终端启动时与父进程共用控制台，保持附着。只有本进程占用该控制台时调用 FreeConsole。
func hideOwnConsole() {
	if os.Getenv("YOYO_CONSOLE") == "1" {
		return
	}
	if consoleProcessCount() != 1 {
		return
	}
	_, _, _ = procFreeConsole.Call()
}

var (
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	procFreeConsole           = kernel32.NewProc("FreeConsole")
)

// consoleProcessCount 调用 GetConsoleProcessList，缓冲区只有两个 pid，返回附着在当前控制台的进程数。
func consoleProcessCount() uint32 {
	var pids [2]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return uint32(n)
}
