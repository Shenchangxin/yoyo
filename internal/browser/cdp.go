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
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type cdpConn struct {
	mu   sync.Mutex
	ws   *websocket.Conn
	next int
}

func (c *cdpConn) close() {
	if c == nil || c.ws == nil {
		return
	}
	_ = c.ws.Close(websocket.StatusNormalClosure, "")
	c.ws = nil
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

func (h *Host) ensureCDP() error {
	h.mu.Lock()
	if h.cdp != nil {
		h.mu.Unlock()
		return nil
	}
	h.mu.Unlock()
	bin := findChrome()
	if bin == "" {
		return fmt.Errorf("browser: no isolated Chromium (set YOYO_CHROME)")
	}
	port, err := freePort()
	if err != nil {
		return err
	}
	args := []string{
		"--user-data-dir=" + h.profile,
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-sync",
		"--disable-extensions",
	}
	h.mu.Lock()
	headed := h.headed
	h.mu.Unlock()
	if !headed {
		args = append(args, "--headless=new", "--disable-gpu")
	}
	args = append(args, "about:blank")
	cmd := exec.Command(bin, args...)
	cmd.Dir = h.dir
	if err := cmd.Start(); err != nil {
		return err
	}
	wsURL := ""
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		wsURL, err = versionWS("127.0.0.1:" + strconv.Itoa(port))
		if err == nil && wsURL != "" {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if wsURL == "" {
		_ = cmd.Process.Kill()
		return fmt.Errorf("browser: chrome debug port did not come up")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	h.mu.Lock()
	h.cmd = cmd
	h.cdp = &cdpConn{ws: ws, next: 1}
	h.port = port
	h.mu.Unlock()
	_, _ = h.cdp.call("Page.enable", nil)
	_, _ = h.cdp.call("Network.enable", nil)
	return nil
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

func (c *cdpConn) call(method string, params any) (json.RawMessage, error) {
	if c == nil || c.ws == nil {
		return nil, fmt.Errorf("browser: no CDP")
	}
	c.mu.Lock()
	c.next++
	id := c.next
	c.mu.Unlock()
	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, c.ws, msg); err != nil {
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
		if err := wsjson.Read(ctx, c.ws, &reply); err != nil {
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
	_, err := h.cdp.call("Page.navigate", map[string]any{"url": raw})
	return err
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
