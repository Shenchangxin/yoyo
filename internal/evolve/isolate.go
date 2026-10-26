// 造一个一次性的 git 仓库，给 Cycle 在调用方没传 IsolateRoot 时当作隔离根。不修改用户仓库。Harbor 任务仍由 eval 盖到分离工作区上，并且不带 tests/。

package evolve

import (
	"os"
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// ScratchGit 在临时目录里 git init、写本地用户并做一个空提交。返回目录和删除函数。PATH 里没有 git，或 init、提交失败时，先删目录再返回错误。
func ScratchGit() (string, func(), error) {
	dir, err := os.MkdirTemp("", "yoyo-cand-git-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if _, err := exec.LookPath("git"); err != nil {
		cleanup()
		return "", nil, err
	}
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		osutil.HideConsole(cmd)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=yoyo", "GIT_AUTHOR_EMAIL=yoyo@local",
			"GIT_COMMITTER_NAME=yoyo", "GIT_COMMITTER_EMAIL=yoyo@local",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return err
		}
		_ = out
		return nil
	}
	if err := run("init", "-b", "main"); err != nil {
		cleanup()
		return "", nil, err
	}
	_ = run("config", "user.email", "yoyo@local")
	_ = run("config", "user.name", "yoyo")
	if err := run("commit", "--allow-empty", "-m", "seed"); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}
