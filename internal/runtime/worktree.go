package runtime

import (
	"fmt"
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

// PersistWorktree creates a durable isolate at dest (git worktree or copy).
// Unlike IsolateWorkspace, dest is operator-owned and is not cleaned up here.
func PersistWorktree(src, dest string) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)
	if src == "" || dest == "" || src == dest {
		return fmt.Errorf("invalid worktree paths")
	}
	if st, err := os.Stat(dest); err == nil && st.IsDir() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(dest)
	if gitWorktree(src, dest) == nil {
		return nil
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := copyTree(src, dest); err != nil {
		_ = os.RemoveAll(dest)
		return err
	}
	return nil
}

// RemoveWorktree drops a persist isolate. src is the origin git repo when known.
func RemoveWorktree(src, dest string) {
	dest = filepath.Clean(dest)
	if dest == "" || dest == "." || dest == string(filepath.Separator) {
		return
	}
	if src != "" {
		_ = exec.Command("git", "-C", src, "worktree", "remove", "--force", dest).Run()
	}
	_ = os.RemoveAll(dest)
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
