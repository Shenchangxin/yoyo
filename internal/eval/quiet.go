// 给评测拉起的外部命令藏控制台窗口。不决定验证是否通过。harbor.go 和 docker.go 调用。

package eval

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// quiet 调用 osutil.HideConsole，并返回同一个命令。
func quiet(cmd *exec.Cmd) *exec.Cmd {
	osutil.HideConsole(cmd)
	return cmd
}
