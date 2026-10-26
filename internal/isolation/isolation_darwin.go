//go:build darwin

// 本文件拥有 Darwin 上用 sandbox-exec 改写 shell 命令的逻辑，以及 codesign 签名探针。
// 不拥有 Job Object；PATH 上没有 sandbox-exec 时命令保持原样。
// 同包的 Status、WaitNotify、SigningStatus 会进到这些函数。

package isolation

import (
	"os"
	"os/exec"
	"strings"
)

// probe 用 exec.LookPath("sandbox-exec") 决定报告。查找没有错误时 Kind 为 sandbox_exec，Sandbox 为 true，Available 为 true。
// 查找有错误时 Kind 为 none，Sandbox 为 false，Available 为 false。Sandbox 为 true 只对应 sandbox-exec 出现在 PATH 上。
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

// assign 在 Darwin 上不记录进程，配置是否套上只由 apply 决定。
// cmd 本函数不读取。
func assign(cmd *exec.Cmd) {}

// release 不结束进程。取消时由 WaitNotify 杀掉当前的 sandbox-exec 进程。
// cmd 本函数不读取。
// kill 本函数不读取。
func release(cmd *exec.Cmd, kill bool) {}

// apply 在 PATH 上找不到 sandbox-exec 时不改命令。找到时用 sandbox-exec -p 套上配置再执行原参数。
// 配置默认拒绝，允许 process-exec、process-fork、signal、sysctl-read、mach-lookup 和 file-read*。
// file-write* 只允许 /private/tmp、/tmp、/var/folders，以及非空的 cmd.Dir。仅当 YOYO_SANDBOX_NET 为 1 时追加 network*。
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

// signingProbe 运行 codesign -dv --verbose=2 查看当前可执行文件。
// 命令失败且输出为空时返回 unsigned。输出含 Signature=adhoc 或 flags=0x2(adhoc) 时返回 adhoc。
// 含 Authority= 或 TeamIdentifier= 时返回 codesign；命令本身成功也返回 codesign。其余返回 unsigned。
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
