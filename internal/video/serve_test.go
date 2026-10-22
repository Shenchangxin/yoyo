package video

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type dirCAS struct{ dir string }

func (d *dirCAS) PutRaw(b []byte) (string, error) {
	h := NewID()
	if err := os.WriteFile(filepath.Join(d.dir, h), b, 0o600); err != nil {
		return "", err
	}
	return h, nil
}

func (d *dirCAS) GetRaw(hash string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.dir, hash))
}

func (d *dirCAS) Path(hash string) string {
	return filepath.Join(d.dir, hash)
}

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
