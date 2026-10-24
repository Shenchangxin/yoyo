//go:build windows

package osutil

import (
	"os/exec"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

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

func TestConsoleHelpersNil(t *testing.T) {
	HideConsole(nil)
	NoConsole(nil)
}
