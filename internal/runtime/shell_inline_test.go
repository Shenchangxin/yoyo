package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteInlineUnescapesOnlyWrappingQuotes(t *testing.T) {
	dir := t.TempDir()
	_, cleanup := rewriteInlineInterpreters(`python -c "print(\"hi\")"`, dir)
	defer cleanup()
	matches, err := filepath.Glob(filepath.Join(dir, ".yoyo", "tmp", "yoyo-inline-*.py"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("tmp scripts %v %v", matches, err)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if got != `print("hi")` {
		t.Fatalf("%q", got)
	}
}

func TestRewriteInlinePythonMaterializesDashC(t *testing.T) {
	dir := t.TempDir()
	cmd, cleanup := rewriteInlineInterpreters(`python -c "import glob,os;print(len(glob.glob('*.txt')))"`, dir)
	defer cleanup()
	if strings.Contains(cmd, " -c ") || strings.HasSuffix(cmd, " -c") {
		t.Fatalf("still -c: %s", cmd)
	}
	if !strings.Contains(cmd, ".py") {
		t.Fatalf("%s", cmd)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".yoyo", "tmp", "yoyo-inline-*.py"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("tmp scripts %v %v", matches, err)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "glob.glob") {
		t.Fatalf("%s", b)
	}
}

func TestRewriteInlinePythonKeepsOrChain(t *testing.T) {
	dir := t.TempDir()
	in := `python -c "import pypdf; print(pypdf.__version__)" 2>&1 || python -c "import fitz; print('ok')" 2>&1`
	cmd, cleanup := rewriteInlineInterpreters(in, dir)
	defer cleanup()
	if strings.Contains(strings.ToLower(cmd), "python -c") {
		t.Fatalf("%s", cmd)
	}
	if !strings.Contains(cmd, "||") {
		t.Fatalf("lost ||: %s", cmd)
	}
	if strings.Count(cmd, ".py") < 2 {
		t.Fatalf("expected two scripts: %s", cmd)
	}
}

func TestRewriteInlinePythonLeavesPlainScripts(t *testing.T) {
	in := `python tools/extract_pdf.py`
	cmd, cleanup := rewriteInlineInterpreters(in, t.TempDir())
	defer cleanup()
	if cmd != in {
		t.Fatalf("%s", cmd)
	}
}

func TestRewriteInlineNodeKeepsRegexEscapes(t *testing.T) {
	dir := t.TempDir()
	in := `node -e "const m=html.match(/<script>([\s\S]*?)<\/script>/);"`
	_, cleanup := rewriteInlineInterpreters(in, dir)
	defer cleanup()
	matches, err := filepath.Glob(filepath.Join(dir, ".yoyo", "tmp", "yoyo-inline-*.js"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("tmp scripts %v %v", matches, err)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, `[\s\S]`) {
		t.Fatalf("lost \\s: %s", got)
	}
	if !strings.Contains(got, `<\/script>`) {
		t.Fatalf("lost \\/: %s", got)
	}
	if strings.Contains(got, `[sS]`) || strings.Contains(got, `</script>`) {
		t.Fatalf("over-unescaped regex: %s", got)
	}
}

func TestRewriteInlineNodeEval(t *testing.T) {
	dir := t.TempDir()
	cmd, cleanup := rewriteInlineInterpreters("node -e \"const fs=require('fs');console.log(fs.readFileSync('x','utf8').length)\"", dir)
	defer cleanup()
	low := strings.ToLower(cmd)
	if strings.Contains(low, "node -e") || strings.Contains(low, " -e ") {
		t.Fatalf("still -e: %s", cmd)
	}
	if !strings.Contains(cmd, ".js") {
		t.Fatalf("%s", cmd)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".yoyo", "tmp", "yoyo-inline-*.js"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("tmp scripts %v %v", matches, err)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "readFileSync") {
		t.Fatalf("%s", b)
	}
}

func TestRewriteInlinePythonLauncherVersion(t *testing.T) {
	dir := t.TempDir()
	cmd, cleanup := rewriteInlineInterpreters(`py -3 -c "print(1)"`, dir)
	defer cleanup()
	if strings.Contains(strings.ToLower(cmd), " -c ") {
		t.Fatalf("still -c: %s", cmd)
	}
	if !strings.Contains(cmd, ".py") {
		t.Fatalf("%s", cmd)
	}
}

func TestStripRedundantWorkspaceCd(t *testing.T) {
	ws := `C:\Users\scx\Desktop\test`
	in := `cd "C:\Users\scx\Desktop\test" && node -e "console.log(1)"`
	got := stripRedundantWorkspaceCd(in, ws)
	if got != `node -e "console.log(1)"` {
		t.Fatalf("%q", got)
	}
	tabbed := `cd "` + unescapeWinPathEscapes(ws) + `" && node .smoke.js`
	got = stripRedundantWorkspaceCd(tabbed, ws)
	if got != `node .smoke.js` {
		t.Fatalf("tab path: %q", got)
	}
	got = stripRedundantWorkspaceCd(`cd . && python app.py`, ws)
	if got != `python app.py` {
		t.Fatalf("%q", got)
	}
}

func TestRepairWorkspacePathEscapes(t *testing.T) {
	ws := `C:\Users\scx\Desktop\test`
	broken := unescapeWinPathEscapes(ws)
	if !strings.Contains(broken, "\t") {
		t.Fatalf("expected tab in %q", broken)
	}
	in := `cd "` + broken + `" && node -e "x"`
	got := repairWorkspacePathEscapes(in, ws)
	if strings.Contains(got, "\t") {
		t.Fatalf("tab remains: %q", got)
	}
	if !strings.Contains(got, ws) {
		t.Fatalf("%q", got)
	}
}

func TestRewriteInlineNodeAfterWorkspaceCd(t *testing.T) {
	dir := t.TempDir()
	ws := `C:\Users\scx\Desktop\test`
	in := `cd "` + ws + `" && node -e "const fs=require('fs');console.log(1)"`
	cmd := stripRedundantWorkspaceCd(in, ws)
	out, cleanup := rewriteInlineInterpreters(cmd, dir)
	defer cleanup()
	if strings.Contains(strings.ToLower(out), "node -e") {
		t.Fatalf("still -e: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "cd ") {
		t.Fatalf("cd remains: %s", out)
	}
}
