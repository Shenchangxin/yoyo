//go:build !windows && !linux

// pdeath_other.go 拥有非 Linux 的 Unix 上的空实现：没有 Pdeathsig，只靠 Kill 结束整组。

package procgroup

import "syscall"

// setDeathSignal 什么都不做。
func setDeathSignal(*syscall.SysProcAttr) {}
