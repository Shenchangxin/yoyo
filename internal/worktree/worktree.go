// Package worktree 负责在工作区目录上做的 git 和文件树操作：
// 给子代理、定时任务和评测建隔离副本（优先 git worktree，失败则整树复制），把隔离副本的改动写回原树，
// 以及把 git diff 拆成 hunk 后按选择打回工作区（见 hunks.go）。
// 这里不认识会话、工具或代理循环，只接收目录路径和 diff 文本。
package worktree

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// skipDirs 是遍历或复制工作区时跳过的目录名：.git、node_modules、vendor、dist、.yoyo、target、.next、__pycache__。
// 只在包内只读使用；包外经 SkipDirNames 拿副本，谁都改不到这张表。
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	".yoyo": true, "target": true, ".next": true, "__pycache__": true,
}

// SkipDirNames 返回 skipDirs 的一份副本。runtime 的 glob、grep 和提及扫描拿它做忽略集的底子，再叠加 .gitignore。
// 每次调用都是新 map，调用方可以随意往里加名字。
func SkipDirNames() map[string]bool {
	out := make(map[string]bool, len(skipDirs))
	for k, v := range skipDirs {
		out[k] = v
	}
	return out
}

// gitCmd 用 git 和给定参数构造命令，并隐藏控制台，避免 Windows 上每次调用都闪出一个窗口。
// args 展开后传给 exec.Command，程序名是 git。
func gitCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	osutil.HideConsole(cmd)
	return cmd
}

// Isolate 把 src 隔到临时目录，避免子代理改用户的树。Eval Harbor 和实时代理任务用同一做法。优先 git worktree add --detach；失败则 copyTree。返回的清理函数会删掉临时目录，worktree 成功时还会 git worktree remove --force。
func Isolate(src string) (string, func(), error) {
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

// Persist 在 dest 做持久隔离，优先 git worktree，失败则复制。dest 已是目录时直接返回，不在这里清理。src 与 dest 为空或相同则返回错误。
func Persist(src, dest string) error {
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

// Remove 去掉持久隔离。dest 为空、为 . 或为路径分隔符时直接返回。src 非空时先 git worktree remove --force，再删除 dest。
func Remove(src, dest string) {
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

// Apply 把隔离目录 dest 的改动写回 origin。gitApply 成功就返回；否则 copyTree 从 dest 覆盖 origin。
func Apply(origin, dest string) error {
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

// copyTree 用 filepath.Walk 复制文件。目录名在 skipDirs 里且不是源根时跳过。文件按原模式覆盖写入。
// src 交给 filepath.Walk；目录名在 skipDirs 里且当前路径不是 src 时跳过该目录。
// dst 和 Walk 得到的相对路径拼成写入目标。
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && skipDirs[info.Name()] && path != src {
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
