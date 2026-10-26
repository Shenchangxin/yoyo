// 子代理用的工作区隔离：临时目录上的 git worktree，失败则整树复制。持久隔离和把改动拷回原树也在这里。hunk 级 git apply 在 hunks.go。

package runtime

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IsolateWorkspace 把 src 隔到临时目录，避免子代理改用户的树。Eval Harbor 已经自行隔离；这里给实时代理任务用同一做法。优先 git worktree add --detach；失败则 copyTree。返回的清理函数会删掉临时目录，worktree 成功时还会 git worktree remove --force。
func IsolateWorkspace(src string) (string, func(), error) {
	dst, err := os.MkdirTemp("", "yoyo-wt-*")
	if err != nil {
		return "", nil, err
	}
	// git worktree add 拒绝已存在的目标目录，所以先删掉 MkdirTemp 创建的空目录。
	_ = os.Remove(dst)
	cleanup := func() { _ = os.RemoveAll(dst) }
	if gitWorktree(src, dst) == nil {
		return dst, func() {
			_ = gitCmd("-C", src, "worktree", "remove", "--force", dst).Run()
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

// PersistWorktree 在 dest 做持久隔离，优先 git worktree，失败则复制。dest 已是目录时直接返回，不在这里清理。src 与 dest 为空或相同则返回错误。
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

// RemoveWorktree 去掉持久隔离。dest 为空、为 . 或为路径分隔符时直接返回。src 非空时先 git worktree remove --force，再删除 dest。
func RemoveWorktree(src, dest string) {
	dest = filepath.Clean(dest)
	if dest == "" || dest == "." || dest == string(filepath.Separator) {
		return
	}
	if src != "" {
		_ = gitCmd("-C", src, "worktree", "remove", "--force", dest).Run()
	}
	_ = os.RemoveAll(dest)
}

// gitWorktree 先确认 src 在 git 工作树里，再执行 git worktree add --detach。
// dst 作为 git worktree add --detach 的目标路径传给 gitCmd。
func gitWorktree(src, dst string) error {
	cmd := gitCmd("-C", src, "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return err
	}
	return gitCmd("-C", src, "worktree", "add", "--detach", dst).Run()
}

// ApplyWorktree 把隔离目录 dest 的改动写回 origin。gitApply 成功就返回；否则 copyTree 从 dest 覆盖 origin。
func ApplyWorktree(origin, dest string) error {
	origin = filepath.Clean(origin)
	dest = filepath.Clean(dest)
	if origin == "" || dest == "" || origin == dest {
		return fmt.Errorf("invalid worktree apply paths")
	}
	if st, err := os.Stat(dest); err != nil || !st.IsDir() {
		return fmt.Errorf("worktree missing: %s", dest)
	}
	if gitApply(origin, dest) == nil {
		return nil
	}
	return copyTree(dest, origin)
}

// gitApply 在两边都是 git 工作树时，用 dest 的 git diff --binary 对 origin 执行 git apply --binary。diff 为空则返回。apply 失败则改为 copyTree。
func gitApply(origin, dest string) error {
	if err := gitCmd("-C", origin, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		return err
	}
	if err := gitCmd("-C", dest, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		return err
	}
	cmd := gitCmd("-C", dest, "diff", "--binary")
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return nil
	}
	apply := gitCmd("-C", origin, "apply", "--binary", "-")
	apply.Stdin = strings.NewReader(string(out))
	if err := apply.Run(); err != nil {
		return copyTree(dest, origin)
	}
	return nil
}

// copyTree 用 filepath.Walk 复制文件。目录名在 skipDirNames 里且不是源根时跳过。文件按原模式覆盖写入。
// src 交给 filepath.Walk；目录名在 skipDirNames 里且当前路径不是 src 时跳过该目录。
// dst 和 Walk 得到的相对路径拼成写入目标。
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
