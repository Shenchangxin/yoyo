// 会话上下文目录的复制和删除。分叉会话时复制 spill 和工作区里的 .yoyo/context，删除会话时两边一起清掉。
// 不改 JSONL。轨迹文件由调用方另行复制。

package runtime

import (
	"os"
	"path/filepath"
)

// CopyTree 把 src 复制到 dst。src 或 dst 为空，或 src 不存在时返回 nil。复制文件时保留原权限。
func CopyTree(src, dst string) error {
	if src == "" || dst == "" {
		return nil
	}
	st, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !st.IsDir() {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		b, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, st.Mode())
	}
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode())
	})
}

// RemoveSessionContext 删除 homeRoot/sessions/<sessionID>，以及工作区下的 .yoyo/context/<sessionID>。sessionID 为空则不动。
// workspace 非空时删除其下的 .yoyo/context/<sessionID>。
func RemoveSessionContext(homeRoot, workspace, sessionID string) {
	if sessionID == "" {
		return
	}
	if homeRoot != "" {
		_ = os.RemoveAll(filepath.Join(homeRoot, "sessions", sessionID))
	}
	if workspace != "" {
		_ = os.RemoveAll(filepath.Join(workspace, ".yoyo", "context", sessionID))
	}
}
