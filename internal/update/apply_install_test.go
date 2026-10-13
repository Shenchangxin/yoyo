package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNSISCmdLineQuotesInstallerAndLeavesDestUnquoted(t *testing.T) {
	got := nsisCmdLine(`C:\Users\Jane Doe\updates\Yoyo-1.2.3-windows-x64-setup.exe`, `C:\Program Files\Yoyo\Yoyo`)
	if got != `"C:\Users\Jane Doe\updates\Yoyo-1.2.3-windows-x64-setup.exe" /S /D=C:\Program Files\Yoyo\Yoyo` {
		t.Fatalf("%s", got)
	}
}

func TestNSISCmdLineOmitsDest(t *testing.T) {
	got := nsisCmdLine(`C:\setup.exe`, "")
	if got != `"C:\setup.exe" /S` {
		t.Fatalf("%s", got)
	}
}

func TestNSISApplySpecRoundTrip(t *testing.T) {
	want := nsisApplySpec{Installer: `C:\a b\setup.exe`, DestDir: `C:\Program Files\Yoyo`, WaitPID: 42}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got nsisApplySpec
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%+v", got)
	}
}

func TestSkipNSISDestDirDevLayouts(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !skipNSISDestDir(bin) {
		t.Fatal("wails bin next to go.mod must skip /D=")
	}
	goBuild := filepath.Join(root, "go-build123", "b001", "exe")
	if !skipNSISDestDir(goBuild) {
		t.Fatal("go run temp must skip /D=")
	}
	installed := filepath.Join(root, "Yoyo")
	if skipNSISDestDir(installed) {
		t.Fatal("real install dir must keep /D=")
	}
	exe := filepath.Join(bin, "Yoyo.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := nsisDestDir(exe); got != "" {
		t.Fatalf("dev bin dest %q", got)
	}
}

func TestNSISDestDirTrimsSlash(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Yoyo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "Yoyo.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := nsisDestDir(exe)
	if strings.HasSuffix(got, `\`) || strings.HasSuffix(got, "/") {
		t.Fatalf("%q", got)
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	want = strings.TrimRight(want, `/\`)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestHandleNSISHelperIgnoresNormalArgs(t *testing.T) {
	if HandleNSISHelper() {
		t.Fatal("normal process must not enter helper")
	}
}
