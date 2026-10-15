package pwsh

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveName(t *testing.T) {
	got, err := archiveName("windows", "amd64")
	if err != nil || got != "PowerShell-7.6.7-win-x64.zip" {
		t.Fatalf("amd64: %q %v", got, err)
	}
	got, err = archiveName("windows", "arm64")
	if err != nil || got != "PowerShell-7.6.7-win-arm64.zip" {
		t.Fatalf("arm64: %q %v", got, err)
	}
	if _, err := archiveName("linux", "amd64"); err == nil {
		t.Fatal("linux should be unsupported")
	}
	if _, err := archiveName("windows", "386"); err == nil {
		t.Fatal("386 should be unsupported")
	}
}

func TestCandidateURLs(t *testing.T) {
	t.Setenv(EnvURL, "")
	t.Setenv(EnvEndpoint, "")
	t.Setenv(EnvRepo, "")
	t.Setenv(EnvRevision, "")
	t.Setenv(EnvPrefix, "")
	m := New(t.TempDir())
	got := m.candidateURLs("PowerShell-7.6.7-win-x64.zip")
	want := []string{
		"https://www.modelscope.cn/datasets/NanoModel/ffmpeg/resolve/master/ps7/PowerShell-7.6.7-win-x64.zip?view=false",
		"https://www.modelscope.cn/api/v1/datasets/NanoModel/ffmpeg/repo?FilePath=ps7%2FPowerShell-7.6.7-win-x64.zip&Revision=master",
		"https://github.com/PowerShell/PowerShell/releases/download/v7.6.7/PowerShell-7.6.7-win-x64.zip",
		"https://ghfast.top/https://github.com/PowerShell/PowerShell/releases/download/v7.6.7/PowerShell-7.6.7-win-x64.zip",
	}
	if len(got) != len(want) {
		t.Fatalf("len %d %v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%d got %s want %s", i, got[i], want[i])
		}
	}
	arm := m.candidateURLs("PowerShell-7.6.7-win-arm64.zip")
	if arm[0] != "https://www.modelscope.cn/datasets/NanoModel/ffmpeg/resolve/master/ps7/PowerShell-7.6.7-win-arm64.zip?view=false" {
		t.Fatalf("arm resolve %s", arm[0])
	}
}

func TestRemotePath(t *testing.T) {
	t.Setenv(EnvPrefix, "")
	if got := remotePath("PowerShell-7.6.7-win-x64.zip"); got != "ps7/PowerShell-7.6.7-win-x64.zip" {
		t.Fatalf("default prefix %q", got)
	}
	t.Setenv(EnvPrefix, " ")
	if got := remotePath(`C:\tmp\PowerShell-7.6.7-win-x64.zip`); got != "ps7/PowerShell-7.6.7-win-x64.zip" {
		t.Fatalf("basename %q", got)
	}
}

func TestOfficialSHAPinned(t *testing.T) {
	x64, _ := archiveName("windows", "amd64")
	arm, _ := archiveName("windows", "arm64")
	if officialSHA256[x64] != "d2f952ebdbf24386cb8923553aa58e41a846db35a6155ea04c58142a4b16dbb7" {
		t.Fatalf("x64 sha %s", officialSHA256[x64])
	}
	if officialSHA256[arm] != "119bd9221254ef3637523cb1990797618f5fd68848b57de40f12b561d743149e" {
		t.Fatalf("arm sha %s", officialSHA256[arm])
	}
}

func TestExtractZipPreservesTreeAndRejectsSlip(t *testing.T) {
	raw := makeZip(t, map[string][]byte{
		"PowerShell/pwsh.exe": []byte("pwsh-bin"),
		"PowerShell/pwsh.dll": []byte("dll"),
		"../evil.exe":         []byte("bad"),
	})
	zipPath := filepath.Join(t.TempDir(), "in.zip")
	if err := os.WriteFile(zipPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	bin, err := extractZip(zipPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(bin) != "pwsh.exe" {
		t.Fatalf("bin %s", bin)
	}
	if _, err := os.Stat(filepath.Join(dest, "PowerShell", "pwsh.dll")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "evil.exe")); err == nil {
		t.Fatal("zip-slip")
	}
}

func TestLookOrderEnvThenManaged(t *testing.T) {
	home := t.TempDir()
	m := New(filepath.Join(home, "runtime"))
	if m.Look() != "" {
		t.Fatal("empty")
	}
	managed := m.installDir()
	if err := os.MkdirAll(managed, 0o755); err != nil {
		t.Fatal(err)
	}
	managedBin := filepath.Join(managed, "pwsh.exe")
	if err := os.WriteFile(managedBin, []byte("m"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := m.Look(); got != managedBin {
		t.Fatalf("managed %s", got)
	}
	envDir := t.TempDir()
	envBin := filepath.Join(envDir, "pwsh.exe")
	if err := os.WriteFile(envBin, []byte("e"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvBin, envBin)
	if got := m.Look(); got != envBin {
		t.Fatalf("env %s", got)
	}
}

func TestInstallBundleFromHTTP(t *testing.T) {
	raw := makeZip(t, map[string][]byte{"pwsh.exe": []byte("pwsh-bin")})
	sum := sha256.Sum256(raw)
	hexSum := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(raw)
	}))
	t.Cleanup(srv.Close)

	archive, _ := archiveName("windows", "amd64")
	prev := officialSHA256[archive]
	officialSHA256[archive] = hexSum
	t.Cleanup(func() { officialSHA256[archive] = prev })

	home := t.TempDir()
	m := New(filepath.Join(home, "runtime"))
	m.GOOS, m.GOARCH = "windows", "amd64"
	m.HTTP = srv.Client()
	m.Verify = func(bin string) error {
		b, err := os.ReadFile(bin)
		if err != nil {
			return err
		}
		if string(b) != "pwsh-bin" {
			return os.ErrInvalid
		}
		return nil
	}
	t.Setenv(EnvURL, srv.URL+"/"+archive)
	t.Setenv(EnvEndpoint, srv.URL)

	if err := m.installBundle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p := m.Look(); p == "" || !strings.HasSuffix(p, "pwsh.exe") {
		t.Fatalf("look %q", p)
	}
	snap := m.Snapshot()
	if snap["source"] != "managed" {
		t.Fatalf("source %+v", snap)
	}
}

func makeZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
