// launch_test.go 锁住启动方式的拼法：经 cmd.exe 启动时的原始命令行引号、追加参数不破坏引号，以及 Windows 参数转义。

package acp_test

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// TestCmdLineViaCmd 锁住 cmd /d /s /c 后面整体加一对外层引号，里面每段按 Windows 规则转义。
func TestCmdLineViaCmd(t *testing.T) {
	got := acp.CmdLineViaCmd(`C:\Windows\system32\cmd.exe`, `C:\Program Files\cursor\cursor-agent.cmd`, []string{"acp", "--model", "a b"})
	want := `C:\Windows\system32\cmd.exe /d /s /c ""C:\Program Files\cursor\cursor-agent.cmd" acp --model "a b""`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

// TestLaunchWithArgsKeepsViaCmd 锁住追加参数后再算命令行，参数仍在外层引号里面。
func TestLaunchWithArgsKeepsViaCmd(t *testing.T) {
	l := acp.Launch{Command: `C:\x\agent.cmd`, Args: []string{"acp"}, ViaCmd: true, Env: []string{`ComSpec=C:\cmd.exe`}}
	opts := l.WithArgs("login").Options()
	if opts.Command != `C:\cmd.exe` || opts.CmdLine != `C:\cmd.exe /d /s /c "C:\x\agent.cmd acp login"` {
		t.Fatalf("options %+v", opts)
	}
	if len(l.Args) != 1 {
		t.Fatal("WithArgs changed the original")
	}
	plain := acp.Launch{Command: "/usr/bin/agent", Args: []string{"acp"}}.Options()
	if plain.Command != "/usr/bin/agent" || plain.CmdLine != "" || len(plain.Args) != 1 {
		t.Fatalf("plain options %+v", plain)
	}
}

// TestQuoteWindowsArg 锁住空串、空格、引号和结尾反斜杠的转义。
func TestQuoteWindowsArg(t *testing.T) {
	cases := map[string]string{
		"":           `""`,
		"plain":      "plain",
		"a b":        `"a b"`,
		`say "hi"`:   `"say \"hi\""`,
		`C:\dir\`:    `C:\dir\`,
		`C:\my dir\`: `"C:\my dir\\"`,
		`a\"b`:       `"a\\\"b"`,
	}
	for in, want := range cases {
		if got := acp.QuoteWindowsArg(in); got != want {
			t.Errorf("%q => %s, want %s", in, got, want)
		}
	}
}
