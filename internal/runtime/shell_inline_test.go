// 内联脚本物化到工作区临时文件。只解开外层引号。指向工作区的 cd 被剥掉。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRewriteInlineUnescapesOnlyWrappingQuotes 确认只解开外层引号，使脚本里是 print("hi")。
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

// TestRewriteInlinePythonMaterializesDashC 确认 python -c 落成 .py，命令改成跑该文件。
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

// TestRewriteInlinePythonKeepsOrChain 确认 || 链上的两段脚本都物化。
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

// TestRewriteInlinePythonLeavesPlainScripts 确认普通 .py 路径不被改写。
func TestRewriteInlinePythonLeavesPlainScripts(t *testing.T) {
	in := `python tools/extract_pdf.py`
	cmd, cleanup := rewriteInlineInterpreters(in, t.TempDir())
	defer cleanup()
	if cmd != in {
		t.Fatalf("%s", cmd)
	}
}

// TestRewriteInlineNodeKeepsRegexEscapes 确认 [\s\S] 和 <\/script> 保留，不收成 [sS] 或 </script>。
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

// TestRewriteInlineNodeEval 确认 node -e 落成 .js。
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

// TestRewriteInlinePythonLauncherVersion 确认 py -3 -c 去掉版本和 -c。
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

// TestStripRedundantWorkspaceCd 确认 cd 到工作区、被制表符弄坏的工作区路径，以及 cd . 被剥掉。
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

// TestRepairWorkspacePathEscapes 确认制表符被换回原始工作区路径。
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

// TestRewriteInlineNodeAfterWorkspaceCd 确认先剥掉指向工作区的 cd，再物化 node -e。
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

// TestRelaxRedundantShellQuotes 确认去掉无空格相对路径的引号，保留带空格的引号。
func TestRelaxRedundantShellQuotes(t *testing.T) {
	in := `node "game-adaptations\xiyou-tianming-guiling\qa\verify.mjs" && echo "ok done"`
	got := relaxRedundantShellQuotes(in)
	if strings.Contains(got, `"game-adaptations`) {
		t.Fatalf("quoted relative path remains: %s", got)
	}
	if !strings.Contains(got, `node game-adaptations\xiyou-tianming-guiling\qa\verify.mjs`) {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(got, `"ok done"`) {
		t.Fatalf("quoted phrase with space must stay: %s", got)
	}
}
