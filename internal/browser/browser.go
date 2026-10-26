// Package browser 拉起一份独立用户目录的 Chromium，用 CDP 打开页面、点击、填表和截图。调试端口由 freePort 分配，写进启动参数 --remote-debugging-port。默认通道是 isolated。
// attached 通道连接 127.0.0.1 上已有的调试端口，默认 9222，环境变量 YOYO_CHROME_DEBUG 可以改这个端口。连不上时错误文案带上这个端口号。
// 命名配置的 cookie 在目录下的 profiles/<name>。配置名 attached 被拒绝。打开、点击和截图走本包的 Host，不调用 runtime.Run。
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Snapshot 是一次页面观察。Actions 每行一个可见控件，选择器可供点击使用。Errors 是上次浏览器调用以来页面抛出的 JavaScript 错误，没有则为空。Elements 是最近一次实况控件遍历。
type Snapshot struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Text    string `json:"text"`
	Profile string `json:"profile"`
	// Actions 每行一个可见的可交互元素，并带上 browser_click 能用的选择器。
	Actions string `json:"actions,omitempty"`
	// Errors 是自上次浏览器调用以来页面抛出的 JavaScript 错误。页面没抛时为空。
	Errors string `json:"errors,omitempty"`
	// Elements 是最近一次实况控件遍历。结构化快照用它。
	Elements []Action `json:"elements,omitempty"`
}

// Host 是本进程里的隔离浏览器。profile 非空时 Isolated 为真。port 是当前调试端口，隔离启动时来自 freePort，attached 通道才是 9222 或 YOYO_CHROME_DEBUG。
type Host struct {
	mu            sync.Mutex
	dir           string
	url           string
	title         string
	profile       string
	profileName   string
	targetID      string
	lane          string
	log           []map[string]any
	cmd           *exec.Cmd
	cdp           *cdpConn
	port          int
	headed        bool
	snap          string
	frame         string
	frameAt       time.Time
	syncing       bool
	preview       string
	fileSrv       *http.Server
	fileRoot      string
	fileURL       string
	killTree      func()
	lastClick     string
	actions       string
	lastShotSum   [32]byte
	hasShot       bool
	lastShotN     int
	shotUnchanged bool
	refs          refCache
	rec           *recorder
	tabCount      int
}

// Open 在 dir 下创建 profile 目录，权限 0o700，并返回宿主。配置名先是 default。
func Open(dir string) (*Host, error) {
	p := filepath.Join(dir, "profile")
	if err := os.MkdirAll(p, 0o700); err != nil {
		return nil, err
	}
	return &Host{dir: dir, profile: p, profileName: "default"}, nil
}

// Isolated 在宿主非空且 profile 非空时返回真。
func (h *Host) Isolated() bool { return h != nil && h.profile != "" }

// Close 关掉本地文件服务，并停掉本进程拉起的 Chrome。
func (h *Host) Close() {
	h.closeFileServer()
	h.stopChrome()
}

// stopChrome 关掉 CDP、结束进程树，把端口和标签计数清零，再等用户目录锁放开。
func (h *Host) stopChrome() {
	h.mu.Lock()
	cmd := h.cmd
	cdp := h.cdp
	killTree := h.killTree
	h.cmd = nil
	h.cdp = nil
	h.killTree = nil
	h.port = 0
	h.tabCount = 0
	h.mu.Unlock()
	if cdp != nil {
		cdp.close()
	}
	stopChromeProc(cmd, killTree)
	waitProfileUnlocked(h.profile, 4*time.Second)
}

// stopChromeProc 先调用 killTree，再杀掉进程并最多等 3 秒。
func stopChromeProc(cmd *exec.Cmd, killTree func()) {
	if killTree != nil {
		killTree()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
}

// OpenURL 只接受 http 或 https。先用本进程的 HTTP 客户端取一页文本，再尝试用 CDP 导航；CDP 成功时改用页面标题、正文、可见控件和页面错误。
func (h *Host) OpenURL(raw string) (Snapshot, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Snapshot{}, fmt.Errorf("browser: need http(s) URL")
	}
	h.mu.Lock()
	h.url = u.String()
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Snapshot{}, err
	}
	req.Header.Set("User-Agent", "YoyoBrowser/0.3")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Snapshot{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	text := strip(string(body))
	if len(text) > 8000 {
		text = text[:8000]
	}
	snap := Snapshot{URL: u.String(), Title: u.Host, Text: text, Profile: h.profile}
	if err := h.navigateCDP(u.String()); err == nil {
		time.Sleep(350 * time.Millisecond)
		h.syncView()
		h.refreshActions()
		snap.Errors = h.takePageErrors()
		h.mu.Lock()
		if h.title != "" {
			snap.Title = h.title
		}
		if h.snap != "" {
			snap.Text = h.snap
		}
		snap.Actions = h.actions
		h.mu.Unlock()
	}
	h.record("open", raw)
	return snap, nil
}

// LastAction 返回最近一次点击的观察文字。宿主为空时返回空字符串。
func (h *Host) LastAction() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastClick
}

// ShotMeta 返回最近一张截图是否与上一张相同、哈希前 8 字节的十六进制，以及字节数。还没有截图时三个结果都是零值。
func (h *Host) ShotMeta() (unchanged bool, sum string, n int) {
	if h == nil {
		return false, "", 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.hasShot {
		return false, "", 0
	}
	return h.shotUnchanged, fmt.Sprintf("%x", h.lastShotSum[:8]), h.lastShotN
}

// Download 把 http 或 https 地址下载到 dest，最多 64 MiB。
func (h *Host) Download(raw, dest string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("browser: need http(s) URL")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "YoyoBrowser/0.3")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, io.LimitReader(resp.Body, 64<<20))
	h.record("download", raw)
	return err
}

// Log 返回操作日志的副本。
func (h *Host) Log() []map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]map[string]any(nil), h.log...)
}

// record 追加一条操作并整份写到目录里的 journal.json，权限 0o600。
func (h *Host) record(op, detail string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.log = append(h.log, map[string]any{"op": op, "detail": detail, "ts": time.Now().UTC(), "url": h.url, "profile": h.profile})
	raw, _ := json.Marshal(h.log)
	_ = os.WriteFile(filepath.Join(h.dir, "journal.json"), raw, 0o600)
}

// strip 去掉尖括号里的内容，把剩下的空白收成单空格。
func strip(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch r {
		case '<':
			in = true
		case '>':
			in = false
			b.WriteByte(' ')
		default:
			if !in {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
