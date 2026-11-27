// 锁住供应商探测的错误文案，以及无密钥和 chat/completions 两条路径。

package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/vault"
)

// TestNormalizeAppearanceTrimsBaseURL 锁住 NormalizeAppearance 去掉 BaseURL 末尾空白。
func TestNormalizeAppearanceTrimsBaseURL(t *testing.T) {
	c := Config{BaseURL: "https://server.flowyaipc.com/claw/v1 ", Theme: "dark"}
	c.NormalizeAppearance()
	if c.BaseURL != "https://server.flowyaipc.com/claw/v1" {
		t.Fatal(c.BaseURL)
	}
}

// TestTestProviderRequiresKey 锁住没有 vault 密钥时 ok 为 false，错误文案是 missing API key — paste the token and Save。
func TestTestProviderRequiresKey(t *testing.T) {
	a := &App{Config: Config{BaseURL: "https://server.flowyaipc.com/claw/v1"}}
	out := a.TestProvider()
	if out["ok"] != false {
		t.Fatalf("%+v", out)
	}
	err, _ := out["error"].(string)
	if err != "missing API key — paste the token and Save" {
		t.Fatal(err)
	}
}

// TestTestProviderUsesChatCompletions 锁住探测走 POST /chat/completions，不把 /models 的 401 当成结果。
func TestTestProviderUsesChatCompletions(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if strings.HasSuffix(r.URL.Path, "/models") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"msg":"no models route"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()
	v := vault.NewFileOnly(t.TempDir())
	v.Set("default", "tok")
	a := &App{Config: Config{BaseURL: srv.URL + "/claw/v1", Model: "openclaw/default", Provider: "custom"}, Vault: v}
	out := a.TestProvider()
	if out["ok"] != true {
		t.Fatalf("%+v path=%s", out, gotPath)
	}
	if gotMethod != http.MethodPost || !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Fatalf("method=%s path=%s", gotMethod, gotPath)
	}
}
