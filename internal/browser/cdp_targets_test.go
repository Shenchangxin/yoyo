// 本文件锁住 page 目标优先于 browser 目标，http 页优先于 about:blank，以及截图帧过大不算连接断开。
// 由 go test 调用。

package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPageWSPrefersPageTarget 锁住同时有 browser 和 page 时选 page 的 WebSocket。
func TestPageWSPrefersPageTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/list" && r.URL.Path != "/json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]cdpTarget{
			{Type: "browser", WS: "ws://127.0.0.1:9/devtools/browser/x"},
			{Type: "page", URL: "about:blank", WS: "ws://127.0.0.1:9/devtools/page/p1"},
		})
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	got, err := pageWS(addr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/devtools/page/p1") {
		t.Fatalf("got %s", got)
	}
}

// TestPageWSPrefersHTTPOverAboutBlank 锁住 http 页优先于 about:blank，指定地址时选中那一页。
func TestPageWSPrefersHTTPOverAboutBlank(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/list" && r.URL.Path != "/json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]cdpTarget{
			{Type: "page", URL: "about:blank", WS: "ws://127.0.0.1:9/devtools/page/blank"},
			{Type: "page", URL: "http://127.0.0.1:9/game.html", WS: "ws://127.0.0.1:9/devtools/page/game"},
		})
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	got, err := pageWS(addr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/devtools/page/game") {
		t.Fatalf("got %s", got)
	}
	want := "http://127.0.0.1:9/game.html"
	got, err = pageWSPrefer(addr, want)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/devtools/page/game") {
		t.Fatalf("prefer %s", got)
	}
}

// TestPageWSPreferWaitsWhenWantedHTTPMissing 锁住只有 about:blank 而想要的 http 页不存在时返回错误。
func TestPageWSPreferWaitsWhenWantedHTTPMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]cdpTarget{
			{Type: "page", URL: "about:blank", WS: "ws://127.0.0.1:9/devtools/page/blank"},
		})
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	_, err := pageWSPrefer(addr, "http://127.0.0.1:9/game.html")
	if err == nil {
		t.Fatal("expected wait error while only about:blank exists")
	}
}

// TestCDPMaxMessageExceedsDefaultWebsocketLimit 锁住 cdpMaxMessage 大于 32768，装得下 PNG 截图。
func TestCDPMaxMessageExceedsDefaultWebsocketLimit(t *testing.T) {
	if cdpMaxMessage <= 32768 {
		t.Fatalf("cdpMaxMessage=%d cannot hold a PNG screenshot", cdpMaxMessage)
	}
}

// TestCDPMessageTooBigIsNotGone 锁住 message too big 会被认出来，并且 isCDPGone 为假，不会因此重启 Chrome。
func TestCDPMessageTooBigIsNotGone(t *testing.T) {
	err := fmt.Errorf("failed to read JSON message: failed to read: websocket: message too big: read limited at 32769 bytes")
	if !isCDPMessageTooBig(err) {
		t.Fatal("too big")
	}
	if isCDPGone(err) {
		t.Fatal("message too big must not trigger chrome relaunch")
	}
}
