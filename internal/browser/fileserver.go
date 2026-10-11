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

func (h *Host) SetPreview(abs string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.preview = abs
	h.mu.Unlock()
}

func (h *Host) PreviewPath() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.preview
}

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
	h.record("open", raw)
	snap := h.snapshotLocked()
	if snap.URL == "" {
		snap.URL = raw
	}
	return snap, nil
}

// EnsureLocal starts isolated Chrome on abs if needed, but does not reload a
// document already showing that file. Playwright page.screenshot / page.click
// never call page.goto; session facb2e21ba0b31a3 lost every click because
// screenshot/click always navigated back to the boot screen.
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

func (h *Host) ShowingLocal(abs string) bool {
	return !shouldReloadLocal(h.chromeAlive(), h.liveURL(), h.wantLocalURL(abs))
}

func (h *Host) chromeAlive() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	c := h.cdp
	h.mu.Unlock()
	return cdpAlive(c)
}

func (h *Host) liveURL() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.url
}

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

func (h *Host) snapshotLocked() Snapshot {
	if h == nil {
		return Snapshot{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return Snapshot{URL: h.url, Title: h.title, Text: h.snap, Profile: h.profile}
}

func shouldReloadLocal(chromeAlive bool, liveURL, wantURL string) bool {
	if !chromeAlive || strings.TrimSpace(wantURL) == "" {
		return true
	}
	return !sameDocumentURL(liveURL, wantURL)
}

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
	pa, _ := url.PathUnescape(strings.TrimSuffix(a.Path, "/"))
	pb, _ := url.PathUnescape(strings.TrimSuffix(b.Path, "/"))
	return pa == pb
}

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

func withinDir(root, p string) bool {
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
