package runtime

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellNeedsWrapper(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{`cd webui && npm install`, true},
		{`cd src`, true},
		{`set FOO=1`, true},
		{`echo hello`, true},
		{`dir`, true},
		{`git status`, false},
		{`go test ./...`, false},
	}
	for _, tc := range cases {
		argv := SplitShellArgv(tc.cmd)
		got := shellNeedsWrapper(tc.cmd, argv)
		if got != tc.want {
			t.Fatalf("%q wrapper=%v want %v argv=%v", tc.cmd, got, tc.want, argv)
		}
	}
}

func TestStripUnixOutputPipes(t *testing.T) {
	got, ok := stripUnixOutputPipes(`cd apps/api && go vet ./... 2>&1 | head -40`)
	if !ok || got != `cd apps/api && go vet ./...` {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	got, ok = stripUnixOutputPipes(`cd apps\api && go test ./... | tail -20`)
	if !ok || !strings.Contains(got, `go test ./...`) || strings.Contains(got, "tail") {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if _, ok := stripUnixOutputPipes(`ls -la | head -20`); ok {
		t.Fatal("ls | head must stay posix — stripping would still be unix")
	}
	if _, ok := stripUnixOutputPipes(`go test ./...`); ok {
		t.Fatal("no pipe")
	}
}

func TestLooksPosixUnix(t *testing.T) {
	posix := []string{
		`find . -maxdepth 2 -type d -not -path '*/node_modules/*' 2>/dev/null | head -60; echo "---"; ls -la`,
		`ls -la`,
		`cat README.md`,
		`pwd`,
	}
	for _, cmd := range posix {
		if !looksPosixUnix(cmd) {
			t.Fatalf("want posix: %s", cmd)
		}
	}
	cmdish := []string{
		`dir /s /b api cmd engine`,
		`go build ./...`,
		`git status`,
		`cd webui && dir /b`,
		`find /c /v "" web\static\app.js`,
		`echo hello`,
	}
	for _, cmd := range cmdish {
		if looksPosixUnix(cmd) {
			t.Fatalf("want cmd: %s", cmd)
		}
	}
}

func TestShellDeniedDoesNotMatchExportFormat(t *testing.T) {
	if err := ShellDenied(`echo export format is json`, t.TempDir(), nil); err != nil {
		t.Fatalf("false destructive: %v", err)
	}
	if err := ShellDenied(`format c:`, t.TempDir(), nil); err == nil {
		t.Fatal("format c: must deny")
	}
}

func TestShellDeniedAllowsLoopbackCurl(t *testing.T) {
	ws := t.TempDir()
	allow := []string{
		`curl -s localhost:18099/`,
		`curl -s -D - -o /dev/null localhost:18099/console/`,
		`curl http://127.0.0.1:8080/console/`,
		`curl -s http://[::1]:8080/`,
	}
	for _, cmd := range allow {
		if err := ShellDenied(cmd, ws, nil); err != nil {
			t.Fatalf("loopback must pass: %s: %v", cmd, err)
		}
	}
	if err := ShellDenied(`curl -s https://example.com/`, ws, nil); err == nil {
		t.Fatal("remote curl must deny")
	}
	if err := ShellDenied(`curl`, ws, nil); err == nil {
		t.Fatal("curl with no host must deny")
	}
}

func TestShellDeniedMsysWorkspaceRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}
	ws := t.TempDir()
	p := filepath.ToSlash(ws)
	msys := "/" + strings.ToLower(p[:1]) + p[2:]
	if err := ShellDenied("cd "+msys+" && pwd", ws, nil); err != nil {
		t.Fatal(err)
	}
}

func TestShellDeniedIgnoresWindowsSwitches(t *testing.T) {
	ws := t.TempDir()
	cmds := []string{
		`taskkill //F //IM permserver.exe`,
		`tasklist //FI "IMAGENAME eq permserver.exe"`,
	}
	for _, cmd := range cmds {
		if err := ShellDenied(cmd, ws, nil); err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
	}
}

func TestShellDeniedAllowsHostInterpreter(t *testing.T) {
	ws := t.TempDir()
	cmds := []string{
		`E:\python-sdk\python.exe scripts\fetch_arxiv.py`,
		`/usr/bin/python3 scripts/fetch_arxiv.py`,
	}
	for _, cmd := range cmds {
		if err := ShellDenied(cmd, ws, nil); err != nil {
			t.Fatalf("interpreter must pass: %s: %v", cmd, err)
		}
	}
}
