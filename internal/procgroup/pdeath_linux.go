//go:build linux

// pdeath_linux.go 拥有 Linux 上的父进程死亡信号：yoyo 意外退出时，子进程收到 SIGKILL。

package procgroup

import "syscall"

// setDeathSignal 设置 Pdeathsig 为 SIGKILL。
func setDeathSignal(attr *syscall.SysProcAttr) {
	attr.Pdeathsig = syscall.SIGKILL
}
