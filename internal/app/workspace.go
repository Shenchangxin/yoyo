package app

import (
	"os"
	"path/filepath"
	"strings"
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
