//go:build linux

// 本文件拥有 Linux 上用 bwrap 改写 shell 命令的逻辑，以及一份固定返回 unsigned 的签名探针。
// 不拥有 Job Object；PATH 上没有 bwrap 时命令保持原样。
// 同包的 Status、WaitNotify 和 SigningStatus 会进到这里。

package isolation

import (
	"os"
	"os/exec"
)

// probe 用 exec.LookPath("bwrap") 决定报告。查找没有错误时 Kind 为 bubblewrap，Sandbox 为 true，Available 为 true。
// 查找有错误时 Kind 为 none，Sandbox 为 false，Available 为 false。Sandbox 为 true 只对应 bwrap 出现在 PATH 上。
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

// assign 在 Linux 上不记录进程，命令是否被 bwrap 包住只由 apply 决定。
func assign(cmd *exec.Cmd) {}

// release 不结束进程。取消时由 WaitNotify 杀掉当前进程。
func release(cmd *exec.Cmd, kill bool) {}

// apply 在 PATH 上找不到 bwrap 时不改命令。找到时把原参数交给 bwrap，并带上 --die-with-parent 和 --unshare-pid。
// 环境变量 YOYO_SANDBOX_NET 不是 1 时追加 --unshare-net。cmd.Dir 非空时 --bind 该目录并 --chdir。
// /usr、/bin、/lib 以只读绑定挂上；/lib64 存在时同样只读绑定。最后挂上 /proc、/dev 和 tmpfs 的 /tmp。
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

// signingProbe 在 Linux 上固定返回 unsigned，不读可执行文件的签名。
func signingProbe() string {
	return "unsigned"
}
