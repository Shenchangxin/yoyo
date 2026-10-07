package connection

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/vault"
)

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
