// 拆开 powershell 和 pwsh 的包装，把命令正文写成工作区里的 .ps1，再以 -File 运行。
// 不执行进程，那在 tools.go 的 bindShell。不调用 WorkspaceTools.resolve。本文件没有 Windows Job Object。

package runtime

import (
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/pwsh"
)

// exeBase 取路径的文件名，去掉 .exe，并转成小写。
func exeBase(p string) string {
	name := strings.ToLower(filepath.Base(p))
	return strings.TrimSuffix(name, ".exe")
}

// isPowerShellExe 判断名字是 powershell 还是 pwsh。
func isPowerShellExe(name string) bool {
	return name == "powershell" || name == "pwsh"
}

// skipPowerShellFlag 跳过 -NoProfile、-WindowStyle 和 -ExecutionPolicy 这类启动开关。带值的开关连同下一个参数一起跳过。
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

// parsePowerShellInvocation 拆开 powershell 或 pwsh 包装，供宿主把正文写成 -File。
// 编码命令原样放进 encoded，不解码。已经是 -File 或 .ps1 时返回文件和其余参数。命令本身不是 PowerShell 时，正文就是原命令。
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

// materializePowerShell 返回要执行的程序和参数。exe 为空时用 pwsh.Fallback。
// 编码命令原样接在 -NoProfile -NoLogo -NonInteractive -ExecutionPolicy Bypass 后面。已有脚本则加 -File。正文则写成带 BOM 的 .ps1 再 -File。
// -ExecutionPolicy Bypass 只是传给 PowerShell 的参数，不是操作系统沙箱。
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
