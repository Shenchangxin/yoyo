// 桌面更新检查、后台下载和安装入口。检查用 update.PublicKey，长度为 32 才继续。下载完成后仍由 ApplyUpdate 交给人确认安装。keymap.yaml 的读写也在这里。

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/isolation"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/update"
)

// isolated 返回 isolation.Status 的 Sandbox 字段。
func isolated() bool {
	return isolation.Status().Sandbox
}

// isolationKind 返回 isolation.Status 的 Kind。
func isolationKind() string {
	return isolation.Status().Kind
}

// loadKeymapFile 在 Config.Keymap 已有内容时不读盘。否则读取家目录 keymap.yaml，解析出非空 map 才写入配置。
func (a *App) loadKeymapFile() {
	if len(a.Config.Keymap) > 0 {
		return
	}
	b, err := os.ReadFile(filepath.Join(a.Home.Root, "keymap.yaml"))
	if err != nil {
		return
	}
	var m map[string]string
	if yaml.Unmarshal(b, &m) == nil && len(m) > 0 {
		a.Config.Keymap = m
	}
}

// writeKeymapFile 把 Config.Keymap 写成家目录 keymap.yaml。Keymap 为 nil 时写成空 map。
func (a *App) writeKeymapFile() {
	km := a.Config.Keymap
	if km == nil {
		km = map[string]string{}
	}
	b, err := yaml.Marshal(km)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(a.Home.Root, "keymap.yaml"), b, 0o644)
}

// updateState 保存检查阶段、进度、错误、上次检查、校验结果和已下载路径。busy 为真时 CheckUpdate 直接返回快照。notified 记住已经广播过的版本。
type updateState struct {
	mu        sync.Mutex
	phase     string
	percent   int
	bytes     int64
	total     int64
	err       string
	lastCheck time.Time
	result    update.CheckResult
	path      string
	busy      bool
	notified  string
}

// feedURL 用去掉空白的 Config.UpdateURL。该项为空时用 update.FeedURL(update.DefaultChannel)。
func (a *App) feedURL() string {
	if a != nil && strings.TrimSpace(a.Config.UpdateURL) != "" {
		return strings.TrimSpace(a.Config.UpdateURL)
	}
	return update.FeedURL(update.DefaultChannel)
}

// startUpdateLoop 只在桌面系统启动 updateLoop。App 为空则返回。
func (a *App) startUpdateLoop() {
	if a == nil || !update.DesktopOS() {
		return
	}
	if a.updateStop == nil {
		a.updateStop = make(chan struct{})
	}
	go a.updateLoop()
}

// updateLoop 首次等待 8 秒，之后间隔 6 小时。失败时退避加倍，最长 24 小时。暂停或 UpdateAutoCheck 为假时只重置到 6 小时，不检查。
func (a *App) updateLoop() {
	const startDelay = 8 * time.Second
	const interval = 6 * time.Hour
	const maxBackoff = 24 * time.Hour
	timer := time.NewTimer(startDelay)
	defer timer.Stop()
	backoff := interval
	for {
		select {
		case <-a.updateStop:
			return
		case <-timer.C:
		}
		if a.IsPaused() || !a.Config.UpdateAutoCheck {
			timer.Reset(interval)
			continue
		}
		out := a.CheckUpdate()
		if err, _ := out["error"].(string); err != "" {
			diaglog.Error("update poll failed", "component", "update", "err", err)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			timer.Reset(backoff)
			continue
		}
		backoff = interval
		timer.Reset(interval)
	}
}

// UpdateStatus 返回 updateSnapshot。
func (a *App) UpdateStatus() map[string]any {
	return a.updateSnapshot()
}

// updateSnapshot 汇总当前版本、暂存文件、订阅地址、自动检查和公钥是否为 32 字节。没有 upd 或 phase 为空时 phase 为 idle。暂存路径存在且 phase 为 ready 或 applying 时 downloaded 为真。
func (a *App) updateSnapshot() map[string]any {
	p := a.StagingPath()
	st, err := os.Stat(p)
	size := int64(0)
	staged := err == nil
	if staged {
		size = st.Size()
	}
	pub := update.PublicKey()
	out := map[string]any{
		"current":           a.Health(),
		"staging":           p,
		"staged":            staged,
		"size":              size,
		"url":               a.Config.UpdateURL,
		"feed":              a.feedURL(),
		"channel":           string(update.DefaultChannel),
		"auto_check":        a.Config.UpdateAutoCheck,
		"pubkey_configured": len(pub) == 32,
	}
	if a.upd == nil {
		out["phase"] = "idle"
		return out
	}
	a.upd.mu.Lock()
	defer a.upd.mu.Unlock()
	out["phase"] = a.upd.phase
	if a.upd.phase == "" {
		out["phase"] = "idle"
	}
	out["percent"] = a.upd.percent
	out["bytes"] = a.upd.bytes
	out["total"] = a.upd.total
	out["error"] = a.upd.err
	if !a.upd.lastCheck.IsZero() {
		out["last_check"] = a.upd.lastCheck.UTC().Format(time.RFC3339)
	}
	if a.upd.result.Verified {
		out["verified"] = true
		out["newer"] = a.upd.result.Newer
		out["manifest"] = a.upd.result.Manifest
		out["asset"] = a.upd.result.Asset
	}
	if a.upd.path != "" {
		if st, err := os.Stat(a.upd.path); err == nil {
			out["staged"] = true
			out["staging"] = a.upd.path
			out["size"] = st.Size()
			out["downloaded"] = a.upd.phase == "ready" || a.upd.phase == "applying"
		}
	}
	return out
}

// CheckUpdate 校验已签名的频道 JSON。公钥长度不是 32 时错误为 update public key missing。检查超时 30 秒。未通过校验或不是更新版本则回到 idle。桌面系统在后台下载，其他系统当场下载。同一版本已有非空暂存文件则直接标为 ready。
func (a *App) CheckUpdate() map[string]any {
	if a.upd == nil {
		a.upd = &updateState{}
	}
	a.upd.mu.Lock()
	if a.upd.busy {
		a.upd.mu.Unlock()
		return a.updateSnapshot()
	}
	a.upd.busy = true
	a.upd.phase = "checking"
	a.upd.err = ""
	a.upd.percent = 0
	a.upd.mu.Unlock()

	pub := update.PublicKey()
	if len(pub) != 32 {
		a.finishUpdateErr("update public key missing")
		return a.updateSnapshot()
	}
	feed := a.feedURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	res, err := update.Check(ctx, feed, nil, nil, pub)
	cancel()
	if err != nil {
		diaglog.Error("update check failed", "component", "update", "err", err)
		a.finishUpdateErr(err.Error())
		return a.updateSnapshot()
	}
	res.Current = update.Current(update.DefaultChannel)
	res.Newer = update.IsNewer(res.Manifest.Version, res.Current.Version)
	diaglog.Info("update check", "component", "update", "verified", res.Verified, "newer", res.Newer, "version", res.Manifest.Version)

	a.upd.mu.Lock()
	a.upd.result = res
	a.upd.lastCheck = time.Now()
	a.upd.mu.Unlock()

	if !res.Verified || !res.Newer {
		a.upd.mu.Lock()
		a.upd.busy = false
		a.upd.phase = "idle"
		a.upd.mu.Unlock()
		return a.updateSnapshot()
	}
	if res.Asset.URL == "" {
		a.finishUpdateErr("no package for this platform")
		return a.updateSnapshot()
	}
	dir := filepath.Dir(a.StagingPath())
	if p, err := update.ReadPending(dir); err == nil && p.Version == res.Manifest.Version && p.Staging != "" {
		if st, err := os.Stat(p.Staging); err == nil && st.Size() > 0 {
			a.upd.mu.Lock()
			a.upd.path = p.Staging
			a.upd.phase = "ready"
			a.upd.busy = false
			a.upd.percent = 100
			a.upd.bytes = st.Size()
			a.upd.total = st.Size()
			a.upd.mu.Unlock()
			a.notifyUpdateReady(res.Manifest.Version)
			return a.updateSnapshot()
		}
	}
	if update.DesktopOS() {
		go a.downloadUpdate(res)
		return a.updateSnapshot()
	}
	a.downloadUpdate(res)
	return a.updateSnapshot()
}

// downloadUpdate 用 25 分钟超时下载。进度百分比在完成前最高写到 99。成功后 MarkPending，phase 改为 ready，并 notifyUpdateReady。
func (a *App) downloadUpdate(res update.CheckResult) {
	dir := filepath.Dir(a.StagingPath())
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	a.upd.mu.Lock()
	a.upd.phase = "downloading"
	a.upd.mu.Unlock()
	path, err := update.Download(ctx, res.Asset, dir, func(done, total int64) {
		pct := 0
		if total > 0 {
			pct = int(done * 100 / total)
			if pct > 99 {
				pct = 99
			}
		}
		a.upd.mu.Lock()
		a.upd.bytes = done
		a.upd.total = total
		a.upd.percent = pct
		a.upd.mu.Unlock()
	})
	if err != nil {
		diaglog.Error("update download failed", "component", "update", "err", err)
		a.finishUpdateErr(err.Error())
		return
	}
	_ = update.MarkPending(dir, update.Pending{
		Staging:  path,
		SHA256:   res.Asset.SHA256,
		Kind:     res.Asset.Kind,
		Version:  res.Manifest.Version,
		Filename: res.Asset.Filename,
	})
	st, _ := os.Stat(path)
	size := int64(0)
	if st != nil {
		size = st.Size()
	}
	a.upd.mu.Lock()
	a.upd.path = path
	a.upd.phase = "ready"
	a.upd.busy = false
	a.upd.err = ""
	a.upd.percent = 100
	a.upd.bytes = size
	a.upd.total = size
	a.upd.mu.Unlock()
	diaglog.Info("update downloaded", "component", "update", "version", res.Manifest.Version, "path", path, "size", size)
	a.notifyUpdateReady(res.Manifest.Version)
}

// finishUpdateErr 把 phase 设为 error，清掉 busy，并记下错误文本。
func (a *App) finishUpdateErr(msg string) {
	a.upd.mu.Lock()
	a.upd.busy = false
	a.upd.phase = "error"
	a.upd.err = msg
	a.upd.mu.Unlock()
}

// notifyUpdateReady 每个版本只向 Hub 发一次 system 事件。Hub 为空则只记下已通知的版本。
func (a *App) notifyUpdateReady(ver string) {
	if a.upd == nil {
		return
	}
	a.upd.mu.Lock()
	if a.upd.notified == ver {
		a.upd.mu.Unlock()
		return
	}
	a.upd.notified = ver
	a.upd.mu.Unlock()
	if a.Hub == nil {
		return
	}
	a.Hub.Publish(trace.Event{
		TS:     time.Now().UTC(),
		Type:   trace.TypeSystem,
		Source: "update",
		Payload: map[string]any{
			"version": ver,
			"title":   "Yoyo " + ver,
			"text":    "Update downloaded. Install from the download icon when ready.",
		},
	})
}

// ApplyUpdate 把 phase 设为 applying，再调用 update.ApplyDownloaded。ErrQuitNeeded 和 ErrOpenedPackage 原样返回。其他错误把 phase 设为 error。
func (a *App) ApplyUpdate(exe string) error {
	if a.upd != nil {
		a.upd.mu.Lock()
		a.upd.phase = "applying"
		a.upd.mu.Unlock()
	}
	diaglog.Info("update apply", "component", "update", "staging", a.StagingPath())
	err := update.ApplyDownloaded(exe, filepath.Dir(a.StagingPath()))
	if err != nil && !errors.Is(err, update.ErrQuitNeeded) && !errors.Is(err, update.ErrOpenedPackage) {
		if a.upd != nil {
			a.upd.mu.Lock()
			a.upd.phase = "error"
			a.upd.err = err.Error()
			a.upd.mu.Unlock()
		}
		return err
	}
	return err
}
