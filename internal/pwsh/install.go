// 本文件下载、校验并解压钉住的 pwsh。Look 已经有结果时不再下载。

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

// Ensure 在 Look 为空时下载并校验钉住的便携 pwsh，返回可执行文件路径。使用进程级 Manager。
func Ensure(ctx context.Context) (string, error) {
	return current().Ensure(ctx)
}

// Prefetch 只在 Windows 且 Look 为空时，在后台调用 Ensure。其他系统直接返回。
func Prefetch() {
	if runtime.GOOS != "windows" {
		return
	}
	if Look() != "" {
		return
	}
	go func() { _, _ = Ensure(context.Background()) }()
}

// Install 在 Look 为空时开始下载，不等待结束，并返回当前 Snapshot。
func Install() map[string]any {
	return current().Install()
}

// Snapshot 返回进程级 Manager 的进度表。
func Snapshot() map[string]any {
	return current().Snapshot()
}

// Ensure 在 Look 已有路径时直接返回它。非 Windows 返回空路径和 nil。否则 beginInstall 并等待。ctx 为 nil 时用 Background。取消时返回 ctx.Err。
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

// Install 在 Look 为空时调用 beginInstall，忽略它的错误，然后返回 Snapshot。
func (m *Manager) Install() map[string]any {
	if m.Look() == "" {
		_, _ = m.beginInstall()
	}
	return m.Snapshot()
}

// beginInstall 在平台不受支持时把 phase 设为 error 并返回该错误。托管目录里已经有 pwsh 时返回 nil, nil。已有 wait 时返回同一个通道。否则后台跑 runInstall。
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

// runInstall 用 25 分钟超时跑 installBundle。成功时 phase 为 ready、percent 为 100。失败时 phase 为 error。结束时清掉自己的 wait、关闭通道并 publish。
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

// installBundle 下载 zip、去掉标记、解到 installDir.staging、校验，再改名为 installDir。失败时删掉 staging。成功后删掉 zip。Look 仍为空时返回 pwsh: install did not produce pwsh.exe。
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

// downloadZip 按 candidateURLs 的顺序尝试。第一个成功就返回。全部失败时返回最后一次错误；一次都没试到时错误文案是 pwsh: download failed。
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

// downloadURL 用 GET 下载到 dest.tmp。状态码大于等于 300 时返回 pwsh: download 加状态。读过 maxZipBytes 就失败。
// wantSHA 非空时比对 SHA-256，不一致返回 pwsh: sha256 mismatch。成功后改名为 dest 并 unblockPath。进度百分比最高 90。
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

// progressReader 在 Read 时累计字节，并在百分比变化或读到错误时调用 on。
type progressReader struct {
	// r 是被计数的读取端。
	r io.Reader
	// n 是已经读到的字节。
	n int64
	// total 是期望总字节，未知时为 0。
	total int64
	// last 是上次回调时的百分比，用来少发重复进度。
	last int
	// on 收到当前字节和总字节。
	on func(n, total int64)
}

// Read 读取底层数据，累加 n，并在百分比变化或 err 非空时调用 on。百分比按 n*90/total 计算。
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

// setProgress 更新进度。phase 为空不改 phase。percent 小于 0 不改 percent。total 小于等于 0 不改 total。archive 为空不改 archive。然后 publish。
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

// publish 在 Hub 非空时把 Snapshot 交给它。
func (m *Manager) publish() {
	if m.Hub == nil {
		return
	}
	m.Hub(m.Snapshot())
}

// Snapshot 返回给设置页和 RPC 的进度表。needed 在目标 GOOS 为空时看 runtime.GOOS 是否为 windows，否则看 m.GOOS。
// phase 为空时：已有二进制，或并不需要 pwsh，则 phase 为 ready、percent 为 100；需要但还没有则为 idle。
// ready 在已有路径或不需要时为 true。source 来自 sourceOf，shell 来自 Fact。
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

// sourceOf 在 bin 为空时返回空。与 YOYO_PWSH 指向同一文件时返回 env。与托管目录里找到的文件相同时返回 managed。否则返回 file。
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

// sameFile 把两边转成绝对路径再 Clean，然后不区分大小写比较。Abs 的错误被丢掉。
func sameFile(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
