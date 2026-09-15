package runtime

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// IsolateWorkspace copies the workspace (or git worktree) so a subagent
// cannot mutate the user's tree. Eval Harbor already isolates; this is the
// same idea for live agent tasks.
func IsolateWorkspace(src string) (string, func(), error) {
	dst, err := os.MkdirTemp("", "yoyo-wt-*")
	if err != nil {
		return "", nil, err
	}
	// git worktree add refuses an existing dest.
	_ = os.Remove(dst)
	cleanup := func() { _ = os.RemoveAll(dst) }
	if gitWorktree(src, dst) == nil {
		return dst, func() {
			_ = exec.Command("git", "-C", src, "worktree", "remove", "--force", dst).Run()
			cleanup()
		}, nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", nil, err
	}
	if err := copyTree(src, dst); err != nil {
		cleanup()
		return "", nil, err
	}
	return dst, cleanup, nil
}

func gitWorktree(src, dst string) error {
	cmd := exec.Command("git", "-C", src, "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return err
	}
	return exec.Command("git", "-C", src, "worktree", "add", "--detach", dst).Run()
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && skipDirNames[info.Name()] && path != src {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
