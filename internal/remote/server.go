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

type sock struct {
	c  *websocket.Conn
	mu sync.Mutex
}

func (s *sock) send(ctx context.Context, v any) error {
	b, err := json.Marshal(Scrub(v))
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.c.Write(ctx, websocket.MessageText, b)
}

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

func rpcMessage(id, method string, params []byte, ts int64, nonce string) []byte {
	sum := sha256.Sum256(params)
	return []byte(fmt.Sprintf("%s\nrpc\n%s\n%s\n%s\n%d\n%s", signDomain, id, method, hex.EncodeToString(sum[:]), ts, nonce))
}

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
