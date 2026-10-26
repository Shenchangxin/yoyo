// Package remote 是本进程里给手机 App 和手机浏览器用的门。
// 手机 App 走 TLS WebSocket，用二维码里的 fp 钉证书。fp 是叶子证书 DER 的 SHA-256。
// 浏览器门是另一端口上的明文 HTTP，不是加密通道。证书 DNS 名只有 yoyo.local，没有 IP，浏览器不会在局域网地址上信任那条 TLS。
// 配对码是 16 字节随机数，从出示起 2 分钟有效。手机 stage 和浏览器 takeOffer 谁先核销谁赢。
// WebSocket 文本帧在 Scrub 成功时按字段名丢掉密钥。浏览器只洗内容类型含 json、且重新编码成功的响应，其余原样出去。
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

// signDomain 是和手持客户端约定的签名域，值是 yoyo.remote.v1。行序是约定的一部分，不能换。
// 配对三行：这个域、pair、nonce。恢复三行：这个域、resume、nonce。
// RPC 七行：这个域、rpc、id、方法、参数 SHA-256 的十六进制、秒级时间戳、nonce。
// 取媒体六行：这个域、get、/remote/media、路径原文、秒级时间戳、nonce。
const signDomain = "yoyo.remote.v1"

// storedDevice 是一台手机。Pub 是 base64 的 Ed25519 公钥。pub 是解开后的公钥，不进 JSON。
type storedDevice struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Pub     string    `json:"pub"`
	Created time.Time `json:"created"`
	Seen    time.Time `json:"seen"`
	Revoked time.Time `json:"revoked,omitempty"`
	pub     ed25519.PublicKey
}

// offer 是 BeginPair 放下、还没被 stage 或 takeOffer 核销的配对码。
type offer struct {
	Nonce string
	Exp   time.Time
}

// pendingPair 是手机已经验签、正在等电脑点允许或拒绝的一次配对。
// Exp 沿用出示时的到期时刻，不从扫码重新计。done 缓冲 1。
type pendingPair struct {
	ID   string
	Name string
	Pub  ed25519.PublicKey
	Exp  time.Time
	done chan bool
	dev  *storedDevice
}

// diskState 是 state.json 的形状，只存开关和 TLS 端口，不存浏览器端口。
type diskState struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// Node 是本进程里的手持门。手机 WebSocket 走 TLS，浏览器走另一端口的明文 HTTP。
type Node struct {
	app *app.App
	dir string

	mu        sync.Mutex
	ln        net.Listener // 手机 TLS
	srv       *http.Server
	httpLn    net.Listener // 浏览器的明文 HTTP
	httpSrv   *http.Server
	fp        string // 叶子证书 DER 的 SHA-256 十六进制
	port      int    // TLS 端口。磁盘上没有时 listen 从 47321 起
	httpPort  int    // 浏览器端口。serveBrowser 没听上时保持 0
	enabled   bool
	listenErr string
	offers    map[string]*offer       // 还没核销的配对码，键是十六进制 nonce
	pending   map[string]*pendingPair // 已验签、等电脑点允许的手机
	devices   map[string]*storedDevice
	browsers  map[string]*browserHold
	sessions  map[string]time.Time // 浏览器令牌到过期时刻
	static    http.Handler
	pages     http.Handler
	pagesFor  http.Handler // 组 pages 时用的那份 static。static 一换就重做
	nonces    map[string]map[string]time.Time
	mdnsStop  func()
}

// current 是本进程 Bind 存下的节点。
var current atomic.Pointer[Node]

// Current 返回本进程里 Bind 存下的节点。还没有就是 nil。
func Current() *Node { return current.Load() }

// Bind 在主目录下建 remote/devices，权限 0o700，载入设备和 state.json，并向 API 登记 handle。
// 磁盘上 enabled 为真就 listen。听失败只写进 listenErr，仍然返回这个节点和空错误。
// home 为空时返回错误，不登记，也不写入 current。
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

// Close 停掉 mDNS。两台服务器各最多等 2 秒关机，然后关掉监听。
// 浏览器会话、等待中的浏览器和页面处理器会清掉。磁盘上的设备和证书不动，current 也不摘。
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

// handle 只处理挂在这个节点自己的 App 上的 remote 方法。
// remote.status、remote.pair.begin、remote.enable、remote.pair.approve、remote.pair.deny、remote.revoke、remote.disable 在这里完成。
// App 对不上，或方法不在这张名单里，就返回 ok 假，让 API 继续往下分。
// method 在 n 非 nil 且 a 就是本节点的 App 时进入 switch。命中 remote.status、remote.pair.begin、remote.enable、remote.pair.approve、remote.pair.deny、remote.revoke、remote.disable 时第二个返回值为真，否则为假。
// params 交给 json.Unmarshal，取出 id 和 name。
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

// Status 返回是否在听、开关、TLS 端口、浏览器端口、证书指纹、局域网主机、待允许项、设备和仍有效的浏览器。
// 还没点完的浏览器放进 pending，名字是浏览器。已允许、令牌还在、且会话没过期的放进 browsers。
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

// BeginPair 需要时先 listen，再签发一枚配对码并返回二维码。
// 随机数是 16 字节，十六进制后放进 offers。到期是这次调用起 2 分钟。手机 stage 和浏览器 takeOffer 谁先把这枚码从 offers 删掉，谁就用掉它，后到的得到配对码已过期。
// http 端口为 0 时链接是 yoyo://pair。否则链接是明文 http://主机:http端口/open，查询串带 v、主机 h、TLS 端口 p、证书指纹 fp、配对码 n 和 Unix 到期秒。这扇 HTTP 不是加密通道。手机在 TLS 端口用 fp 钉证书，fp 是叶子证书 DER 的 SHA-256。
// 主机取 lanHosts 的第一项，没有则是空字符串。返回的 exp 是 UTC 的 RFC3339。png 是边长 256、纠错 Medium 的 PNG，再用标准 base64。二维码编码失败时 offers 里已经有这枚码，调用方拿不到它。
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

// Approve 先允许同号的浏览器。没有再允许等待中的手机，并把它写进设备。两边都没有就返回没有等待中的手机。
// id 传给 approveBrowser。没有浏览器时用来查 pending，并写入返回值的 id。
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

// Deny 先拒绝同号的浏览器。没有再拒绝等待中的手机。两边都没有就返回没有等待中的手机。
// id 传给 denyBrowser。没有浏览器时用来查 pending。
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

// finish 在允许时把设备放进内存并写盘，记一条 pair 审计，再从 pending 去掉。拒绝时只去掉 pending，不写审计。
// 结果送进 done。缓冲已满就丢掉，避免卡住电脑这边。
// ok 为真时把设备放进 devices、写盘、记 pair 审计，并从 pending 去掉。为假时只去掉 pending。然后送进 done。
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

// Revoke 把这台设备的收回时间写成现在并写盘。没有这台设备就返回错误。
// 已经连着的套接字要等下一帧 RPC 才看见收回。
// id 用来查 devices。没有就返回错误。找到时把 Revoked 写成现在。
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

// Disable 把 enabled 写成假，清掉 listenErr，保存后再 Close。Close 会停掉两扇门并清掉浏览器会话。
func (n *Node) Disable() error {
	n.mu.Lock()
	n.enabled = false
	n.listenErr = ""
	n.mu.Unlock()
	n.saveState()
	return n.Close()
}

// listen 已经在听就直接返回。端口为 0 时用 47321。这个端口绑不上就改听系统分配的端口。
// 证书装上后提供 /remote/ws 和 /remote/media，读请求头超时 10 秒，地址是所有网卡上的这个端口。
// 随后打开浏览器门并做 mDNS，再把 enabled 和 TLS 端口写盘。
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

// loadState 从 state.json 读 enabled 和 TLS 端口。文件没有或解不开就保持零值。
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

// saveState 把 enabled 和 TLS 端口写成 state.json，权限 0o600。浏览器端口不写。写失败忽略。
func (n *Node) saveState() {
	n.mu.Lock()
	st := diskState{Enabled: n.enabled, Port: n.port}
	dir := n.dir
	n.mu.Unlock()
	b, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "state.json"), b, 0o600)
}

// loadDevices 读取 devices 目录里的 JSON。公钥解不开，或长度不是 ed25519.PublicKeySize，就跳过。
// 已经收回的设备也载入，之后由 device 滤掉。
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

// saveDevice 把设备写成 devices 下以 ID 命名的 JSON，权限 0o600。解开的 pub 不进文件。
func (n *Node) saveDevice(d *storedDevice) {
	b, _ := json.MarshalIndent(d, "", "  ")
	_ = os.WriteFile(filepath.Join(n.dir, "devices", d.ID+".json"), b, 0o600)
}

// audit 往 audit.jsonl 追加一行：时间、动作、设备号、名字、是否成功。打开失败就放弃。文件权限 0o600。
// action 写入 action。
// id 写入 id。
// name 写入 name。
// ok 写入 ok。
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

// stage 核验配对签名。签名原文是 signDomain、pair、nonce 三行。
// 配对码不存在，或此刻已经晚于 Exp，就返回过期。检查通过后立刻从 offers 删掉，不放回去。和浏览器抢同一枚码时，谁先删谁用掉。
// 设备号是公钥 SHA-256 的前 8 字节，写成十六进制。这台手机已经收回也返回错误，但码已经消耗。名字空着就写成手机。到期时刻沿用出示时的 Exp。
// name 去掉空白后写入设备名。空字符串写成手机。
// pubB64 用标准 Base64 解码，长度必须等于 ed25519.PublicKeySize，并写入设备的 Pub。
// sigB64 用标准 Base64 解码后参与 ed25519.Verify。
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

// resume 让已经允许过的手机另开一条连接。配对码只负责介绍，之后的凭据是设备密钥。
// 签名原文是 signDomain、resume、nonce 三行。nonce 用 takeNonce 记在现在，不看客户端时间戳。还没允许就让它重新扫配对码。传来的名字非空就改显示名。
// name 去掉空白后非空时写入设备的 Name。
// pubB64 用标准 Base64 解码，长度必须等于 ed25519.PublicKeySize。
// sigB64 用标准 Base64 解码后参与 ed25519.Verify。
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

// device 返回还没收回的设备。没有，或 Revoked 不是零值，就返回 nil。
// id 用来查 devices。
func (n *Node) device(id string) *storedDevice {
	n.mu.Lock()
	defer n.mu.Unlock()
	d := n.devices[id]
	if d == nil || !d.Revoked.IsZero() {
		return nil
	}
	return d
}

// seen 把这台设备的 Seen 改成现在。只改内存，不写盘。
// id 用来查 devices。找到时把 Seen 写成现在。
func (n *Node) seen(id string) {
	n.mu.Lock()
	d := n.devices[id]
	if d != nil {
		d.Seen = time.Now().UTC()
	}
	n.mu.Unlock()
}

// takeNonce 收下这枚 nonce。空字符串直接拒绝。时间戳早于现在超过 2 分钟，或晚于现在超过 1 分钟，也拒绝。
// 同一设备 5 分钟内见过的 nonce 不能再用。顺手删掉这只袋子里早于 5 分钟的记录。
// id 用来取 nonces 里的袋子，没有就新建。
// ts 早于现在超过 2 分钟，或晚于现在超过 1 分钟时返回 false。
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
