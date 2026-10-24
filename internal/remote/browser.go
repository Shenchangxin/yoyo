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
	browserCookie = "yoyo_session"
	waitCookie    = "yoyo_wait"
)

// browserHold is a phone browser waiting for the operator. It has no device key.
// The one-time nonce and the approval are the credential.
type browserHold struct {
	ID    string
	exp   time.Time
	done  bool
	allow bool
	token string
	ch    chan struct{}
}

// UseStatic is the desktop UI served after the operator allows a browser.
// Bind may already be listening, so the files are read on the next request.
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

// enterBrowser sets the session cookie on a document response. WeChat and other
// in-app browsers often drop Set-Cookie on fetch(), so the next page has to be
// a real navigation.
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
	// WeChat often drops Set-Cookie on fetch and omits cookies on later fetch.
	// The document navigation stores the token where the page script can send it.
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>Yoyo</title><script>try{sessionStorage.setItem("yoyo_browser",%q)}catch(e){}location.replace("/")</script></html>`, token)
}

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

func (n *Node) approveBrowser(id string) bool { return n.settleBrowser(id, true) }
func (n *Node) denyBrowser(id string) bool    { return n.settleBrowser(id, false) }

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

// browserPresented is the session from the cookie, the page header, or the
// event-stream query. Fetch in WeChat may not send the cookie the document stored.
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

func sameSite(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	// In-app browsers send Origin: null, or an origin without the port, on the
	// same page that just loaded. A foreign site still cannot present the
	// session cookie: it is SameSite=Strict.
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return hostMatches(u.Host, r.Host)
}

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

func isAsset(p string) bool {
	return strings.Contains(path.Base(p), ".")
}

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

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeBrowserPage(w http.ResponseWriter, status int, title, body, extra string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover"><title>Yoyo</title><style>
:root{color-scheme:dark}body{margin:0;min-height:100dvh;display:grid;place-items:center;background:#1c1c1a;color:#f4f4f0;font:16px/1.5 system-ui,sans-serif}main{width:min(28rem,calc(100%% - 2rem));padding-bottom:env(safe-area-inset-bottom)}h1{font-size:21px;font-weight:500;letter-spacing:-.02em;margin:0 0 8px}p{margin:0;color:#8c8b85}.live{margin-top:18px;color:#5c99d6}
</style></head><body><main><h1>%s</h1><p>%s</p>%s</main></body></html>`, htmlText(title), htmlText(body), extra)
}

func htmlText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;").Replace(s)
}

const browserTouch = `<style id="yoyo-browser-touch">.drag{-webkit-app-region:no-drag!important;app-region:no-drag!important;--wails-draggable:no-drag!important}</style><script>document.documentElement.dataset.browser="1"</script>`

// markBrowser rewrites only the HTML this door serves. The desktop window uses
// another server, so its drag region stays intact. WeChat treats a drag region
// as dead to touch, including when the value is unset.
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

type htmlInject struct {
	http.ResponseWriter
	buf     bytes.Buffer
	code    int
	html    bool
	decided bool
}

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

func joinAt(body []byte, at int) []byte {
	out := make([]byte, 0, len(body)+len(browserTouch))
	out = append(out, body[:at]...)
	out = append(out, browserTouch...)
	return append(out, body[at:]...)
}

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

// scrubResponse buffers JSON so vault fields can be removed before the bytes leave.
// Websocket upgrades pass through. json.Encoder may split one value across writes.
type scrubResponse struct {
	http.ResponseWriter
	buf      []byte
	code     int
	hold     bool
	decided  bool
	hijacked bool
}

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

func (s *scrubResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("websocket")
	}
	s.hijacked = true
	return h.Hijack()
}

func (s *scrubResponse) Flush() {
	if s.hold || s.hijacked {
		return
	}
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *scrubResponse) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func jsonBody(ct string) bool {
	return strings.Contains(strings.ToLower(ct), "json")
}
