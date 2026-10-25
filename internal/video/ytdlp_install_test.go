package video

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestInstallYTDLPStoresTheRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("yt-dlp-bytes"))
	}))
	defer srv.Close()
	prev := ytdlpAssetURL
	ytdlpAssetURL = func() string { return srv.URL }
	t.Cleanup(func() { ytdlpAssetURL = prev })
	e := testEngine(t)
	e.YTDLP = ""
	path, err := e.downloadYTDLP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "yt-dlp-bytes" || e.YTDLP != path {
		t.Fatalf("%v %s %s", err, path, e.YTDLP)
	}
}
