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

func (h *Host) StartTakeover() error {
	return h.StartTakeoverAt("")
}

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

func isBlankURL(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return true
	}
	low := strings.ToLower(s)
	return low == "about:blank" || strings.HasPrefix(low, "about:blank?")
}

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

func dirOf(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i <= 0 {
		return "."
	}
	return path[:i]
}
