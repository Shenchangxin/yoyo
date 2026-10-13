//go:build !windows

package update

import (
	"fmt"
	"os/exec"
	"time"
)

func hideWindow(cmd *exec.Cmd) {}

func startElevatedApply(exe, spec string) error {
	return fmt.Errorf("update: nsis apply is windows-only")
}

func waitForPID(pid int, timeout time.Duration) {
	_ = pid
	_ = timeout
}

func runNSISInstaller(installer, destDir string) error {
	return fmt.Errorf("update: nsis apply is windows-only")
}
