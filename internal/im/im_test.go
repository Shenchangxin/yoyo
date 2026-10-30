package im

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type memVault struct {
	mu sync.Mutex
	m  map[string]string
}

func (v *memVault) Get(name string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	s := v.m[name]
	if s == "" {
		return "", fmt.Errorf("missing")
	}
	return s, nil
}

func (v *memVault) Set(name, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.m == nil {
		v.m = map[string]string{}
	}
	if value == "" {
		delete(v.m, name)
		return
	}
	v.m[name] = value
}

func TestCatalogMatchesRunners(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Catalog() {
		if p.ID == "" || seen[p.ID] {
			t.Fatalf("bad catalog id %q", p.ID)
		}
		seen[p.ID] = true
		if _, ok := runners[p.ID]; !ok {
			t.Fatalf("no runner for %s", p.ID)
		}
		if p.Name.Zh == "" || p.Name.En == "" || p.Summary.Zh == "" || p.Summary.En == "" {
			t.Fatalf("%s missing copy", p.ID)
		}
	}
	if len(seen) != len(runners) {
		t.Fatalf("catalog %d runners %d", len(seen), len(runners))
	}
	if len(seen) < 21 {
		t.Fatalf("catalog shrunk to %d", len(seen))
	}
}

func TestAdmit(t *testing.T) {
	rt, err := Open(t.TempDir(), &memVault{})
	if err != nil {
		t.Fatal(err)
	}
	rt.settings = DefaultSettings()
	base := Inbound{AccountID: "a", SenderID: "u", Text: "hello", Kind: "direct"}
	if got, ok := rt.Admit(base); !ok || got.Text != "hello" {
		t.Fatalf("direct %+v %v", got, ok)
	}
	group := base
	group.Kind = "group"
	if _, ok := rt.Admit(group); ok {
		t.Fatal("group without wake")
	}
	group.Text = "/hello"
	if got, ok := rt.Admit(group); !ok || got.Text != "hello" {
		t.Fatalf("prefix %+v %v", got, ok)
	}
	group.Text = "/"
	if _, ok := rt.Admit(group); ok {
		t.Fatal("empty after prefix")
	}
	group.Text = "hello"
	group.Mentioned = true
	if _, ok := rt.Admit(group); !ok {
		t.Fatal("mention")
	}
	group.AtAll = true
	rt.settings.IgnoreAtAll = true
	group.Mentioned = true
	if _, ok := rt.Admit(group); ok {
		t.Fatal("@all should not wake when ignored and there is no other wake")
	}
	self := base
	self.SelfID = "u"
	if _, ok := rt.Admit(self); ok {
		t.Fatal("self")
	}
	rt.settings.FriendNeedsWake = true
	if _, ok := rt.Admit(base); ok {
		t.Fatal("direct now needs a prefix")
	}
	rt.settings.FriendNeedsWake = false
	rt.rates = nil
	rt.settings.RatePerMinute = 1
	if _, ok := rt.Admit(base); !ok {
		t.Fatal("first")
	}
	if _, ok := rt.Admit(base); ok {
		t.Fatal("rate")
	}
}

func TestSecretsStayOutOfFile(t *testing.T) {
	dir := t.TempDir()
	vault := &memVault{}
	rt, err := Open(dir, vault)
	if err != nil {
		t.Fatal(err)
	}
	acct, err := rt.SaveAccount(Account{Platform: "telegram", Label: "bot", Enabled: false}, map[string]string{"token": "secret-token"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(rt.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-token") || strings.Contains(string(raw), "has_secret") {
		t.Fatalf("secret leaked: %s", raw)
	}
	if vault.m[secretName(acct.ID, "token")] != "secret-token" {
		t.Fatalf("vault %+v", vault.m)
	}
	pub := rt.publicAccount(acct.ID)
	if !pub.HasSecret["token"] {
		t.Fatal("has_secret")
	}
}

func TestTelegramPoll(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"id": 9, "username": "yoyobot"}})
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{}})
		default:
			n++
			if n == 1 {
				fmt.Fprint(w, `{"ok":true,"result":[{"update_id":7,"message":{"message_id":3,"text":"hello","chat":{"id":5,"type":"private"},"from":{"id":4,"first_name":"Ada"}}}]}`)
				return
			}
			<-r.Context().Done()
		}
	}))
	defer srv.Close()
	vault := &memVault{}
	rt, err := Open(t.TempDir(), vault)
	if err != nil {
		t.Fatal(err)
	}
	rt.bases["telegram"] = srv.URL
	acct := Account{ID: "tg", Platform: "telegram", Enabled: true}
	vault.Set(secretName(acct.ID, "token"), "tok")
	got := make(chan Inbound, 1)
	rt.SetHandler(func(in Inbound) {
		got <- in
		if in.Reply != nil {
			_ = in.Reply("pong")
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- rt.runTelegram(ctx, acct) }()
	select {
	case in := <-got:
		if in.ChatID != "5" || in.Text != "hello" || in.SenderName != "Ada" {
			t.Fatalf("%+v", in)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runner")
	}
}

func TestOneBotReverse(t *testing.T) {
	rt, err := Open(t.TempDir(), &memVault{})
	if err != nil {
		t.Fatal(err)
	}
	acct := Account{ID: "ob", Platform: "onebot", Enabled: true, Config: map[string]string{"ws_host": "127.0.0.1", "ws_port": "0"}}
	rt.accounts = []Account{acct}
	got := make(chan Inbound, 1)
	rt.SetHandler(func(in Inbound) {
		got <- in
		if in.Reply != nil {
			_ = in.Reply("pong")
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = rt.runOneBot(ctx, acct) }()
	var addr string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, st := range rt.Snapshot().Statuses {
			if strings.HasPrefix(st.Detail, "ws://") {
				addr = st.Detail
			}
		}
		if addr != "" {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if addr == "" {
		t.Fatal("no listener")
	}
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()
	c, _, err := websocket.Dial(dialCtx, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")
	event := []byte(`{"post_type":"message","message_type":"private","user_id":42,"self_id":7,"message_id":1,"raw_message":"hi","sender":{"nickname":"ada"},"message":[{"type":"text","data":{"text":"hi"}}]}`)
	if err := c.Write(dialCtx, websocket.MessageText, event); err != nil {
		t.Fatal(err)
	}
	select {
	case in := <-got:
		if in.ChatID != "private:42" || in.Text != "hi" || in.SenderName != "ada" {
			t.Fatalf("%+v", in)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("event")
	}
	_, reply, err := c.Read(dialCtx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(reply), "send_private_msg") || !strings.Contains(string(reply), "pong") {
		t.Fatalf("reply %s", reply)
	}
}

func TestLineAndWeixinSignatures(t *testing.T) {
	vault := &memVault{}
	rt, err := Open(t.TempDir(), vault)
	if err != nil {
		t.Fatal(err)
	}
	rt.accounts = []Account{
		{ID: "line1", Platform: "line", Enabled: true},
		{ID: "wx1", Platform: "weixin_official_account", Enabled: true},
	}
	vault.Set(secretName("line1", "channel_secret"), "sec")
	vault.Set(secretName("wx1", "token"), "tok")
	body := []byte(`{"events":[{"type":"message","source":{"type":"user","userId":"U1"},"message":{"type":"text","id":"m1","text":"hi"}}]}`)
	bad := httptest.NewRequest(http.MethodPost, "/im/line1", bytes.NewReader(body))
	bad.Header.Set("X-Line-Signature", "nope")
	rec := httptest.NewRecorder()
	rt.onHook(rec, bad)
	if rec.Code != 401 {
		t.Fatalf("line bad %d", rec.Code)
	}
	got := make(chan Inbound, 1)
	rt.SetHandler(func(in Inbound) { got <- in })
	okReq := httptest.NewRequest(http.MethodPost, "/im/line1", bytes.NewReader(body))
	okReq.Header.Set("X-Line-Signature", hmacSHA256("sec", body))
	rec = httptest.NewRecorder()
	rt.onHook(rec, okReq)
	if rec.Code != 200 {
		t.Fatalf("line %d %s", rec.Code, rec.Body.String())
	}
	select {
	case in := <-got:
		if in.Platform != "line" || in.ChatID != "U1" || in.Text != "hi" {
			t.Fatalf("%+v", in)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("line event")
	}
	sig := sha1Join("tok", "1", "2")
	wx := httptest.NewRequest(http.MethodGet, "/im/wx1?signature="+sig+"&timestamp=1&nonce=2&echostr=ping", nil)
	rec = httptest.NewRecorder()
	rt.onHook(rec, wx)
	if rec.Code != 200 || rec.Body.String() != "ping" {
		t.Fatalf("weixin %d %q", rec.Code, rec.Body.String())
	}
}

func TestWeComDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	aesKey := strings.TrimSuffix(base64.StdEncoding.EncodeToString(key), "=")
	msg := strings.Repeat("a", 12)
	receive := "corp"
	var raw []byte
	raw = append(raw, key[:16]...)
	var nbuf [4]byte
	binary.BigEndian.PutUint32(nbuf[:], uint32(len(msg)))
	raw = append(raw, nbuf[:]...)
	raw = append(raw, msg...)
	raw = append(raw, receive...)
	pad := 32 - len(raw)%32
	if pad == 0 {
		pad = 32
	}
	for i := 0; i < pad; i++ {
		raw = append(raw, byte(pad))
	}
	if pad <= 16 {
		t.Fatalf("pad %d", pad)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, len(raw))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(dst, raw)
	got, err := decryptWeCom(aesKey, base64.StdEncoding.EncodeToString(dst))
	if err != nil {
		t.Fatal(err)
	}
	if got != msg {
		t.Fatalf("got %q", got)
	}
}
