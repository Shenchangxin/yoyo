// 检查公钥、空正文、latest 回退、暂存，以及 ApplyStaged 留下 .old。

package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPublicKeyFromEnv 确认空环境变量没有公钥，未设置时用编译进来的 32 字节，设置十六进制后能解析。
func TestPublicKeyFromEnv(t *testing.T) {
	t.Setenv(EnvPubkey, "")
	if PublicKeyFromEnv() != nil {
		t.Fatal("empty env")
	}
	if len(PublicKey()) != ed25519.PublicKeySize {
		t.Fatal("embedded key")
	}
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvPubkey, hex.EncodeToString(pub))
	got := PublicKeyFromEnv()
	if len(got) != ed25519.PublicKeySize {
		t.Fatalf("%d", len(got))
	}
}

// TestFetchRejectsEmptyBody 确认空响应返回含 "empty body" 的错误。
func TestFetchRejectsEmptyBody(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/empty.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	})
	SetHTTPClient(srv.Client())
	defer SetHTTPClient(nil)
	_, err := Fetch(t.Context(), srv.URL+"/empty.json", 1<<20, true)
	if err == nil || !strings.Contains(err.Error(), "empty body") {
		t.Fatalf("got %v", err)
	}
}

// TestCheckFallsBackFromEmptyLatest 确认 latest.json 正文为空时改用 nightly.json。
func TestCheckFallsBackFromEmptyLatest(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"version":"9.9.9","channel":"nightly"}`)
	sig := Sign(priv, payload)
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/datasets/acme/yoyo/resolve/master/channels/latest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	})
	mux.HandleFunc("/datasets/acme/yoyo/resolve/master/channels/latest.json.sig", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/datasets/acme/yoyo/resolve/master/channels/nightly.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/datasets/acme/yoyo/resolve/master/channels/nightly.json.sig", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(sig)
	})
	SetHTTPClient(srv.Client())
	defer SetHTTPClient(nil)
	url := srv.URL + "/datasets/acme/yoyo/resolve/master/channels/latest.json"
	res, err := Check(t.Context(), url, nil, nil, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Verified || res.Manifest.Version != "9.9.9" {
		t.Fatalf("%+v", res)
	}
}

// TestCheckAndStage 确认直接给出的清单能验过，Stage 接受匹配的 SHA-256 并拒绝错误摘要。
func TestCheckAndStage(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"version":"9.0.0","channel":"nightly","sha256":"abc"}`)
	sig := Sign(priv, payload)
	res, err := Check(context.Background(), "", payload, sig, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Verified || !res.Newer || res.Manifest.Version != "9.0.0" {
		t.Fatalf("%+v", res)
	}
	bin := []byte("fake-binary")
	sum := sha256.Sum256(bin)
	path, err := Stage(t.TempDir(), bin, hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("empty stage path")
	}
	if _, err := Stage(t.TempDir(), bin, "deadbeef"); err == nil {
		t.Fatal("bad sha accepted")
	}
}

// TestApplyStagedRenamesRunningPath 确认原路径变成新字节，旧字节留在路径加 .old 的文件里。
func TestApplyStagedRenamesRunningPath(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "yoyo.bin")
	if err := os.WriteFile(exe, []byte("old-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(dir, StagingName())
	if err := os.WriteFile(stage, []byte("new-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ApplyStaged(exe, stage); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-bytes" {
		t.Fatalf("%q", got)
	}
	old, err := os.ReadFile(exe + ".old")
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "old-bytes" {
		t.Fatalf("backup %q", old)
	}
}
