package video

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFFmpegArchiveName(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"windows", "amd64", "ffmpeg-windows-x86_64.zip"},
		{"windows", "arm64", "ffmpeg-windows-aarch64.zip"},
		{"linux", "amd64", "ffmpeg-linux-x86_64.zip"},
		{"linux", "arm64", "ffmpeg-linux-aarch64.zip"},
		{"darwin", "amd64", "ffmpeg-macos-x86_64.zip"},
		{"darwin", "arm64", "ffmpeg-macos-aarch64.zip"},
	}
	for _, c := range cases {
		got, err := ffmpegArchiveName(c.goos, c.goarch)
		if err != nil || got != c.want {
			t.Fatalf("%s/%s: got %q %v want %q", c.goos, c.goarch, got, err, c.want)
		}
	}
	if _, err := ffmpegArchiveName("plan9", "amd64"); err == nil {
		t.Fatal("expected unsupported os")
	}
	if _, err := ffmpegArchiveName("linux", "386"); err == nil {
		t.Fatal("expected unsupported arch")
	}
}

func TestPickFFmpegArchive(t *testing.T) {
	files := []string{
		"README.md",
		"ffmpeg-linux-aarch64.zip",
		"ffmpeg-linux-x86_64.zip",
		"ffmpeg-macos-aarch64.zip",
		"ffmpeg-macos-x86_64.zip",
		"ffmpeg-windows-aarch64.zip",
		"ffmpeg-windows-x86_64.zip",
	}
	got, err := pickFFmpegArchive(files, "windows", "amd64")
	if err != nil || got != "ffmpeg-windows-x86_64.zip" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = pickFFmpegArchive(files, "darwin", "arm64")
	if err != nil || got != "ffmpeg-macos-aarch64.zip" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = pickFFmpegArchive(nil, "linux", "amd64")
	if err != nil || got != "ffmpeg-linux-x86_64.zip" {
		t.Fatalf("fallback %q %v", got, err)
	}
}

func TestExtractFFmpegZipNestedAndDLL(t *testing.T) {
	raw := makeFFmpegZip(t, map[string][]byte{
		"ffmpeg-7.1/bin/ffmpeg.exe":  []byte("ffmpeg-bin"),
		"ffmpeg-7.1/bin/ffprobe.exe": []byte("ffprobe-bin"),
		"ffmpeg-7.1/bin/avcodec.dll": []byte("dll"),
		"ffmpeg-7.1/README.txt":      []byte("nope"),
		"../evil.exe":                []byte("bad"),
	})
	zipPath := filepath.Join(t.TempDir(), "in.zip")
	if err := os.WriteFile(zipPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	ff, probe, err := extractFFmpegZip(zipPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ff) != "ffmpeg.exe" {
		t.Fatalf("ffmpeg %s", ff)
	}
	if filepath.Base(probe) != "ffprobe.exe" {
		t.Fatalf("ffprobe %s", probe)
	}
	if _, err := os.Stat(filepath.Join(dest, "avcodec.dll")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "README.txt")); err == nil {
		t.Fatal("should skip readme")
	}
	if _, err := os.Stat(filepath.Join(dest, "evil.exe")); err == nil {
		t.Fatal("zip-slip entry should be skipped")
	}
	if _, _, err := extractFFmpegZip(zipPath, dest); err != nil {
		t.Fatalf("re-extract: %v", err)
	}
}

func TestExtractFFmpegZipBackslashNames(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name string, body []byte) {
		t.Helper()
		fh := &zip.FileHeader{Name: name, Method: zip.Store}
		fw, err := w.CreateHeader(fh)
		if err != nil {
			t.Skipf("zip writer rejected %q: %v", name, err)
		}
		if _, err := fw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	add(`ffmpeg-8.1.2-essentials_build\bin\ffmpeg.exe`, []byte("ffmpeg-bin"))
	add(`ffmpeg-8.1.2-essentials_build\bin\ffprobe.exe`, []byte("ffprobe-bin"))
	add(`ffmpeg-8.1.2-essentials_build\bin\avcodec.dll`, []byte("dll"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "in.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	ff, probe, err := extractFFmpegZip(zipPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ff) != "ffmpeg.exe" {
		t.Fatalf("ffmpeg %s", ff)
	}
	if filepath.Base(probe) != "ffprobe.exe" {
		t.Fatalf("ffprobe %s", probe)
	}
}

func TestFFmpegInstallSnapshotClearsStaleError(t *testing.T) {
	e := testEngine(t)
	e.FFMPEG = filepath.Join(e.Dir, "bin", "ffmpeg.exe")
	e.ffPhase = "error"
	e.ffErr = "ffmpeg: failed to extract ffmpeg"
	snap := e.FFmpegInstallSnapshot()
	if snap["phase"] != "ready" || snap["error"] != "" {
		t.Fatalf("%+v", snap)
	}
}

func TestParseFFmpegRepoTree(t *testing.T) {
	raw := []byte(`{"Code":200,"Data":{"Files":[
		{"Name":"ffmpeg-windows-x86_64.zip","Path":"ffmpeg-windows-x86_64.zip","Type":"blob","Size":123,"Sha256":"abc","IsLFS":true},
		{"Name":"docs","Path":"docs","Type":"tree"}
	]}}`)
	files := parseFFmpegRepoTree(raw)
	if len(files) != 1 || files[0].Path != "ffmpeg-windows-x86_64.zip" || files[0].Sha256 != "abc" {
		t.Fatalf("%+v", files)
	}
}

func TestInstallFFmpegBundleFromModelScopeShape(t *testing.T) {
	payload := makeFFmpegZip(t, map[string][]byte{
		"bin/ffmpeg":  []byte("ffmpeg-bin"),
		"bin/ffprobe": []byte("ffprobe-bin"),
	})
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/repo/tree"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"Code":200,"Data":{"Files":[{"Name":"ffmpeg-linux-x86_64.zip","Path":"ffmpeg-linux-x86_64.zip","Type":"blob","Size":%d,"Sha256":"%s"}]}}`, len(payload), sha)
		case strings.HasSuffix(r.URL.Path, "/ffmpeg-linux-x86_64.zip"):
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	prevEndpoint, prevVerify := ffmpegEndpoint, ffmpegVerify
	ffmpegEndpoint = srv.URL
	ffmpegVerify = func(string) error { return nil }
	t.Cleanup(func() {
		ffmpegEndpoint = prevEndpoint
		ffmpegVerify = prevVerify
	})

	e := testEngine(t)
	e.ffmpegHTTP = srv.Client()
	e.ffmpegOS = "linux"
	e.ffmpegArch = "amd64"
	if err := e.installFFmpegBundle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.Dir, "bin", "ffmpeg")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.Dir, "bin", "ffprobe")); err != nil {
		t.Fatal(err)
	}
	snap := e.FFmpegInstallSnapshot()
	if snap["archive"] != "ffmpeg-linux-x86_64.zip" {
		t.Fatalf("archive %+v", snap["archive"])
	}
}

func makeFFmpegZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		fh, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fh.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
