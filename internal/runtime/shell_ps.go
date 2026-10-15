package runtime

import (
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/pwsh"
)

func exeBase(p string) string {
	name := strings.ToLower(filepath.Base(p))
	return strings.TrimSuffix(name, ".exe")
}

func isPowerShellExe(name string) bool {
	return name == "powershell" || name == "pwsh"
}

func skipPowerShellFlag(argv []string, i int) (next int, skip bool) {
	a := strings.ToLower(argv[i])
	switch a {
	case "-noprofile", "-nologo", "-noninteractive", "-nop", "-nol":
		return i + 1, true
	case "-windowstyle", "-executionpolicy", "-ep":
		if i+1 < len(argv) {
			return i + 2, true
		}
		return i + 1, true
	}
	return i, false
}

// parsePowerShellInvocation unwraps powershell/pwsh wrappers so the host can
// run a body as -File. Encoded commands are passed through unchanged.
func parsePowerShellInvocation(command string) (file string, extra []string, body string, encoded []string) {
	argv := SplitShellArgv(command)
	if len(argv) == 0 {
		return "", nil, command, nil
	}
	if !isPowerShellExe(exeBase(argv[0])) {
		return "", nil, command, nil
	}
	rest := argv[1:]
	for i := 0; i < len(rest); {
		next, skip := skipPowerShellFlag(rest, i)
		if skip {
			i = next
			continue
		}
		a := strings.ToLower(rest[i])
		switch {
		case a == "-encodedcommand" || a == "-enc" || a == "-e":
			return "", nil, "", rest[i:]
		case a == "-file" || a == "-f":
			if i+1 < len(rest) {
				return rest[i+1], rest[i+2:], "", nil
			}
			return "", nil, command, nil
		case a == "-command" || a == "-c" || a == "/command" || a == "/c":
			return "", nil, strings.TrimSpace(strings.Join(rest[i+1:], " ")), nil
		case strings.HasSuffix(a, ".ps1"):
			return rest[i], rest[i+1:], "", nil
		default:
			return "", nil, strings.TrimSpace(strings.Join(rest[i:], " ")), nil
		}
	}
	return "", nil, command, nil
}

func materializePowerShell(exe, command, workspace string) (string, []string, func(), error) {
	nop := func() {}
	if strings.TrimSpace(exe) == "" {
		exe = pwsh.Fallback()
	}
	flags := []string{"-NoProfile", "-NoLogo", "-NonInteractive", "-ExecutionPolicy", "Bypass"}
	file, extra, body, encoded := parsePowerShellInvocation(command)
	if len(encoded) > 0 {
		return exe, append(append([]string{}, flags...), encoded...), nop, nil
	}
	if file != "" {
		args := append(append([]string{}, flags...), "-File", file)
		args = append(args, extra...)
		return exe, args, nop, nil
	}
	if strings.TrimSpace(body) == "" {
		body = command
	}
	path, done, err := writeInlineScript(workspace, "\ufeff"+body+"\n", ".ps1")
	if err != nil {
		return "", nil, nop, err
	}
	return exe, append(flags, "-File", path), done, nil
}
