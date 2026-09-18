//go:build linux

package isolation

import (
	"os"
	"os/exec"
)

func probe() Report {
	if _, err := exec.LookPath("bwrap"); err == nil {
		return Report{
			Kind:      "bubblewrap",
			Sandbox:   true,
			Available: true,
			Note:      "bwrap --die-with-parent around shell. network unshared unless YOYO_SANDBOX_NET=1.",
		}
	}
	return Report{Kind: "none", Sandbox: false, Available: false, Note: "bubblewrap not on PATH; policy jail only"}
}

func assign(cmd *exec.Cmd) {}

func release(cmd *exec.Cmd) {}

func apply(cmd *exec.Cmd) {
	bin, err := exec.LookPath("bwrap")
	if err != nil {
		return
	}
	inner := append([]string(nil), cmd.Args...)
	if len(inner) == 0 {
		inner = []string{cmd.Path}
	}
	args := []string{"--die-with-parent", "--unshare-pid"}
	if os.Getenv("YOYO_SANDBOX_NET") != "1" {
		args = append(args, "--unshare-net")
	}
	if cmd.Dir != "" {
		args = append(args, "--bind", cmd.Dir, cmd.Dir, "--chdir", cmd.Dir)
	}
	args = append(args, "--ro-bind", "/usr", "/usr", "--ro-bind", "/bin", "/bin", "--ro-bind", "/lib", "/lib")
	if _, err := os.Stat("/lib64"); err == nil {
		args = append(args, "--ro-bind", "/lib64", "/lib64")
	}
	args = append(args, "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp")
	args = append(args, inner...)
	cmd.Path = bin
	cmd.Args = append([]string{"bwrap"}, args...)
}

func signingProbe() string {
	return "unsigned"
}
