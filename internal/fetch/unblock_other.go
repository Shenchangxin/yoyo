//go:build !windows

// unblock_other.go 拥有 macOS 上去掉隔离标记；其他系统什么都不做。
// buddy 安装完成后调用 Unblock。

package fetch

import (
	"os/exec"
	"runtime"
)

// Unblock 在 macOS 上对 dir 尽力执行 xattr -dr com.apple.quarantine。失败不报错。
func Unblock(dir string) {
	if runtime.GOOS != "darwin" {
		return
	}
	_ = exec.Command("xattr", "-dr", "com.apple.quarantine", dir).Run()
}
