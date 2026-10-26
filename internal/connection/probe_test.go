// 锁住聊天探测走 POST /chat/completions，以及 Pexels、Pixabay、Unsplash 的鉴权头。

package connection

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/vault"
)

// TestTestConnectionFallsBackToDefaultKey 确认连接自己没有钥匙时，探测使用 default 里的钥匙，并且 POST 到 /chat/completions。
func TestTestConnectionFallsBackToDefaultKey(t *testing.T) {
	isolateVaultEnv(t)
	var gotAuth, gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotMethod = r.Method
		if gotAuth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"msg":"Authorization header is required"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()

	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	c, err := r.SeedChat("custom", srv.URL+"/claw/v1", "m")
	if err != nil {
		t.Fatal(err)
	}
	v.Set("default", "gw-token")
	out := TestConnection(c, r)
	if out["ok"] != true {
		t.Fatalf("%+v auth=%q path=%s", out, gotAuth, gotPath)
	}
	if gotAuth != "Bearer gw-token" {
		t.Fatalf("auth %q", gotAuth)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Fatalf("path %s", gotPath)
	}
}

// TestProbeChatCompletionsSkipsModelsRoute 确认探测不请求 /models。
func TestProbeChatCompletionsSkipsModelsRoute(t *testing.T) {
	isolateVaultEnv(t)
	var modelsHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			modelsHits++
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"msg":"Authorization header is required"}`))
			return
		}
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model"`) {
			t.Fatalf("body %s", body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	out := ProbeChatCompletions(srv.URL+"/claw/v1", "tok", "openclaw/default", "custom")
	if out["ok"] != true {
		t.Fatalf("%+v", out)
	}
	if modelsHits != 0 {
		t.Fatalf("GET /models hit %d times", modelsHits)
	}
}

// TestProbeChatCompletionsTreatsModel400AsReachable 确认模型错误的 400 仍算可达。
func TestProbeChatCompletionsTreatsModel400AsReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model not found","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()
	out := ProbeChatCompletions(srv.URL+"/v1", "tok", "missing-model", "custom")
	if out["ok"] != true {
		t.Fatalf("%+v", out)
	}
}

// TestProbePexelsUsesRawAuthorization 确认 Pexels 的 Authorization 是钥匙原文，不是 Bearer。
func TestProbePexelsUsesRawAuthorization(t *testing.T) {
	isolateVaultEnv(t)
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"photos":[]}`))
	}))
	defer srv.Close()

	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	v.Set("default", "px-secret")
	out := TestConnection(Connection{
		Vendor: "pexels", Protocol: "pexels", Endpoint: srv.URL + "/v1",
		Capabilities: []string{CapStock},
	}, r)
	if out["ok"] != true {
		t.Fatalf("%+v auth=%q path=%s", out, gotAuth, gotPath)
	}
	if gotAuth != "px-secret" {
		t.Fatalf("pexels must send the raw key, got %q", gotAuth)
	}
	if !strings.Contains(gotPath, "/curated") {
		t.Fatalf("path %s", gotPath)
	}
}

// TestProbePixabayAndUnsplashAuth 确认 Pixabay 把钥匙放进查询参数，Unsplash 使用 Client-ID。
func TestProbePixabayAndUnsplashAuth(t *testing.T) {
	isolateVaultEnv(t)
	var pixKey, unsplashAuth string
	pix := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pixKey = r.URL.Query().Get("key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":[]}`))
	}))
	defer pix.Close()
	uns := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unsplashAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer uns.Close()

	v := vault.NewFileOnly(t.TempDir())
	r, err := Open(t.TempDir(), v)
	if err != nil {
		t.Fatal(err)
	}
	v.Set("default", "stock-key")
	if out := TestConnection(Connection{
		Vendor: "pixabay", Protocol: "pixabay", Endpoint: pix.URL,
		Capabilities: []string{CapStock},
	}, r); out["ok"] != true {
		t.Fatalf("pixabay %+v", out)
	}
	if pixKey != "stock-key" {
		t.Fatalf("pixabay key %q", pixKey)
	}
	if out := TestConnection(Connection{
		Vendor: "unsplash", Protocol: "unsplash", Endpoint: uns.URL,
		Capabilities: []string{CapStock},
	}, r); out["ok"] != true {
		t.Fatalf("unsplash %+v", out)
	}
	if unsplashAuth != "Client-ID stock-key" {
		t.Fatalf("unsplash auth %q", unsplashAuth)
	}
}

// TestProbeChatCompletionsFailsAuth 确认过期令牌的响应使探测失败，并把错误文本带出来。
func TestProbeChatCompletionsFailsAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"msg":"Invalid or expired token"}`))
	}))
	defer srv.Close()
	out := ProbeChatCompletions(srv.URL+"/v1", "bad", "m", "custom")
	if out["ok"] != false {
		t.Fatalf("%+v", out)
	}
	err, _ := out["error"].(string)
	if !strings.Contains(err, "Invalid or expired token") {
		t.Fatal(err)
	}
}
