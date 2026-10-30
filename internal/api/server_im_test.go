package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func TestIMHTTPRoundTrip(t *testing.T) {
	home := t.TempDir()
	a, err := app.Open(home, evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.IM == nil {
		t.Fatal("im runtime missing")
	}
	h := Handler(a, nil)

	res := serveIM(t, h, http.MethodGet, "/api/im", nil)
	if res.Code != 200 {
		t.Fatalf("snapshot %d %s", res.Code, res.Body.String())
	}
	var snap struct {
		Catalog []struct {
			ID string `json:"id"`
		} `json:"catalog"`
		Settings struct {
			ListenHost string `json:"listen_host"`
		} `json:"settings"`
		HookBase string `json:"hook_base"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if len(snap.Catalog) < 21 {
		t.Fatalf("catalog %d", len(snap.Catalog))
	}
	if snap.Settings.ListenHost != "127.0.0.1" {
		t.Fatalf("host %q", snap.Settings.ListenHost)
	}
	if !strings.Contains(snap.HookBase, "127.0.0.1:8787/im/") {
		t.Fatalf("hook %q", snap.HookBase)
	}

	save := serveIM(t, h, http.MethodPut, "/api/im/accounts", map[string]any{
		"account": map[string]any{
			"platform": "telegram",
			"label":    "desk",
			"enabled":  false,
			"config":   map[string]string{},
		},
		"secrets": map[string]string{"token": "secret-token-value"},
	})
	if save.Code != 200 {
		t.Fatalf("save %d %s", save.Code, save.Body.String())
	}
	if strings.Contains(save.Body.String(), "secret-token-value") {
		t.Fatal("token leaked in save response")
	}
	var acct struct {
		ID        string          `json:"id"`
		HasSecret map[string]bool `json:"has_secret"`
	}
	if err := json.Unmarshal(save.Body.Bytes(), &acct); err != nil {
		t.Fatal(err)
	}
	if acct.ID == "" || !acct.HasSecret["token"] {
		t.Fatalf("%+v", acct)
	}
	raw, err := os.ReadFile(filepath.Join(home, "im", "im.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-token-value") {
		t.Fatal("token leaked into im.json")
	}

	again := serveIM(t, h, http.MethodGet, "/api/im", nil)
	if strings.Contains(again.Body.String(), "secret-token-value") {
		t.Fatal("token leaked in snapshot")
	}

	del := serveIM(t, h, http.MethodDelete, "/api/im/accounts/"+acct.ID, nil)
	if del.Code != 200 {
		t.Fatalf("delete %d %s", del.Code, del.Body.String())
	}
}

func serveIM(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rdr)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
