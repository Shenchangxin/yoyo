// 本文件拥有手机浏览器的明文 HTTP 门：扫码打开、等电脑允许、会话 cookie、改写页面上的拖拽，以及按字段名洗 JSON 响应。
// 不拥有手机 WebSocket。监听用 net.Listen，不是加密通道。证书 DNS 名只有 yoyo.local，没有 IP，浏览器不会在局域网地址上信任那条 TLS，所以扫码页用 HTTP。
// listen 调用 serveBrowser。桌面通过 UseStatic 把静态文件交进来。

package remote

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/api"
)

const (
	// browserCookie 是电脑允许之后的会话令牌。waitCookie 是还在等允许时的浏览器号。
	browserCookie = "yoyo_session"
	waitCookie    = "yoyo_wait"
)

// browserHold 是正在等电脑允许的手机浏览器。它没有设备密钥，凭的是一次性配对码和电脑上的允许。
type browserHold struct {
	ID    string
	exp   time.Time // 沿用配对码的到期时刻
	done  bool
	allow bool
	token string // 允许成功后写入的会话令牌
	ch    chan struct{}
}

// UseStatic 换上允许浏览器之后要提供的桌面静态文件。
// Bind 可能已经在听，所以清掉已组好的页面处理器，下一次请求才按新文件重做。
// h 在 n 非 nil 时写入 static，并清空 pages 和 pagesFor。
func (n *Node) UseStatic(h http.Handler) {
	if n == nil {
		return
	}
	n.mu.Lock()
	n.static = h
	n.pages = nil
	n.pagesFor = nil
	n.mu.Unlock()
}

// serveBrowser 已经在听就直接返回。端口先取 TLS 端口加 1，结果小于等于 1 或绑不上时改听 :0。
// 用 net.Listen 提供明文 HTTP，读请求头超时 10 秒。两次都失败时，只在 listenErr 还空着的情况下记下错误。
func (n *Node) serveBrowser() {
	n.mu.Lock()
	if n.httpLn != nil {
		n.mu.Unlock()
		return
	}
	tlsPort := n.port
	n.mu.Unlock()

	httpPort := tlsPort + 1
	if httpPort <= 1 {
		httpPort = 0
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", httpPort))
	if err != nil {
		ln, err = net.Listen("tcp", ":0")
	}
	if err != nil {
		n.mu.Lock()
		if n.listenErr == "" {
			n.listenErr = err.Error()
		}
		n.mu.Unlock()
		return
	}
	tcp, _ := ln.Addr().(*net.TCPAddr)
	srv := &http.Server{Handler: http.HandlerFunc(n.serveBrowserHTTP), ReadHeaderTimeout: 10 * time.Second}
	n.mu.Lock()
	n.httpLn = ln
	n.httpSrv = srv
	if tcp != nil {
		n.httpPort = tcp.Port
	}
	n.mu.Unlock()
	go func() { _ = srv.Serve(ln) }()
}

// serveBrowserHTTP 先拒绝非私网来源。/open、/open/wait、/open/enter 在允许之前就能进。
// 其余路径要同时有有效会话和 sameSite。静态资源、以 /api 开头的路径、/rpc 和 /ws 未通过时回 401，别的路径回扫码说明页。
// 通过之后，最后一段里没有点的响应加上 no-store，再交给会洗 JSON 的页面处理器。
func (n *Node) serveBrowserHTTP(w http.ResponseWriter, r *http.Request) {
	if !privatePeer(r.RemoteAddr) {
		http.Error(w, "只接受局域网里的手机", http.StatusForbidden)
		return
	}
	switch r.URL.Path {
	case "/open":
		n.openBrowser(w, r)
		return
	case "/open/wait":
		n.waitBrowser(w, r)
		return
	case "/open/enter":
		n.enterBrowser(w, r)
		return
	}
	if !n.browserAuthed(r) || !sameSite(r) {
		if isAsset(r.URL.Path) || strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/rpc" || r.URL.Path == "/ws" {
			http.Error(w, "先在电脑上允许这个浏览器", http.StatusUnauthorized)
			return
		}
		writeBrowserPage(w, http.StatusUnauthorized, "先扫电脑上的配对码", "在电脑的设置里出示配对码，再用微信或其他扫码工具打开。允许之后，这个浏览器里的操作写回那台电脑。", "")
		return
	}
	if !isAsset(r.URL.Path) {
		w.Header().Set("Cache-Control", "no-store")
	}
	rec := &scrubResponse{ResponseWriter: w}
	n.pageHandler().ServeHTTP(rec, r)
	rec.finish()
}

// pageHandler 在 static 没变时复用已经包好的 API 处理器。变了就用 spa 和 markBrowser 重新包一层。
func (n *Node) pageHandler() http.Handler {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.pages != nil && n.pagesFor == n.static {
		return n.pages
	}
	n.pagesFor = n.static
	n.pages = api.Handler(n.app, spa(markBrowser(n.static)))
	return n.pages
}

// openBrowser 只接受 GET。用查询串 n 核销配对码，失败就说明这枚码只能用一次，并且两分钟后过期。
// 浏览器号是 b- 加上 4 字节随机数的十六进制。等待 cookie 的 MaxAge 是 180。页面每 1500 毫秒带 cookie 问一次 /open/wait。
func (n *Node) openBrowser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	exp, err := n.takeOffer(r.URL.Query().Get("n"))
	if err != nil {
		writeBrowserPage(w, http.StatusBadRequest, "配对码已经不能用", "在电脑上重新出示配对码，再扫一次。一个码只能用一次，两分钟后过期。", "")
		return
	}
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		http.Error(w, "稍后再试", http.StatusInternalServerError)
		return
	}
	id := "b-" + hex.EncodeToString(buf)
	hold := &browserHold{ID: id, exp: exp, ch: make(chan struct{})}
	n.mu.Lock()
	n.browsers[id] = hold
	n.mu.Unlock()
	setBrowserCookie(w, waitCookie, id, 180)
	writeBrowserPage(w, http.StatusOK, "等这台电脑允许", "回到电脑，在设置里的「手机」点允许。允许之后，这个浏览器就是那台工作站，你在这里做的操作写回电脑。", fmt.Sprintf(`<p class="live" id="state">正在等。</p><script>
const id = %q;
async function poll(){
  try {
    const res = await fetch("/open/wait?id=" + encodeURIComponent(id), {credentials:"include"});
    const body = await res.json();
    if (body.ok) { location.replace("/open/enter?id=" + encodeURIComponent(id)); return; }
    if (body.denied) {
      document.getElementById("state").textContent = "这台电脑没有允许。重新出示配对码后再扫一次。";
      return;
    }
  } catch (e) {}
  setTimeout(poll, 1500);
}
poll();
</script>`, id))
}

// enterBrowser 在文档响应上写会话 cookie。微信等应用内浏览器经常丢掉 fetch 的 Set-Cookie，之后的 fetch 也不带上，所以必须是一次真正的页面跳转。
// 还没允许就回说明页。允许之后会话 cookie 的 MaxAge 是 12*60*60，等待 cookie 用 MaxAge -1 清掉，并加上 Refresh 跳到 /。
// 正文把令牌写进 sessionStorage 的 yoyo_browser，供页面以后放进请求头。
func (n *Node) enterBrowser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	id := n.waitID(r)
	n.mu.Lock()
	b := n.browsers[id]
	allow, token := false, ""
	if b != nil {
		allow, token = b.allow && b.token != "", b.token
	}
	n.mu.Unlock()
	if !allow || token == "" {
		writeBrowserPage(w, http.StatusForbidden, "还没有允许", "回到电脑点允许。如果这一页停太久，重新出示配对码再扫一次。", "")
		return
	}
	setBrowserCookie(w, browserCookie, token, 12*60*60)
	setBrowserCookie(w, waitCookie, "", -1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Refresh", "0;url=/")
	// 微信经常丢掉 fetch 上的 Set-Cookie，之后的 fetch 也不带 cookie。这次文档跳转把令牌放进页面脚本能读到的地方。
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>Yoyo</title><script>try{sessionStorage.setItem("yoyo_browser",%q)}catch(e){}location.replace("/")</script></html>`, token)
}

// waitBrowser 只接受 GET。没有合法等待号，或找不到这次等待，就回 401 且 denied。
// 已经过了配对码的到期时刻还没完成，就标成结束。最多堵 15 秒等电脑点允许。
// 允许后同样写下 MaxAge 为 12*60*60 的会话 cookie，并清掉等待 cookie。被拒绝回 403。还在等就回 pending。
func (n *Node) waitBrowser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	id := n.waitID(r)
	if id == "" {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"ok": false, "denied": true})
		return
	}
	n.mu.Lock()
	b := n.browsers[id]
	if b != nil && !b.done && time.Now().After(b.exp) {
		b.done = true
		close(b.ch)
	}
	n.mu.Unlock()
	if b == nil {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"ok": false, "denied": true})
		return
	}
	if !b.done {
		select {
		case <-b.ch:
		case <-time.After(15 * time.Second):
		case <-r.Context().Done():
			return
		}
	}
	n.mu.Lock()
	allow, token := b.allow, b.token
	n.mu.Unlock()
	if !allow || token == "" {
		if b.done {
			writeJSONStatus(w, http.StatusForbidden, map[string]any{"ok": false, "denied": true})
			return
		}
		writeJSONStatus(w, http.StatusOK, map[string]any{"ok": false, "pending": true})
		return
	}
	setBrowserCookie(w, browserCookie, token, 12*60*60)
	setBrowserCookie(w, waitCookie, "", -1)
	writeJSONStatus(w, http.StatusOK, map[string]any{"ok": true})
}

// takeOffer 核销一枚配对码并返回它的到期时刻。不存在，或此刻已经晚于 Exp，就返回过期。
// 删掉之后不放回去。手机 stage 再来，或浏览器再打开同一次，都会失败。
// nonce 用来查 offers。不存在，或此刻已经晚于 Exp，就返回过期。否则删掉该条并返回 Exp。
func (n *Node) takeOffer(nonce string) (time.Time, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	off := n.offers[nonce]
	if off == nil || time.Now().After(off.Exp) {
		return time.Time{}, errors.New("配对码已过期")
	}
	delete(n.offers, nonce)
	return off.Exp, nil
}

// settleBrowser 结束一次还没完成的浏览器等待。已经完成或不存在就返回假。
// 允许时用 32 字节随机数做令牌，sessions 里记 12 小时。随机数失败就把允许改回假。然后关掉等待通道并记审计。
// id 用来查 browsers，并传给 audit。
// allow 写入这条等待的 allow。为真时读 32 字节随机数做令牌，放进 sessions，到期加上 12 小时。随机数失败就把 allow 改回假。
func (n *Node) settleBrowser(id string, allow bool) bool {
	n.mu.Lock()
	b := n.browsers[id]
	if b == nil || b.done {
		n.mu.Unlock()
		return false
	}
	b.done = true
	b.allow = allow
	if allow {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			b.allow = false
		} else {
			b.token = hex.EncodeToString(buf)
			n.sessions[b.token] = time.Now().Add(12 * time.Hour)
		}
	}
	close(b.ch)
	ok := b.allow
	n.mu.Unlock()
	n.audit("browser", id, "浏览器", ok)
	return true
}

// approveBrowser 把这次浏览器等待记成允许。
// id 和 true 传给 settleBrowser。
func (n *Node) approveBrowser(id string) bool { return n.settleBrowser(id, true) }

// denyBrowser 把这次浏览器等待记成拒绝。
// id 和 false 传给 settleBrowser。
func (n *Node) denyBrowser(id string) bool { return n.settleBrowser(id, false) }

// browserAuthed 看呈现出来的令牌是否还在 sessions 里，并且此刻没有晚于过期时间。过期的会从 sessions 删掉。
func (n *Node) browserAuthed(r *http.Request) bool {
	token := browserPresented(r)
	if token == "" {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	exp, ok := n.sessions[token]
	if !ok || time.Now().After(exp) {
		if ok {
			delete(n.sessions, token)
		}
		return false
	}
	return true
}

// browserPresented 按 cookie、请求头 X-Yoyo-Session、查询参数 ys 的顺序取会话令牌。
// 微信里的 fetch 可能不带文档写下的那枚 cookie。
func browserPresented(r *http.Request) string {
	if c, err := r.Cookie(browserCookie); err == nil {
		if tok := cleanSession(c.Value); tok != "" {
			return tok
		}
	}
	if tok := cleanSession(r.Header.Get("X-Yoyo-Session")); tok != "" {
		return tok
	}
	return cleanSession(r.URL.Query().Get("ys"))
}

// cleanSession 去掉首尾空白。长度要在 32 到 128 之间，并且只含 0-9 和 a-f。大写十六进制会当成没有令牌。
// v 去掉首尾空白。长度小于 32 或大于 128，或含有 0-9 和 a-f 以外的字符时返回空字符串。
func cleanSession(v string) string {
	v = strings.TrimSpace(v)
	if len(v) < 32 || len(v) > 128 {
		return ""
	}
	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ""
		}
	}
	return v
}

// waitID 先看查询参数 id。不合法再看等待 cookie。都不是合法浏览器号就返回空字符串。
// r 的查询参数 id 先交给 validBrowserID。不合法再读等待 cookie 的值。
func (n *Node) waitID(r *http.Request) string {
	if id := r.URL.Query().Get("id"); validBrowserID(id) {
		return id
	}
	c, err := r.Cookie(waitCookie)
	if err != nil || !validBrowserID(c.Value) {
		return ""
	}
	return c.Value
}

// validBrowserID 要求前缀是 b-，总长 10，后面每一位都是 0-9 或 a-f。4 字节随机数编码出来正好是这个形状。
func validBrowserID(id string) bool {
	if !strings.HasPrefix(id, "b-") || len(id) != 10 {
		return false
	}
	for _, c := range id[2:] {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

// sameSite 在 Origin 为空或值为 null 时放行。应用内浏览器会这样送，也会送一个不带端口的源。
// 解析失败或没有主机就拒绝。主机名对上才放行。cookie 是 SameSite 严格，外来站点带不来这枚 cookie。请求头里的令牌也要过这里。
func sameSite(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return hostMatches(u.Host, r.Host)
}

// hostMatches 先比整串。不一致时去掉端口再比主机名。去掉之后主机为空则不算相同。
// b 与 a 先做不区分大小写的整串比较。不一致时去掉端口后再比。
func hostMatches(a, b string) bool {
	if strings.EqualFold(a, b) {
		return true
	}
	ah, _, aerr := net.SplitHostPort(a)
	if aerr != nil {
		ah = a
	}
	bh, _, berr := net.SplitHostPort(b)
	if berr != nil {
		bh = b
	}
	return ah != "" && strings.EqualFold(ah, bh)
}

// isAsset 看路径最后一段里有没有点。有点才当成静态文件。
// p 取最后一段，看里面有没有点。
func isAsset(p string) bool {
	return strings.Contains(path.Base(p), ".")
}

// setBrowserCookie 写 Path 为 /、HttpOnly、SameSite 严格的 cookie。不设 Secure，因为这扇门是明文 HTTP。
// MaxAge 由调用方传入。负值用来删掉 cookie。
// name 写入 Cookie 的 Name。
// value 写入 Cookie 的 Value。
// maxAge 写入 Cookie 的 MaxAge。
func setBrowserCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	})
}

// writeJSONStatus 写 JSON 正文和给定的状态码。
// status 传给 WriteHeader。
// v 传给 json.NewEncoder 的 Encode。
func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeBrowserPage 写一页不缓存的说明 HTML。标题和正文会做转义，extra 原样拼进页面。
// status 传给 WriteHeader。
// title 传给 htmlText 后写入 h1。
// body 传给 htmlText 后写入段落。
func writeBrowserPage(w http.ResponseWriter, status int, title, body, extra string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover"><title>Yoyo</title><style>
:root{color-scheme:dark}body{margin:0;min-height:100dvh;display:grid;place-items:center;background:#1c1c1a;color:#f4f4f0;font:16px/1.5 system-ui,sans-serif}main{width:min(28rem,calc(100%% - 2rem));padding-bottom:env(safe-area-inset-bottom)}h1{font-size:21px;font-weight:500;letter-spacing:-.02em;margin:0 0 8px}p{margin:0;color:#8c8b85}.live{margin-top:18px;color:#5c99d6}
</style></head><body><main><h1>%s</h1><p>%s</p>%s</main></body></html>`, htmlText(title), htmlText(body), extra)
}

// htmlText 把 &、<、> 和双引号换成字符引用。
// s 交给 strings.NewReplacer，替换 &、<、> 和双引号。
func htmlText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;").Replace(s)
}

// browserTouch 插进这扇门送出的 HTML：关掉拖拽区域，并标上 browser。桌面那条服务器不走这里。
const browserTouch = `<style id="yoyo-browser-touch">.drag{-webkit-app-region:no-drag!important;app-region:no-drag!important;--wails-draggable:no-drag!important}</style><script>document.documentElement.dataset.browser="1"</script>`

// markBrowser 只改写这扇门送出的 HTML。桌面窗口用的是另一台服务器，拖拽区域保持原样。
// 微信把拖拽区域当成点不了的地方，没写值也一样。HEAD 原样转发。其它请求去掉 Range，好把整页攒起来再插入。
// files 为 nil 时返回 nil。否则返回的处理器在 HEAD 时直接 ServeHTTP。其他请求去掉 Range 后，用 htmlInject 包住 ResponseWriter 再 ServeHTTP。
func markBrowser(files http.Handler) http.Handler {
	if files == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			files.ServeHTTP(w, r)
			return
		}
		clone := r.Clone(r.Context())
		clone.Header.Del("Range")
		rec := &htmlInject{ResponseWriter: w}
		files.ServeHTTP(rec, clone)
		rec.finish()
	})
}

// htmlInject 把 HTML 响应攒起来，插入浏览器触摸样式。非 HTML 直接往下写。
type htmlInject struct {
	http.ResponseWriter
	buf     bytes.Buffer
	code    int
	html    bool
	decided bool
}

// WriteHeader 只认第一次。HTML 先攒着，其它类型马上写下状态码。
// code 在尚未决定时写入 code 字段。内容类型不含 text/html 时传给底层 WriteHeader。
func (m *htmlInject) WriteHeader(code int) {
	if m.decided {
		return
	}
	m.code = code
	m.html = strings.Contains(strings.ToLower(m.Header().Get("Content-Type")), "text/html")
	m.decided = true
	if !m.html {
		m.ResponseWriter.WriteHeader(code)
	}
}

// Write 在还没决定时补上内容类型。HTML 进缓冲区，其它字节直接写出。
// p 在尚未决定且 Content-Type 为空时传给 http.DetectContentType。是 HTML 时写入缓冲区，否则传给底层 Write。
func (m *htmlInject) Write(p []byte) (int, error) {
	if !m.decided {
		if m.Header().Get("Content-Type") == "" {
			m.Header().Set("Content-Type", http.DetectContentType(p))
		}
		code := m.code
		if code == 0 {
			code = http.StatusOK
		}
		m.WriteHeader(code)
	}
	if m.html {
		return m.buf.Write(p)
	}
	return m.ResponseWriter.Write(p)
}

// finish 把插入触摸样式后的 HTML 一次写出，并加上 no-store。不是 HTML 就什么都不做。
func (m *htmlInject) finish() {
	if !m.html {
		return
	}
	body := injectBrowserTouch(m.buf.Bytes())
	m.Header().Set("Cache-Control", "no-store")
	m.Header().Set("Content-Length", strconv.Itoa(len(body)))
	code := m.code
	if code == 0 {
		code = http.StatusOK
	}
	m.ResponseWriter.WriteHeader(code)
	_, _ = m.ResponseWriter.Write(body)
}

// injectBrowserTouch 把 browserTouch 放进 head 开始标签后面。没有 head 就接到正文最前。
// body 转成小写后找 <head> 或 <head。找到就在标签结束后传给 joinAt。找不到就把 browserTouch 接到 body 前面。
func injectBrowserTouch(body []byte) []byte {
	lower := bytes.ToLower(body)
	if i := bytes.Index(lower, []byte("<head>")); i >= 0 {
		at := i + len("<head>")
		return joinAt(body, at)
	}
	if i := bytes.Index(lower, []byte("<head")); i >= 0 {
		if j := bytes.IndexByte(body[i:], '>'); j >= 0 {
			return joinAt(body, i+j+1)
		}
	}
	out := make([]byte, 0, len(browserTouch)+len(body))
	out = append(out, browserTouch...)
	return append(out, body...)
}

// joinAt 在 at 这个下标处插入 browserTouch。
// body 在 at 处切开，中间插入 browserTouch。
func joinAt(body []byte, at int) []byte {
	out := make([]byte, 0, len(body)+len(browserTouch))
	out = append(out, body[:at]...)
	out = append(out, browserTouch...)
	return append(out, body[at:]...)
}

// spa 把最后一段里没有点、且不是 / 的路径改去要 /，让前端路由落到同一份入口。有点的路径原样要文件。files 为空就返回 nil。
func spa(files http.Handler) http.Handler {
	if files == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.URL.Path)
		if p != "/" && !strings.Contains(path.Base(p), ".") {
			clone := r.Clone(r.Context())
			u := *r.URL
			u.Path = "/"
			clone.URL = &u
			files.ServeHTTP(w, clone)
			return
		}
		files.ServeHTTP(w, r)
	})
}

// scrubResponse 把 JSON 正文攒起来，写出前按字段名丢掉密钥。WebSocket 升级直接透传。
// json.Encoder 可能把一个值拆成多次 Write，所以要等到 finish。内容类型不含 json 的正文不洗。
// 正文不是合法 JSON，解不开，或重新编码失败时，原样写出，密钥还在。
type scrubResponse struct {
	http.ResponseWriter
	buf      []byte
	code     int
	hold     bool
	decided  bool
	hijacked bool
}

// WriteHeader 只认第一次。内容类型含 json 时先不往下写状态码，等洗完再写。
// code 在尚未劫持且尚未决定时写入 code 字段。内容类型含 json 时只标记 hold。否则传给底层 WriteHeader。
func (s *scrubResponse) WriteHeader(code int) {
	if s.hijacked || s.decided {
		return
	}
	s.code = code
	if jsonBody(s.Header().Get("Content-Type")) {
		s.hold = true
		s.decided = true
		return
	}
	s.ResponseWriter.WriteHeader(code)
	s.decided = true
}

// Write 在攒 JSON 时只追加缓冲区。已经 Hijack，或内容类型不是 JSON，就直接写出。
// p 在已劫持或不是 JSON 时传给底层 Write。正在攒时追加到缓冲区并返回长度。
func (s *scrubResponse) Write(p []byte) (int, error) {
	if s.hijacked {
		return s.ResponseWriter.Write(p)
	}
	if !s.decided {
		if jsonBody(s.Header().Get("Content-Type")) {
			s.hold = true
		} else {
			if s.code == 0 {
				s.code = http.StatusOK
			}
			s.ResponseWriter.WriteHeader(s.code)
		}
		s.decided = true
	}
	if s.hold {
		s.buf = append(s.buf, p...)
		return len(p), nil
	}
	return s.ResponseWriter.Write(p)
}

// finish 在攒了 JSON 且没有 Hijack 时，合法 JSON 会按 scrub 重编码再写。洗不成就把原来的字节写出。
func (s *scrubResponse) finish() {
	if s.hijacked || !s.hold {
		return
	}
	cleaned := s.buf
	if json.Valid(s.buf) {
		var v any
		if json.Unmarshal(s.buf, &v) == nil {
			if b, err := json.Marshal(scrub(v)); err == nil {
				cleaned = append(b, '\n')
			}
		}
	}
	s.Header().Set("Content-Length", strconv.Itoa(len(cleaned)))
	code := s.code
	if code == 0 {
		code = http.StatusOK
	}
	s.ResponseWriter.WriteHeader(code)
	_, _ = s.ResponseWriter.Write(cleaned)
}

// Hijack 标记这条响应已经交给 WebSocket，finish 不再把缓冲区写出。内层没有 Hijacker 就返回错误。
func (s *scrubResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("websocket")
	}
	s.hijacked = true
	return h.Hijack()
}

// Flush 在正在攒 JSON 或已经 Hijack 时不刷。否则调用内层的 Flusher。
func (s *scrubResponse) Flush() {
	if s.hold || s.hijacked {
		return
	}
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap 返回内层 ResponseWriter，外层才能找到 Hijacker 和 Flusher。
func (s *scrubResponse) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// jsonBody 把内容类型转成小写后，看里面有没有 json。
// ct 转成小写后看是否包含 json。
func jsonBody(ct string) bool {
	return strings.Contains(strings.ToLower(ct), "json")
}
