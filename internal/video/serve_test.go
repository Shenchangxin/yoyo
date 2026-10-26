// serve_test.go 检查媒体响应的类型，以及中间件是否放过应用自己的路径。

package video

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// dirCAS 把字节写进一个目录，用随机标识当文件名。
type dirCAS struct{ dir string }

// PutRaw 用 NewID 当文件名，把字节写成 0600 的文件。
func (d *dirCAS) PutRaw(b []byte) (string, error) {
	h := NewID()
	if err := os.WriteFile(filepath.Join(d.dir, h), b, 0o600); err != nil {
		return "", err
	}
	return h, nil
}

// GetRaw 按标识读回该目录里的文件。
func (d *dirCAS) GetRaw(hash string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.dir, hash))
}

// Path 返回标识在该目录下的路径。
func (d *dirCAS) Path(hash string) string {
	return filepath.Join(d.dir, hash)
}

// TestMediaHandlerServesMP4NotOctetStream 确认带 ftyp 的字节以 video/mp4 返回，状态为 200。
func TestMediaHandlerServesMP4NotOctetStream(t *testing.T) {
	e := testEngine(t)
	e.CAS = &dirCAS{dir: t.TempDir()}
	raw := make([]byte, 32)
	copy(raw[4:8], []byte("ftyp"))
	hash, err := e.PutBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	e.mediaHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/"+hash, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("content-type %q", ct)
	}
}

// TestMediaAssetMiddlewareLeavesTheAppAlone 确认请求根路径时仍交给下游，并得到 204。
func TestMediaAssetMiddlewareLeavesTheAppAlone(t *testing.T) {
	e := testEngine(t)
	hit := false
	h := MediaAssetMiddleware(e, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !hit || rec.Code != http.StatusNoContent {
		t.Fatalf("hit %v status %d", hit, rec.Code)
	}
}

// TestMediaTypeSniffsContainers 确认 ftyp 是 video/mp4，EBML 头是 video/webm，空字节返回空串。
func TestMediaTypeSniffsContainers(t *testing.T) {
	mp4 := make([]byte, 16)
	copy(mp4[4:8], []byte("ftyp"))
	if mediaType(mp4) != "video/mp4" {
		t.Fatal(mediaType(mp4))
	}
	webm := []byte{0x1a, 0x45, 0xdf, 0xa3, 0, 0, 0, 0}
	if mediaType(webm) != "video/webm" {
		t.Fatal(mediaType(webm))
	}
	if mediaType(nil) != "" {
		t.Fatal("empty")
	}
}
