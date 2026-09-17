package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/home"
)

// WorkspaceReady reports whether p is a real directory the harness can use.
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

func unsetWorkspace(p string) bool {
	p = strings.TrimSpace(p)
	return p == "" || p == "." || p == "./" || p == ".\\"
}

// applyDefaultWorkspace makes Config.Workspace a usable directory so the
// desktop can open on the agent instead of a first-run gate.
func applyDefaultWorkspace(h *home.Dir, cfg *Config) bool {
	if h == nil || cfg == nil {
		return false
	}
	if WorkspaceReady(cfg.Workspace) {
		return false
	}
	p := strings.TrimSpace(cfg.Workspace)
	if filepath.IsAbs(p) && !unsetWorkspace(p) {
		if err := os.MkdirAll(p, 0o755); err == nil && WorkspaceReady(p) {
			return true
		}
	}
	fallback := h.Workspace()
	if err := os.MkdirAll(fallback, 0o755); err != nil {
		return false
	}
	if cfg.Workspace == fallback {
		return false
	}
	cfg.Workspace = fallback
	return true
}
