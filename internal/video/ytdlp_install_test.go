// ytdlp_install_test.go 用本地服务器代替发布地址，检查 yt-dlp 下载是否落盘。

package video

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestInstallYTDLPStoresTheRelease 确认 downloadYTDLP 把响应字节写入目标文件，并记到 Engine.YTDLP。
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
