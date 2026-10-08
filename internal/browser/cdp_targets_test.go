package browser

import (
	"encoding/json"
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
