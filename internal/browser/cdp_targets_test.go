package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestCDPMaxMessageExceedsDefaultWebsocketLimit(t *testing.T) {
	if cdpMaxMessage <= 32768 {
		t.Fatalf("cdpMaxMessage=%d cannot hold a PNG screenshot", cdpMaxMessage)
	}
}

func TestCDPMessageTooBigIsNotGone(t *testing.T) {
	err := fmt.Errorf("failed to read JSON message: failed to read: websocket: message too big: read limited at 32769 bytes")
	if !isCDPMessageTooBig(err) {
		t.Fatal("too big")
	}
	if isCDPGone(err) {
		t.Fatal("message too big must not trigger chrome relaunch")
	}
}
