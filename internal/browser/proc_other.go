//go:build !windows

// 本文件拥有 Windows 以外平台的空进程准备。不建作业，也不改启动标志。
// startChromeCDP 在这些平台上仍会调用 prepareChromeCmd 和 bindChromeJob。

package browser

import "os/exec"

// prepareChromeCmd 在非 Windows 上不改命令。
// cmd 没有被读写。
// headed 没有被读写。
func prepareChromeCmd(cmd *exec.Cmd, headed bool) {}

// bindChromeJob 在非 Windows 上返回 nil，调用方不会拆进程树。
// cmd 没有被读写。
func bindChromeJob(cmd *exec.Cmd) func() {
	return nil
}
