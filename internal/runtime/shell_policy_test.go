package runtime

import "testing"

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
