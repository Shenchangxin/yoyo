// 本文件锁住 atlas 页名修正、立绘只下载一次，以及 /idol-stand/ 之外的请求交给下一个处理器。
// 由 go test 调用。测试里的假 CDN 字节是数据。

package idol

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPointAtlasAtDataPNG 锁住下一行是 size: 的 date.png 会改成 data.png，其他页名不动。
func TestPointAtlasAtDataPNG(t *testing.T) {
	in := "date.png\nsize: 1024,1024\narm\n  rotate: true\n"
	got := pointAtlasAtDataPNG(in)
	if strings.Contains(got, "date.png") || !strings.Contains(got, "data.png\nsize:") {
		t.Fatalf("%q", got)
	}
	kept := "other.png\nsize: 4,4\n"
	if pointAtlasAtDataPNG(kept) != kept {
		t.Fatal("renamed a page that is not the known slip")
	}
}

// TestStandCacheFetchesOnce 锁住同一份 atlas 第二次从磁盘读，并把 date.png 改成 data.png。逃出服装 id 的路径被拒绝。
func TestStandCacheFetchesOnce(t *testing.T) {
	var hits int
	png := []byte("\x89PNG\r\n\x1a\nrest")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/stand/1030010010/data.json":
			_, _ = w.Write([]byte(`{"skeleton":{"spine":"3.6.53"}}`))
		case "/stand/1030010010/data.atlas":
			_, _ = w.Write([]byte("date.png\nsize: 8,8\n"))
		case "/stand/1030010010/data.png":
			_, _ = w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	cache := &StandCache{Dir: dir, Base: srv.URL + "/", Client: srv.Client()}
	body, ctype, err := cache.Open("stand/1030010010", "data.atlas")
	if err != nil {
		t.Fatal(err)
	}
	if ctype != "text/plain; charset=utf-8" || strings.Contains(string(body), "date.png") {
		t.Fatalf("%s %s", ctype, body)
	}
	if _, _, err := cache.Open("stand/1030010010", "data.atlas"); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "stand", "1030010010", "data.atlas"))
	if err != nil || strings.Contains(string(saved), "date.png") {
		t.Fatalf("saved %v %s", err, saved)
	}
	if _, _, err := cache.Open("stand/../secret", "data.json"); err == nil {
		t.Fatal("accepted a path outside the outfit id")
	}
}

// TestStandMiddleware 锁住 /idol-stand/stand/9/data.json 返回缓存里的 JSON，其他路径交给 next。
func TestStandMiddleware(t *testing.T) {
	dir := t.TempDir()
	outfit := filepath.Join(dir, "stand", "9")
	if err := os.MkdirAll(outfit, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outfit, "data.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	h := StandMiddleware(dir, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "next", http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/idol-stand/stand/9/data.json", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/other", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("passed %d", rec.Code)
	}
}
