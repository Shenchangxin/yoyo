// quiet.go 给剪辑拉起的外部命令隐藏控制台窗口。

package cut

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// quiet 调用 osutil.HideConsole 后返回原命令。
// cmd 传给 osutil.HideConsole，再原样返回。
func quiet(cmd *exec.Cmd) *exec.Cmd {
	osutil.HideConsole(cmd)
	return cmd
}
