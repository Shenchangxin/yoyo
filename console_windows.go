//go:build windows

package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func init() {
	hideOwnConsole()
}

// hideOwnConsole closes the console Windows attaches when a console-subsystem
// binary is double-clicked. Production builds are linked with -H windowsgui
// and have no console, so this does nothing. A dev launch from an existing
// terminal shares that console with the parent and stays attached.
// YOYO_CONSOLE=1 keeps a private console.
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

func consoleProcessCount() uint32 {
	var pids [2]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return uint32(n)
}
