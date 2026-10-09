//go:build !windows

package browser

import "os/exec"

func prepareChromeCmd(cmd *exec.Cmd, headed bool) {}

func bindChromeJob(cmd *exec.Cmd) func() {
	return nil
}
