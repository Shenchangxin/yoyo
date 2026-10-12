package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeedURL(t *testing.T) {
	t.Setenv(EnvRepo, "acme/yoyo-releases")
	t.Setenv(EnvEndpoint, "https://www.modelscope.cn")
	u := FeedURL(ChannelStable)
	if !strings.Contains(u, "/datasets/acme/yoyo-releases/resolve/master/channels/stable.json") {
		t.Fatal(u)
	}
	if SigURL(u) != u+".sig" {
		t.Fatal(SigURL(u))
	}
}

func TestCheckDownloadRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("installer-bytes")
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])
	var manifest []byte
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	m := Manifest{
		Schema:  1,
		Version: "9.9.9",
		Channel: "stable",
		Platforms: map[string]Asset{
			DesktopPlatform(): {URL: srv.URL + "/pkg.bin", Filename: "pkg.bin", SHA256: sha, Size: int64(len(payload)), Kind: "bin"},
			CLIPlatform():     {URL: srv.URL + "/pkg.bin", Filename: "pkg.bin", SHA256: sha, Size: int64(len(payload)), Kind: "bin"},
		},
	}
	manifest, err = json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	sig := Sign(priv, manifest)
	mux.HandleFunc("/latest.json", func(w http.ResponseWriter, r *http.Request) { w.Write(manifest) })
	mux.HandleFunc("/latest.json.sig", func(w http.ResponseWriter, r *http.Request) { w.Write(sig) })
	mux.HandleFunc("/pkg.bin", func(w http.ResponseWriter, r *http.Request) { io.Copy(w, strings.NewReader(string(payload))) })

	SetHTTPClient(srv.Client())
	defer SetHTTPClient(nil)

	res, err := Check(t.Context(), srv.URL+"/latest.json", nil, nil, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Newer || res.Asset.URL == "" {
		t.Fatalf("%+v", res)
	}
	dir := t.TempDir()
	path, err := Download(t.Context(), res.Asset, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("%q", got)
	}
	if filepath.Base(path) != "pkg.bin" {
		t.Fatal(path)
	}
}

func TestAPIFileURLFromResolve(t *testing.T) {
	got := APIFileURLFromResolve("https://www.modelscope.cn/datasets/acme/yoyo-releases/resolve/master/channels/stable.json")
	if !strings.Contains(got, "/api/v1/datasets/acme/yoyo-releases/repo?") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "FilePath=channels%2Fstable.json") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "Revision=master") {
		t.Fatal(got)
	}
	if APIFileURLFromResolve("https://example.com/file.bin") != "" {
		t.Fatal("non-resolve")
	}
}

func TestFetchFallsBackToAPI(t *testing.T) {
	body := []byte(`{"version":"9.9.9","channel":"stable"}`)
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/datasets/acme/yoyo-releases/resolve/master/channels/stable.json", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	})
	mux.HandleFunc("/api/v1/datasets/acme/yoyo-releases/repo", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("FilePath") != "channels/stable.json" {
			http.Error(w, "bad path", 400)
			return
		}
		w.Write(body)
	})
	SetHTTPClient(srv.Client())
	defer SetHTTPClient(nil)
	raw := srv.URL + "/datasets/acme/yoyo-releases/resolve/master/channels/stable.json"
	got, err := Fetch(t.Context(), raw, 1<<20, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("%q", got)
	}
}

func TestDownloadRangeResume(t *testing.T) {
	payload := []byte("abcdefghijklmnop")
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/pkg.bin", func(w http.ResponseWriter, r *http.Request) {
		rng := r.Header.Get("Range")
		if rng == "" {
			http.Error(w, "range required", http.StatusBadRequest)
			return
		}
		if rng != "bytes=8-" {
			http.Error(w, rng, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Range", "bytes 8-15/16")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[8:])
	})
	SetHTTPClient(srv.Client())
	defer SetHTTPClient(nil)
	dir := t.TempDir()
	tmp := filepath.Join(dir, "pkg.bin.tmp")
	if err := os.WriteFile(tmp, payload[:8], 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := Download(t.Context(), Asset{
		URL: srv.URL + "/pkg.bin", Filename: "pkg.bin", SHA256: sha, Size: int64(len(payload)), Kind: "bin",
	}, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("%q", got)
	}
}
