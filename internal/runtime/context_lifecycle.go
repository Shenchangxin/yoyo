package runtime

import (
	"os"
	"path/filepath"
)

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
