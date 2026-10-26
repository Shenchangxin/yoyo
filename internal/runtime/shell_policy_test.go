// 拒绝名单不是操作系统沙箱。本文件不调用路径 resolve，也没有 Windows Job Object。相对路径放行。

package runtime

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestBindShellStatementSepUsesPowerShellFile 确认未加引号的分号在 Windows 上走 PowerShell -File，不用 cmd /C。
func TestBindShellStatementSepUsesPowerShellFile(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("cmd vs PowerShell routing is Windows-only")
	}
	command := `node --version; node -p 1`
	cmd, cleanup, err := bindShell(context.Background(), command, SplitShellArgv(command), false, t.TempDir())
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.ToLower(strings.Join(cmd.Args, " "))
	if !strings.Contains(joined, "-file") {
		t.Fatalf("want PowerShell -File, got %v", cmd.Args)
	}
	if strings.Contains(joined, "cmd") && strings.Contains(joined, "/c") {
		t.Fatalf("must not use cmd /C: %v", cmd.Args)
	}
}

// TestShellHasStatementSep 确认未加引号的分号和 && 算语句分隔，引号内的不算。
func TestShellHasStatementSep(t *testing.T) {
	if !shellHasStatementSep(`node --version; node -p 1`) {
		t.Fatal("trailing/command sep")
	}
	if !shellHasStatementSep(`node --version;`) {
		t.Fatal("trailing semicolon")
	}
	if shellHasStatementSep(`findstr /C:"a;b" file.txt`) {
		t.Fatal("quoted semicolon is not a statement sep")
	}
	if shellHasStatementSep(`echo hello`) {
		t.Fatal("no sep")
	}
}

// TestShellNeedsWrapper 确认 cd、set、echo、dir 要包装，git status 和 go test 不要。
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

// TestStripUnixOutputPipes 确认 head 和 tail 被去掉；剩余仍像 posix 的管道不剥。
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

// TestLooksPosixUnix 确认 find -maxdepth、ls、cat、pwd 为真，dir、go build、git、find /c /v 和 echo 为假。
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

// TestShellDeniedDoesNotMatchExportFormat 确认 echo export format is json 不因 format 子串被拒，format c: 被拒。
func TestShellDeniedDoesNotMatchExportFormat(t *testing.T) {
	if err := ShellDenied(`echo export format is json`, t.TempDir(), nil); err != nil {
		t.Fatalf("false destructive: %v", err)
	}
	if err := ShellDenied(`format c:`, t.TempDir(), nil); err == nil {
		t.Fatal("format c: must deny")
	}
}

// TestShellDeniedAllowsPythonHTTPServerWithLoopbackFetch 确认 python -m http.server 与回环 Invoke-WebRequest 放行。
func TestShellDeniedAllowsPythonHTTPServerWithLoopbackFetch(t *testing.T) {
	ws := t.TempDir()
	cmds := []string{
		`python -m http.server 8777 --directory build/app`,
		`powershell -NoProfile -Command "Start-Process -WindowStyle Hidden -FilePath python -ArgumentList '-m','http.server','8777','--directory','build/app'; Start-Sleep -Seconds 2; (Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8777/index.html).StatusCode"`,
	}
	for _, cmd := range cmds {
		if err := ShellDenied(cmd, ws, nil); err != nil {
			t.Fatalf("python -m http.server + loopback fetch must pass: %s: %v", cmd, err)
		}
	}
}

// TestShellDeniedAllowsLoopbackWithLocalFilename 确认带本地文件名的回环命令放行。
func TestShellDeniedAllowsLoopbackWithLocalFilename(t *testing.T) {
	ws := t.TempDir()
	cmds := []string{
		`Invoke-WebRequest http://127.0.0.1:8765/pelican-bicycle.html`,
		`curl -s http://127.0.0.1:8765/pelican-bicycle.html`,
		`powershell -NoProfile -Command "Invoke-WebRequest http://127.0.0.1:8765/index.html"`,
	}
	for _, cmd := range cmds {
		if err := ShellDenied(cmd, ws, nil); err != nil {
			t.Fatalf("loopback+filename must pass: %s: %v", cmd, err)
		}
	}
}

// TestLooksLikePowerShell 确认 Get-Content 与 ForEach 像 PowerShell，ls $(pwd) 不像。
func TestLooksLikePowerShell(t *testing.T) {
	if !looksLikePowerShell(`$c=Get-Content pelican-bicycle.html; $c.Count`) {
		t.Fatal("assignment")
	}
	if !looksLikePowerShell(`Get-Process | ForEach-Object { $_.Id }`) {
		t.Fatal("foreach")
	}
	if looksLikePowerShell(`ls $(pwd)`) {
		t.Fatal("posix subshell is not powershell")
	}
	got := powershellCommandArg(`powershell -NoProfile -Command Start-Process x`)
	if got != "Start-Process x" {
		t.Fatalf("unwrap %q", got)
	}
}

// TestShellDeniedAllowsLoopbackCurl 确认回环 curl 放行，https://example.com 与无主机的 curl 拒绝。
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

// TestShellDeniedMsysWorkspaceRoot 确认 Windows 上工作区内的 MSYS 路径 cd 放行。
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

// TestShellDeniedIgnoresWindowsSwitches 确认 taskkill //F 与 tasklist //FI 不因看起来像 UNC 被拒。
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

// TestShellDeniedAllowsHostInterpreter 确认工作区外的 python.exe 与 /usr/bin/python3 作为宿主解释器放行。
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
