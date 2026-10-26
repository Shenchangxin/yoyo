//go:build windows

package desktop

import (
	"os/exec"
	"syscall"
)

// createNoWindow 是 0x08000000，用来不让 PowerShell 子进程闪出控制台。
const createNoWindow = 0x08000000

// clearToastHistory 用 Windows PowerShell 5.1 执行清空脚本。进程带 HideWindow 和 createNoWindow。
// appID 传给 clearToastHistoryScript。
func clearToastHistory(appID string) error {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", clearToastHistoryScript(appID))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Run()
}
