//go:build !windows

// 非 Windows 上 NSIS 提权不可用，对应函数返回 windows-only 或什么也不做。

package update

import (
	"fmt"
	"os/exec"
	"time"
)

// hideWindow 在非 Windows 上什么也不做。
// cmd 本函数不读取。
func hideWindow(cmd *exec.Cmd) {}

// startElevatedApply 返回 "update: nsis apply is windows-only"。
// exe 本函数不读取。
// spec 本函数不读取。
func startElevatedApply(exe, spec string) error {
	return fmt.Errorf("update: nsis apply is windows-only")
}

// waitForPID 在非 Windows 上什么也不做。
// pid 赋给空白标识符，不另作使用。
// timeout 赋给空白标识符，不另作使用。
func waitForPID(pid int, timeout time.Duration) {
	_ = pid
	_ = timeout
}

// runNSISInstaller 返回 "update: nsis apply is windows-only"。
// installer 本函数不读取。
// destDir 本函数不读取。
func runNSISInstaller(installer, destDir string) error {
	return fmt.Errorf("update: nsis apply is windows-only")
}
