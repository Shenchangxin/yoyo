// 给子进程隐藏控制台，并构造已经隐藏控制台的 git 命令。
// 不执行 git，也不检查路径。worktree.go、hunks.go、tools.go 和 reach_status.go 调用这里。

package runtime

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// hideConsole 把 cmd 交给 osutil.HideConsole。
func hideConsole(cmd *exec.Cmd) {
	osutil.HideConsole(cmd)
}

// gitCmd 用 git 和给定参数构造命令，并隐藏控制台。
// args 展开后传给 exec.Command，程序名是 git。
func gitCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	hideConsole(cmd)
	return cmd
}
