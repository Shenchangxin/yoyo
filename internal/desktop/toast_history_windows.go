//go:build windows

package desktop

import (
	"os/exec"
	"syscall"
)

// createNoWindow stops the PowerShell child from flashing a console window.
const createNoWindow = 0x08000000

// clearToastHistory runs the clear in Windows PowerShell 5.1, which ships with Windows and
// projects the WinRT History API. wintoast's own PowerShell fallback already relies on it.
func clearToastHistory(appID string) error {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", clearToastHistoryScript(appID))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Run()
}
