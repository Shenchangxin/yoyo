package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
