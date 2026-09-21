//go:build darwin

package isolation

import (
	"os"
	"os/exec"
	"strings"
)

func probe() Report {
	if _, err := exec.LookPath("sandbox-exec"); err == nil {
		return Report{
			Kind:      "sandbox_exec",
			Sandbox:   true,
			Available: true,
			Note:      "sandbox-exec profile: process + workspace writes; network denied unless YOYO_SANDBOX_NET=1.",
		}
	}
	return Report{Kind: "none", Sandbox: false, Available: false, Note: "sandbox-exec not on PATH"}
}

func assign(cmd *exec.Cmd) {}

func release(cmd *exec.Cmd, kill bool) {}

func apply(cmd *exec.Cmd) {
	bin, err := exec.LookPath("sandbox-exec")
	if err != nil {
		return
	}
	innerPath := cmd.Path
	innerArgs := append([]string(nil), cmd.Args...)
	if innerPath == "" && len(innerArgs) > 0 {
		innerPath = innerArgs[0]
	}
	if len(innerArgs) == 0 {
		innerArgs = []string{innerPath}
	}
	profile := `(version 1)
(deny default)
(allow process-exec)
(allow process-fork)
(allow signal)
(allow sysctl-read)
(allow mach-lookup)
(allow file-read*)
(allow file-write* (subpath "/private/tmp") (subpath "/tmp") (subpath "/var/folders"))`
	if cmd.Dir != "" {
		profile += "\n(allow file-write* (subpath \"" + cmd.Dir + "\"))"
	}
	if os.Getenv("YOYO_SANDBOX_NET") == "1" {
		profile += "\n(allow network*)"
	}
	cmd.Path = bin
	cmd.Args = append([]string{"sandbox-exec", "-p", profile}, innerArgs...)
}

func signingProbe() string {
	exe, err := os.Executable()
	if err != nil {
		return "unsigned"
	}
	out, err := exec.Command("codesign", "-dv", "--verbose=2", exe).CombinedOutput()
	blob := string(out)
	if err != nil && blob == "" {
		return "unsigned"
	}
	if strings.Contains(blob, "Signature=adhoc") || strings.Contains(blob, "flags=0x2(adhoc)") {
		return "adhoc"
	}
	if strings.Contains(blob, "Authority=") || strings.Contains(blob, "TeamIdentifier=") {
		return "codesign"
	}
	if err == nil {
		return "codesign"
	}
	return "unsigned"
}
