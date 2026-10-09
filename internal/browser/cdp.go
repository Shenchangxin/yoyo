package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type cdpConn struct {
	mu     sync.Mutex
	callMu sync.Mutex
	ws     *websocket.Conn
	next   int
}

func (c *cdpConn) close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	ws := c.ws
	c.ws = nil
	c.mu.Unlock()
	if ws != nil {
		_ = ws.Close(websocket.StatusNormalClosure, "")
	}
}

func findChrome() string {
	if v := os.Getenv("YOYO_CHROME"); v != "" {
		return v
	}
	var cands []string
	switch runtime.GOOS {
	case "windows":
		cands = []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		}
	case "darwin":
		cands = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default:
		cands = []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge"}
	}
	for _, c := range cands {
		if c == "" {
			continue
		}
		if filepath.IsAbs(c) {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (h *Host) chromeProfile(headed bool) string {
	if headed {
		return filepath.Join(h.dir, "profile-headed")
	}
	if h.dir != "" {
		return filepath.Join(h.dir, "profile")
	}
	return h.profile
}

func (h *Host) ensureCDP() error {
	return h.ensureChrome("")
}

func (h *Host) ensureChrome(startURL string) error {
	headed := false
	if h != nil {
		h.mu.Lock()
		headed = h.headed
		h.mu.Unlock()
	}
	err := h.startChromeCDP(startURL, headed)
	if err == nil {
		return nil
	}
	if headed {
		return err
	}
	if err2 := h.startChromeCDP(startURL, true); err2 != nil {
		return err
	}
	return nil
}

func (h *Host) startChromeCDP(startURL string, headed bool) error {
	h.mu.Lock()
	c := h.cdp
	same := c != nil && h.headed == headed
	h.mu.Unlock()
	if same && cdpAlive(c) {
		return nil
	}
	if h.cdp != nil || h.cmd != nil {
		h.stopChrome()
	}
	bin := findChrome()
	if bin == "" {
		return fmt.Errorf("browser: no isolated Chromium (set YOYO_CHROME)")
	}
	profile := h.chromeProfile(headed)
	if err := os.MkdirAll(profile, 0o700); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(profile, "DevToolsActivePort"))
	if headed {
		resetChromeSession(profile)
	}
	waitProfileUnlocked(profile, 3*time.Second)
	port := 0
	if p, err := freePort(); err == nil {
		port = p
	}
	args := chromeLaunchArgs(profile, port, headed, startURL)
	cmd := exec.Command(bin, args...)
	cmd.Dir = h.dir
	prepareChromeCmd(cmd, headed)
	errPath := filepath.Join(profile, "chrome.err.log")
	errFile, err := os.Create(errPath)
	if err == nil {
		cmd.Stderr = errFile
	}
	if err := cmd.Start(); err != nil {
		if errFile != nil {
			_ = errFile.Close()
		}
		return err
	}
	var killTree func()
	if !headed {
		killTree = bindChromeJob(cmd)
	}
	wait := chromeReadyWait
	if headed {
		wait = chromeHeadedReadyWait
	}
	deadline := time.Now().Add(wait)
	got, err := waitChromeDebug(profile, port, deadline)
	if err != nil {
		stopChromeProc(cmd, killTree)
		if errFile != nil {
			_ = errFile.Close()
		}
		return chromeReadyErr(err, readFileTail(errPath, 400))
	}
	wsURL, err := waitDebugPageWS("127.0.0.1:"+strconv.Itoa(got), startURL, deadline)
	if err != nil || wsURL == "" {
		stopChromeProc(cmd, killTree)
		if errFile != nil {
			_ = errFile.Close()
		}
		if err == nil {
			err = fmt.Errorf("browser: chrome debug port did not come up")
		}
		return chromeReadyErr(err, readFileTail(errPath, 400))
	}
	if err := h.dialWS(wsURL); err != nil {
		stopChromeProc(cmd, killTree)
		if errFile != nil {
			_ = errFile.Close()
		}
		return err
	}
	h.mu.Lock()
	h.cmd = cmd
	h.port = got
	h.killTree = killTree
	h.profile = profile
	h.headed = headed
	h.mu.Unlock()
	return nil
}

func chromeReadyErr(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	if len(stderr) > 400 {
		stderr = stderr[len(stderr)-400:]
	}
	return fmt.Errorf("%w (%s)", err, stderr)
}

func readFileTail(path string, n int) string {
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if n > 0 && len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

func (h *Host) dialWS(wsURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.cdp = &cdpConn{ws: ws, next: 1}
	h.mu.Unlock()
	_, _ = h.cdp.call("Page.enable", nil)
	_, _ = h.cdp.call("Network.enable", nil)
	return nil
}

func (h *Host) dropDeadCDP() {
	if h == nil {
		return
	}
	h.mu.Lock()
	c := h.cdp
	h.cdp = nil
	h.mu.Unlock()
	if c != nil {
		c.close()
	}
}

func versionWS(addr string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/json/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var doc struct {
		WS string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", err
	}
	if doc.WS == "" {
		return "", fmt.Errorf("no websocket")
	}
	return doc.WS, nil
}

func cdpAlive(c *cdpConn) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	ws := c.ws
	c.mu.Unlock()
	if ws == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	return ws.Ping(ctx) == nil
}

func isCDPGone(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, n := range []string{
		"use of closed network connection",
		"failed to write frame",
		"failed to write json",
		"failed to write msg",
		"websocket: close",
		"wsasend",
		"wsarecv",
		"broken pipe",
		"connection reset",
		"connection aborted",
	} {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func (h *Host) previewStartURL() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	preview := h.preview
	cur := h.url
	h.mu.Unlock()
	if preview != "" {
		if u, err := h.serveFileURL(preview); err == nil {
			return u
		}
	}
	if !isBlankURL(cur) {
		return cur
	}
	return ""
}

func (h *Host) reconnectChrome() error {
	start := h.previewStartURL()
	h.stopChrome()
	return h.ensureChrome(start)
}

func (h *Host) callCDP(method string, params any) (json.RawMessage, error) {
	if h == nil {
		return nil, fmt.Errorf("browser: no host")
	}
	h.mu.Lock()
	c := h.cdp
	h.mu.Unlock()
	if c == nil {
		return nil, fmt.Errorf("browser: no CDP")
	}
	raw, err := c.call(method, params)
	if err == nil || !isCDPGone(err) {
		return raw, err
	}
	if recErr := h.reconnectChrome(); recErr != nil {
		return nil, fmt.Errorf("browser: chrome disconnected; relaunch failed: %w", recErr)
	}
	h.mu.Lock()
	c = h.cdp
	h.mu.Unlock()
	if c == nil {
		return nil, fmt.Errorf("browser: chrome disconnected")
	}
	return c.call(method, params)
}

func (c *cdpConn) call(method string, params any) (json.RawMessage, error) {
	if c == nil {
		return nil, fmt.Errorf("browser: no CDP")
	}
	c.callMu.Lock()
	defer c.callMu.Unlock()
	c.mu.Lock()
	ws := c.ws
	c.next++
	id := c.next
	c.mu.Unlock()
	if ws == nil {
		return nil, fmt.Errorf("browser: no CDP")
	}
	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, ws, msg); err != nil {
		return nil, err
	}
	for {
		var reply struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
			Method string `json:"method"`
		}
		if err := wsjson.Read(ctx, ws, &reply); err != nil {
			return nil, err
		}
		if reply.ID != id {
			continue
		}
		if reply.Error != nil {
			return nil, fmt.Errorf("cdp %s: %s", method, reply.Error.Message)
		}
		return reply.Result, nil
	}
}

func (h *Host) navigateCDP(raw string) error {
	if err := h.ensureCDP(); err != nil {
		return err
	}
	return h.pageNavigate(raw)
}

func (h *Host) pageNavigate(raw string) error {
	raw = strings.TrimSpace(raw)
	if h.cdp == nil {
		return fmt.Errorf("browser: no CDP")
	}
	if isBlankURL(raw) {
		return fmt.Errorf("browser: refusing blank navigation")
	}
	if _, err := h.callCDP("Page.navigate", map[string]any{"url": raw}); err == nil {
		h.mu.Lock()
		h.url = raw
		h.mu.Unlock()
		return nil
	} else if _, err2 := h.callCDP("Target.createTarget", map[string]any{"url": raw}); err2 == nil {
		h.mu.Lock()
		h.url = raw
		h.mu.Unlock()
		return nil
	} else {
		return err
	}
}

func (h *Host) evalJS(expr string) (string, error) {
	if err := h.ensureCDP(); err != nil {
		return "", err
	}
	return h.evalOnConn(expr)
}

func (h *Host) evalOnConn(expr string) (string, error) {
	if h.cdp == nil {
		return "", fmt.Errorf("browser: no CDP")
	}
	raw, err := h.cdp.call("Runtime.evaluate", map[string]any{
		"expression":    expr,
		"returnByValue": true,
	})
	if err != nil {
		return "", err
	}
	var out struct {
		Result struct {
			Value any `json:"value"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &out)
	return fmt.Sprint(out.Result.Value), nil
}

func (h *Host) clickCDP(sel string) error {
	js := fmt.Sprintf(`(() => { const el = document.querySelector(%q); if (!el) return "missing"; el.click(); return "ok"; })()`, sel)
	out, err := h.evalJS(js)
	if err != nil {
		return err
	}
	if out != "ok" {
		return fmt.Errorf("browser: selector %s not found", sel)
	}
	return nil
}

func (h *Host) typeCDP(sel, text string) error {
	js := fmt.Sprintf(`(() => { const el = document.querySelector(%q); if (!el) return "missing"; el.focus(); el.value = %q; el.dispatchEvent(new Event("input", {bubbles:true})); return "ok"; })()`, sel, text)
	out, err := h.evalJS(js)
	if err != nil {
		return err
	}
	if out != "ok" {
		return fmt.Errorf("browser: selector %s not found", sel)
	}
	return nil
}
