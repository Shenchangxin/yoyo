// 本文件拥有手机的 TLS WebSocket 和视频下载。第一帧是配对或恢复，之后是签名 RPC，另有 /remote/media。
// 不签发配对码，也不接待浏览器的明文 HTTP。listen 把这两个路径挂到 TLS 端口。

package remote

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
)

// sock 是一条已接受的 WebSocket。写出前先洗密钥，写操作互斥。
type sock struct {
	c  *websocket.Conn
	mu sync.Mutex
}

// send 先 Scrub 再编码成文本帧。这次编码失败就不写。Scrub 因解码失败返回原值，而这次编码又能成功时，发出去的值没洗过。
// ctx 传给连接的 Write。
// v 传给 Scrub，再交给 json.Marshal。
func (s *sock) send(ctx context.Context, v any) error {
	b, err := json.Marshal(Scrub(v))
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.c.Write(ctx, websocket.MessageText, b)
}

// inbound 是手机发来的一帧。配对和恢复的随机数先看 n，空了再看 nonce。RPC 只用 nonce。
type inbound struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Name    string          `json:"name"`
	Pub     string          `json:"pub"`
	Nonce   string          `json:"nonce"`
	N       string          `json:"n"`
	Sig     string          `json:"sig"`
	Ts      int64           `json:"ts"`
	Confirm bool            `json:"confirm"`
}

// serveWS 先看私网来源，公网地址回 403。通过之后 WebSocket 的 Origin 用通配，不再按站点拒绝。
// 第一帧必须是 pair 或 resume，否则回先配对并结束。恢复失败不会再当成配对。
// pair 交给 stage，发 pair_wait，然后等到电脑允许、到达出示时的 Exp，或连接取消。拒绝回电脑拒绝了这台手机，到期回配对码已过期。
// resume 验过已允许的设备密钥就直接 pair_ok。允许之后 pump 推送事件，并按帧调用 onRPC。
func (n *Node) serveWS(w http.ResponseWriter, r *http.Request) {
	if !privatePeer(r.RemoteAddr) {
		http.Error(w, "lan only", http.StatusForbidden)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	s := &sock{c: conn}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		return
	}
	var frame inbound
	if json.Unmarshal(data, &frame) != nil {
		_ = s.send(ctx, map[string]any{"type": "error", "error": "先配对"})
		return
	}
	if frame.Type == "resume" {
		nonce := frame.N
		if nonce == "" {
			nonce = frame.Nonce
		}
		dev, err := n.resume(frame.Name, nonce, frame.Pub, frame.Sig)
		if err != nil {
			_ = s.send(ctx, map[string]any{"type": "error", "error": err.Error()})
			return
		}
		_ = s.send(ctx, map[string]any{"type": "pair_ok", "id": dev.ID, "name": dev.Name, "epoch": n.app.StateEpoch()})
		n.pump(ctx, s)
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			n.onRPC(ctx, s, dev, data)
		}
	}
	if frame.Type != "pair" {
		_ = s.send(ctx, map[string]any{"type": "error", "error": "先配对"})
		return
	}
	nonce := frame.N
	if nonce == "" {
		nonce = frame.Nonce
	}
	pending, err := n.stage(frame.Name, nonce, frame.Pub, frame.Sig)
	if err != nil {
		_ = s.send(ctx, map[string]any{"type": "error", "error": err.Error()})
		return
	}
	_ = s.send(ctx, map[string]any{"type": "pair_wait", "id": pending.ID, "name": pending.Name})
	var allowed bool
	select {
	case allowed = <-pending.done:
	case <-time.After(time.Until(pending.Exp)):
		n.mu.Lock()
		delete(n.pending, pending.ID)
		n.mu.Unlock()
		_ = s.send(ctx, map[string]any{"type": "error", "error": "配对码已过期"})
		return
	case <-ctx.Done():
		return
	}
	if !allowed {
		_ = s.send(ctx, map[string]any{"type": "error", "error": "电脑拒绝了这台手机"})
		return
	}
	_ = s.send(ctx, map[string]any{"type": "pair_ok", "id": pending.ID, "name": pending.Name, "epoch": n.app.StateEpoch()})
	n.pump(ctx, s)
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		n.onRPC(ctx, s, pending.dev, data)
	}
}

// pump 把 Hub 的全部事件收成 type 为 live 的帧，把状态订阅收成 type 为 state 的帧。Hub 为空就跳过实时这一路。app 为空就两路都不订。
// ctx 在两个订阅循环里看 Done，并传给 send。
func (n *Node) pump(ctx context.Context, s *sock) {
	if n.app != nil && n.app.Hub != nil {
		ch, unsub := n.app.Hub.Subscribe("*")
		go func() {
			defer unsub()
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-ch:
					if !ok {
						return
					}
					_ = s.send(ctx, map[string]any{"type": "live", "event": app.NoticeOf(ev)})
				}
			}
		}()
	}
	if n.app == nil {
		return
	}
	ch, unsub := n.app.SubscribeState()
	go func() {
		defer unsub()
		for {
			select {
			case <-ctx.Done():
				return
			case change, ok := <-ch:
				if !ok {
					return
				}
				_ = s.send(ctx, map[string]any{"type": "state", "epoch": change.Epoch, "topic": change.Topic, "object": change.Object})
			}
		}
	}()
}

// onRPC 处理配对之后的一帧。解不开就回无法解析，不记审计。
// 设备已收回、nonce 不合用、验签失败、PhoneDenied，或需要确认却没带 confirm，都记失败审计，不更新 Seen。
// 参数为空时按 {} 参与摘要。通过这些检查后交给 api.Dispatch，无论业务是否出错都更新 Seen。成功的回复带上当前 epoch。写出前仍走 send 的洗密钥。
// ctx 传给 send 和 api.Dispatch。
// dev 的 ID 传给 device、takeNonce、seen 和 audit，Name 传给 audit，pub 参与验签。
// data 交给 json.Unmarshal。解不开就回无法解析。
func (n *Node) onRPC(ctx context.Context, s *sock, dev *storedDevice, data []byte) {
	var frame inbound
	if json.Unmarshal(data, &frame) != nil {
		_ = s.send(ctx, map[string]any{"type": "error", "error": "无法解析"})
		return
	}
	reply := map[string]any{"type": "rpc", "id": frame.ID, "method": frame.Method}
	fail := func(msg string) {
		reply["error"] = msg
		n.audit(frame.Method, dev.ID, dev.Name, false)
		_ = s.send(ctx, reply)
	}
	if n.device(dev.ID) == nil {
		fail("设备已收回")
		return
	}
	raw := frame.Params
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	ts := time.Unix(frame.Ts, 0)
	if !n.takeNonce(dev.ID, frame.Nonce, ts) {
		fail("签名过期或已用过")
		return
	}
	sig, err := base64.StdEncoding.DecodeString(frame.Sig)
	if err != nil || !ed25519.Verify(dev.pub, rpcMessage(frame.ID, frame.Method, raw, frame.Ts, frame.Nonce), sig) {
		fail("签名对不上")
		return
	}
	if PhoneDenied(frame.Method) {
		fail("配对只能在电脑上开始")
		return
	}
	if NeedsConfirm(frame.Method) && !frame.Confirm {
		fail("这一步需要在手机上确认")
		return
	}
	res := api.Dispatch(ctx, n.app, api.RPCRequest{JSONRPC: "2.0", ID: frame.ID, Method: frame.Method, Params: raw})
	n.seen(dev.ID)
	if res.Error != nil {
		reply["error"] = res.Error.Message
		n.audit(frame.Method, dev.ID, dev.Name, false)
		_ = s.send(ctx, reply)
		return
	}
	reply["result"] = res.Result
	reply["epoch"] = n.app.StateEpoch()
	n.audit(frame.Method, dev.ID, dev.Name, true)
	_ = s.send(ctx, reply)
}

// rpcMessage 拼 RPC 的签名原文：signDomain、rpc、id、方法、参数 SHA-256 的十六进制、秒级时间戳、nonce。行序不能换。
// method 写进签名原文。
// params 做 SHA-256，十六进制写进签名原文。
// ts 按十进制写进签名原文。
func rpcMessage(id, method string, params []byte, ts int64, nonce string) []byte {
	sum := sha256.Sum256(params)
	return []byte(fmt.Sprintf("%s\nrpc\n%s\n%s\n%s\n%d\n%s", signDomain, id, method, hex.EncodeToString(sum[:]), ts, nonce))
}

// serveMedia 只接受私网来源。设备号、时间戳、nonce 和签名都在请求头里。
// 签名原文是 signDomain、get、/remote/media、查询参数 path 的原文、时间戳、nonce。验签通过后才打开文件。文件正文不洗密钥。
// w 在来源不是私网、设备无效、验签失败或打开文件失败时写错误。通过后传给 http.ServeContent。
func (n *Node) serveMedia(w http.ResponseWriter, r *http.Request) {
	if !privatePeer(r.RemoteAddr) {
		http.Error(w, "lan only", http.StatusForbidden)
		return
	}
	dev := n.device(r.Header.Get("X-Yoyo-Device"))
	if dev == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	path := r.URL.Query().Get("path")
	ts, _ := strconv.ParseInt(r.Header.Get("X-Yoyo-Ts"), 10, 64)
	nonce := r.Header.Get("X-Yoyo-Nonce")
	sig, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Yoyo-Sig"))
	msg := []byte(fmt.Sprintf("%s\nget\n/remote/media\n%s\n%d\n%s", signDomain, path, ts, nonce))
	if err != nil || !n.takeNonce(dev.ID, nonce, time.Unix(ts, 0)) || !ed25519.Verify(dev.pub, msg, sig) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	f, err := n.openMedia(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

// openMedia 只打开主目录 video 下的普通文件。相对路径接到这个目录上，绝对路径也必须落在里面。
// 越界、目录，或没有 home，都拒绝。
// path 不是绝对路径时接到 video 目录上，是绝对路径时也必须落在该目录里，然后交给 os.Open。
func (n *Node) openMedia(path string) (*os.File, error) {
	if n.app == nil || n.app.Home == nil {
		return nil, errors.New("no home")
	}
	root := filepath.Clean(filepath.Join(n.app.Home.Root, "video"))
	full := filepath.Clean(path)
	if !filepath.IsAbs(full) {
		full = filepath.Clean(filepath.Join(root, path))
	}
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("媒体只在这台电脑的视频目录里")
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		_ = f.Close()
		return nil, errors.New("不是文件")
	}
	return f, nil
}
