//go:build !windows && !darwin && !linux

package isolation

import "os/exec"

func probe() Report {
	return Report{Kind: "none", Sandbox: false, Available: false, Note: "no OS isolation backend on this GOOS"}
}

func assign(cmd *exec.Cmd) {}

func release(cmd *exec.Cmd) {}

func apply(cmd *exec.Cmd) {}

func signingProbe() string { return "unsigned" }
