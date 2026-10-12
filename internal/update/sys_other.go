//go:build !windows

package update

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
