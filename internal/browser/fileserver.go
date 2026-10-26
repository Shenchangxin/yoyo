// 本文件用 127.0.0.1 上的临时端口提供工作区里的本地 HTML，并判断当前文档要不要重新导航。
// 同一目录复用已有文件服务。路径逃出根目录返回 404。Chrome 把 index.html 报成 / 时，sameDocumentURL 仍视为同一份文档，避免点击或截图把页面重新加载掉。

package browser

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SetPreview 记下要预览的本地绝对路径。
// abs 在宿主非空时写入 preview。宿主为空时直接返回。
func (h *Host) SetPreview(abs string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.preview = abs
	h.mu.Unlock()
}

// PreviewPath 返回当前预览路径。宿主为空时返回空字符串。
func (h *Host) PreviewPath() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.preview
}

// OpenLocal 把本地文件设成预览，用文件服务得到地址，再确保 Chrome 并导航过去。目录会被拒绝。这里不取走页面错误，留给之后报告打开结果的调用方用 PageErrors。
// abs 先 filepath.Clean，再 os.Stat。是目录时返回 browser: need a file, got directory。然后传给 SetPreview 和 serveFileURL。断开时错误文案用它的 Base。
func (h *Host) OpenLocal(abs string) (Snapshot, error) {
	if h == nil {
		return Snapshot{}, fmt.Errorf("browser: no host")
	}
	abs = filepath.Clean(abs)
	st, err := os.Stat(abs)
	if err != nil {
		return Snapshot{}, err
	}
	if st.IsDir() {
		return Snapshot{}, fmt.Errorf("browser: need a file, got directory")
	}
	h.SetPreview(abs)
	raw, err := h.serveFileURL(abs)
	if err != nil {
		return Snapshot{}, err
	}
	if err := h.ensureChrome(raw); err != nil {
		return Snapshot{}, err
	}
	if err := h.navigateCDP(raw); err != nil {
		if isCDPGone(err) {
			return Snapshot{}, fmt.Errorf("browser: chrome disconnected while opening %s", filepath.Base(abs))
		}
		return Snapshot{}, err
	}
	h.waitDocumentComplete(4 * time.Second)
	h.syncView()
	h.refreshActions()
	h.record("open", raw)
	snap := h.snapshotLocked()
	if snap.URL == "" {
		snap.URL = raw
	}
	// 这里不取走页面错误。EnsurePreview 会丢掉这份快照，取走的错误就没了；报告这次打开的调用方再用 PageErrors 取。
	return snap, nil
}

// EnsureLocal 在需要时启动隔离 Chrome 打开 abs。当前文档已经是这个文件时不重新加载，只同步视图。
func (h *Host) EnsureLocal(abs string) (Snapshot, error) {
	if h == nil {
		return Snapshot{}, fmt.Errorf("browser: no host")
	}
	abs = filepath.Clean(abs)
	if !shouldReloadLocal(h.chromeAlive(), h.liveURL(), h.wantLocalURL(abs)) {
		h.SetPreview(abs)
		h.syncView()
		return h.snapshotLocked(), nil
	}
	return h.OpenLocal(abs)
}

// EnsurePreview 有预览路径时 EnsureLocal，否则按 previewStartURL 确保 Chrome 已启动。
func (h *Host) EnsurePreview() error {
	if h == nil {
		return fmt.Errorf("browser: no host")
	}
	if preview := h.PreviewPath(); preview != "" {
		_, err := h.EnsureLocal(preview)
		return err
	}
	return h.ensureChrome(h.previewStartURL())
}

// ShowingLocal 在 Chrome 还活着且当前地址和这个文件是同一份文档时返回真。
// abs 传给 wantLocalURL。
func (h *Host) ShowingLocal(abs string) bool {
	return !shouldReloadLocal(h.chromeAlive(), h.liveURL(), h.wantLocalURL(abs))
}

// chromeAlive 看当前 CDP 连接的读循环是否还在。
func (h *Host) chromeAlive() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	c := h.cdp
	h.mu.Unlock()
	return cdpAlive(c)
}

// liveURL 返回宿主记下的当前地址。
func (h *Host) liveURL() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.url
}

// wantLocalURL 在文件服务的根就是该文件所在目录时，返回文件服务上的地址。否则返回空字符串。
// abs 先 filepath.Clean。宿主为空、文件服务为 nil 或 fileRoot 为空时返回空字符串。fileRoot 与 abs 所在目录不同时也返回空字符串，否则返回 fileURL、斜杠和转义后的文件名。
func (h *Host) wantLocalURL(abs string) string {
	if h == nil {
		return ""
	}
	abs = filepath.Clean(abs)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.fileSrv == nil || h.fileRoot == "" {
		return ""
	}
	if filepath.Clean(h.fileRoot) != filepath.Clean(filepath.Dir(abs)) {
		return ""
	}
	return h.fileURL + "/" + url.PathEscape(filepath.Base(abs))
}

// snapshotLocked 在锁内抄一份地址、标题、正文、配置路径和控件文本。
func (h *Host) snapshotLocked() Snapshot {
	if h == nil {
		return Snapshot{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return Snapshot{URL: h.url, Title: h.title, Text: h.snap, Profile: h.profile, Actions: h.actions}
}

// shouldReloadLocal 在 Chrome 不在、想要的地址为空，或两份地址不是同一文档时返回真。
// chromeAlive 为假时返回真。
// liveURL 传给 sameDocumentURL。
// wantURL 去掉空白后为空时返回真。否则和 liveURL 一起传给 sameDocumentURL，不是同一文档时返回真。
func shouldReloadLocal(chromeAlive bool, liveURL, wantURL string) bool {
	if !chromeAlive || strings.TrimSpace(wantURL) == "" {
		return true
	}
	return !sameDocumentURL(liveURL, wantURL)
}

// sameDocumentURL 比较方案、主机和规范化后的路径。解析失败时只比较去掉末尾斜杠的原文字符串。
// live 去掉空白后为空则返回假。解析失败时用去掉末尾斜杠的原文比较。解析成功时用 EqualFold 比较方案和主机，路径传给 normalizeDocPath。
// want 去掉空白后为空则返回假。解析失败时用去掉末尾斜杠的原文比较。解析成功时用 EqualFold 比较方案和主机，路径传给 normalizeDocPath。
func sameDocumentURL(live, want string) bool {
	live = strings.TrimSpace(live)
	want = strings.TrimSpace(want)
	if live == "" || want == "" {
		return false
	}
	a, err1 := url.Parse(live)
	b, err2 := url.Parse(want)
	if err1 != nil || err2 != nil {
		return strings.TrimRight(live, "/") == strings.TrimRight(want, "/")
	}
	if !strings.EqualFold(a.Scheme, b.Scheme) || !strings.EqualFold(a.Host, b.Host) {
		return false
	}
	return normalizeDocPath(a.Path) == normalizeDocPath(b.Path)
}

// normalizeDocPath 把同一份文档的路径收成一个键。末尾的 /index.html 和 /index.htm 去掉，这样 Chrome 重定向后的 / 和 index.html 算同一页。
// p 先 PathUnescape 并去掉末尾斜杠。以 /index.html 或 /index.htm 结尾时去掉这段，结尾比较不区分大小写，然后再去掉末尾斜杠。
func normalizeDocPath(p string) string {
	p, _ = url.PathUnescape(strings.TrimSuffix(p, "/"))
	for _, idx := range []string{"/index.html", "/index.htm"} {
		if strings.HasSuffix(strings.ToLower(p), idx) {
			p = p[:len(p)-len(idx)]
		}
	}
	return strings.TrimSuffix(p, "/")
}

// serveFileURL 在 127.0.0.1 的临时端口上提供该文件所在目录。根目录没变时复用已有服务。空路径对应该文件本身。含 .. 或逃出目录的请求返回 404。
// abs 用 Dir 当根、Base 当文件名。根与已有 fileRoot 相同时返回已有 fileURL、斜杠和转义后的文件名。
func (h *Host) serveFileURL(abs string) (string, error) {
	root := filepath.Dir(abs)
	base := filepath.Base(abs)
	h.mu.Lock()
	if h.fileSrv != nil && h.fileRoot == root {
		u := h.fileURL + "/" + url.PathEscape(base)
		h.mu.Unlock()
		return u, nil
	}
	old := h.fileSrv
	h.fileSrv = nil
	h.mu.Unlock()
	if old != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = old.Shutdown(ctx)
		cancel()
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	mux := http.NewServeMux()
	fileRoot := root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/")
		if rel == "" {
			rel = base
		}
		rel = filepath.Clean(rel)
		if strings.Contains(rel, "..") {
			http.NotFound(w, r)
			return
		}
		p := filepath.Join(fileRoot, rel)
		if !withinDir(fileRoot, p) {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, p)
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	u := "http://" + ln.Addr().String()
	h.mu.Lock()
	h.fileSrv = srv
	h.fileRoot = root
	h.fileURL = u
	h.mu.Unlock()
	return u + "/" + url.PathEscape(base), nil
}

// closeFileServer 关掉文件服务并清掉根和地址。最多等 500 毫秒。
func (h *Host) closeFileServer() {
	h.mu.Lock()
	srv := h.fileSrv
	h.fileSrv = nil
	h.fileRoot = ""
	h.fileURL = ""
	h.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = srv.Shutdown(ctx)
		cancel()
	}
}

// withinDir 判断 p 是否仍在 root 里面。相对路径是 .. 或以 .. 加分隔符开头时返回假。
func withinDir(root, p string) bool {
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
