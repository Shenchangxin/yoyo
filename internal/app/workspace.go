package app

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/office"
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

// applyDefaultVideoWorkspace gives Video its own folder, independent of the
// Agent workspace, so video chats and file tools never share the coding repo.
func applyDefaultVideoWorkspace(h *home.Dir, cfg *Config) bool {
	if h == nil || cfg == nil {
		return false
	}
	if WorkspaceReady(cfg.VideoWorkspace) {
		return false
	}
	p := strings.TrimSpace(cfg.VideoWorkspace)
	if filepath.IsAbs(p) && !unsetWorkspace(p) {
		if err := os.MkdirAll(p, 0o755); err == nil && WorkspaceReady(p) {
			return true
		}
	}
	fallback := h.VideoWorkspace()
	if err := os.MkdirAll(fallback, 0o755); err != nil {
		return false
	}
	if cfg.VideoWorkspace == fallback {
		return false
	}
	cfg.VideoWorkspace = fallback
	return true
}

const previewFileBytes = 80_000
const blobFileBytes = 16 << 20

func (a *App) resolveWorkspaceFile(workspace, rel string) (abs, slashRel string, err error) {
	if a != nil && strings.TrimSpace(workspace) == "" {
		workspace = a.Workspace()
	}
	rel = strings.TrimSpace(rel)
	if workspace == "" || rel == "" {
		return "", "", fmt.Errorf("empty path")
	}
	p := rel
	if !filepath.IsAbs(p) {
		p = filepath.Join(workspace, rel)
	}
	p = filepath.Clean(p)
	if !capability.WithinWorkspace(workspace, p) {
		return "", "", fmt.Errorf("path escapes workspace")
	}
	return p, filepath.ToSlash(rel), nil
}

// PreviewWorkspaceFile returns a bounded text preview of a workspace file
// for the session inspector and artifact cards.
func (a *App) PreviewWorkspaceFile(workspace, rel string) (map[string]any, error) {
	p, slashRel, err := a.resolveWorkspaceFile(workspace, rel)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(p))
	kind := previewKind(ext)
	meta := map[string]any{
		"path":  slashRel,
		"kind":  kind,
		"mime":  mimeFromExt(ext),
		"lang":  langFromExt(ext),
		"html":  kind == "html",
		"bytes": st.Size(),
	}
	if kind == "image" || kind == "pdf" || kind == "audio" || kind == "video" {
		meta["binary"] = true
		return meta, nil
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
	meta["truncated"] = truncated
	if officeText, oerr := officePreviewText(p, ext); oerr == nil && officeText != "" {
		meta["text"] = officeText
		meta["lang"] = "text"
		meta["html"] = false
		meta["office"] = true
		meta["kind"] = "office"
		return meta, nil
	}
	if looksBinary(raw) {
		meta["binary"] = true
		meta["kind"] = "binary"
		return meta, nil
	}
	meta["text"] = string(raw)
	return meta, nil
}

// ReadWorkspaceBlob returns a bounded base64 payload so the inspector can
// render PDF, images, and other binary previews.
func (a *App) ReadWorkspaceBlob(workspace, rel string) (map[string]any, error) {
	p, slashRel, err := a.resolveWorkspaceFile(workspace, rel)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	truncated := false
	if len(raw) > blobFileBytes {
		raw = raw[:blobFileBytes]
		truncated = true
	}
	ext := strings.ToLower(filepath.Ext(p))
	mimeType := mimeFromExt(ext)
	if mimeType == "" || mimeType == "application/octet-stream" {
		if d := http.DetectContentType(raw); d != "" {
			mimeType = d
		}
	}
	return map[string]any{
		"path":      slashRel,
		"kind":      previewKind(ext),
		"mime":      mimeType,
		"bytes":     st.Size(),
		"truncated": truncated,
		"base64":    base64.StdEncoding.EncodeToString(raw),
	}, nil
}

func officePreviewText(path, ext string) (string, error) {
	switch ext {
	case ".docx", ".xlsx", ".pptx":
		return office.Query(path)
	default:
		return "", os.ErrInvalid
	}
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
	case ".md", ".markdown":
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

func previewKind(ext string) string {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".ico", ".avif":
		return "image"
	case ".pdf":
		return "pdf"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac":
		return "audio"
	case ".mp4", ".webm", ".mov", ".m4v":
		return "video"
	case ".html", ".htm", ".xhtml":
		return "html"
	case ".md", ".markdown":
		return "markdown"
	case ".docx", ".xlsx", ".pptx":
		return "office"
	default:
		return "text"
	}
}

func mimeFromExt(ext string) string {
	switch ext {
	case ".md", ".markdown":
		return "text/markdown"
	case ".ts", ".tsx":
		return "text/typescript"
	case ".svg":
		return "image/svg+xml"
	}
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "application/octet-stream"
}
