package browser

import (
	"context"
	"encoding/json"
	"errors"
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

// cdpCallTimeout bounds one CDP round trip. A slow page must not tear the
// connection down; only a dead socket does.
const cdpCallTimeout = 12 * time.Second

// cdpConn is one CDP websocket. A single readLoop goroutine owns every Read:
// replies are matched to callers by id and the connection is alive exactly
// until the reader observes a failure or the host closes it.
//
// Liveness used to be probed with ws.Ping. coder/websocket only processes a
// pong while something is reading, so an idle, healthy connection always
// looked dead. The host then relaunched Chrome onto about:blank on every
// click (session 3b9f1bbb660e4236: every click reported "selector not found").
type cdpConn struct {
	ws      *websocket.Conn
	writeMu sync.Mutex

	mu      sync.Mutex
	next    int
	pending map[int]chan cdpReply
	done    chan struct{}
	err     error

	// errs holds the JavaScript errors the page raised since the last browser
	// call. The reader fills it from protocol events; the host drains it.
	errs pageErrors
}

type cdpReply struct {
	result json.RawMessage
	err    error
}

func newCDPConn(ws *websocket.Conn) *cdpConn {
	c := &cdpConn{
		ws:      ws,
		next:    1,
		pending: map[int]chan cdpReply{},
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c
}

func (c *cdpConn) readLoop() {
	for {
		var msg struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := wsjson.Read(context.Background(), c.ws, &msg); err != nil {
			c.fail(err)
			return
		}
		if msg.ID == 0 {
			// Protocol event. Only page errors are kept; the rest is ignored.
			if msg.Method != "" {
				c.errs.add(msg.Method, msg.Params)
			}
			continue
		}
		c.mu.Lock()
		ch := c.pending[msg.ID]
		delete(c.pending, msg.ID)
		c.mu.Unlock()
		if ch == nil {
			continue
		}
		r := cdpReply{result: msg.Result}
		if msg.Error != nil {
			r.err = errors.New(msg.Error.Message)
		}
		ch <- r
	}
}

// fail marks the connection dead exactly once, wakes every waiter, and
// releases the socket. Waiters receive an error that isCDPGone recognises.
func (c *cdpConn) fail(cause error) {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}
	c.err = fmt.Errorf("websocket: closed: %w", cause)
	e := c.err
	pending := c.pending
	c.pending = map[int]chan cdpReply{}
	close(c.done)
	c.mu.Unlock()
	_ = c.ws.CloseNow()
	for _, ch := range pending {
		ch <- cdpReply{err: e}
	}
}

func (c *cdpConn) close() {
	if c == nil {
		return
	}
	c.fail(errors.New("closed by host"))
}

func (c *cdpConn) alive() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err == nil
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

// headedForAutomation keeps a live takeover window, but drops a dead headed
// profile so screenshot/click can relaunch isolated Chrome (session ac0738fbdfb801ac).
func headedForAutomation(headed, cdpAlive bool) bool {
	if headed && !cdpAlive {
		return false
	}
	return headed
}

func (h *Host) ensureChrome(startURL string) error {
	headed := false
	alive := false
	if h != nil {
		h.mu.Lock()
		headed = h.headed
		alive = cdpAlive(h.cdp)
		h.mu.Unlock()
	}
	headed = headedForAutomation(headed, alive)
	if h != nil && !headed {
		h.mu.Lock()
		h.headed = false
		h.mu.Unlock()
	}
	err := h.startChromeCDP(startURL, headed)
	if err == nil {
		return nil
	}
	if headed {
		h.mu.Lock()
		h.headed = false
		h.mu.Unlock()
		if err2 := h.startChromeCDP(startURL, false); err2 == nil {
			return nil
		}
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
	if strings.TrimSpace(startURL) == "" {
		// A relaunch with no URL used to land on about:blank, silently
		// replacing the game page the agent was operating on.
		startURL = h.previewStartURL()
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

// cdpMaxMessage is the CDP websocket read cap. coder/websocket defaults to
// 32KiB+1, which cannot hold Page.captureScreenshot PNG/JPEG (typically 100KiB–8MiB).
// Playwright/Puppeteer raise this into the 100–256MiB range; 64MiB covers 4K PNG.
const cdpMaxMessage = 64 << 20

func (h *Host) dialWS(wsURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	ws.SetReadLimit(cdpMaxMessage)
	conn := newCDPConn(ws)
	h.mu.Lock()
	old := h.cdp
	h.cdp = conn
	h.mu.Unlock()
	old.close() // a replaced socket must not keep its reader or target alive
	_, _ = conn.call("Page.enable", nil)
	_, _ = conn.call("Network.enable", nil)
	// Runtime events carry uncaught exceptions and console.error calls. They
	// are the only signal a page failure leaves behind; see page_errors.go.
	_, _ = conn.call("Runtime.enable", nil)
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

// cdpAlive reports whether the connection's reader is still running. It is a
// local check: no network round trip and no false negatives on idle sockets.
func cdpAlive(c *cdpConn) bool {
	return c.alive()
}

func isCDPMessageTooBig(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "message too big")
}

func isCDPGone(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if isCDPMessageTooBig(err) {
		return false
	}
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
	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return nil, err
	}
	id := c.next
	c.next++
	ch := make(chan cdpReply, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	ctx, cancel := context.WithTimeout(context.Background(), cdpCallTimeout)
	defer cancel()
	c.writeMu.Lock()
	err := wsjson.Write(ctx, c.ws, msg)
	c.writeMu.Unlock()
	if err != nil {
		c.drop(id)
		return nil, err
	}
	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("cdp %s: %w", method, r.err)
		}
		return r.result, nil
	case <-c.done:
		// fail() sends to every registered waiter, so this receive completes.
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		return r.result, nil
	case <-ctx.Done():
		c.drop(id)
		return nil, fmt.Errorf("cdp %s: no reply within %s", method, cdpCallTimeout)
	}
}

func (c *cdpConn) drop(id int) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
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
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception *struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.ExceptionDetails != nil {
		msg := out.ExceptionDetails.Text
		if e := out.ExceptionDetails.Exception; e != nil && e.Description != "" {
			msg = e.Description
		}
		return "", fmt.Errorf("browser: page script error: %s", strings.TrimSpace(msg))
	}
	return fmt.Sprint(out.Result.Value), nil
}

func (h *Host) waitDocumentComplete(d time.Duration) {
	if h == nil || d <= 0 {
		return
	}
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		state, err := h.evalOnConn("document.readyState")
		if err == nil && strings.ToLower(cleanJS(state)) == "complete" {
			time.Sleep(80 * time.Millisecond)
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
}
