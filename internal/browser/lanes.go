// 本文件拥有通道、接管窗口、截图和 cookie 导入。
// 默认通道是 isolated，调试端口由 startChromeCDP 用 freePort 分配。attached 才去连已有的调试端口，默认 9222，环境变量 YOYO_CHROME_DEBUG 可以改成别的正整数。连不上时报错，文案里带上这个端口。
// ImportCookies 读 Netscape 格式的 cookie 文件，经 Network.setCookie 写入当前页。不调用 runtime.Run。

package browser

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
)

// OpenURLLane 按通道打开地址。地址里含 lane=attached 时改走 attached，并去掉这段字。attached 调用 attachDebugPort，其余把 lane 写成 isolated，然后 OpenURL。
func (h *Host) OpenURLLane(raw, lane string) (Snapshot, error) {
	lane = strings.ToLower(strings.TrimSpace(lane))
	if strings.Contains(strings.ToLower(raw), "lane=attached") {
		lane = "attached"
		raw = strings.TrimSpace(strings.ReplaceAll(raw, "lane=attached", ""))
	}
	if lane == "attached" {
		if err := h.attachDebugPort(); err != nil {
			return Snapshot{}, err
		}
	} else {
		h.mu.Lock()
		h.lane = "isolated"
		h.mu.Unlock()
	}
	return h.OpenURL(raw)
}

// attachDebugPort 连接 127.0.0.1 上已有的 Chrome 调试端口。默认 9222；YOYO_CHROME_DEBUG 是正整数时用那个端口。失败时错误是 browser: attached lane needs Chrome --remote-debugging-port=%d。成功前会 Close，profile 写成 attached:<端口>，lane 写成 attached。
func (h *Host) attachDebugPort() error {
	port := 9222
	if v := os.Getenv("YOYO_CHROME_DEBUG"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			port = n
		}
	}
	wsURL, err := debugWS(fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("browser: attached lane needs Chrome --remote-debugging-port=%d: %w", port, err)
	}
	h.Close()
	h.mu.Lock()
	h.port = port
	h.profile = "attached:" + strconv.Itoa(port)
	h.lane = "attached"
	h.mu.Unlock()
	return h.dialWS(wsURL)
}

// StartTakeover 用当前预览路径打开有头窗口。等价于 StartTakeoverAt 传入空字符串。
func (h *Host) StartTakeover() error {
	return h.StartTakeoverAt("")
}

// StartTakeoverAt 停掉当前 Chrome，再用有头模式打开接管页。CDP 打不开时改用系统浏览器。abs 非空时先设成预览路径。
func (h *Host) StartTakeoverAt(abs string) error {
	h.mu.Lock()
	raw := strings.TrimSpace(h.url)
	preview := h.preview
	h.mu.Unlock()
	if abs != "" {
		h.SetPreview(abs)
		preview = abs
	}
	target := takeoverTarget(raw, preview, h.serveFileURL)
	if isBlankURL(target) {
		if preview != "" {
			return h.openTakeoverSystem(target, preview)
		}
		if !isBlankURL(raw) {
			target = raw
		}
	}
	if isBlankURL(target) {
		return fmt.Errorf("browser: nothing to take over")
	}
	h.mu.Lock()
	h.url = target
	h.mu.Unlock()
	h.stopChrome()
	if err := h.startChromeCDP(target, true); err != nil {
		if openErr := h.openTakeoverSystem(target, preview); openErr == nil {
			return nil
		}
		return err
	}
	if err := h.showTakeover(target); err != nil {
		h.stopChrome()
		if openErr := h.openTakeoverSystem(target, preview); openErr == nil {
			return nil
		}
		return err
	}
	h.record("takeover", "headed")
	h.syncView()
	return nil
}

// takeoverTarget 优先用预览文件经 serve 得到的地址。预览为空或 serve 失败时，返回非空白的 liveURL。
func takeoverTarget(liveURL, preview string, serve func(string) (string, error)) string {
	liveURL = strings.TrimSpace(liveURL)
	if isBlankURL(liveURL) {
		liveURL = ""
	}
	preview = strings.TrimSpace(preview)
	if preview != "" && serve != nil {
		if u, err := serve(preview); err == nil && !isBlankURL(u) {
			return strings.TrimSpace(u)
		}
	}
	return liveURL
}

// isBlankURL 在空白、about:blank，或以 about:blank? 开头时返回真。
func isBlankURL(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return true
	}
	low := strings.ToLower(s)
	return low == "about:blank" || strings.HasPrefix(low, "about:blank?")
}

// showTakeover 最多导航 6 次，直到 location.href 不再是空白页。
func (h *Host) showTakeover(target string) error {
	if isBlankURL(target) {
		return fmt.Errorf("browser: blank takeover url")
	}
	var last error
	for i := 0; i < 6; i++ {
		if err := h.pageNavigate(target); err != nil {
			last = err
			time.Sleep(200 * time.Millisecond)
			continue
		}
		time.Sleep(180 * time.Millisecond)
		href, _ := h.evalOnConn("location.href")
		if !isBlankURL(cleanJS(href)) {
			return nil
		}
		last = fmt.Errorf("browser: takeover stayed at about:blank")
		time.Sleep(200 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("browser: takeover stayed at about:blank")
	}
	return last
}

// openTakeoverSystem 用 hostopen 打开地址或本地预览文件，并记下 takeover 为 system-browser。两边都打不开就返回错误。
func (h *Host) openTakeoverSystem(target, preview string) error {
	opened := false
	if !isBlankURL(target) && hostopen.URL(target) == nil {
		opened = true
	} else if strings.TrimSpace(preview) != "" && hostopen.Path(preview) == nil {
		opened = true
	}
	if !opened {
		if strings.TrimSpace(preview) != "" {
			return fmt.Errorf("browser: could not open %s", preview)
		}
		return fmt.Errorf("browser: could not open takeover page")
	}
	h.record("takeover", "system-browser")
	h.mu.Lock()
	h.headed = true
	if isBlankURL(h.url) {
		h.url = target
	}
	h.mu.Unlock()
	h.syncView()
	return nil
}

// Screenshot 对当前页截图写到 path。PNG 帧过大时改用质量 72 的 JPEG。连接已断时错误文案是 isolated Chrome closed the debug connection。成功后记下 SHA-256，与上一张相同时 ShotMeta 的 unchanged 为真。
func (h *Host) Screenshot(path string) error {
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return err
	}
	params := map[string]any{"format": "png"}
	raw, err := h.callCDP("Page.captureScreenshot", params)
	if err != nil && isCDPMessageTooBig(err) {
		raw, err = h.callCDP("Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 72})
	}
	if err != nil {
		if isCDPGone(err) {
			return fmt.Errorf("browser screenshot failed: isolated Chrome closed the debug connection; retry after the page loads")
		}
		if isCDPMessageTooBig(err) {
			return fmt.Errorf("browser screenshot failed: CDP frame exceeded %d bytes after jpeg fallback", cdpMaxMessage)
		}
		return err
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	b, err := base64.StdEncoding.DecodeString(out.Data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	h.mu.Lock()
	h.shotUnchanged = h.hasShot && h.lastShotSum == sum
	h.lastShotSum = sum
	h.hasShot = true
	h.lastShotN = len(b)
	h.mu.Unlock()
	h.syncView()
	return nil
}

// ImportCookies 按 Netscape 的制表符行调用 Network.setCookie。至少 7 列，# 开头的行跳过。返回成功写入的条数，并在日志里记下 cookies。
func (h *Host) ImportCookies(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if err := h.ensureCDP(); err != nil {
		return 0, err
	}
	n := 0
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}
		cookie := map[string]any{
			"name":   parts[5],
			"value":  parts[6],
			"domain": parts[0],
			"path":   parts[2],
			"secure": strings.EqualFold(parts[3], "TRUE"),
		}
		if _, err := h.cdp.call("Network.setCookie", cookie); err == nil {
			n++
		}
	}
	h.record("cookies", path)
	return n, nil
}

// dirOf 返回路径里最后一个斜杠之前的目录。没有斜杠或斜杠在开头时返回 "."。
func dirOf(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i <= 0 {
		return "."
	}
	return path[:i]
}
