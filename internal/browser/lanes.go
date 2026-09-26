package browser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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
	wsURL, err := versionWS(fmt.Sprintf("127.0.0.1:%d", port))
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
	h.mu.Lock()
	h.headed = true
	raw := strings.TrimSpace(h.url)
	cmd := h.cmd
	cdp := h.cdp
	h.cmd = nil
	h.cdp = nil
	h.mu.Unlock()
	if cdp != nil {
		cdp.close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	time.Sleep(200 * time.Millisecond)
	if err := h.ensureCDP(); err != nil {
		return err
	}
	if raw != "" {
		_ = h.navigateCDP(raw)
		time.Sleep(350 * time.Millisecond)
	}
	h.record("takeover", "headed")
	h.syncView()
	return nil
}

func (h *Host) Screenshot(path string) error {
	if err := h.ensureCDP(); err != nil {
		return err
	}
	raw, err := h.cdp.call("Page.captureScreenshot", map[string]any{"format": "png"})
	if err != nil {
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
