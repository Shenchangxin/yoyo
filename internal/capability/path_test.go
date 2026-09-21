package capability

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMSYSToWindowsPath(t *testing.T) {
	got, ok := MSYSToWindowsPath(`/c/Users/scx/Desktop/codex/api/api.go`)
	if !ok || got != `C:\Users\scx\Desktop\codex\api\api.go` {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	got, ok = MSYSToWindowsPath(`/cygdrive/c/Users/x`)
	if !ok || got != `C:\Users\x` {
		t.Fatalf("cygdrive %q ok=%v", got, ok)
	}
	if _, ok = MSYSToWindowsPath(`/dev/null`); ok {
		t.Fatal("/dev/null must not look like a drive")
	}
	if _, ok = MSYSToWindowsPath(`/tmp/t.txt`); ok {
		t.Fatal("/tmp must not look like a drive")
	}
	if _, ok = MSYSToWindowsPath(`//F`); ok {
		t.Fatal("//F is a switch, not a drive path")
	}
}

func TestLooksLikeWindowsSwitch(t *testing.T) {
	for _, s := range []string{`//F`, `//IM`, `//FI`, `/C`, `/B`, `/IM`} {
		if !LooksLikeWindowsSwitch(s) {
			t.Fatalf("want switch %q", s)
		}
	}
	for _, s := range []string{`/c/Users/scx`, `/etc/passwd`, `C:\Windows\System32`, `permserver.exe`, `/console/`} {
		if LooksLikeWindowsSwitch(s) {
			t.Fatalf("not a switch: %q", s)
		}
	}
}

func TestWithinWorkspaceGitBashPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}
	root := t.TempDir()
	inner := filepath.Join(root, "a.txt")
	if err := os.WriteFile(inner, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.ToSlash(inner)
	msys := "/" + strings.ToLower(p[:1]) + p[2:]
	if !WithinWorkspace(root, msys) {
		t.Fatalf("msys %s not in %s", msys, root)
	}
}
