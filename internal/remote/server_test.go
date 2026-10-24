package remote

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func evalsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestPairModelAndStepUp(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	a.Config.SearchKey = "search-secret"
	n, err := Bind(a)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Close() })

	offer, err := n.BeginPair()
	if err != nil {
		t.Fatal(err)
	}
	uri := offer["uri"].(string)
	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if q.Get("fp") == "" || q.Get("n") == "" || offer["png"] == "" {
		t.Fatalf("offer %+v", offer)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pairSig := ed25519.Sign(priv, []byte(signDomain+"\npair\n"+q.Get("n")))
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			sum := sha256.Sum256(raw[0])
			if hex.EncodeToString(sum[:]) != q.Get("fp") {
				return fmt.Errorf("fingerprint")
			}
			return nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "wss://127.0.0.1:"+q.Get("p")+"/remote/ws", &websocket.DialOptions{
		HTTPClient: &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	write := func(v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
			t.Fatal(err)
		}
	}
	read := func() map[string]any {
		t.Helper()
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if json.Unmarshal(data, &m) != nil {
			t.Fatalf("bad frame %s", data)
		}
		return m
	}
	readType := func(want string) map[string]any {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			m := read()
			if m["type"] == want {
				return m
			}
		}
		t.Fatalf("missing %s", want)
		return nil
	}
	write(map[string]any{
		"type": "pair",
		"name": "Pixel",
		"n":    q.Get("n"),
		"pub":  base64.StdEncoding.EncodeToString(pub),
		"sig":  base64.StdEncoding.EncodeToString(pairSig),
	})
	wait := readType("pair_wait")
	if _, err := n.Approve(fmt.Sprint(wait["id"])); err != nil {
		t.Fatal(err)
	}
	if ok := readType("pair_ok"); ok["name"] != "Pixel" {
		t.Fatalf("pair_ok %+v", ok)
	}

	call := func(method string, params []byte, confirm bool) map[string]any {
		t.Helper()
		id := method
		ts := time.Now().Unix()
		nonce := method + fmt.Sprint(ts)
		sum := sha256.Sum256(params)
		msg := fmt.Sprintf("%s\nrpc\n%s\n%s\n%s\n%d\n%s", signDomain, id, method, hex.EncodeToString(sum[:]), ts, nonce)
		sig := ed25519.Sign(priv, []byte(msg))
		body := map[string]any{
			"type": "rpc", "id": id, "method": method, "params": json.RawMessage(params),
			"ts": ts, "nonce": nonce, "sig": base64.StdEncoding.EncodeToString(sig), "confirm": confirm,
		}
		write(body)
		return readType("rpc")
	}

	started := call("thread.start", []byte(`{"workspace":"","channel":"agent"}`), false)
	if started["error"] != nil {
		t.Fatalf("start %v", started["error"])
	}
	result := started["result"].(map[string]any)
	sid := fmt.Sprint(result["id"])
	if sid == "" {
		t.Fatalf("result %+v", result)
	}
	model := call("thread.model.set", []byte(fmt.Sprintf(`{"session":%q,"model":"gpt-handheld"}`, sid)), false)
	if model["error"] != nil {
		t.Fatalf("model %v", model["error"])
	}
	meta, err := a.GetSession(sid)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "gpt-handheld" {
		t.Fatalf("pc model %q", meta.Model)
	}
	if a.StateEpoch() == 0 {
		t.Fatal("model change did not publish state")
	}
	denied := call("key.set", []byte(`{"key":"sk-test"}`), false)
	if denied["error"] == nil {
		t.Fatal("key write without confirmation")
	}
	mint := call("remote.pair.begin", []byte(`{}`), true)
	if mint["error"] == nil {
		t.Fatal("phone minted a pairing")
	}
	raw, _ := json.Marshal(Scrub(a.Config))
	if strings.Contains(string(raw), "search-secret") {
		t.Fatalf("search key survived scrub %s", raw)
	}

	conn.Close(websocket.StatusNormalClosure, "")
	again, _, err := websocket.Dial(ctx, "wss://127.0.0.1:"+q.Get("p")+"/remote/ws", &websocket.DialOptions{
		HTTPClient: &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close(websocket.StatusNormalClosure, "")
	resumeNonce := hex.EncodeToString([]byte("resume-nonce-1"))
	resumeSig := ed25519.Sign(priv, []byte(signDomain+"\nresume\n"+resumeNonce))
	rb, _ := json.Marshal(map[string]any{
		"type": "resume",
		"name": "Pixel",
		"n":    resumeNonce,
		"pub":  base64.StdEncoding.EncodeToString(pub),
		"sig":  base64.StdEncoding.EncodeToString(resumeSig),
	})
	if err := again.Write(ctx, websocket.MessageText, rb); err != nil {
		t.Fatal(err)
	}
	_, data, err := again.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var resumed map[string]any
	if json.Unmarshal(data, &resumed) != nil || resumed["type"] != "pair_ok" {
		t.Fatalf("resume %s", data)
	}
	readAgain := func(want string) map[string]any {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			_, data, err := again.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if json.Unmarshal(data, &m) != nil {
				t.Fatalf("bad frame %s", data)
			}
			if m["type"] == want {
				return m
			}
		}
		t.Fatalf("missing %s", want)
		return nil
	}
	writeRPC := func(nonce string) {
		t.Helper()
		params := []byte(`{}`)
		ts := time.Now().Unix()
		sum := sha256.Sum256(params)
		msg := fmt.Sprintf("%s\nrpc\nreplay\nhealth\n%s\n%d\n%s", signDomain, hex.EncodeToString(sum[:]), ts, nonce)
		sig := ed25519.Sign(priv, []byte(msg))
		body, _ := json.Marshal(map[string]any{
			"type": "rpc", "id": "replay", "method": "health", "params": json.RawMessage(params),
			"ts": ts, "nonce": nonce, "sig": base64.StdEncoding.EncodeToString(sig),
		})
		if err := again.Write(ctx, websocket.MessageText, body); err != nil {
			t.Fatal(err)
		}
	}
	writeRPC("once-nonce")
	if got := readAgain("rpc"); got["error"] != nil {
		t.Fatalf("first health %v", got["error"])
	}
	writeRPC("once-nonce")
	if got := readAgain("rpc"); !strings.Contains(fmt.Sprint(got["error"]), "已用过") {
		t.Fatalf("replay %v", got["error"])
	}
	if _, err := n.Revoke(fmt.Sprint(resumed["id"])); err != nil {
		t.Fatal(err)
	}
	writeRPC("after-revoke")
	if got := readAgain("rpc"); !strings.Contains(fmt.Sprint(got["error"]), "收回") {
		t.Fatalf("revoked %v", got["error"])
	}
	closed, _, err := websocket.Dial(ctx, "wss://127.0.0.1:"+q.Get("p")+"/remote/ws", &websocket.DialOptions{
		HTTPClient: &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer closed.Close(websocket.StatusNormalClosure, "")
	if err := closed.Write(ctx, websocket.MessageText, []byte(`{"type":"rpc","id":"x","method":"health","params":{}}`)); err != nil {
		t.Fatal(err)
	}
	_, closedData, err := closed.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var closedFrame map[string]any
	if json.Unmarshal(closedData, &closedFrame) != nil || closedFrame["type"] != "error" {
		t.Fatalf("unpaired %s", closedData)
	}
}
