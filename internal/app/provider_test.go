package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/vault"
)

func TestProviderProbeErrorReadsGatewayMsg(t *testing.T) {
	got := providerProbeError(401, `{"code":401,"msg":"Authorization header is required","error_key":"error.auth.authorization_required"}`)
	if got != "HTTP 401: Authorization header is required" {
		t.Fatal(got)
	}
}

func TestNormalizeAppearanceTrimsBaseURL(t *testing.T) {
	c := Config{BaseURL: "https://server.flowyaipc.com/claw/v1 ", Theme: "dark"}
	c.NormalizeAppearance()
	if c.BaseURL != "https://server.flowyaipc.com/claw/v1" {
		t.Fatal(c.BaseURL)
	}
}

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
