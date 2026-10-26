//go:build windows

// 本文件锁住 Windows 上 HideConsole 会保留已有 CreationFlags，并同时打开 HideWindow 与 CREATE_NO_WINDOW。
// NoConsole 只或上 CREATE_NO_WINDOW，不把 HideWindow 设为真。由 go test 调用。

package osutil

import (
	"os/exec"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

// TestHideConsolePreservesExistingFlags 要求 HideWindow 为真，CreationFlags 同时含 CREATE_NO_WINDOW 和原先的 CREATE_NEW_PROCESS_GROUP。
func TestHideConsolePreservesExistingFlags(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "echo hi")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
	HideConsole(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("HideWindow not set")
	}
	flags := cmd.SysProcAttr.CreationFlags
	if flags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatal("CREATE_NO_WINDOW not set")
	}
	if flags&windows.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Fatal("existing creation flags were cleared")
	}
}

// TestNoConsoleDoesNotHideGUI 要求 NoConsole 之后 HideWindow 仍为假，并且 CreationFlags 含 CREATE_NO_WINDOW。
func TestNoConsoleDoesNotHideGUI(t *testing.T) {
	cmd := exec.Command("Yoyo.exe")
	NoConsole(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr not set")
	}
	if cmd.SysProcAttr.HideWindow {
		t.Fatal("NoConsole must not hide a GUI window")
	}
	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatal("CREATE_NO_WINDOW not set")
	}
}

// TestConsoleHelpersNil 对两个函数传入 nil，确认不会崩。
func TestConsoleHelpersNil(t *testing.T) {
	HideConsole(nil)
	NoConsole(nil)
}
