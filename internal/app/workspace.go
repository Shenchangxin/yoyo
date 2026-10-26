// 代理工作区和视频工作区的默认目录，以及检查器要用的有界文件预览。路径必须落在 capability.WithinWorkspace 里。

package app

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/office"
)

// WorkspaceReady 判断 p 是否为绝对路径上的真实目录。空串、.、./、.\ 和非绝对路径返回 false。
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

// WorkspaceReady 报告 Config.Workspace 是否为可用目录。App 为空返回 false。
func (a *App) WorkspaceReady() bool {
	if a == nil {
		return false
	}
	return WorkspaceReady(a.Config.Workspace)
}

// unsetWorkspace 判断路径是否为空白、.、./ 或 .\。
func unsetWorkspace(p string) bool {
	p = strings.TrimSpace(p)
	return p == "" || p == "." || p == "./" || p == ".\\"
}

// applyDefaultWorkspace 在 Config.Workspace 还不是可用目录时，先尝试创建已有绝对路径，失败再用 Home.Workspace。返回值表示是否改写了配置里的路径。
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

// applyDefaultVideoWorkspace 给视频单独一个目录，和代理工作区分开。规则与 applyDefaultWorkspace 相同，回落目录是 Home.VideoWorkspace。
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

// previewFileBytes 是文本预览的字节上限，值为 80000。
const previewFileBytes = 80_000

// blobFileBytes 是二进制预览的字节上限，值为 16 MiB。
const blobFileBytes = 16 << 20

// readCapped 最多读取 limit 字节，并用多读 1 字节判断是否截断。limit 小于等于 0 时读完全部。
func readCapped(path string, limit int) ([]byte, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	if limit <= 0 {
		b, err := io.ReadAll(f)
		return b, false, err
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, false, err
	}
	trunc := len(raw) > limit
	if trunc {
		raw = raw[:limit]
	}
	return raw, trunc, nil
}

// resolveWorkspaceFile 把相对路径拼到工作区。工作区参数为空时用 App.Workspace。逃出 capability.WithinWorkspace 的路径返回 path escapes workspace。
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

// PreviewWorkspaceFile 为检查器和产物卡返回有界预览。图片、PDF、音频、视频只回元数据。docx、xlsx、pptx 走 office 文本。前 800 字节含 NUL 则标成 binary。
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
	raw, truncated, err := readCapped(p, previewFileBytes)
	if err != nil {
		return nil, err
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

// ReadWorkspaceBlob 返回不超过 blobFileBytes 的 base64，供检查器显示 PDF、图片和其他二进制。扩展名没有类型时用 http.DetectContentType。
func (a *App) ReadWorkspaceBlob(workspace, rel string) (map[string]any, error) {
	p, slashRel, err := a.resolveWorkspaceFile(workspace, rel)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	raw, truncated, err := readCapped(p, blobFileBytes)
	if err != nil {
		return nil, err
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

// officePreviewText 只对 .docx、.xlsx、.pptx 调用 office.Query，其他扩展名返回 os.ErrInvalid。
func officePreviewText(path, ext string) (string, error) {
	switch ext {
	case ".docx", ".xlsx", ".pptx":
		return office.Query(path)
	default:
		return "", os.ErrInvalid
	}
}

// looksBinary 只看前 800 字节，出现 NUL 则视为二进制。
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

// langFromExt 按扩展名给出预览语言名。未列出的扩展名去掉点号后原样返回。
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

// previewKind 按扩展名分成 image、pdf、audio、video、html、markdown、office 或 text。
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

// mimeFromExt 为 markdown、typescript 和 svg 写明类型，其余用 mime.TypeByExtension，没有则 application/octet-stream。
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
