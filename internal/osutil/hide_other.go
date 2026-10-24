//go:build !windows

package osutil

import "os/exec"

func HideConsole(cmd *exec.Cmd) {}

func NoConsole(cmd *exec.Cmd) {}
