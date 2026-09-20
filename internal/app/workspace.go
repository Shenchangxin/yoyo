package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
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

const previewFileBytes = 80_000

// PreviewWorkspaceFile returns a bounded text preview of a workspace file
// for the session inspector and artifact cards.
func (a *App) PreviewWorkspaceFile(workspace, rel string) (map[string]any, error) {
	if a != nil && strings.TrimSpace(workspace) == "" {
		workspace = a.Workspace()
	}
	rel = strings.TrimSpace(rel)
	if workspace == "" || rel == "" {
		return nil, fmt.Errorf("empty path")
	}
	p := rel
	if !filepath.IsAbs(p) {
		p = filepath.Join(workspace, rel)
	}
	p = filepath.Clean(p)
	if !capability.WithinWorkspace(workspace, p) {
		return nil, fmt.Errorf("path escapes workspace")
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	truncated := false
	if len(raw) > previewFileBytes {
		raw = raw[:previewFileBytes]
		truncated = true
	}
	if looksBinary(raw) {
		return map[string]any{
			"path":      filepath.ToSlash(rel),
			"bytes":     len(raw),
			"binary":    true,
			"truncated": truncated,
		}, nil
	}
	ext := strings.ToLower(filepath.Ext(p))
	html := ext == ".html" || ext == ".htm" || ext == ".xhtml"
	return map[string]any{
		"path":      filepath.ToSlash(rel),
		"text":      string(raw),
		"lang":      langFromExt(ext),
		"html":      html,
		"bytes":     len(raw),
		"truncated": truncated,
	}, nil
}

func looksBinary(b []byte) bool {
	n := len(b)
	if n > 800 {
		n = 800
	}
	for i := 0; i < n; i++ {
		if b[i] == 0 {
			return true
		}
	}
	return false
}

func langFromExt(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "ts"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "js"
	case ".json":
		return "json"
	case ".css":
		return "css"
	case ".html", ".htm", ".xhtml":
		return "html"
	case ".md":
		return "md"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".yml", ".yaml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sh":
		return "bash"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}
