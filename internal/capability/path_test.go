// 锁住这些不变量：/c/Users/... 译成 C:\Users\...，/cygdrive/c/Users/x 译成 C:\Users\x。
// /dev/null、/tmp、//F 不是盘符路径。//F、//IM、//FI、/C、/B、/IM 是开关；/c/Users/scx、/etc/passwd、Windows 绝对路径、无斜杠文件名、/console/ 不是。
// Windows 上，工作区内文件的 Git Bash 路径仍算在工作区内。
// ForbiddenDiagPath 拒绝 home 下的 logs、journal、observe、pages、learn、profiles 以及 vault.json，放行 workspace 里的文件；home 为空时不拒绝。

package capability

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMSYSToWindowsPath 确认盘符翻译成功，并确认 /dev/null、/tmp、//F 翻译失败。
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

// TestLooksLikeWindowsSwitch 确认样例里的开关为真，带多段路径的字符串为假。
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

// TestWithinWorkspaceGitBashPath 只在 Windows 上跑，确认工作区内文件的 /盘符 路径仍被 WithinWorkspace 接受。
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

// TestForbiddenDiagPath 确认诊断目录和 vault.json 被拒绝，工作区文件与空 home 不被拒绝。
func TestForbiddenDiagPath(t *testing.T) {
	home := t.TempDir()
	for _, p := range []string{
		filepath.Join(home, "logs"),
		filepath.Join(home, "logs", "yoyo.log"),
		filepath.Join(home, "journal", "chain.jsonl"),
		filepath.Join(home, "observe", "spans.jsonl"),
		filepath.Join(home, "pages", "home", "note.md"),
		filepath.Join(home, "learn", "learn.sqlite"),
		filepath.Join(home, "profiles", "assistant.json"),
		filepath.Join(home, "vault.json"),
	} {
		if !ForbiddenDiagPath(home, p) {
			t.Fatalf("want deny %s", p)
		}
	}
	if ForbiddenDiagPath(home, filepath.Join(home, "workspace", "readme.md")) {
		t.Fatal("workspace file must be allowed")
	}
	if ForbiddenDiagPath("", filepath.Join(home, "logs")) {
		t.Fatal("empty home must not deny")
	}
}
