package pwsh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Ensure downloads and verifies the pinned portable pwsh if Look is empty.
func Ensure(ctx context.Context) (string, error) {
	return current().Ensure(ctx)
}

// Prefetch starts Ensure in the background when the binary is missing.
func Prefetch() {
	if runtime.GOOS != "windows" {
		return
	}
	if Look() != "" {
		return
	}
	go func() { _, _ = Ensure(context.Background()) }()
}

// Install kicks off a download without waiting (settings button).
func Install() map[string]any {
	return current().Install()
}

// Snapshot is the settings/RPC progress blob.
func Snapshot() map[string]any {
	return current().Snapshot()
}

func (m *Manager) Ensure(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p := m.Look(); p != "" {
		return p, nil
	}
	goos, _ := m.platform()
	if goos != "windows" {
		return "", nil
	}
	wait, err := m.beginInstall()
	if err != nil {
		return "", err
	}
	if wait == nil {
		if p := m.Look(); p != "" {
			return p, nil
		}
		return "", fmt.Errorf("pwsh is not available")
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-wait:
	}
	if p := m.Look(); p != "" {
		return p, nil
	}
	m.mu.Lock()
	msg := strings.TrimSpace(m.err)
	m.mu.Unlock()
	if msg == "" {
		msg = "pwsh install failed"
	}
	return "", fmt.Errorf("%s", msg)
}

func (m *Manager) Install() map[string]any {
	if m.Look() == "" {
		_, _ = m.beginInstall()
	}
	return m.Snapshot()
}

func (m *Manager) beginInstall() (<-chan struct{}, error) {
	goos, goarch := m.platform()
	if _, err := archiveName(goos, goarch); err != nil {
		m.mu.Lock()
		m.phase = "error"
		m.err = err.Error()
		m.mu.Unlock()
		m.publish()
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lookManaged() != "" {
		return nil, nil
	}
	if m.wait != nil {
		return m.wait, nil
	}
	ch := make(chan struct{})
	m.wait = ch
	m.phase = "resolving"
	m.percent = 0
	m.bytes = 0
	m.total = 0
	m.err = ""
	go m.runInstall(ch)
	return ch, nil
}

func (m *Manager) runInstall(done chan struct{}) {
	defer func() {
		m.mu.Lock()
		if m.wait == done {
			m.wait = nil
		}
		m.mu.Unlock()
		close(done)
		m.publish()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	err := m.installBundle(ctx)
	m.mu.Lock()
	if err != nil {
		m.phase = "error"
		m.err = err.Error()
	} else {
		m.phase = "ready"
		m.err = ""
		m.percent = 100
	}
	m.mu.Unlock()
}

func (m *Manager) installBundle(ctx context.Context) error {
	goos, goarch := m.platform()
	archive, err := archiveName(goos, goarch)
	if err != nil {
		return err
	}
	wantSHA := officialSHA256[archive]
	m.setProgress("resolving", 0, 0, 0, archive)
	destDir := m.installDir()
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return err
	}
	zipPath := filepath.Join(m.runtimeDir(), "pwsh", archive)
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return err
	}
	m.setProgress("downloading", 0, 0, 0, archive)
	if err := m.downloadZip(ctx, archive, zipPath, wantSHA); err != nil {
		return err
	}
	unblockPath(zipPath)
	m.setProgress("extracting", 92, 0, 0, archive)
	staging := destDir + ".staging"
	_ = os.RemoveAll(staging)
	bin, err := extractZip(zipPath, staging)
	if err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	unblockTree(staging)
	m.setProgress("verifying", 96, 0, 0, archive)
	if err := m.verify()(bin); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	_ = os.RemoveAll(destDir)
	if err := os.Rename(staging, destDir); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	_ = os.Remove(zipPath)
	if m.Look() == "" {
		return fmt.Errorf("pwsh: install did not produce pwsh.exe")
	}
	return nil
}

func (m *Manager) downloadZip(ctx context.Context, archive, dest, wantSHA string) error {
	var last error
	for _, rawURL := range m.candidateURLs(archive) {
		err := m.downloadURL(ctx, rawURL, dest, wantSHA)
		if err == nil {
			return nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("pwsh: download failed")
	}
	return last
}

func (m *Manager) downloadURL(ctx context.Context, rawURL, dest, wantSHA string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	endpoint, _, _ := repoSpec()
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", referer(endpoint))
	res, err := m.client().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("pwsh: download %s", res.Status)
	}
	total := res.ContentLength
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	ok := false
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	h := sha256.New()
	pr := &progressReader{r: res.Body, total: total, on: func(n, tot int64) {
		pct := 0
		if tot > 0 {
			pct = int(n * 90 / tot)
			if pct > 90 {
				pct = 90
			}
		}
		m.setProgress("downloading", pct, n, tot, "")
	}}
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(pr, maxZipBytes+1))
	if err != nil {
		return err
	}
	if n > maxZipBytes {
		return fmt.Errorf("pwsh: archive larger than %d bytes", maxZipBytes)
	}
	if err := f.Close(); err != nil {
		return err
	}
	closed = true
	if wantSHA != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, wantSHA) {
			_ = os.Remove(tmp)
			return fmt.Errorf("pwsh: sha256 mismatch")
		}
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		return err
	}
	ok = true
	unblockPath(dest)
	return nil
}

type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	last  int
	on    func(n, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.on != nil {
		pct := 0
		if p.total > 0 {
			pct = int(p.n * 90 / p.total)
		}
		if pct != p.last || err != nil {
			p.last = pct
			p.on(p.n, p.total)
		}
	}
	return n, err
}

func (m *Manager) setProgress(phase string, percent int, bytes, total int64, archive string) {
	m.mu.Lock()
	if phase != "" {
		m.phase = phase
	}
	if percent >= 0 {
		m.percent = percent
	}
	m.bytes = bytes
	if total > 0 {
		m.total = total
	}
	if archive != "" {
		m.archive = archive
	}
	m.mu.Unlock()
	m.publish()
}

func (m *Manager) publish() {
	if m.Hub == nil {
		return
	}
	m.Hub(m.Snapshot())
}

func (m *Manager) Snapshot() map[string]any {
	needed := runtime.GOOS == "windows"
	if m.GOOS != "" {
		needed = m.GOOS == "windows"
	}
	p := m.Look()
	m.mu.Lock()
	phase := m.phase
	percent := m.percent
	bytes := m.bytes
	total := m.total
	errMsg := m.err
	archive := m.archive
	m.mu.Unlock()
	if phase == "" {
		if p != "" {
			phase = "ready"
			percent = 100
		} else if needed {
			phase = "idle"
		} else {
			phase = "ready"
			percent = 100
		}
	}
	ready := p != "" || !needed
	return map[string]any{
		"ok":      true,
		"needed":  needed,
		"ready":   ready,
		"bin":     p,
		"version": Version,
		"phase":   phase,
		"percent": percent,
		"bytes":   bytes,
		"total":   total,
		"error":   errMsg,
		"archive": archive,
		"source":  sourceOf(p, m.installDir()),
		"shell":   Fact(),
	}
}

func sourceOf(bin, managedDir string) string {
	if bin == "" {
		return ""
	}
	if env := lookEnv(); env != "" && sameFile(env, bin) {
		return "env"
	}
	if managed := lookDir(managedDir); managed != "" && sameFile(managed, bin) {
		return "managed"
	}
	return "file"
}

func sameFile(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
