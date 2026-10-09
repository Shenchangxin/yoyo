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
	time.Sleep(350 * time.Millisecond)
	h.syncView()
	h.record("open", raw)
	h.mu.Lock()
	snap := Snapshot{URL: h.url, Title: h.title, Text: h.snap, Profile: h.profile}
	h.mu.Unlock()
	if snap.URL == "" {
		snap.URL = raw
	}
	return snap, nil
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
