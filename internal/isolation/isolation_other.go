//go:build !windows && !darwin && !linux

// 本文件拥有 Windows、Darwin、Linux 以外 GOOS 的空后端。
// probe 的 Sandbox 固定为 false，Available 固定为 false，不改写命令。
// 同包的 Status、WaitNotify、SigningStatus 在这些平台上进到这里。

package isolation

import "os/exec"

// probe 报告 Kind 为 none，Sandbox 为 false，Available 为 false，说明当前 GOOS 没有隔离后端。
func probe() Report {
	return Report{Kind: "none", Sandbox: false, Available: false, Note: "no OS isolation backend on this GOOS"}
}

// assign 在没有后端的平台上留空。
func assign(cmd *exec.Cmd) {}

// release 在没有后端的平台上留空，kill 也不会拆树。
func release(cmd *exec.Cmd, kill bool) {}

// apply 在没有后端的平台上不改命令。
func apply(cmd *exec.Cmd) {}

// signingProbe 在没有后端的平台上固定返回 unsigned。
func signingProbe() string { return "unsigned" }
