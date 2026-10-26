// 本文件拥有 CDP WebSocket、查找浏览器和启动隔离 Chrome。一次调用超时 12 秒，读取上限是 64 MiB。
// 隔离启动的调试端口来自 freePort，写进 --remote-debugging-port。找不到浏览器时错误是 browser: no isolated Chromium (set YOYO_CHROME)。
// 空白地址拒绝导航。不调用 runtime.Run。

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

// cdpCallTimeout 是一次 CDP 往返的上限，12 秒。慢页面不会因此拆掉连接，只有套接字死了才算断开。
const cdpCallTimeout = 12 * time.Second

// cdpConn 是一条 CDP WebSocket。唯一的 readLoop 负责全部读取，按 id 把回复对上调用方。连接在读循环看到失败或宿主关闭之前都算活着。
// 空闲时不再用 Ping 探活：coder/websocket 只在有人读取时处理 pong，空闲的健康连接会被误判成断开，然后把 Chrome 重新开到 about:blank。
type cdpConn struct {
	ws      *websocket.Conn
	writeMu sync.Mutex

	mu      sync.Mutex
	next    int
	pending map[int]chan cdpReply
	done    chan struct{}
	err     error

	// errs 记下上次浏览器调用以来页面抛出的 JavaScript 错误。读循环从协议事件里填它，宿主再取走。
	errs pageErrors

	onEvent func(method string, params json.RawMessage)
}

// cdpReply 是一次调用的结果或错误。
type cdpReply struct {
	result json.RawMessage
	err    error
}

// newCDPConn 建一条连接并启动 readLoop。下一个 id 从 1 开始。
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

// readLoop 一直读 JSON。id 为 0 的是协议事件，页面错误总会记下，录制和导航失效在宿主装了 onEvent 时再交给它。其余按 id 交给等待的调用方。
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
			// 协议事件。页面错误总会记下；录制和导航失效在宿主装了 onEvent 时再交给它。
			if msg.Method != "" {
				c.errs.add(msg.Method, msg.Params)
				c.mu.Lock()
				fn := c.onEvent
				c.mu.Unlock()
				if fn != nil {
					fn(msg.Method, msg.Params)
				}
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

// fail 只把连接标死一次，唤醒所有等待者，并关掉套接字。等待者收到的错误能被 isCDPGone 认出来。
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

// close 用「closed by host」调用 fail。空连接什么也不做。
func (c *cdpConn) close() {
	if c == nil {
		return
	}
	c.fail(errors.New("closed by host"))
}

// alive 在读循环还没记错时返回真。
func (c *cdpConn) alive() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err == nil
}

// findChrome 先看 YOYO_CHROME。否则按 GOOS 在常见的 Chrome、Chromium、Edge 路径里找第一个存在的可执行文件。
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

// freePort 在 127.0.0.1:0 上听一下，返回系统分配的端口，然后立刻关掉。隔离 Chrome 的调试端口用它，不固定 9222。
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// chromeProfile 返回有头或无头对应的用户目录。
func (h *Host) chromeProfile(headed bool) string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.profileDirLocked(headed)
}

// ensureCDP 用空的起始地址确保 Chrome 已连上，空地址会改走 previewStartURL。
func (h *Host) ensureCDP() error {
	return h.ensureChrome("")
}

// headedForAutomation 在有头窗口还活着时保持有头。有头但 CDP 已死时返回假，让截图和点击改拉隔离 Chrome。
func headedForAutomation(headed, cdpAlive bool) bool {
	if headed && !cdpAlive {
		return false
	}
	return headed
}

// ensureChrome 按当前有头状态启动。有头启动失败时再试无头；无头失败时再试有头。
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

// startChromeCDP 在同一种有头状态的连接还活着时直接返回。否则停掉旧进程，用 freePort 的端口启动 Chrome，等调试端口和 page WebSocket，再 dialWS。找不到浏览器时错误是 browser: no isolated Chromium (set YOYO_CHROME)。
func (h *Host) startChromeCDP(startURL string, headed bool) error {
	h.mu.Lock()
	c := h.cdp
	same := c != nil && h.headed == headed
	h.mu.Unlock()
	if same && cdpAlive(c) {
		return nil
	}
	if strings.TrimSpace(startURL) == "" {
		// 以前不带地址的重启会落到 about:blank，把正在操作的页面换掉。
		startURL = h.previewStartURL()
	}
	if h.cdp != nil || h.cmd != nil {
		h.stopChrome()
	}
	h.invalidateRefs()
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

// chromeReadyErr 把 stderr 末尾最多 400 个字符附在错误后面。stderr 为空时原样返回。
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

// readFileTail 返回文件末尾最多 n 个字符。读不到或为空时返回空字符串。
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

// cdpMaxMessage 是 CDP WebSocket 的读取上限，64 MiB。coder/websocket 的默认上限装不下 Page.captureScreenshot 的 PNG 或 JPEG。
const cdpMaxMessage = 64 << 20

// dialWS 连上 WebSocket，把读取上限设成 cdpMaxMessage，换上新的 cdpConn，并启用 Page、Network、Runtime。旧套接字会关掉，避免它的读循环还活着。
func (h *Host) dialWS(wsURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	ws.SetReadLimit(cdpMaxMessage)
	conn := newCDPConn(ws)
	conn.mu.Lock()
	conn.onEvent = h.onCDPEvent
	conn.mu.Unlock()
	h.mu.Lock()
	old := h.cdp
	h.cdp = conn
	h.targetID = targetIDFromWS(wsURL)
	if h.tabCount < 1 {
		h.tabCount = 1
	}
	h.mu.Unlock()
	old.close() // 换掉的套接字不能再留着读循环或目标
	_, _ = conn.call("Page.enable", nil)
	_, _ = conn.call("Network.enable", nil)
	// Runtime 事件带上未捕获异常和 console.error。页面失败只留下这个信号，见 page_errors.go。
	_, _ = conn.call("Runtime.enable", nil)
	h.applyRecordHooks(conn)
	return nil
}

// dropDeadCDP 清掉当前连接和标签计数，并关掉那条 WebSocket。
func (h *Host) dropDeadCDP() {
	if h == nil {
		return
	}
	h.mu.Lock()
	c := h.cdp
	h.cdp = nil
	h.tabCount = 0
	h.mu.Unlock()
	if c != nil {
		c.close()
	}
}

// versionWS 读 /json/version 里的 webSocketDebuggerUrl。800 毫秒超时。没有这个字段时返回 no websocket。
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

// cdpAlive 报告读循环是否还在。这是本地检查，不打网络，空闲套接字也不会被误判成断开。
func cdpAlive(c *cdpConn) bool {
	return c.alive()
}

// isCDPMessageTooBig 在错误文本里含有 message too big 时返回真。
func isCDPMessageTooBig(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "message too big")
}

// isCDPGone 判断套接字是不是已经断了。message too big 返回假，避免因此重启 Chrome。
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

// previewStartURL 优先返回预览文件的本地服务地址，否则返回当前非空白地址。都没有时返回空字符串。
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

// reconnectChrome 停掉 Chrome，再用 previewStartURL 重新确保启动。
func (h *Host) reconnectChrome() error {
	start := h.previewStartURL()
	h.stopChrome()
	return h.ensureChrome(start)
}

// callCDP 调用一条 CDP 方法。连接已断时先 reconnectChrome 再试一次。没有宿主或没有连接时返回错误。
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

// call 发出一条带 id 的 CDP 消息并等待回复。超时文案带上方法名和 cdpCallTimeout。连接已死时返回 fail 记下的错误。
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
		// fail 会向每个已登记的等待者发送，所以这次接收会完成。
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

// drop 从等待表里去掉这个 id，用于写失败或超时之后。
func (c *cdpConn) drop(id int) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// navigateCDP 先 ensureCDP，再 pageNavigate。
func (h *Host) navigateCDP(raw string) error {
	if err := h.ensureCDP(); err != nil {
		return err
	}
	return h.pageNavigate(raw)
}

// pageNavigate 用 Page.navigate 打开地址。空白页被拒绝，错误是 browser: refusing blank navigation。Page.navigate 失败时再试 Target.createTarget。成功后清掉 ref 并记下地址。
func (h *Host) pageNavigate(raw string) error {
	raw = strings.TrimSpace(raw)
	if h.cdp == nil {
		return fmt.Errorf("browser: no CDP")
	}
	if isBlankURL(raw) {
		return fmt.Errorf("browser: refusing blank navigation")
	}
	if _, err := h.callCDP("Page.navigate", map[string]any{"url": raw}); err == nil {
		h.invalidateRefs()
		h.mu.Lock()
		h.url = raw
		h.mu.Unlock()
		return nil
	} else if _, err2 := h.callCDP("Target.createTarget", map[string]any{"url": raw}); err2 == nil {
		h.invalidateRefs()
		h.mu.Lock()
		h.url = raw
		h.mu.Unlock()
		return nil
	} else {
		return err
	}
}

// evalJS 先确保有连接，再在当前连接上执行表达式。
func (h *Host) evalJS(expr string) (string, error) {
	if err := h.ensureCDP(); err != nil {
		return "", err
	}
	return h.evalOnConn(expr)
}

// evalOnConn 用 Runtime.evaluate 执行表达式并返回值的文本。页面抛异常时返回 browser: page script error。
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

// waitDocumentComplete 在 d 之内轮询 document.readyState，变成 complete 后再等 80 毫秒。d 小于等于 0 或宿主为空时立即返回。
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
