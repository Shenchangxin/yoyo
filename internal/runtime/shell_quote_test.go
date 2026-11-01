// 含反斜杠或空格的 Windows 路径加引号。看起来像 posix 的命令保留引号。/dev/null 使内联 node 走 Git Bash。

package runtime

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestInlineNodeScriptRunsThroughGitBash 确认 Windows 上有 Git Bash 和 node 时，node -e 经 shell 跑出 bash-ok，/dev/null 使命令走 bash。
func TestInlineNodeScriptRunsThroughGitBash(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git Bash path quoting is a Windows problem")
	}
	if windowsPosixShell() == "" {
		t.Skip("no Git Bash on this machine")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not on PATH")
	}
	tools := &WorkspaceTools{Workspace: t.TempDir()}
	res := tools.Call("shell", `{"command":"node -e \"console.log('bash-ok')\" 2>/dev/null"}`)
	if res.Err != nil {
		t.Fatalf("shell: %v", res.Err)
	}
	if !strings.Contains(res.Content, "bash-ok") {
		t.Fatalf("inline script did not run under bash:\n%s", res.Content)
	}
}

// TestQuoteShellPathKeepsWindowsPathsIntact 确认含反斜杠或空格的路径加引号，/tmp 和空串不加。
func TestQuoteShellPathKeepsWindowsPathsIntact(t *testing.T) {
	cases := map[string]string{
		`C:\Users\scx\.yoyo\tmp\yoyo-inline-1.js`: `"C:\Users\scx\.yoyo\tmp\yoyo-inline-1.js"`,
		`C:\Users\scx\My Docs\x.js`:               `"C:\Users\scx\My Docs\x.js"`,
		`/tmp/yoyo-inline-1.js`:                   `/tmp/yoyo-inline-1.js`,
		``:                                        ``,
	}
	for in, want := range cases {
		if got := quoteShellPath(in); got != want {
			t.Fatalf("quoteShellPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNormalizeWindowsQuotesKeepsQuotesForBash 确认看起来像 posix 的命令保留引号。
func TestNormalizeWindowsQuotesKeepsQuotesForBash(t *testing.T) {
	bash := `node "C:\Users\scx\qa\run.js" > /dev/null`
	if got := normalizeWindowsQuotes(bash); got != bash {
		t.Fatalf("bash command lost its quotes:\n got %s\nwant %s", got, bash)
	}
}

// TestNormalizeWindowsQuotesRelaxesForCmd 确认 cmd 命令去掉无空格引号，相对路径的引号也去掉。
func TestNormalizeWindowsQuotesRelaxesForCmd(t *testing.T) {
	in := `node "C:\Users\scx\qa\run.js"`
	want := `node C:\Users\scx\qa\run.js`
	if got := normalizeWindowsQuotes(in); got != want {
		t.Fatalf("cmd command: got %s want %s", got, want)
	}
	// 带引号的相对路径同样去掉引号。
	rel := `node "rel\file.mjs"`
	if got := normalizeWindowsQuotes(rel); strings.Contains(got, `"rel`) {
		t.Fatalf("quoted relative path remains: %s", got)
	}
}

// TestPrepareShellCommandRelaxesInlinePath 确认交给 cmd 的内联脚本路径不再带着引号，bash 命令仍保留引号。
func TestPrepareShellCommandRelaxesInlinePath(t *testing.T) {
	ws := t.TempDir()
	got, done := prepareShellCommand(`python fetch.py && python -c "print(1)"`, ws)
	defer done()
	if !strings.Contains(got, "yoyo-inline-") || !strings.Contains(got, "fetch.py") {
		t.Fatalf("rewrite dropped the script or the first stage: %s", got)
	}
	if strings.Contains(got, `\"`) {
		t.Fatalf("escaped quote remains: %s", got)
	}
	if runtime.GOOS == "windows" && strings.Contains(got, `"`) {
		t.Fatalf("cmd path still quoted: %s", got)
	}
	bash, bashDone := prepareShellCommand(`python -c "print(1)" 2>/dev/null`, ws)
	defer bashDone()
	if runtime.GOOS == "windows" && !strings.Contains(bash, `"`) {
		t.Fatalf("bash path lost its quotes: %s", bash)
	}
}

// TestInlineScriptPathSurvivesRewrite 确认改写后的脚本在 workspace/.yoyo/tmp，含反斜杠的路径以引号开头。
func TestInlineScriptPathSurvivesRewrite(t *testing.T) {
	ws := t.TempDir()
	out, done := rewriteInlineInterpreters(`node -e "console.log(1)"`, ws)
	defer done()
	if !strings.HasPrefix(out, "node ") {
		t.Fatalf("rewrite changed the interpreter: %s", out)
	}
	arg := strings.TrimPrefix(out, "node ")
	if !strings.Contains(arg, filepath.Join(ws, ".yoyo", "tmp")) {
		t.Fatalf("script path is not under the workspace temp dir: %s", arg)
	}
	if strings.Contains(arg, `\`) && !strings.HasPrefix(arg, `"`) {
		t.Fatalf("Windows script path must be quoted for bash: %s", arg)
	}
}
