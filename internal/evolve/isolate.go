package evolve

import (
	"os"
	"os/exec"
)

// ScratchGit is a throwaway git repo used as IsolateRoot so Harbor tasks
// for a candidate run inside a detached worktree, not the user's tree.
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
