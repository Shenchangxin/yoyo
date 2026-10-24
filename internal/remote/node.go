package remote

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
)

// Signing domain shared with the handheld client. Do not reorder the lines.
const signDomain = "yoyo.remote.v1"

type storedDevice struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Pub     string    `json:"pub"`
	Created time.Time `json:"created"`
	Seen    time.Time `json:"seen"`
	Revoked time.Time `json:"revoked,omitempty"`
	pub     ed25519.PublicKey
}

type offer struct {
	Nonce string
	Exp   time.Time
}

type pendingPair struct {
	ID   string
	Name string
	Pub  ed25519.PublicKey
	Exp  time.Time
	done chan bool
	dev  *storedDevice
}

type diskState struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// Node is the handheld door inside the process that owns the App.
type Node struct {
	app *app.App
	dir string

	mu        sync.Mutex
	ln        net.Listener
	srv       *http.Server
	httpLn    net.Listener
	httpSrv   *http.Server
	fp        string
	port      int
	httpPort  int
	enabled   bool
	listenErr string
	offers    map[string]*offer
	pending   map[string]*pendingPair
	devices   map[string]*storedDevice
	browsers  map[string]*browserHold
	sessions  map[string]time.Time
	static    http.Handler
	pages     http.Handler
	pagesFor  http.Handler
	nonces    map[string]map[string]time.Time
	mdnsStop  func()
}

var current atomic.Pointer[Node]

// Current is the node bound in this process, if any.
func Current() *Node { return current.Load() }

// Bind registers the node and resumes listening when the operator left it on.
func Bind(a *app.App) (*Node, error) {
	if a == nil || a.Home == nil {
		return nil, errors.New("remote needs a home")
	}
	dir := filepath.Join(a.Home.Root, "remote")
	if err := os.MkdirAll(filepath.Join(dir, "devices"), 0o700); err != nil {
		return nil, err
	}
	n := &Node{
		app:      a,
		dir:      dir,
		offers:   map[string]*offer{},
		pending:  map[string]*pendingPair{},
		devices:  map[string]*storedDevice{},
		browsers: map[string]*browserHold{},
		sessions: map[string]time.Time{},
		nonces:   map[string]map[string]time.Time{},
	}
	n.loadDevices()
	n.loadState()
	api.SetRemoteHandler(n.handle)
	current.Store(n)
	if n.enabled {
		if err := n.listen(); err != nil {
			n.listenErr = err.Error()
		}
	}
	return n, nil
}

// Close stops the listener.
func (n *Node) Close() error {
	if n == nil {
		return nil
	}
	n.mu.Lock()
	stop := n.mdnsStop
	n.mdnsStop = nil
	srv := n.srv
	ln := n.ln
	httpSrv := n.httpSrv
	httpLn := n.httpLn
	n.ln = nil
	n.httpLn = nil
	n.httpPort = 0
	n.sessions = map[string]time.Time{}
	n.browsers = map[string]*browserHold{}
	n.pages = nil
	n.pagesFor = nil
	n.mu.Unlock()
	if stop != nil {
		stop()
	}
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
	}
	if httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = httpSrv.Shutdown(ctx)
		cancel()
	}
	if ln != nil {
		_ = ln.Close()
	}
	if httpLn != nil {
		_ = httpLn.Close()
	}
	return nil
}

func (n *Node) handle(_ context.Context, a *app.App, method string, params json.RawMessage) (any, bool, error) {
	if n == nil || a != n.app {
		return nil, false, nil
	}
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.Unmarshal(params, &body)
	switch method {
	case "remote.status":
		return n.Status(), true, nil
	case "remote.pair.begin", "remote.enable":
		out, err := n.BeginPair()
		return out, true, err
	case "remote.pair.approve":
		out, err := n.Approve(body.ID)
		return out, true, err
	case "remote.pair.deny":
		out, err := n.Deny(body.ID)
		return out, true, err
	case "remote.revoke":
		out, err := n.Revoke(body.ID)
		return out, true, err
	case "remote.disable":
		return map[string]any{"ok": true}, true, n.Disable()
	default:
		return nil, false, nil
	}
}

func (n *Node) Status() map[string]any {
	n.mu.Lock()
	defer n.mu.Unlock()
	hosts := lanHosts()
	pending := make([]map[string]any, 0, len(n.pending))
	for _, p := range n.pending {
		pending = append(pending, map[string]any{"id": p.ID, "name": p.Name, "exp": p.Exp.UTC().Format(time.RFC3339)})
	}
	devices := make([]map[string]any, 0, len(n.devices))
	for _, d := range n.devices {
		devices = append(devices, map[string]any{
			"id":      d.ID,
			"name":    d.Name,
			"seen":    d.Seen.UTC().Format(time.RFC3339),
			"revoked": !d.Revoked.IsZero(),
		})
	}
	now := time.Now()
	browsers := make([]map[string]any, 0)
	for _, b := range n.browsers {
		if !b.done {
			pending = append(pending, map[string]any{"id": b.ID, "name": "浏览器", "exp": b.exp.UTC().Format(time.RFC3339)})
			continue
		}
		if !b.allow || b.token == "" {
			continue
		}
		exp, ok := n.sessions[b.token]
		if !ok || now.After(exp) {
			continue
		}
		browsers = append(browsers, map[string]any{"id": b.ID, "name": "浏览器"})
	}
	return map[string]any{
		"listening":   n.ln != nil,
		"enabled":     n.enabled,
		"port":        n.port,
		"http_port":   n.httpPort,
		"fingerprint": n.fp,
		"hosts":       hosts,
		"pending":     pending,
		"devices":     devices,
		"browsers":    browsers,
		"error":       n.listenErr,
		"epoch":       n.app.StateEpoch(),
	}
}

// BeginPair opens the listener if needed and returns a two-minute QR.
func (n *Node) BeginPair() (map[string]any, error) {
	if err := n.listen(); err != nil {
		return nil, err
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	nonce := hex.EncodeToString(buf)
	exp := time.Now().Add(2 * time.Minute)
	n.mu.Lock()
	n.offers[nonce] = &offer{Nonce: nonce, Exp: exp}
	host := ""
	hosts := lanHosts()
	if len(hosts) > 0 {
		host = hosts[0]
	}
	port, fp, httpPort := n.port, n.fp, n.httpPort
	n.enabled = true
	n.mu.Unlock()
	n.saveState()
	hostPart := host
	uri := fmt.Sprintf("http://%s:%d/open?v=1&h=%s&p=%d&fp=%s&n=%s&exp=%d", hostPart, httpPort, host, port, fp, nonce, exp.Unix())
	if httpPort == 0 {
		uri = fmt.Sprintf("yoyo://pair?v=1&h=%s&p=%d&fp=%s&n=%s&exp=%d", host, port, fp, nonce, exp.Unix())
	}
	png, err := qrcode.Encode(uri, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"uri":         uri,
		"png":         base64.StdEncoding.EncodeToString(png),
		"fingerprint": fp,
		"host":        host,
		"port":        port,
		"http_port":   httpPort,
		"exp":         exp.UTC().Format(time.RFC3339),
	}, nil
}

func (n *Node) Approve(id string) (map[string]any, error) {
	if n.approveBrowser(id) {
		return map[string]any{"ok": true, "id": id}, nil
	}
	n.mu.Lock()
	p := n.pending[id]
	n.mu.Unlock()
	if p == nil {
		return nil, fmt.Errorf("没有等待中的手机")
	}
	n.finish(p, true)
	return map[string]any{"ok": true, "id": id}, nil
}

func (n *Node) Deny(id string) (map[string]any, error) {
	if n.denyBrowser(id) {
		return map[string]any{"ok": true}, nil
	}
	n.mu.Lock()
	p := n.pending[id]
	n.mu.Unlock()
	if p == nil {
		return nil, fmt.Errorf("没有等待中的手机")
	}
	n.finish(p, false)
	return map[string]any{"ok": true}, nil
}

func (n *Node) finish(p *pendingPair, ok bool) {
	if ok {
		n.mu.Lock()
		n.devices[p.dev.ID] = p.dev
		delete(n.pending, p.ID)
		n.mu.Unlock()
		n.saveDevice(p.dev)
		n.audit("pair", p.dev.ID, p.dev.Name, true)
	} else {
		n.mu.Lock()
		delete(n.pending, p.ID)
		n.mu.Unlock()
	}
	select {
	case p.done <- ok:
	default:
	}
}

func (n *Node) Revoke(id string) (map[string]any, error) {
	n.mu.Lock()
	d := n.devices[id]
	if d != nil {
		d.Revoked = time.Now().UTC()
	}
	n.mu.Unlock()
	if d == nil {
		return nil, fmt.Errorf("没有这台设备")
	}
	n.saveDevice(d)
	n.audit("revoke", d.ID, d.Name, true)
	return map[string]any{"ok": true}, nil
}

func (n *Node) Disable() error {
	n.mu.Lock()
	n.enabled = false
	n.listenErr = ""
	n.mu.Unlock()
	n.saveState()
	return n.Close()
}

func (n *Node) listen() error {
	n.mu.Lock()
	if n.ln != nil {
		n.mu.Unlock()
		return nil
	}
	port := n.port
	if port == 0 {
		port = 47321
	}
	dir := n.dir
	n.mu.Unlock()

	cert, fp, err := loadCert(dir)
	if err != nil {
		return err
	}
	ln, err := listenTLS(port, cert)
	if err != nil {
		ln, err = listenTLS(0, cert)
		if err != nil {
			n.mu.Lock()
			n.listenErr = err.Error()
			n.mu.Unlock()
			return err
		}
	}
	tcp, _ := ln.Addr().(*net.TCPAddr)
	mux := http.NewServeMux()
	mux.HandleFunc("/remote/ws", n.serveWS)
	mux.HandleFunc("/remote/media", n.serveMedia)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	n.mu.Lock()
	n.ln = ln
	n.srv = srv
	n.fp = fp
	n.port = tcp.Port
	n.enabled = true
	n.listenErr = ""
	n.mu.Unlock()
	go func() { _ = srv.Serve(ln) }()
	n.serveBrowser()
	n.announce()
	n.saveState()
	return nil
}

func (n *Node) loadState() {
	b, err := os.ReadFile(filepath.Join(n.dir, "state.json"))
	if err != nil {
		return
	}
	var st diskState
	if json.Unmarshal(b, &st) == nil {
		n.enabled = st.Enabled
		n.port = st.Port
	}
}

func (n *Node) saveState() {
	n.mu.Lock()
	st := diskState{Enabled: n.enabled, Port: n.port}
	dir := n.dir
	n.mu.Unlock()
	b, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "state.json"), b, 0o600)
}

func (n *Node) loadDevices() {
	entries, err := os.ReadDir(filepath.Join(n.dir, "devices"))
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(n.dir, "devices", e.Name()))
		if err != nil {
			continue
		}
		var d storedDevice
		if json.Unmarshal(b, &d) != nil || d.ID == "" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(d.Pub)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		d.pub = ed25519.PublicKey(raw)
		n.devices[d.ID] = &d
	}
}

func (n *Node) saveDevice(d *storedDevice) {
	b, _ := json.MarshalIndent(d, "", "  ")
	_ = os.WriteFile(filepath.Join(n.dir, "devices", d.ID+".json"), b, 0o600)
}

func (n *Node) audit(action, id, name string, ok bool) {
	line, _ := json.Marshal(map[string]any{
		"t":      time.Now().UTC().Format(time.RFC3339),
		"action": action,
		"id":     id,
		"name":   name,
		"ok":     ok,
	})
	f, err := os.OpenFile(filepath.Join(n.dir, "audit.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	_, _ = f.Write(append(line, '\n'))
	_ = f.Close()
}

func (n *Node) stage(name, nonce, pubB64, sigB64 string) (*pendingPair, error) {
	pubRaw, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pubRaw) != ed25519.PublicKeySize {
		return nil, errors.New("设备公钥无效")
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("签名无效")
	}
	msg := []byte(signDomain + "\npair\n" + nonce)
	if !ed25519.Verify(ed25519.PublicKey(pubRaw), msg, sig) {
		return nil, errors.New("配对签名对不上")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	off := n.offers[nonce]
	if off == nil || time.Now().After(off.Exp) {
		return nil, errors.New("配对码已过期")
	}
	delete(n.offers, nonce)
	sum := sha256.Sum256(pubRaw)
	id := hex.EncodeToString(sum[:8])
	if old := n.devices[id]; old != nil && !old.Revoked.IsZero() {
		return nil, errors.New("这台手机已被收回")
	}
	dev := &storedDevice{
		ID: id, Name: strings.TrimSpace(name), Pub: pubB64,
		Created: time.Now().UTC(), Seen: time.Now().UTC(), pub: ed25519.PublicKey(pubRaw),
	}
	if dev.Name == "" {
		dev.Name = "手机"
	}
	p := &pendingPair{ID: id, Name: dev.Name, Pub: dev.pub, Exp: off.Exp, done: make(chan bool, 1), dev: dev}
	n.pending[id] = p
	return p, nil
}

// resume lets a phone that was already allowed open a new socket. The QR is
// only the introduction; the device key is the credential after that.
func (n *Node) resume(name, nonce, pubB64, sigB64 string) (*storedDevice, error) {
	pubRaw, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pubRaw) != ed25519.PublicKeySize {
		return nil, errors.New("设备公钥无效")
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("签名无效")
	}
	msg := []byte(signDomain + "\nresume\n" + nonce)
	if !ed25519.Verify(ed25519.PublicKey(pubRaw), msg, sig) {
		return nil, errors.New("签名对不上")
	}
	sum := sha256.Sum256(pubRaw)
	id := hex.EncodeToString(sum[:8])
	if !n.takeNonce(id, nonce, time.Now()) {
		return nil, errors.New("签名过期或已用过")
	}
	d := n.device(id)
	if d == nil {
		return nil, errors.New("这台手机还没被允许。重新扫配对码。")
	}
	if strings.TrimSpace(name) != "" {
		d.Name = strings.TrimSpace(name)
	}
	n.seen(id)
	return d, nil
}

func (n *Node) device(id string) *storedDevice {
	n.mu.Lock()
	defer n.mu.Unlock()
	d := n.devices[id]
	if d == nil || !d.Revoked.IsZero() {
		return nil
	}
	return d
}

func (n *Node) seen(id string) {
	n.mu.Lock()
	d := n.devices[id]
	if d != nil {
		d.Seen = time.Now().UTC()
	}
	n.mu.Unlock()
}

func (n *Node) takeNonce(id, nonce string, ts time.Time) bool {
	if nonce == "" || time.Since(ts) > 2*time.Minute || time.Until(ts) > time.Minute {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	bag := n.nonces[id]
	if bag == nil {
		bag = map[string]time.Time{}
		n.nonces[id] = bag
	}
	now := time.Now()
	for k, at := range bag {
		if now.Sub(at) > 5*time.Minute {
			delete(bag, k)
		}
	}
	if _, ok := bag[nonce]; ok {
		return false
	}
	bag[nonce] = now
	return true
}
