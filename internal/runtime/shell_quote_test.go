package runtime

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// End to end through the shell tool: an inline `node -e` is rewritten to a
// temp script under the workspace, and the command runs under Git Bash. Before
// the fix, the unquoted path lost its backslashes and node could not find it.
// The /dev/null makes the command posix, which is what sends it to bash.
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

// Git Bash reads an unquoted backslash as an escape. Quotes must survive any
// command that will run under bash, or the path loses its separators.
func TestNormalizeWindowsQuotesKeepsQuotesForBash(t *testing.T) {
	bash := `node "C:\Users\scx\qa\run.js" > /dev/null`
	if got := normalizeWindowsQuotes(bash); got != bash {
		t.Fatalf("bash command lost its quotes:\n got %s\nwant %s", got, bash)
	}
}

func TestNormalizeWindowsQuotesRelaxesForCmd(t *testing.T) {
	in := `node "C:\Users\scx\qa\run.js"`
	want := `node C:\Users\scx\qa\run.js`
	if got := normalizeWindowsQuotes(in); got != want {
		t.Fatalf("cmd command: got %s want %s", got, want)
	}
	// The original cmd fix still applies: a quoted relative path is unquoted.
	rel := `node "rel\file.mjs"`
	if got := normalizeWindowsQuotes(rel); strings.Contains(got, `"rel`) {
		t.Fatalf("quoted relative path remains: %s", got)
	}
}

// The inline interpreter rewrite must leave the temp script path readable by
// the shell. On Windows the path has backslashes, so it must be quoted.
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
