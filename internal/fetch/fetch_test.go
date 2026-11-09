// fetch_test.go 锁住下载回退、哈希和大小校验，以及解压对 zip-slip、符号链接逃逸、bz2、裸二进制的处理。

package fetch

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// tarBz2Fixture 是 pkg/bin/tool（内容 "hello bz2\n"）的 tar.bz2，用 bsdtar -cjf 生成。标准库只能解 bzip2，所以放成定值。
const tarBz2Fixture = "QlpoOTFBWSZTWRayobIAAJtbkNKQQAD/hAREcu3eEAQAAIggAJINqiJ6epMQGDRBo9J6gikgJk9QAyGjQBY9bSUzQ/kusBCczw1g0iFqtN47NpIarXWV0GeeTFzfCtTArDxYQ5Y44VtjEoHCVDb03phEyL3oeVFRBk47jMznwTMPVC5CUiKs9/D9spEfu0k6iybLcmTOcy7qJfxdyRThQkBayobI"

// serve 起一个测试服务：/bad 回 500，/ok 回 body。
func serve(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			http.Error(w, "nope", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestDownloadFallsBack 锁住前一个地址失败时换下一个，返回实际地址和哈希，不留临时文件。
func TestDownloadFallsBack(t *testing.T) {
	body := []byte("agent binary")
	srv := serve(t, body)
	dest := filepath.Join(t.TempDir(), "sub", "agent.bin")
	var seen int64
	res, err := Download(context.Background(), Spec{
		URLs:     []string{srv.URL + "/bad", srv.URL + "/ok"},
		Dest:     dest,
		Progress: func(done, _ int64) { seen = done },
	})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if res.URL != srv.URL+"/ok" || res.SHA256 != hex.EncodeToString(sum[:]) || res.Size != int64(len(body)) || seen != int64(len(body)) {
		t.Fatalf("result %+v seen %d", res, seen)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, body) {
		t.Fatalf("content %q", got)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("temporary file left behind")
	}
}

// TestDownloadHashMismatch 锁住哈希对不上时不写出目标文件，错误是 ErrHashMismatch。
func TestDownloadHashMismatch(t *testing.T) {
	srv := serve(t, []byte("tampered"))
	dest := filepath.Join(t.TempDir(), "agent.bin")
	_, err := Download(context.Background(), Spec{URLs: []string{srv.URL + "/ok"}, Dest: dest, SHA256: hex.EncodeToString(make([]byte, 32))})
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("mismatched file was kept")
	}
}

// TestDownloadTooLarge 锁住超过 MaxBytes 时放弃，错误是 ErrTooLarge。
func TestDownloadTooLarge(t *testing.T) {
	srv := serve(t, bytes.Repeat([]byte("x"), 64))
	_, err := Download(context.Background(), Spec{URLs: []string{srv.URL + "/ok"}, Dest: filepath.Join(t.TempDir(), "big"), MaxBytes: 16})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

// writeZip 把 files 写成 zip。键是包内路径，值是内容；mode 给可执行文件。
func writeZip(t *testing.T, path string, files map[string]string, mode os.FileMode) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.SetMode(mode)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestExtractRejectsZipSlip 锁住带 .. 的条目整个包都不解，目标目录不出现。
func TestExtractRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.zip")
	writeZip(t, archive, map[string]string{"pkg/../../evil.txt": "x"}, 0o644)
	dest := filepath.Join(dir, "out")
	if err := Extract(archive, dest, KindZip, 0); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("destination created for a bad archive")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("zip-slip wrote outside")
	}
}

// TestExtractZipStripsAndKeepsExec 锁住 strip 去掉顶层目录，可执行位保留。
func TestExtractZipStripsAndKeepsExec(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "agent.zip")
	writeZip(t, archive, map[string]string{"agent-1.0/bin/run": "#!/bin/sh\n"}, 0o755)
	dest := filepath.Join(dir, "out")
	if err := Extract(archive, dest, "", 1); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "bin", "run"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("mode %v", info.Mode())
	}
}

// TestExtractRejectsTarSymlinkEscape 锁住指向目标目录外的符号链接让整个包失败。
func TestExtractRejectsTarSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "pkg/", Typeflag: tar.TypeDir, Mode: 0o755})
	_ = tw.WriteHeader(&tar.Header{Name: "pkg/link", Typeflag: tar.TypeSymlink, Linkname: "../../outside"})
	_ = tw.Close()
	_ = gz.Close()
	archive := filepath.Join(dir, "evil.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := Extract(archive, dest, "", 0); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("destination created for a bad archive")
	}
}

// TestExtractTarBz2 锁住 tar.bz2 能解开，strip 后路径正确。
func TestExtractTarBz2(t *testing.T) {
	dir := t.TempDir()
	raw, err := base64.StdEncoding.DecodeString(tarBz2Fixture)
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "fix.tar.bz2")
	if err := os.WriteFile(archive, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := Extract(archive, dest, "", 1); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "bin", "tool"))
	if err != nil || string(got) != "hello bz2\n" {
		t.Fatalf("got %q %v", got, err)
	}
}

// TestExtractRawBinary 锁住裸二进制按原文件名放进目标目录，旧目录被替换。
func TestExtractRawBinary(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "agent-x86_64.exe")
	if err := os.WriteFile(archive, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "stale"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Extract(archive, dest, KindOf(archive), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "agent-x86_64.exe")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "stale")); !os.IsNotExist(err) {
		t.Fatal("old install left behind")
	}
}

// TestMirrorURLs 锁住只有 GitHub 的 https 地址追加镜像。
func TestMirrorURLs(t *testing.T) {
	gh := "https://github.com/o/r/releases/download/v1/a.zip"
	if got := MirrorURLs(gh); len(got) != 2 || got[1] != "https://ghfast.top/"+gh {
		t.Fatalf("github %v", got)
	}
	obj := "https://objects.githubusercontent.com/x"
	if got := MirrorURLs(obj); len(got) != 2 {
		t.Fatalf("objects %v", got)
	}
	other := "https://registry.npmmirror.com/x.tgz"
	if got := MirrorURLs(other); len(got) != 1 || got[0] != other {
		t.Fatalf("other %v", got)
	}
}
