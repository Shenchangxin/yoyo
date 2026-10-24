//go:build windows

package osutil

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// HideConsole keeps a console child (cmd, powershell, git, ffmpeg, …) from
// opening a visible console when the desktop app starts it. Pipes still work.
// Do not use this for a GUI the user must see: HideWindow asks that process
// to start hidden. Use NoConsole for those.
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

// NoConsole stops Windows from allocating a console for the child without
// asking a GUI process to hide its window. CREATE_NO_WINDOW is ignored for
// GUI binaries, so relaunching Yoyo.exe stays visible.
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
