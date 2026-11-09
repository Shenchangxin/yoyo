// fs.go 拥有外部 agent 经 ACP 读写文件时的目录限制：只认绝对路径；读可以落在工单目录、可写目录和只读目录，最多 2 MB；
// 写只能落在工单目录和可写目录，先经 CheckWrite（Broker 的 WriteWorkspace 检查），再写临时文件后改名。
// 和 Broker、工作区工具一样按字面路径判断，不调用 filepath.EvalSymlinks。
// 不拥有审批界面（app），也不拥有 agent 自己用 shell 写的文件（那些走权限请求，见 perm.go）。

package acpworker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// maxReadBytes 是一次读文件的上限。
const maxReadBytes = 2 << 20

// errNotAbsolute 是 agent 给了相对路径。
var errNotAbsolute = errors.New("path must be absolute")

// readFile 处理 fs/read_text_file。Line 从 1 起，Limit 是最多几行。
func (w *Worker) readFile(_ context.Context, req acp.ReadTextFileRequest) (string, error) {
	path, err := w.fsPath(req.Path, false)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", fmt.Errorf("is a directory: %s", path)
	}
	if st.Size() > maxReadBytes {
		return "", fmt.Errorf("file is larger than 2 MB: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sliceLines(string(body), req.Line, req.Limit), nil
}

// writeFile 处理 fs/write_text_file。写成后记进这一轮，没有工具调用覆盖到的会在回合结束时补记。
func (w *Worker) writeFile(ctx context.Context, req acp.WriteTextFileRequest) error {
	path, err := w.fsPath(req.Path, true)
	if err != nil {
		return err
	}
	if w.cfg.CheckWrite != nil {
		if err := w.cfg.CheckWrite(ctx, path); err != nil {
			return err
		}
	}
	if err := atomicWrite(path, []byte(req.Content)); err != nil {
		return err
	}
	w.jr.noteWrite(path)
	return nil
}

// fsPath 检查 raw 能不能读或写，返回清理过的路径。只读目录压过可写目录。
func (w *Worker) fsPath(raw string, write bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !filepath.IsAbs(raw) {
		return "", errNotAbsolute
	}
	path := filepath.Clean(raw)
	inRead := withinAny(w.spec.ReadRoots, path)
	inWrite := within(w.spec.Dir, path) || withinAny(w.spec.Roots, path)
	if write {
		if inWrite && !inRead {
			return path, nil
		}
		return "", fmt.Errorf("writing outside the ticket directories is not allowed: %s", path)
	}
	if inWrite || inRead {
		return path, nil
	}
	return "", fmt.Errorf("reading outside the ticket directories is not allowed: %s", path)
}

// within 在 path 就是 root 或在它下面时为真。root 为空时为假。Windows 上不分大小写。
func within(root, path string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

// withinAny 在 path 落在 roots 任一目录里时为真。
func withinAny(roots []string, path string) bool {
	for _, root := range roots {
		if within(root, path) {
			return true
		}
	}
	return false
}

// atomicWrite 先在同一目录写临时文件，再改名盖过 path。中途失败时删掉临时文件，原文件不动。
func atomicWrite(path string, body []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".yoyo-write-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if st, err := os.Stat(path); err == nil {
		_ = os.Chmod(name, st.Mode().Perm())
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}

// sliceLines 按 line（从 1 起）和 limit 取行。都为 nil 时原样返回。
func sliceLines(text string, line, limit *int) string {
	if line == nil && limit == nil {
		return text
	}
	lines := strings.SplitAfter(text, "\n")
	start := 0
	if line != nil && *line > 1 {
		start = *line - 1
	}
	if start >= len(lines) {
		return ""
	}
	end := len(lines)
	if limit != nil && *limit >= 0 && start+*limit < end {
		end = start + *limit
	}
	return strings.Join(lines[start:end], "")
}
