// 给子进程隐藏控制台。不执行命令，也不检查路径。tools.go、tools_git.go 和 reach_status.go 调用这里。
// git worktree 和 hunk 操作自带的 git 命令构造在 internal/worktree。

package runtime

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// hideConsole 把 cmd 交给 osutil.HideConsole，避免 Windows 上每个子进程都闪出一个控制台窗口。
func hideConsole(cmd *exec.Cmd) {
	osutil.HideConsole(cmd)
}
