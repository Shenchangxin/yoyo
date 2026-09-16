package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/home"
)

// WorkspaceReady reports whether p is a real directory the operator picked,
// not the implicit cwd placeholder that used to skip FirstRun.
func WorkspaceReady(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" || p == "." || p == "./" || p == ".\\" {
		return false
	}
	if !filepath.IsAbs(p) {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func (a *App) WorkspaceReady() bool {
	if a == nil {
		return false
	}
	return WorkspaceReady(a.Config.Workspace)
}

// applyDefaultWorkspace fills an empty config path with $YOYO_HOME/workspace.
// A non-empty path that is missing on disk is left alone so setup can still run.
func applyDefaultWorkspace(h *home.Dir, cfg *Config) bool {
	if h == nil || cfg == nil {
		return false
	}
	if strings.TrimSpace(cfg.Workspace) != "" {
		return false
	}
	p := h.Workspace()
	if err := os.MkdirAll(p, 0o755); err != nil {
		return false
	}
	cfg.Workspace = p
	return true
}
